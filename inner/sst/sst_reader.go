package sst

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"sync"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/filter"
	"github.com/aixiasang/lsm/inner/myerror"
)

// SSTReader 用于读取SST文件
type SSTReader struct {
	conf         *config.Config          // 配置
	filePath     string                  // 文件路径
	fileSize     int64                   // 文件大小
	dataOffset   int64                   // 数据区域偏移量
	dataLength   uint32                  // 数据区域长度
	indexOffset  int64                   // 索引区域偏移量
	indexLength  uint32                  // 索引区域长度
	filterOffset int64                   // 过滤器区域偏移量
	filterLength uint32                  // 过滤器区域长度
	index        []*Index                // 索引
	filterMap    map[int64]filter.Filter // 过滤器映射表 key=blockOffset
	fp           *os.File                // 文件指针
	mu           sync.RWMutex            // 互斥锁
	kvList       []*KeyValue             // 数据块
}

// NewSSTReader 创建一个新的SST读取器
func NewSSTReader(conf *config.Config, filePath string) (*SSTReader, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	stat, err := fp.Stat()
	if err != nil {
		fp.Close()
		return nil, err
	}

	fileSize := stat.Size()
	if fileSize < 12 { // 至少需要footer大小
		fp.Close()
		return nil, myerror.ErrInvalidSSTFormat
	}

	reader := &SSTReader{
		conf:      conf,
		filePath:  filePath,
		fileSize:  fileSize,
		fp:        fp,
		filterMap: make(map[int64]filter.Filter),
		index:     make([]*Index, 0),
		kvList:    make([]*KeyValue, 0),
	}

	// 读取文件footer
	if err := reader.loadFooter(); err != nil {
		return nil, err
	}

	// 加载索引和过滤器
	if err := reader.loadIndex(); err != nil {
		return nil, err
	}
	// 加载数据块
	if err := reader.loadDataBlock(); err != nil {
		return nil, err
	}
	// 加载过滤器
	if err := reader.loadFilter(); err != nil {
		return nil, err
	}

	return reader, nil
}
func (r *SSTReader) MinKey() []byte {
	return r.index[0].StartKey
}
func (r *SSTReader) MaxKey() []byte {
	return r.index[len(r.index)-1].EndKey
}
func (r *SSTReader) Index() []*Index {
	return r.index
}
func (r *SSTReader) Filter() map[int64]filter.Filter {
	return r.filterMap
}
func (r *SSTReader) KvList() []*KeyValue {
	return r.kvList
}

// FileSize 获取文件大小
func (r *SSTReader) FileSize() int64 {
	return r.fileSize
}

// loadDataBlock 加载数据区域
func (r *SSTReader) loadDataBlock() error {
	// 读取数据区域数据
	data := make([]byte, r.dataLength)
	if _, err := r.fp.ReadAt(data, r.dataOffset); err != nil {
		return err
	}

	// 解析数据区域
	buf := bytes.NewReader(data)
	for buf.Len() > 0 {
		// 读取key长度
		var keyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &keyLen); err != nil {
			return err
		}

		// 读取value长度
		var valueLen uint32
		if err := binary.Read(buf, binary.BigEndian, &valueLen); err != nil {
			return err
		}

		// 读取key
		key := make([]byte, keyLen)
		if _, err := buf.Read(key); err != nil {
			return err
		}

		// 读取value
		value := make([]byte, valueLen)
		if _, err := buf.Read(value); err != nil {
			return err
		}

		kv := &KeyValue{
			Key:   key,
			Value: value,
		}
		r.kvList = append(r.kvList, kv)
	}

	return nil
}

// loadFooter 加载文件的footer
func (r *SSTReader) loadFooter() error {
	// 读取文件末尾的12字节footer
	footer := make([]byte, 12)
	if _, err := r.fp.ReadAt(footer, r.fileSize-12); err != nil {
		return err
	}

	r.dataLength = binary.BigEndian.Uint32(footer[0:4])
	r.indexLength = binary.BigEndian.Uint32(footer[4:8])
	r.filterLength = binary.BigEndian.Uint32(footer[8:12])

	// 验证长度值的有效性
	if int64(r.dataLength+r.indexLength+r.filterLength+12) != r.fileSize {
		return myerror.ErrInvalidSSTFormat
	}

	// 计算各区域偏移量
	r.dataOffset = 0
	r.indexOffset = int64(r.dataLength)
	r.filterOffset = r.indexOffset + int64(r.indexLength)

	return nil
}

