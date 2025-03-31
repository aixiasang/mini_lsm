package sst

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/myerror"
)

func TestSSTReaderBasic(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sst_reader_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 5 // Small block size for testing

	// Create a test SST file using SSTWriter
	sstFile := filepath.Join(tempDir, "reader_test.sst")
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTWriter: %v", err)
	}

	// Add test data
	testData := map[string]string{
		"key1":   "value1",
		"key2":   "value2",
		"key3":   "value3",
		"key100": "value100",
		"key200": "value200",
	}

	for k, v := range testData {
		if err := writer.Add([]byte(k), []byte(v)); err != nil {
			t.Fatalf("Failed to add key-value pair to writer: %v", err)
		}
	}

	// Ensure data blocks are rotated
	if err := writer.mustRotateDataBlock(); err != nil {
		t.Fatalf("Failed to rotate data block: %v", err)
	}

	// Add more data to create multiple blocks
	for i := 0; i < 10; i++ {
		key := []byte(string(rune('a' + i)))
		value := []byte(string(rune('A' + i)))
		if err := writer.Add(key, value); err != nil {
			t.Fatalf("Failed to add additional key-value pair: %v", err)
		}
	}

	// Flush the writer to disk
	if err := writer.Flush(); err != nil {
		t.Fatalf("Failed to flush writer: %v", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Now create an SSTReader to read the file
	reader, err := NewSSTReader(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTReader: %v", err)
	}
	defer reader.Close()

	// Verify that we can read back the data
	for k, v := range testData {
		value, err := reader.SlowGet([]byte(k))
		if err != nil {
			t.Errorf("Failed to get value for key %s: %v", k, err)
			continue
		}

		if !bytes.Equal(value, []byte(v)) {
			t.Errorf("Value mismatch for key %s: expected %s, got %s", k, v, string(value))
		}
	}

	// Test a non-existent key
	_, err = reader.SlowGet([]byte("non_existent_key"))
	if err != myerror.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound for non-existent key, got: %v", err)
	}

	// Also verify the single-character keys we added
	for i := 0; i < 10; i++ {
		key := []byte(string(rune('a' + i)))
		expectedValue := []byte(string(rune('A' + i)))

		value, err := reader.SlowGet(key)
		if err != nil {
			t.Errorf("Failed to get value for key %s: %v", string(key), err)
			continue
		}

		if !bytes.Equal(value, expectedValue) {
			t.Errorf("Value mismatch for key %s: expected %s, got %s",
				string(key), string(expectedValue), string(value))
		}
	}
	kvList := reader.KvList()

	t.Logf("kvList: %v", kvList)
	for _, kv := range kvList {
		t.Logf("key: %s, value: %s", string(kv.Key), string(kv.Value))
	}
}

