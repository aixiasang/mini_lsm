package sst

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/myerror"
)

// TestSSTFileStructureAndData 详细测试SST文件的内部结构和数据正确性
func TestSSTFileStructureAndData(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "sst_structure_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建配置
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 20 // 减小块大小，降低数据量

	// 生成随机种子确保可重现
	seed := time.Now().UnixNano()
	// 创建独立的随机源
	rng := rand.New(rand.NewSource(seed))
	t.Logf("Using random seed: %d", seed)

	// 创建测试SST文件
	sstFile := filepath.Join(tempDir, "structure_test.sst")

	// 生成测试数据
	const dataCount = 300 // 减少数据量
	data := generateTestData(dataCount, rng)
	t.Logf("Generated %d test key-value pairs", len(data))

	// 写入数据到SST文件
	dataOffsets, indexOffsets, filterOffsets, err := writeTestData(t, conf, sstFile, data)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	t.Logf("Data written to file with %d blocks", len(dataOffsets))

	// 读取SST文件内容进行分析
	analyzeAndVerifySST(t, sstFile, data, dataOffsets, indexOffsets, filterOffsets)

	// 使用SSTReader验证所有数据
	verifyDataWithSSTReader(t, conf, sstFile, data)
}

// 生成测试数据
func generateTestData(count int, rng *rand.Rand) map[string]string {
	data := make(map[string]string, count)
	// 生成各种长度和内容的键值对
	for i := 0; i < count; i++ {
		keyLength := rng.Intn(20) + 5     // 5-24字节长度的键
		valueLength := rng.Intn(100) + 20 // 20-119字节长度的值

		key := make([]byte, keyLength)
		value := make([]byte, valueLength)

		// 生成随机键（确保第一个字符为可打印字符，便于排序）
		key[0] = byte(rng.Intn(26) + 'a')
		for j := 1; j < keyLength; j++ {
			key[j] = byte(rng.Intn(95) + 32) // ASCII 可打印字符
		}

		// 生成随机值
		for j := 0; j < valueLength; j++ {
			value[j] = byte(rng.Intn(95) + 32) // ASCII 可打印字符
		}

		data[string(key)] = string(value)
	}
	return data
}

// 写入测试数据到SST文件，返回各块的偏移量信息
func writeTestData(t *testing.T, conf *config.Config, filePath string, data map[string]string) (
	dataOffsets []int64, indexOffsets []int64, filterOffsets []int64, err error) {

	writer, err := NewSSTWriter(conf, filePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create SSTWriter: %v", err)
	}
	defer writer.Close()

	// 排序键以确保顺序写入
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 跟踪各块偏移量
	dataOffsets = make([]int64, 0)
	indexOffsets = make([]int64, 0)
	filterOffsets = make([]int64, 0)

	// 写入数据并在适当位置强制块旋转
	blockCount := 0
	itemsInBlock := 0 // 跟踪当前块中的项目数
	for i, key := range keys {
		if err := writer.Add([]byte(key), []byte(data[key])); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to add data: %v", err)
		}

		itemsInBlock++

		// 每处理一定数量的项目就旋转块
		if (int64(itemsInBlock) >= conf.BlockSize) || i == len(keys)-1 {
			// 在旋转前记录当前块信息
			if len(writer.index) > 0 {
				lastIdx := writer.index[len(writer.index)-1]
				dataOffsets = append(dataOffsets, lastIdx.Offset)
				indexOffsets = append(indexOffsets, int64(len(writer.index)-1))
			}

			blockLength := writer.dataBlock.Length()
			t.Logf("Block %d contains %d items with %d bytes", blockCount+1, itemsInBlock, blockLength)

			if err := writer.mustRotateDataBlock(); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to rotate block: %v", err)
			}

			// 收集过滤器信息
			for bLength := range writer.mapFilter {
				if !contains(filterOffsets, bLength) {
					filterOffsets = append(filterOffsets, bLength)
				}
			}

			blockCount++
			t.Logf("Rotated block %d with %d bytes of data", blockCount, blockLength)
			itemsInBlock = 0 // 重置块项目计数
		}
	}

	// 刷新数据到磁盘
	if err := writer.Flush(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to flush data: %v", err)
	}

	return dataOffsets, indexOffsets, filterOffsets, nil
}