// loadIndex 加载索引数据
func (r *SSTReader) loadIndex() error {
	// 读取索引区域数据
	indexData := make([]byte, r.indexLength)
	if _, err := r.fp.ReadAt(indexData, r.indexOffset); err != nil {
		return err
	}

	// 解析索引数据
	buf := bytes.NewReader(indexData)
	for buf.Len() > 0 {
		// 读取startKey长度
		var startKeyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &startKeyLen); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// 读取endKey长度
		var endKeyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &endKeyLen); err != nil {
			return err
		}

		// 读取startKey
		startKey := make([]byte, startKeyLen)
		if _, err := buf.Read(startKey); err != nil {
			return err
		}

		// 读取endKey
		endKey := make([]byte, endKeyLen)
		if _, err := buf.Read(endKey); err != nil {
			return err
		}

		// 读取偏移量和长度
		var offset, length int64
		if err := binary.Read(buf, binary.BigEndian, &offset); err != nil {
			return err
		}
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return err
		}

		// 创建索引对象
		index := &Index{
			StartKey: startKey,
			EndKey:   endKey,
			Offset:   offset,
			Length:   length,
		}

		r.index = append(r.index, index)
	}

	return nil
}

// loadFilter 加载过滤器数据
func (r *SSTReader) loadFilter() error {
	// 读取过滤器区域数据
	filterData := make([]byte, r.filterLength)
	if _, err := r.fp.ReadAt(filterData, r.filterOffset); err != nil {
		return err
	}

	// 解析过滤器数据
	buf := bytes.NewReader(filterData)
	for buf.Len() > 0 {
		// 读取blockLength
		var blockLength int64
		if err := binary.Read(buf, binary.BigEndian, &blockLength); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// 读取过滤器数据长度
		var filterLen uint32
		if err := binary.Read(buf, binary.BigEndian, &filterLen); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// 检查数据长度是否合理
		if filterLen == 0 || filterLen > uint32(buf.Len()) {
			return myerror.ErrSSTReaderFilter
		}

		// 读取过滤器数据
		filterBytes := make([]byte, filterLen)
		if _, err := buf.Read(filterBytes); err != nil {
			return err
		}

		// 创建并加载过滤器
		bloomFilter := r.conf.FilterConstructor(1024, 3)
		if err := bloomFilter.Load(filterBytes); err != nil {
			return err
		}

		// 存储过滤器 - 使用blockLength作为映射键
		r.filterMap[blockLength] = bloomFilter
	}

	return nil
}

// Get 通过key获取value [比较慢速的查找 后期进行优化修改]
func (r *SSTReader) Get(key []byte) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 遍历所有索引块查找
	for _, idx := range r.index {
		// 检查key是否在当前索引的范围内
		// 注意：索引的范围判断是包括边界的
		if bytes.Compare(key, idx.StartKey) >= 0 && bytes.Compare(key, idx.EndKey) <= 0 {
			// 检查bloom filter，快速过滤不存在的key
			filter, exists := r.filterMap[idx.Length]
			if exists && !filter.Contains(key) {
				continue // 根据bloom filter判断key不在这个块中
			}

			// 读取对应数据块
			block := make([]byte, idx.Length)
			if _, err := r.fp.ReadAt(block, r.dataOffset+idx.Offset); err != nil {
				return nil, err
			}

			// 在数据块中查找key
			value, err := r.searchInBlock(block, key)
			if err == nil {
				return value, nil
			} else if err != myerror.ErrKeyNotFound {
				return nil, err
			}
			// 如果在当前块中没找到，继续查找下一个块
		}
	}

	// 如果所有索引块都没找到，尝试全面搜索所有数据区
	// 这是为了确保我们不会遗漏任何数据
	dataBytes := make([]byte, r.dataLength)
	if _, err := r.fp.ReadAt(dataBytes, r.dataOffset); err != nil {
		return nil, err
	}

	// 通过完整读取数据区来查找key
	buf := bytes.NewReader(dataBytes)
	for buf.Len() > 0 {
		// 读取key长度
		var keyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &keyLen); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// 读取value长度
		var valueLen uint32
		if err := binary.Read(buf, binary.BigEndian, &valueLen); err != nil {
			return nil, err
		}

		// 读取key
		currentKey := make([]byte, keyLen)
		if _, err := buf.Read(currentKey); err != nil {
			return nil, err
		}

		// 如果找到匹配的key，返回对应的value
		if bytes.Equal(currentKey, key) {
			value := make([]byte, valueLen)
			if _, err := buf.Read(value); err != nil {
				return nil, err
			}
			return value, nil
		}

		// 如果不匹配，跳过value部分
		if _, err := buf.Seek(int64(valueLen), io.SeekCurrent); err != nil {
			return nil, err
		}
	}

	return nil, myerror.ErrKeyNotFound
}

