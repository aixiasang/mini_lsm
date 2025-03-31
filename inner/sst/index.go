package sst

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Index
type Index struct {
	StartKey []byte //最小的key
	EndKey   []byte //最大的key
	Offset   int64  //偏移量
	Length   int64  //长度
}

func (i *Index) String() string {
	return fmt.Sprintf("StartKey: %s, EndKey: %s, Offset: %d, Length: %d", i.StartKey, i.EndKey, i.Offset, i.Length)
}

func (i *Index) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint32(len(i.StartKey))); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(len(i.EndKey))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(i.StartKey); err != nil {
		return nil, err
	}
	if _, err := buf.Write(i.EndKey); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, i.Offset); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, i.Length); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
