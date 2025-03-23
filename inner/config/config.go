package config

import "github.com/aixiasang/lsm/inner/filter"

type Config struct {
	DataDir   string // 数据目录
	WalDir    string // WAL目录
	AutoSync  bool   // 是否自动同步
	BlockSize int64  // 块大小
	WalSize   uint32 // WAL大小

	FilterConstructor func(m uint64, k uint) filter.Filter // 过滤器构造函数
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:           "./data",
		WalDir:            "./wal",
		AutoSync:          true,
		BlockSize:         1024 * 1024,
		FilterConstructor: filter.NewBloomFilter,
	}
}