// searchInBlock 在数据块中搜索指定的key
func (r *SSTReader) searchInBlock(block []byte, searchKey []byte) ([]byte, error) {
	buf := bytes.NewReader(block)

	for buf.Len() > 0 {
		// 读取key长度
		var keyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &keyLen); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// 读取value长度
		var valueLen uint32
		if err := binary.Read(buf, binary.BigEndian, &valueLen); err != nil {
			return nil, err
		}

		// 读取key
		key := make([]byte, keyLen)
		if _, err := buf.Read(key); err != nil {
			return nil, err
		}

		// 如果找到匹配的key，返回对应的value
		if bytes.Equal(key, searchKey) {
			value := make([]byte, valueLen)
			if _, err := buf.Read(value); err != nil {
				return nil, err
			}
			return value, nil
		}

		// 跳过value
		if _, err := buf.Seek(int64(valueLen), io.SeekCurrent); err != nil {
			return nil, err
		}
	}

	return nil, myerror.ErrKeyNotFound
}

// Close 关闭SST读取器
func (r *SSTReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.fp != nil {
		return r.fp.Close()
	}
	return nil
}

// GetIterator 返回一个迭代器，用于遍历所有的key-value对
func (r *SSTReader) GetIterator() (*SSTIterator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 读取整个数据区
	data := make([]byte, r.dataLength)
	if _, err := r.fp.ReadAt(data, r.dataOffset); err != nil {
		return nil, err
	}

	// 创建迭代器
	it := &SSTIterator{
		reader:    r,
		dataBuf:   bytes.NewReader(data),
		currKey:   nil,
		currValue: nil,
		err:       nil,
	}

	return it, nil
}

// todo:后续补充使用
// SSTIterator SST迭代器
type SSTIterator struct {
	reader    *SSTReader
	dataBuf   *bytes.Reader // 整个数据区的读取器
	currKey   []byte        // 当前key
	currValue []byte        // 当前value
	err       error         // 迭代过程中的错误
}

// Next 移动到下一个key-value对
func (it *SSTIterator) Next() bool {
	// 如果已经有错误，不再继续
	if it.err != nil {
		return false
	}

	// 读取下一个key-value对
	return it.readNextKeyValue()
}

// readNextKeyValue 读取下一对key-value
func (it *SSTIterator) readNextKeyValue() bool {
	// 如果buffer已经读完，则结束
	if it.dataBuf.Len() == 0 {
		return false
	}

	// 读取key长度
	var keyLen uint32
	if err := binary.Read(it.dataBuf, binary.BigEndian, &keyLen); err != nil {
		if err == io.EOF {
			return false
		}
		it.err = err
		return false
	}

	// 读取value长度
	var valueLen uint32
	if err := binary.Read(it.dataBuf, binary.BigEndian, &valueLen); err != nil {
		it.err = err
		return false
	}

	// 读取key
	it.currKey = make([]byte, keyLen)
	if _, err := it.dataBuf.Read(it.currKey); err != nil {
		it.err = err
		return false
	}

	// 读取value
	it.currValue = make([]byte, valueLen)
	if _, err := it.dataBuf.Read(it.currValue); err != nil {
		it.err = err
		return false
	}

	return true
}

// Key 获取当前key
func (it *SSTIterator) Key() []byte {
	return it.currKey
}

// Value 获取当前value
func (it *SSTIterator) Value() []byte {
	return it.currValue
}

// Error 获取遍历过程中的错误
func (it *SSTIterator) Error() error {
	return it.err
}