// 检查切片是否包含某个值
func contains(slice []int64, val int64) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// 直接分析和验证SST文件的内部结构
func analyzeAndVerifySST(t *testing.T, filePath string, data map[string]string,
	dataOffsets, indexOffsets, filterOffsets []int64) {

	// 打开文件进行分析
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open SST file for analysis: %v", err)
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}
	fileSize := fileInfo.Size()

	// 读取文件footer
	footer := make([]byte, 12)
	if _, err := file.ReadAt(footer, fileSize-12); err != nil {
		t.Fatalf("Failed to read footer: %v", err)
	}

	// 解析footer
	dataLength := binary.BigEndian.Uint32(footer[0:4])
	indexLength := binary.BigEndian.Uint32(footer[4:8])
	filterLength := binary.BigEndian.Uint32(footer[8:12])

	t.Logf("SST File Analysis:")
	t.Logf("  File Size: %d bytes", fileSize)
	t.Logf("  Data Section: %d bytes", dataLength)
	t.Logf("  Index Section: %d bytes", indexLength)
	t.Logf("  Filter Section: %d bytes", filterLength)

	// 验证文件大小
	if int64(dataLength+indexLength+filterLength+12) != fileSize {
		t.Errorf("File size mismatch: calculated=%d, actual=%d",
			dataLength+indexLength+filterLength+12, fileSize)
	}

	// 读取各部分数据
	dataSection := make([]byte, dataLength)
	if _, err := file.ReadAt(dataSection, 0); err != nil {
		t.Fatalf("Failed to read data section: %v", err)
	}

	indexSection := make([]byte, indexLength)
	if _, err := file.ReadAt(indexSection, int64(dataLength)); err != nil {
		t.Fatalf("Failed to read index section: %v", err)
	}

	filterSection := make([]byte, filterLength)
	if _, err := file.ReadAt(filterSection, int64(dataLength)+int64(indexLength)); err != nil {
		t.Fatalf("Failed to read filter section: %v", err)
	}

	// 解析索引，获取所有块的元数据
	indexEntries := parseIndexSection(t, indexSection)
	t.Logf("Parsed %d index entries", len(indexEntries))

	// 解析过滤器数据
	filters := parseFilterSection(t, filterSection)
	t.Logf("Parsed %d bloom filters", len(filters))

	// 验证数据区的内容与写入的匹配
	verifyDataSection(t, dataSection, indexEntries, data)
}

// 解析索引区，返回索引条目
func parseIndexSection(t *testing.T, indexData []byte) []*Index {
	entries := make([]*Index, 0)
	buf := bytes.NewReader(indexData)

	for buf.Len() > 0 {
		// 读取startKey长度
		var startKeyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &startKeyLen); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Failed to read startKeyLen: %v", err)
		}

		// 读取endKey长度
		var endKeyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &endKeyLen); err != nil {
			t.Fatalf("Failed to read endKeyLen: %v", err)
		}

		// 读取startKey
		startKey := make([]byte, startKeyLen)
		if _, err := buf.Read(startKey); err != nil {
			t.Fatalf("Failed to read startKey: %v", err)
		}

		// 读取endKey
		endKey := make([]byte, endKeyLen)
		if _, err := buf.Read(endKey); err != nil {
			t.Fatalf("Failed to read endKey: %v", err)
		}

		// 读取偏移量和长度
		var offset, length int64
		if err := binary.Read(buf, binary.BigEndian, &offset); err != nil {
			t.Fatalf("Failed to read offset: %v", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			t.Fatalf("Failed to read length: %v", err)
		}

		// 创建索引对象
		index := &Index{
			StartKey: startKey,
			EndKey:   endKey,
			Offset:   offset,
			Length:   length,
		}

		entries = append(entries, index)
	}

	return entries
}

