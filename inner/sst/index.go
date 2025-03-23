package sst

import (
	"bytes"
	"encoding/binary"
)

type Index struct {
	StartKey []byte
	EndKey   []byte
	Offset   int64
	Length   int64
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
