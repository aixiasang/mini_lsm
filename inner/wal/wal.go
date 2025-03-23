package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/memtable"
)

type Wal struct {
	conf   *config.Config // 配置
	fileId uint32         // 文件ID
	offset uint32         // 偏移量
	fp     *os.File       // 文件
	mu     sync.RWMutex   // 互斥锁
}

func NewWal(conf *config.Config, fileId uint32) (*Wal, error) {
	filePath := filepath.Join(conf.DataDir, conf.WalDir, fmt.Sprintf("wal-%d.log", fileId))
	fp, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Wal{conf: conf, fileId: fileId, fp: fp}, nil
}

func (w *Wal) Write(key, value []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	rec := NewRecord(key, value)
	encoded, err := rec.Encode()
	if err != nil {
		return err
	}
	length, err := w.fp.Write(encoded)
	if err != nil {
		return err
	}
	if w.conf.AutoSync {
		if err := w.fp.Sync(); err != nil {
			return err
		}
	}
	w.offset += uint32(length)
	return nil
}

func (w *Wal) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.fp.Sync(); err != nil {
		return err
	}
	return w.fp.Close()
}

func (w *Wal) ReadAll(memTable memtable.MemTable) error {
	// 将文件指针移到开始位置
	if _, err := w.fp.Seek(0, 0); err != nil {
		return err
	}
	if w.conf.IsDebug {
		fmt.Printf("开始从文件ID=%d读取全部记录\n", w.fileId)
	}

	// 获取文件大小
	fileInfo, err := w.fp.Stat()
	if err != nil {
		return fmt.Errorf("无法获取文件大小: %v", err)
	}
	fileSize := fileInfo.Size()

	// 读取整个文件
	buffer := make([]byte, fileSize)
	n, err := w.fp.ReadAt(buffer, 0)
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取文件内容失败: %v", err)
	}
	if int64(n) < fileSize {
		fmt.Printf("警告：仅读取了文件部分内容: %d 字节，总大小 %d 字节\n", n, fileSize)
	}

	// 逐条解析记录并保存最新的记录位置
	var offset uint32 = 0
	for offset < uint32(n) {
		// 确保至少能读取头部
		if offset+9 > uint32(n) {
			fmt.Printf("文件末尾不完整，停止解析: 剩余 %d 字节\n", uint32(n)-offset)
			break
		}

		// 读取记录类型
		recordType := RecordType(buffer[offset])

		// 读取 key 长度
		keyLength := binary.BigEndian.Uint32(buffer[offset+1 : offset+5])

		// 读取 value 长度
		valueLength := binary.BigEndian.Uint32(buffer[offset+5 : offset+9])

		// 检查 key 和 value 长度的合理性
		if keyLength > 10*1024*1024 || valueLength > 100*1024*1024 {
			fmt.Printf("警告: 可能的数据损坏 - key长度: %d, value长度: %d\n", keyLength, valueLength)
			break
		}

		// 计算记录总长度
		recordLength := 9 + keyLength + valueLength + 4

		// 确保能读取完整的记录
		if offset+recordLength > uint32(n) {
			fmt.Printf("文件末尾记录不完整，停止解析: 需要 %d 字节，剩余 %d 字节\n",
				recordLength, uint32(n)-offset)
			break
		}

		// 读取 key 和 value
		key := buffer[offset+9 : offset+9+keyLength]
		value := buffer[offset+9+keyLength : offset+9+keyLength+valueLength]

		// 读取 CRC
		crc := binary.BigEndian.Uint32(buffer[offset+9+keyLength+valueLength : offset+recordLength])

		// 计算CRC进行验证
		computedCrc := crc32.ChecksumIEEE(buffer[offset : offset+9+keyLength+valueLength])
		if crc != computedCrc {
			fmt.Printf("警告: CRC校验失败 (offset=%d) - 存储的: %d, 计算的: %d\n",
				offset, crc, computedCrc)
			// 继续处理，但记录警告
		}
		if w.conf.IsDebug {
			fmt.Printf("解析记录: type=%d, key=%s, keyLen=%d, valueLen=%d, offset=%d, len=%d\n",
				recordType, string(key), keyLength, valueLength, offset, recordLength)
		}

		// 基于记录类型处理
		if recordType == RecordTypeDelete {
			if w.conf.IsDebug {
				fmt.Printf("处理删除记录: key=%s\n", string(key))
			}
			_ = memTable.Delete(key)
		} else {
			if w.conf.IsDebug {
				fmt.Printf("处理普通记录: key=%s, value=%s\n", string(key), string(value))
			}
			// 关键修复: 使用当前记录在文件中的实际位置，而不是旧位置
			if err := memTable.Put(key, value); err != nil {
				return fmt.Errorf("更新索引失败: %v", err)
			}
		}

		// 更新偏移量
		offset += recordLength
	}

	if w.conf.IsDebug {
		fmt.Printf("文件ID=%d读取完成，处理了 %d 字节\n", w.fileId, offset)
	}

	// 更新WAL实例的offset以反映文件的实际大小
	w.offset = offset

	return nil
}

func (w *Wal) Size() uint32 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.offset
}

func (w *Wal) Sync() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.fp.Sync()
}

func (w *Wal) FileId() uint32 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.fileId
}
func (w *Wal) UpdateOffset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	fileInfo, err := w.fp.Stat()
	if err != nil {
		return
	}
	w.offset = uint32(fileInfo.Size())
}
func (w *Wal) Delete() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.fp.Sync(); err != nil {
		return err
	}
	if err := w.fp.Close(); err != nil {
		return err
	}
	return os.Remove(w.fp.Name())
}