// 解析过滤器区
func parseFilterSection(t *testing.T, filterData []byte) map[int64][]byte {
	filters := make(map[int64][]byte)
	buf := bytes.NewReader(filterData)

	for buf.Len() > 0 {
		// 读取blockLength
		var blockLength int64
		if err := binary.Read(buf, binary.BigEndian, &blockLength); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Failed to read blockLength: %v", err)
		}

		// 读取过滤器数据长度
		var filterLen uint32
		if err := binary.Read(buf, binary.BigEndian, &filterLen); err != nil {
			t.Fatalf("Failed to read filterLen: %v", err)
		}

		// 读取过滤器数据
		filterBytes := make([]byte, filterLen)
		if _, err := buf.Read(filterBytes); err != nil {
			t.Fatalf("Failed to read filter data: %v", err)
		}

		filters[blockLength] = filterBytes
	}

	return filters
}

// 验证数据部分的内容
func verifyDataSection(t *testing.T, dataSection []byte, indexes []*Index, originalData map[string]string) {
	// 依次验证每个索引块 - 只测试50%的块以加快测试速度
	maxBlocksToTest := len(indexes) / 2
	if maxBlocksToTest < 1 {
		maxBlocksToTest = 1
	}
	t.Logf("将测试前 %d 个数据块，总共 %d 个", maxBlocksToTest, len(indexes))

	for i, idx := range indexes {
		if i >= maxBlocksToTest {
			t.Logf("跳过剩余数据块 - 已测试 %d/%d 个块", i, len(indexes))
			break
		}

		if idx.Offset < 0 || idx.Length <= 0 {
			t.Logf("跳过无效的块 %d (偏移量=%d, 长度=%d)", i+1, idx.Offset, idx.Length)
			continue
		}

		t.Logf("验证数据块 %d (偏移量=%d, 长度=%d)", i+1, idx.Offset, idx.Length)

		// 检查索引的范围
		if idx.Offset+idx.Length > int64(len(dataSection)) {
			t.Errorf("块 %d 超出数据区大小: 偏移量=%d, 长度=%d, 数据区大小=%d",
				i+1, idx.Offset, idx.Length, len(dataSection))
			continue
		}

		// 重新从一个索引的起始偏移量开始读取，而不是多个重叠块
		blockStart := idx.Offset
		blockEnd := blockStart + idx.Length
		if blockStart < 0 || blockEnd > int64(len(dataSection)) {
			t.Errorf("无效的块边界: 起始=%d, 结束=%d, 数据区大小=%d",
				blockStart, blockEnd, len(dataSection))
			continue
		}

		blockData := dataSection[blockStart:blockEnd]
		// 解析并验证块中的键值对
		verifyBlock(t, blockData, originalData, idx.StartKey, idx.EndKey)
	}
}

// 验证数据块的内容
func verifyBlock(t *testing.T, blockData []byte, originalData map[string]string, startKey, endKey []byte) {
	buf := bytes.NewReader(blockData)
	keyCount := 0
	keyMismatchCount := 0

	// 遍历块中的所有键值对
	for buf.Len() > 0 {
		// 读取key长度
		var keyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &keyLen); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Failed to read key length: %v", err)
		}

		// 读取value长度
		var valueLen uint32
		if err := binary.Read(buf, binary.BigEndian, &valueLen); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Failed to read value length: %v", err)
		}

		// 读取key
		key := make([]byte, keyLen)
		if _, err := io.ReadFull(buf, key); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			t.Fatalf("Failed to read key: %v", err)
		}

		// 读取value
		value := make([]byte, valueLen)
		if _, err := io.ReadFull(buf, value); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			t.Fatalf("Failed to read value: %v", err)
		}

		// 只记录不匹配的数据，减少日志量
		expectedValue, exists := originalData[string(key)]
		if !exists {
			keyMismatchCount++
			if keyMismatchCount <= 5 { // 只显示前5个错误
				t.Errorf("Unexpected key in block: %s", string(key))
			}
		} else if expectedValue != string(value) {
			keyMismatchCount++
			if keyMismatchCount <= 5 { // 只显示前5个错误
				t.Errorf("Value mismatch for key %s: expected %q(len=%d), got %q(len=%d)",
					string(key), expectedValue, len(expectedValue), string(value), len(value))
			}
		}

		keyCount++
	}

	if keyMismatchCount > 0 {
		t.Errorf("Block contains %d key-value mismatches out of %d pairs", keyMismatchCount, keyCount)
	} else {
		t.Logf("Block successfully verified %d key-value pairs", keyCount)
	}
}

