package myerror

import "errors"

var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrValueNil         = errors.New("value has been deleted")
	ErrKeyNil           = errors.New("key is nil")
	ErrInvalidSSTFormat = errors.New("invalid SST format")
)
