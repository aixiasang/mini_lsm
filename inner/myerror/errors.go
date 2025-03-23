package myerror

import "errors"

var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrValueNil         = errors.New("value has been deleted")
	ErrKeyNil           = errors.New("key is nil")
	ErrInvalidSSTFormat = errors.New("invalid SST format")

	ErrWalCorrupted = errors.New("wal corrupted")
	ErrSSTCorrupted = errors.New("sst corrupted")

	ErrInvalidBloomFilter    = errors.New("invalid bloom filter")
	ErrBloomFilterIncomplete = errors.New("unexpected end of data, bloom filter data incomplete")

	ErrEncodeRecordType     = errors.New("failed to write record type")
	ErrEncodeKeyLength      = errors.New("failed to write key length")
	ErrEncodeValueLength    = errors.New("failed to write value length")
	ErrEncodeKey            = errors.New("failed to write key")
	ErrEncodeValue          = errors.New("failed to write value")
	ErrEncodeCrc            = errors.New("failed to write crc")
	ErrRecordDataTooShort   = errors.New("record data too short")
	ErrDecodeKeyLength      = errors.New("failed to read key length")
	ErrDecodeValueLength    = errors.New("failed to read value length")
	ErrDecodeKey            = errors.New("failed to read key")
	ErrDecodeValue          = errors.New("failed to read value")
	ErrDecodeCrc            = errors.New("failed to read crc")
	ErrRecordDataIncomplete = errors.New("record data incomplete")
	ErrCrcMismatch          = errors.New("crc mismatch")

	ErrSSTReaderFilter = errors.New("invalid filter length")
)