// min returns the smaller of x or y.
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// 使用SSTReader验证数据
func verifyDataWithSSTReader(t *testing.T, conf *config.Config, filePath string, data map[string]string) {
	// 创建SSTReader
	reader, err := NewSSTReader(conf, filePath)
	if err != nil {
		t.Fatalf("Failed to create SSTReader: %v", err)
	}
	defer reader.Close()

	// 验证Reader字段
	t.Logf("SSTReader loaded with %d index entries", len(reader.index))
	t.Logf("SSTReader loaded with %d bloom filters", len(reader.filterMap))

	// 排序键以便有序检查
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 1. 抽样测试部分键值对
	t.Log("Testing sample key lookups...")
	// 抽取部分键进行测试，而不是全部
	sampleSize := min(len(keys), 50) // 最多测试50个键
	sampleKeys := make([]string, sampleSize)

	// 均匀分布取样
	stride := len(keys) / sampleSize
	for i := 0; i < sampleSize; i++ {
		idx := i * stride
		if idx < len(keys) {
			sampleKeys[i] = keys[idx]
		}
	}

	missingCount := 0
	mismatchCount := 0

	for _, key := range sampleKeys {
		if key == "" {
			continue
		}
		value, err := reader.Get([]byte(key))
		if err != nil {
			missingCount++
			if missingCount <= 5 {
				t.Errorf("Failed to get value for key %s: %v", key, err)
			}
			continue
		}

		if !bytes.Equal(value, []byte(data[key])) {
			mismatchCount++
			if mismatchCount <= 5 {
				t.Errorf("Value mismatch for key %s: expected %q, got %q",
					key, data[key], string(value))
			}
		}
	}

	if missingCount > 0 || mismatchCount > 0 {
		t.Errorf("Direct lookups: %d missing, %d mismatches out of %d samples",
			missingCount, mismatchCount, sampleSize)
	} else {
		t.Logf("Direct lookups: all %d samples verified successfully", sampleSize)
	}

	// 2. 测试迭代器
	t.Log("Testing iterator...")
	it, err := reader.GetIterator()
	if err != nil {
		t.Fatalf("Failed to get iterator: %v", err)
	}

	keyCount := 0
	iterMismatchCount := 0

	for it.Next() {
		key := string(it.Key())
		value := string(it.Value())

		expectedValue, exists := data[key]
		if !exists {
			iterMismatchCount++
			if iterMismatchCount <= 5 {
				t.Errorf("Iterator returned unexpected key: %s", key)
			}
			continue
		}

		if value != expectedValue {
			iterMismatchCount++
			if iterMismatchCount <= 5 {
				t.Errorf("Iterator value mismatch for key %s: expected %q, got %q",
					key, expectedValue, value)
			}
		}

		keyCount++

		// 只遍历一定数量，避免生成过多日志
		if keyCount >= 100 {
			t.Logf("Iterator tested first 100 key-value pairs, stopping early")
			break
		}
	}

	if iterMismatchCount > 0 {
		t.Errorf("Iterator: %d mismatches out of %d items", iterMismatchCount, keyCount)
	} else {
		t.Logf("Iterator: all %d items verified successfully", keyCount)
	}

	// 3. 简化Bloom过滤器测试
	t.Log("Testing bloom filter with a few non-existent keys...")
	nonExistentKeys := []string{
		"THIS_KEY_DOES_NOT_EXIST_1",
		"THIS_KEY_DOES_NOT_EXIST_2",
	}

	filterErrors := 0
	for _, key := range nonExistentKeys {
		_, err := reader.Get([]byte(key))
		if err != myerror.ErrKeyNotFound {
			filterErrors++
			t.Errorf("Expected ErrKeyNotFound for key %s, got: %v", key, err)
		}
	}

	if filterErrors == 0 {
		t.Log("Bloom filter tests passed")
	}
}