func TestSSTReaderIterator(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sst_reader_iterator_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 3 // Small block size to force multiple blocks

	// Create a test SST file
	sstFile := filepath.Join(tempDir, "iterator_test.sst")
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTWriter: %v", err)
	}

	// Add ordered data for testing the iterator
	orderedData := []struct {
		key   string
		value string
	}{
		{"a", "A"},
		{"b", "B"},
		{"c", "C"},
		{"d", "D"},
		{"e", "E"},
		{"f", "F"},
		{"g", "G"},
		{"h", "H"},
		{"i", "I"},
	}

	// Add the data to the writer
	for _, kv := range orderedData {
		if err := writer.Add([]byte(kv.key), []byte(kv.value)); err != nil {
			t.Fatalf("Failed to add key-value pair to writer: %v", err)
		}
	}

	// Flush and close
	if err := writer.Flush(); err != nil {
		t.Fatalf("Failed to flush writer: %v", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Create a reader
	reader, err := NewSSTReader(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTReader: %v", err)
	}
	defer reader.Close()

	// Get an iterator
	it, err := reader.GetIterator()
	if err != nil {
		t.Fatalf("Failed to get iterator: %v", err)
	}

	// Iterate over all key-value pairs
	index := 0
	for it.Next() {
		if index >= len(orderedData) {
			t.Fatalf("Iterator returned more data than expected")
		}

		key := it.Key()
		value := it.Value()

		expectedKey := []byte(orderedData[index].key)
		expectedValue := []byte(orderedData[index].value)

		if !bytes.Equal(key, expectedKey) {
			t.Errorf("Iterator key mismatch at index %d: expected %s, got %s",
				index, string(expectedKey), string(key))
		}

		if !bytes.Equal(value, expectedValue) {
			t.Errorf("Iterator value mismatch at index %d: expected %s, got %s",
				index, string(expectedValue), string(value))
		}

		index++
	}

	// Check for any errors during iteration
	if err := it.Error(); err != nil {
		t.Errorf("Iterator error: %v", err)
	}

	// Ensure we read all the data
	if index != len(orderedData) {
		t.Errorf("Iterator only returned %d items, expected %d", index, len(orderedData))
	}
}

func TestSSTReaderBloomFilter(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sst_reader_bloom_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 4 // Small block size to test filters

	// Create a test SST file
	sstFile := filepath.Join(tempDir, "bloom_test.sst")
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTWriter: %v", err)
	}

	// Add specific test data that will be added to the bloom filter
	bloomTestData := []struct {
		key   string
		value string
	}{
		{"bloom1", "value1"},
		{"bloom2", "value2"},
		{"bloom3", "value3"},
		{"bloom4", "value4"},
		// Force a block rotation
		{"bloom5", "value5"},
		{"bloom6", "value6"},
		{"bloom7", "value7"},
		{"bloom8", "value8"},
	}

	// Add data to writer
	for _, kv := range bloomTestData {
		if err := writer.Add([]byte(kv.key), []byte(kv.value)); err != nil {
			t.Fatalf("Failed to add key-value pair to writer: %v", err)
		}
	}

	// Force another rotation to make sure the filter data is captured
	if err := writer.mustRotateDataBlock(); err != nil {
		t.Fatalf("Failed to rotate data block: %v", err)
	}

	// Flush and close
	if err := writer.Flush(); err != nil {
		t.Fatalf("Failed to flush writer: %v", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Create a reader
	reader, err := NewSSTReader(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTReader: %v", err)
	}
	defer reader.Close()

	// Test successful lookups (should match bloom filter)
	for _, kv := range bloomTestData {
		value, err := reader.Get([]byte(kv.key))
		if err != nil {
			t.Errorf("Failed to get value for key %s: %v", kv.key, err)
			continue
		}

		if !bytes.Equal(value, []byte(kv.value)) {
			t.Errorf("Value mismatch for key %s: expected %s, got %s",
				kv.key, kv.value, string(value))
		}
	}

	// Test keys that definitely don't exist
	// These should be filtered out by the bloom filter
	nonExistentKeys := []string{
		"nonexistent1",
		"nonexistent2",
		"nonexistent3",
		"xyz123",
	}

	for _, key := range nonExistentKeys {
		_, err := reader.Get([]byte(key))
		if err != myerror.ErrKeyNotFound {
			t.Errorf("Expected ErrKeyNotFound for key %s, got: %v", key, err)
		}
	}
}

// TestSSTReaderInvalidFile tests the reader's behavior with invalid files
func TestSSTReaderInvalidFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sst_reader_invalid_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	conf := config.DefaultConfig()
	conf.DataDir = tempDir

	// Test case 1: Non-existent file
	_, err = NewSSTReader(conf, filepath.Join(tempDir, "nonexistent.sst"))
	if err == nil {
		t.Error("Expected an error when opening non-existent file, but got nil")
	}

	// Test case 2: Empty file (too small to be valid)
	emptyFile := filepath.Join(tempDir, "empty.sst")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	_, err = NewSSTReader(conf, emptyFile)
	if err != myerror.ErrInvalidSSTFormat {
		t.Errorf("Expected ErrInvalidSSTFormat for empty file, got: %v", err)
	}

	// Test case 3: File with invalid format (footer doesn't match content)
	invalidFile := filepath.Join(tempDir, "invalid.sst")
	// Create some invalid data with footer values that don't match the file size
	invalidData := []byte{
		// Some content
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		// Invalid footer (data length = 100, index length = 200, filter length = 300)
		0, 0, 0, 100, // Data length
		0, 0, 0, 200, // Index length
		0, 0, 1, 44, // Filter length
	}
	if err := os.WriteFile(invalidFile, invalidData, 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	_, err = NewSSTReader(conf, invalidFile)
	if err != myerror.ErrInvalidSSTFormat {
		t.Errorf("Expected ErrInvalidSSTFormat for invalid footer, got: %v", err)
	}
}

// TestSSTReaderLargeDataset 测试SSTReader处理大量数据的能力
func TestSSTReaderLargeDataset(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "sst_reader_large_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建配置
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 20 // 较小的块大小以确保创建多个块

	// 创建测试SST文件
	sstFile := filepath.Join(tempDir, "large_dataset.sst")
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTWriter: %v", err)
	}

	// 生成至少100条测试数据
	const dataCount = 150
	testData := make(map[string]string, dataCount)
	expectedKeys := make([]string, 0, dataCount)

	// 添加数字键数据
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("num_key_%04d", i)
		value := fmt.Sprintf("num_value_%04d", i)
		testData[key] = value
		expectedKeys = append(expectedKeys, key)
	}

	// 添加字母键数据
	for i := 0; i < 50; i++ {
		// 生成'a'到'z'的字符
		char := rune('a' + (i % 26))
		repeat := i/26 + 1
		key := strings.Repeat(string(char), repeat) // 如 "a", "b", ..., "aa", "bb", ...
		value := fmt.Sprintf("alpha_value_%s", key)
		testData[key] = value
		expectedKeys = append(expectedKeys, key)
	}

	// 排序键，确保有序写入
	sort.Strings(expectedKeys)

	// 写入所有测试数据
	for i, key := range expectedKeys {
		if err := writer.Add([]byte(key), []byte(testData[key])); err != nil {
			t.Fatalf("Failed to add key-value pair to writer: %v", err)
		}

		// 每写入10条数据就强制旋转数据块，以创建多个数据块和过滤器
		if i > 0 && i%10 == 0 {
			if err := writer.mustRotateDataBlock(); err != nil {
				t.Fatalf("Failed to rotate data block: %v", err)
			}
		}
	}

	// 刷新并关闭
	if err := writer.Flush(); err != nil {
		t.Fatalf("Failed to flush writer: %v", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// 计算文件大小
	fileInfo, err := os.Stat(sstFile)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}
	fileSize := fileInfo.Size()
	t.Logf("SST文件大小: %d 字节", fileSize)

	// 创建SSTReader读取文件
	reader, err := NewSSTReader(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTReader: %v", err)
	}
	defer reader.Close()

	// 测试1：直接读取所有键值对
	t.Log("Testing direct key lookups...")
	for _, key := range expectedKeys {
		value, err := reader.Get([]byte(key))
		if err != nil {
			t.Errorf("Failed to get value for key %s: %v", key, err)
			continue
		}

		if !bytes.Equal(value, []byte(testData[key])) {
			t.Errorf("Value mismatch for key %s: expected %s, got %s",
				key, testData[key], string(value))
		}
	}

	// 测试2：使用迭代器遍历所有键值对
	t.Log("Testing iterator...")
	it, err := reader.GetIterator()
	if err != nil {
		t.Fatalf("Failed to get iterator: %v", err)
	}

	keyCount := 0
	for it.Next() {
		key := string(it.Key())
		value := string(it.Value())

		expectedValue, exists := testData[key]
		if !exists {
			t.Errorf("Iterator returned unexpected key: %s", key)
			continue
		}

		if value != expectedValue {
			t.Errorf("Iterator value mismatch for key %s: expected %s, got %s",
				key, expectedValue, value)
		}

		keyCount++
	}

	// 确保遍历了所有期望的键
	if keyCount != len(expectedKeys) {
		t.Errorf("Iterator only returned %d items, expected %d", keyCount, len(expectedKeys))
	}

	// 测试3：验证非存在键的查询
	t.Log("Testing non-existent keys...")
	nonExistentKeys := []string{
		"nonexistent_key_1",
		"nonexistent_key_2",
		"zzzzzzzzzzzzzz",
	}

	for _, key := range nonExistentKeys {
		_, err := reader.Get([]byte(key))
		if err != myerror.ErrKeyNotFound {
			t.Errorf("Expected ErrKeyNotFound for key %s, got: %v", key, err)
		}
	}

	// 报告索引和过滤器的统计信息
	t.Logf("SST Reader loaded with %d index entries", len(reader.index))
	t.Logf("SST Reader loaded with %d bloom filters", len(reader.filterMap))
	t.Logf("Total data verified: %d key-value pairs", len(expectedKeys))

	// 修复文件大小统计
	t.Logf("\n====== 大规模数据测试结果汇总 ======")
	t.Logf("数据总量: %d 条", len(expectedKeys))
	t.Logf("SST文件大小: %d 字节", fileInfo.Size())

	// 计算内部各段大小总和
	segmentSize := int64(0)
	segmentSize += int64(reader.dataLength)
	segmentSize += int64(reader.indexLength)
	segmentSize += int64(reader.filterLength)
	segmentSize += 12 // footer size

	t.Logf("数据段大小: %d 字节", reader.dataLength)
	t.Logf("索引段大小: %d 字节", reader.indexLength)
	t.Logf("过滤器段大小: %d 字节", reader.filterLength)
	t.Logf("总段大小(含footer): %d 字节", segmentSize)
	t.Logf("数据结构一致性: %v", segmentSize == fileInfo.Size())
	t.Logf("==================================\n")

	t.Logf("大规模数据测试成功完成，所有 %d 条数据处理无误!", len(expectedKeys))
}

// TestSSTReaderLargeScale 测试SSTReader处理大规模数据的能力(至少300条数据)
func TestSSTReaderLargeScale(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "sst_reader_large_scale_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建配置
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 25 // 设置较小的块大小以创建多个数据块

	// 创建测试SST文件
	sstFile := filepath.Join(tempDir, "large_scale_test.sst")

	// 创建写入器
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("创建SSTWriter失败: %v", err)
	}

	// 生成大规模测试数据(至少300条)
	const dataCount = 350
	testData := make(map[string]string, dataCount)
	expectedKeys := make([]string, 0, dataCount)

	// 1. 添加数字键数据 (200条)
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("num_key_%06d", i)
		value := fmt.Sprintf("num_value_%08d", i*i) // 使值更复杂
		testData[key] = value
		expectedKeys = append(expectedKeys, key)
	}

	// 2. 添加字母键数据 (50条)
	for i := 0; i < 50; i++ {
		// 生成'a'到'z'的字符
		char := rune('a' + (i % 26))
		repeat := i/26 + 1
		key := strings.Repeat(string(char), repeat) // 如 "a", "b", ..., "aa", "bb", ...
		value := fmt.Sprintf("alpha_value_%s_%d", key, i*3)
		testData[key] = value
		expectedKeys = append(expectedKeys, key)
	}

	// 3. 添加混合数据 (100条)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("mixed_%c%d_%s", rune('A'+i%26), i, strings.Repeat("*", i%5+1))
		value := fmt.Sprintf("mixed_value_%d_%s", i, strings.Repeat("@", i%7+1))
		testData[key] = value
		expectedKeys = append(expectedKeys, key)
	}

	// 对键进行排序，确保有序写入
	sort.Strings(expectedKeys)

	t.Logf("准备写入 %d 条数据...", len(expectedKeys))

	// 写入所有测试数据
	dataWritten := 0
	for i, key := range expectedKeys {
		if err := writer.Add([]byte(key), []byte(testData[key])); err != nil {
			t.Fatalf("添加键值对到writer失败: %v", err)
		}
		dataWritten++

		// 每写入15条数据就强制旋转数据块，以创建多个数据块和过滤器
		if i > 0 && i%15 == 0 {
			if err := writer.mustRotateDataBlock(); err != nil {
				t.Fatalf("旋转数据块失败: %v", err)
			}
		}
	}

	t.Logf("成功写入 %d 条数据", dataWritten)

	// 刷新并关闭
	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush writer失败: %v", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭writer失败: %v", err)
	}

	// 计算文件大小
	fileInfo, err := os.Stat(sstFile)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}
	fileSize := fileInfo.Size()
	t.Logf("SST文件大小: %d 字节", fileSize)

	// 创建SSTReader读取文件
	reader, err := NewSSTReader(conf, sstFile)
	if err != nil {
		t.Fatalf("创建SSTReader失败: %v", err)
	}
	defer reader.Close()

	// 统计和记录SST文件内部结构
	t.Logf("SST文件内部结构:")
	t.Logf("  - 数据段大小: %d 字节", reader.dataLength)
	t.Logf("  - 索引段大小: %d 字节", reader.indexLength)
	t.Logf("  - 过滤器段大小: %d 字节", reader.filterLength)
	t.Logf("  - 索引条目数量: %d", len(reader.index))
	t.Logf("  - 布隆过滤器数量: %d", len(reader.filterMap))

	// 测试1：直接读取所有键值对进行验证
	t.Log("开始验证所有键值对...")
	successCount := 0
	failCount := 0

	for _, key := range expectedKeys {
		value, err := reader.Get([]byte(key))
		if err != nil {
			t.Errorf("获取键 %s 的值失败: %v", key, err)
			failCount++
			continue
		}

		expectedValue := testData[key]
		if !bytes.Equal(value, []byte(expectedValue)) {
			t.Errorf("键 %s 的值不匹配: 期望 %s, 实际 %s",
				key, expectedValue, string(value))
			failCount++
		} else {
			successCount++
		}

		// 只打印部分验证信息，避免输出过多
		if successCount <= 5 || successCount%50 == 0 {
			t.Logf("键 %s 验证成功", key)
		}
	}

	t.Logf("验证完成: 成功 %d 条, 失败 %d 条", successCount, failCount)

	if failCount > 0 {
		t.Errorf("数据验证失败: 有 %d 条数据不匹配", failCount)
	}

	// 测试2：使用迭代器遍历所有键值对
	t.Log("使用迭代器验证数据...")
	it, err := reader.GetIterator()
	if err != nil {
		t.Fatalf("获取迭代器失败: %v", err)
	}

	itCount := 0
	itSuccess := 0
	itFailed := 0

	for it.Next() {
		itCount++
		key := string(it.Key())
		value := string(it.Value())

		expectedValue, exists := testData[key]
		if !exists {
			t.Errorf("迭代器返回了意外的键: %s", key)
			itFailed++
			continue
		}

		if value != expectedValue {
			t.Errorf("迭代器键 %s 的值不匹配: 期望 %s, 实际 %s",
				key, expectedValue, value)
			itFailed++
		} else {
			itSuccess++
		}
	}

	// 确保遍历了所有期望的键
	if itCount != len(expectedKeys) {
		t.Errorf("迭代器只返回了 %d 项，期望 %d 项", itCount, len(expectedKeys))
	}

	t.Logf("迭代器验证完成: 总计 %d 项, 成功 %d 项, 失败 %d 项",
		itCount, itSuccess, itFailed)

	// 测试3：验证非存在键的查询
	t.Log("测试不存在的键...")
	nonExistKeys := []string{
		"nonexistent_key_1",
		"nonexistent_key_2",
		"zzzzzzzzzzzzzz",
		// 添加更多边界情况的键
		"num_key_999999", // 不存在的数字键
		"mixed_ZZZ_999",  // 不存在的混合键
		"",               // 空键
		"混合中文键_abc_123",  // 包含中文的键
	}
	nonExistSuccess := 0
	for _, k := range nonExistKeys {
		_, err := reader.Get([]byte(k))
		if err == myerror.ErrKeyNotFound {
			t.Logf("成功识别不存在的键: %s", k)
			nonExistSuccess++
		} else {
			t.Errorf("应该返回键不存在错误，但得到: %v，键: %s", err, k)
		}
	}

	// 测试结果汇总
	t.Logf("\n====== 大规模数据测试结果汇总 ======")
	t.Logf("数据总量: %d 条", len(expectedKeys))
	t.Logf("SST文件大小: %d 字节", fileInfo.Size())

	// 计算内部各段大小总和
	segmentSize := int64(0)
	segmentSize += int64(reader.dataLength)
	segmentSize += int64(reader.indexLength)
	segmentSize += int64(reader.filterLength)
	segmentSize += 12 // footer size

	t.Logf("数据段大小: %d 字节", reader.dataLength)
	t.Logf("索引段大小: %d 字节", reader.indexLength)
	t.Logf("过滤器段大小: %d 字节", reader.filterLength)
	t.Logf("总段大小(含footer): %d 字节", segmentSize)
	t.Logf("数据结构一致性: %v", segmentSize == fileInfo.Size())
	t.Logf("直接查询成功率: %.2f%%", float64(successCount)/float64(len(expectedKeys))*100)
	t.Logf("迭代器成功率: %.2f%%", float64(itSuccess)/float64(len(expectedKeys))*100)
	t.Logf("不存在键识别成功率: %.2f%%", float64(nonExistSuccess)/float64(len(nonExistKeys))*100)
	t.Logf("==================================\n")

	t.Logf("大规模数据测试成功完成，所有 %d 条数据处理无误!", len(expectedKeys))
}
