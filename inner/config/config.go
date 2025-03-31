package config

import (
	"github.com/aixiasang/lsm/inner/filter"
	"github.com/aixiasang/lsm/inner/memtable"
)

const (
	DefaultDataDir        = "./data"          // 默认数据目录
	DefaultWalDir         = "./wal"           // 默认WAL目录
	DefaultSSTDir         = "./sst"           // 默认SST目录
	DefaultBlockSize      = 1024 * 1024       // 默认块大小
	DefaultWalSize        = 1024 * 1024 * 10  // 默认WAL大小
	DefaultMemTableDegree = 16                // 默认内存表度
	DefaultMemTableType   = MemTableTypeBTree // 内存表类型
)

// MemTableType 内存表类型
type MemTableType int8

const (
	MemTableTypeBTree    MemTableType = iota // 默认B树
	MemTableTypeSkipList                     // 跳表
)

// FilterConstructor 过滤器构造函数
type FilterConstructor func(m uint64, k uint) filter.Filter

// MemTableConstructor 内存表构造函数
type MemTableConstructor func(mtType memtable.MemTableType, degree int) memtable.MemTable

// Config 配置
type Config struct {
	DataDir             string              // 数据目录
	WalDir              string              // WAL目录
	SSTDir              string              // SST目录
	AutoSync            bool                // 是否自动同步
	BlockSize           int64               // 块大小
	WalSize             uint32              // WAL大小
	MemTableType        MemTableType        // 内存表类型
	MemTableDegree      int                 // 内存表度
	LevelSize           int                 // 层级大小
	FilterConstructor   FilterConstructor   // 过滤器构造函数
	MemTableConstructor MemTableConstructor // 内存表构造函数
	IsDebug             bool                // 是否调试
}

// DefaultConfig 默认配置
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
