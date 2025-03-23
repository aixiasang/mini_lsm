package sst

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/filter"
)

type SSTWriter struct {
	conf           *config.Config   // 配置
	filename       string           // 文件名
	sstWriter      *os.File         // 写入的文件
	dataBuf        *bytes.Buffer    // 数据缓冲区
	indexBuf       *bytes.Buffer    // 索引缓冲区
	filterBuf      *bytes.Buffer    // 过滤器缓冲区
	dataBlock      *Block           // 数据块
	filterBlock    *Block           // 过滤器块
	indexBlock     *Block           // 索引块
	filter         filter.Filter    // 过滤器
	mapFilter      map[int64][]byte // 映射过滤器
	curBlockLength int64            // 当前数据块的长度
	curBlockOffset int64            // 当前数据块的偏移量
	index          []*Index         // 索引
}

func NewSSTWriter(conf *config.Config, filename string) (*SSTWriter, error) {
	fp, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &SSTWriter{
		conf:           conf,
		filename:       filename,
		sstWriter:      fp,
		filter:         conf.FilterConstructor(1024, 3),
		dataBuf:        bytes.NewBuffer(nil),
		indexBuf:       bytes.NewBuffer(nil),
		filterBuf:      bytes.NewBuffer(nil),
		dataBlock:      NewBlock(conf),
		filterBlock:    NewBlock(conf),
		indexBlock:     NewBlock(conf),
		mapFilter:      make(map[int64][]byte),
		curBlockLength: 0,
		curBlockOffset: 0,
		index:          make([]*Index, 0),
	}, nil
}
func (s *SSTWriter) mustRotateDataBlock() error {
	// 当前数据块的长度
	currBlockLength := s.dataBlock.Length()
	if currBlockLength == 0 {
		return nil
	}
	// 将过滤器数据进行存储
	currFilter := s.filter.Save()
	s.mapFilter[currBlockLength] = currFilter
	// filterblock 添加到过滤器块
	if err := s.filterBlock.FilterAdd(currBlockLength, currFilter); err != nil {
		return err
	}

	s.filter.Reset()

	prevLength, err := s.dataBlock.Flush(s.dataBuf)
	if err != nil {
		return err
	}
	s.curBlockLength = currBlockLength
	s.curBlockOffset = prevLength - currBlockLength

	currIndex := &Index{
		StartKey: s.dataBlock.FirstKey(),
		EndKey:   s.dataBlock.LastKey(),
		Offset:   s.curBlockOffset,
		Length:   s.curBlockLength,
	}
	s.index = append(s.index, currIndex)
	// indexblock 添加到索引块
	if err := s.indexBlock.IndexAdd(currIndex); err != nil {
		return err
	}
	// 将数据块写入到数据缓冲区
	if _, err := s.dataBuf.Write(s.dataBlock.Bytes()); err != nil {
		return err
	}
	return nil
}
func (s *SSTWriter) tryRotateDataBlock() error {

	if s.dataBlock.EntriesCnt() <= s.conf.BlockSize {
		return nil
	}

	return s.mustRotateDataBlock()
}
func (s *SSTWriter) Add(key, value []byte) error {
	// 如果数据块满了，则创建新的数据块
	if err := s.dataBlock.Add(key, value); err != nil {
		return err
	}
	s.filter.Add(key)
	if err := s.tryRotateDataBlock(); err != nil {
		return err
	}
	return nil
}

func (s *SSTWriter) Flush() error {
	// 如果数据块满了，则创建新的数据块
	if err := s.mustRotateDataBlock(); err != nil {
		return err
	}
	// 将数据块写入到数据缓冲区
	if _, err := s.dataBuf.Write(s.dataBlock.Bytes()); err != nil {
		return err
	}
	// 将索引块写入到索引缓冲区
	if _, err := s.indexBuf.Write(s.indexBlock.Bytes()); err != nil {
		return err
	}
	// 将过滤器块写入到过滤器缓冲区
	if _, err := s.filterBuf.Write(s.filterBlock.Bytes()); err != nil {
		return err
	}

	// 写入footer
	footerBuffer := bytes.NewBuffer(nil)
	dataLength, err := s.sstWriter.Write(s.dataBuf.Bytes())
	if err != nil {
		return err
	}
	if err := binary.Write(footerBuffer, binary.BigEndian, uint32(dataLength)); err != nil {
		return err
	}

	indexLength, err := s.sstWriter.Write(s.indexBuf.Bytes())
	if err != nil {
		return err
	}
	if err := binary.Write(footerBuffer, binary.BigEndian, uint32(indexLength)); err != nil {
		return err
	}

	filterLength, err := s.sstWriter.Write(s.filterBuf.Bytes())
	if err != nil {
		return err
	}
	if err := binary.Write(footerBuffer, binary.BigEndian, uint32(filterLength)); err != nil {
		return err
	}

	if _, err := s.sstWriter.Write(footerBuffer.Bytes()); err != nil {
		return err
	}

	return nil
}

func (s *SSTWriter) Close() error {

	return nil
}
