package config

import (
	"github.com/aixiasang/lsm/inner/filter"
	"github.com/aixiasang/lsm/inner/memtable"
)

const (
	DefaultDataDir        = "./data"
	DefaultWalDir         = "./wal"
	DefaultSSTDir         = "./sst"
	DefaultBlockSize      = 1024 * 1024
	DefaultWalSize        = 1024 * 1024 * 10
	DefaultMemTableDegree = 16
	DefaultMemTableType   = MemTableTypeBTree
)

type MemTableType int8

const (
	MemTableTypeBTree MemTableType = iota
	MemTableTypeSkipList
)

type Config struct {
	DataDir             string                                                           // 数据目录
	WalDir              string                                                           // WAL目录
	SSTDir              string                                                           // SST目录
	AutoSync            bool                                                             // 是否自动同步
	BlockSize           int64                                                            // 块大小
	WalSize             uint32                                                           // WAL大小
	MemTableType        MemTableType                                                     // 内存表类型
	MemTableDegree      int                                                              // 内存表度
	LevelSize           int                                                              // 层级大小
	FilterConstructor   func(m uint64, k uint) filter.Filter                             // 过滤器构造函数
	MemTableConstructor func(mtType memtable.MemTableType, degree int) memtable.MemTable // 内存表构造函数
	IsDebug             bool                                                             // 是否调试
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:             DefaultDataDir,
		WalDir:              DefaultWalDir,
		SSTDir:              DefaultSSTDir,
		MemTableType:        DefaultMemTableType,
		MemTableDegree:      DefaultMemTableDegree,
		AutoSync:            true,
		BlockSize:           1024 * 1024,
		FilterConstructor:   filter.NewBloomFilter,
		MemTableConstructor: memtable.NewMemTable,
		LevelSize:           5,
		WalSize:             1024 * 1,
		IsDebug:             true,
	}
}
