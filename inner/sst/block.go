package sst

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"

	"github.com/aixiasang/lsm/inner/config"
)

type Block struct {
	conf       *config.Config // 配置
	dataBuf    *bytes.Buffer  // 数据缓冲区
	entriesCnt int64          // 条目数量
	firstKey   []byte         // 第一个写入的key
	lastKey    []byte         // 最后写一个写入的key
	mu         sync.RWMutex   // 互斥锁
}

func NewBlock(conf *config.Config) *Block {
	return &Block{
		conf:       conf,
		dataBuf:    bytes.NewBuffer(nil),
		entriesCnt: 0,
		firstKey:   nil,
		lastKey:    nil,
	}
}

func (b *Block) Add(key, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.entriesCnt == 0 {
		b.firstKey = key
	}

	b.lastKey = key
	b.entriesCnt++
	if err := binary.Write(b.dataBuf, binary.BigEndian, uint32(len(key))); err != nil {
		return err
	}
	if err := binary.Write(b.dataBuf, binary.BigEndian, uint32(len(value))); err != nil {
		return err
	}
	if _, err := b.dataBuf.Write(key); err != nil {
		return err
	}
	if _, err := b.dataBuf.Write(value); err != nil {
		return err
	}
	return nil
}

// FilterAdd 添加过滤器数据
func (b *Block) FilterAdd(length int64, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 写入blockLength
	if err := binary.Write(b.dataBuf, binary.BigEndian, length); err != nil {
		return err
	}

	// 写入过滤器数据的长度
	filterLen := uint32(len(value))
	if err := binary.Write(b.dataBuf, binary.BigEndian, filterLen); err != nil {
		return err
	}

	// 写入过滤器数据
	if _, err := b.dataBuf.Write(value); err != nil {
		return err
	}
	return nil
}

func (b *Block) IndexAdd(index *Index) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	encoded, err := index.Encode()
	if err != nil {
		return err
	}
	if _, err := b.dataBuf.Write(encoded); err != nil {
		return err
	}
	return nil
}

func (b *Block) Bytes() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.dataBuf.Bytes()
}

func (b *Block) Length() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return int64(b.dataBuf.Len())
}

func (b *Block) FirstKey() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.firstKey
}

func (b *Block) LastKey() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastKey
}

func (b *Block) EntriesCnt() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.entriesCnt
}

func (b *Block) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dataBuf.Reset()
	b.entriesCnt = 0
	b.firstKey = nil
	b.lastKey = nil
}

func (b *Block) Flush(fp io.Writer) (int64, error) {
	defer b.Clear()

	length, err := fp.Write(b.Bytes())
	return int64(length), err
}
