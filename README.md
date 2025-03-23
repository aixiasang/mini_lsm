# 🚀 Go LSM-Tree 数据库

![Language](https://img.shields.io/badge/Language-Go-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

📚 基于LSM树（Log-Structured Merge Tree）实现的高性能键值数据库，完全使用Go语言开发。

## 📖 简介

LSM-Tree是一种日志结构的合并树，是一种用于持久化存储的数据结构，特别适合写入密集型的应用场景。它将随机写入转换为顺序写入，显著提高了写入性能。

## 🏗️ 系统架构

该项目由以下核心模块组成：

- **📝 内存表(MemTable)**: 用于临时存储写入数据
- **📒 预写日志(WAL)**: 确保数据持久性和崩溃恢复
- **📚 排序字符串表(SST)**: 磁盘上的不可变数据结构
- **🔍 布隆过滤器(BloomFilter)**: 优化键值查找性能
- **🔄 合并机制(Compaction)**: 自动整理和优化数据

## ✨ 特性

- 🌲 **LSM-Tree架构**：高效写入优化，写放大较小
- 📝 **日志结构化存储**：所有写入都是顺序追加，提高写入性能
- 🔍 **布隆过滤器**：快速识别不存在的键，减少不必要的磁盘访问
- 📑 **SST文件格式**：数据以排序字符串表格式存储，支持快速查询
- 🧠 **多级内存/磁盘存储**：平衡内存占用与查询性能
- 💾 **WAL预写日志**：确保数据持久性和崩溃恢复能力
- 🔄 **自动压缩/合并**：后台自动合并，优化存储空间和查询性能
- 🔒 **并发安全**：支持多协程并发访问
- 🛠️ **可配置性**：丰富的配置选项，适应不同场景

## 📦 当前实现状态

目前的LSM-Tree实现已完成以下基础功能：

- **✅ 基本键值操作**：支持Put、Get、Delete等基本操作
- **✅ 日志预写（WAL）**：确保数据持久性，防止系统崩溃数据丢失
- **✅ 内存表管理**：支持可变和不可变内存表，高效处理写入操作
- **✅ SST文件持久化**：将内存表数据持久化到SST文件
- **✅ 多级存储**：支持多层级SST文件组织
- **✅ 异步压缩**：后台异步将不可变内存表压缩到SST文件

### ❌ 尚未实现的功能

- **层次合并（Leveled Compaction）**：目前尚未实现不同层级之间的SST文件合并
- **范围查询**：尚未实现范围查询功能
- **迭代器接口**：尚未提供标准的迭代器接口用于数据遍历

## 🗂️ 项目结构

```
├── 📁 cmd            # 命令行工具和示例
├── 📁 inner          # 核心实现
│   ├── 📁 config     # 配置管理
│   ├── 📁 filter     # 布隆过滤器
│   ├── 📁 memtable   # 内存表实现
│   ├── 📁 myerror    # 错误处理
│   ├── 📁 sst        # 排序字符串表
│   ├── 📁 utils      # 实用工具
│   ├── 📁 wal        # 预写日志
│   └── 📄 lsm.go     # LSM树主结构
└── 📁 test           # 测试用例
```

## 🧩 核心组件实现

### 🌲 LsmTree 主结构

```go
type LsmTree struct {
    conf           *config.Config    // 配置
    mutableIndex   memtable.MemTable // 内存表
    walId          uint32            // 写日志文件id
    curWal         *wal.Wal          // 当前写日志
    immutableIndex []*immutable      // 不可变索引
    compactCh      chan *immutable   // 压缩通道，用于异步传递不可变索引进行压缩
    stopCh         chan struct{}     // 停止信号通道
    nodes          [][]*sst.Node     // 节点 - 每层的节点切片数组
    seq            []*atomic.Uint32  // 序列号
    levelSize      int               // 层级大小
}
```

### 🧊 不可变索引

```go
type immutable struct {
    wal   *wal.Wal          // 关联的WAL
    index memtable.MemTable // 内存表
}
```

### 🔄 工作流程

1. **📥 写入流程**
   - 先写入WAL，确保数据持久性
   - 然后更新内存表MemTable
   - 当WAL大小达到阈值时，触发MemTable转换为不可变索引

2. **📤 读取流程**
   - 首先在活跃MemTable中查找
   - 然后按照从新到旧的顺序在不可变索引中查找
   - 最后在SST文件节点中按层查找

3. **⚙️ 异步压缩**
   - 不可变索引通过通道传递给压缩工作线程
   - 压缩工作线程将不可变索引转换为SST文件
   - 完成压缩后，关闭WAL并从immutableIndex中移除

## 🚀 快速开始

### 📋 先决条件

- Go 1.16+
- Git

### 🔧 安装

```bash
# 克隆仓库
git clone https://github.com/aixiasang/lsm.git
cd lsm

# 构建项目
go build -o lsm ./cmd/main.go
```

### 📝 基本使用

```go
package main

import (
    "fmt"
    "github.com/aixiasang/lsm/inner"
    "github.com/aixiasang/lsm/inner/config"
)

func main() {
    // 创建默认配置
    conf := config.DefaultConfig()
    
    // 创建LSM-Tree实例
    lsm, err := inner.NewLsmTree(conf)
    if err != nil {
        panic(err)
    }
    defer lsm.Close()
    
    // 写入键值对
    err = lsm.Put([]byte("hello"), []byte("world"))
    if err != nil {
        panic(err)
    }
    
    // 读取键值对
    value, err := lsm.Get([]byte("hello"))
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Value for 'hello': %s\n", string(value))
    
    // 删除键值对
    err = lsm.Delete([]byte("hello"))
    if err != nil {
        panic(err)
    }
}
```

## 🔍 模块详解

### 🌲 LSM-Tree 核心实现

LSM-Tree是一种写优化的数据结构，通过将随机写转换为顺序写来提高写入性能。主要方法包括：

```go
// 创建LSM-Tree
func NewLsmTree(conf *config.Config) (*LsmTree, error)

// 基本操作
func (t *LsmTree) Put(key, value []byte) error
func (t *LsmTree) Get(key []byte) ([]byte, error)
func (t *LsmTree) Delete(key []byte) error

// 内部操作
func (t *LsmTree) rotateWal() error
func (t *LsmTree) doCompact(imm *immutable) error
func (t *LsmTree) compactWorker()
func (t *LsmTree) Close() error

// 加载操作
func (t *LsmTree) load() error
func (t *LsmTree) loadSST() error
func (t *LsmTree) loadWAL() error
```

### 📝 WAL (预写日志)

WAL确保数据持久性，所有写操作先写入WAL再修改内存，提供崩溃恢复能力。系统启动时会从WAL中恢复数据。

### 🧠 MemTable (内存表)

MemTable是内存中的数据结构，支持高效的插入和查询。本实现提供两种内存表实现：
- **B树实现**: 适合范围查询
- **跳表实现**: 适合随机访问

当内存表达到阈值后，会被转换为不可变内存表，并通过异步方式创建新的SST文件。

### 📑 SST (排序字符串表)

SST是磁盘上的不可变数据结构，包含排序的键值对。SST文件由以下组件组成：

- **数据块**: 存储实际的键值对
- **索引块**: 存储数据块的索引信息
- **元数据块**: 存储文件的元数据
- **过滤器块**: 存储布隆过滤器等数据结构，加速查找

### 🔍 SST节点

```go
type Node struct {
    conf     *config.Config          // 配置
    filename string                  // 文件名
    level    int                     // 层级
    seq      int32                   // 序列号
    size     int64                   // 大小
    minKey   []byte                  // 最小键
    maxKey   []byte                  // 最大键
    index    []*Index                // 索引
    filter   map[int64]filter.Filter // 过滤器
    reader   *SSTReader              // 读取器
    kvList   []*KeyValue             // 数据块
}
```

### 🔍 BloomFilter (布隆过滤器)

布隆过滤器用于快速判断键是否可能存在，减少不必要的磁盘访问。

## 🛠️ 高级配置

```go
// 创建自定义配置
conf := &config.Config{
    // 数据目录
    DataDir:             "./data",
    WalDir:              "wal",
    SSTDir:              "sst",
    
    // 内存表配置
    MemTableType:        config.MemTableTypeBTree,
    MemTableDegree:      16,
    
    // 预写日志配置
    WalSize:             1024 * 1024,  // 1MB
    AutoSync:            true,
    
    // SST文件参数
    BlockSize:           1024 * 1024,  // 1MB
    
    // LSM-Tree配置
    LevelSize:           5,
    
    // 过滤器配置
    FilterConstructor:   filter.NewBloomFilter,
    
    // 调试选项
    IsDebug:             true,
}
```

## 📊 性能优化

### 📝 写优化

- 将随机写转换为顺序写
- 批量写入减少WAL同步次数
- 自动调整WAL和SST大小

### 🔍 读优化

- 布隆过滤器减少磁盘访问
- 多级缓存减少IO操作
- 并行查询多个SST文件

## 📜 许可证

本项目基于MIT许可证开源 - 详见 [LICENSE](LICENSE) 文件

## 🔰 使用示例

```go
package main

import (
    "github.com/aixiasang/lsm/inner"
    "github.com/aixiasang/lsm/inner/config"
)

func main() {
    // 创建配置
    conf := config.DefaultConfig()
    conf.DataDir = "./data"
    conf.WalDir = "wal"
    conf.SSTDir = "sst"
    conf.WalSize = 4 * 1024 * 1024 // 4MB
    conf.MemTableType = config.MemTableTypeBTree
    conf.MemTableDegree = 16
    
    // 初始化LSM-Tree
    lsm, err := inner.NewLsmTree(conf)
    if err != nil {
        panic(err)
    }
    defer lsm.Close()
    
    // 写入数据
    if err := lsm.Put([]byte("key1"), []byte("value1")); err != nil {
        panic(err)
    }
    
    // 读取数据
    value, err := lsm.Get([]byte("key1"))
    if err != nil {
        panic(err)
    }
    
    // 删除数据
    if err := lsm.Delete([]byte("key1")); err != nil {
        panic(err)
    }
}
```

## 📚 模块说明

详情请参阅各模块的README文件：

- [🌟 inner](./inner/README.md) - LSM-Tree核心实现
- [⚙️ config](./inner/config/README.md) - 配置相关
- [🔍 filter](./inner/filter/README.md) - 布隆过滤器实现
- [📝 memtable](./inner/memtable/README.md) - 内存表实现
- [⚠️ myerror](./inner/myerror/README.md) - 错误定义
- [📚 sst](./inner/sst/README.md) - 排序字符串表实现
- [🛠️ utils](./inner/utils/README.md) - 工具函数
- [📒 wal](./inner/wal/README.md) - 预写日志实现

## 🚀 未来规划

1. **✨ 实现层次合并**：基于大小和层级触发SST文件之间的合并，优化存储效率
2. **✨ 添加范围查询支持**：实现高效的范围查询功能
3. **✨ 提供迭代器接口**：用于高效遍历数据
4. **✨ 优化读取性能**：通过缓存、索引优化等手段提升读取性能
5. **✨ 增强并发控制**：优化多线程下的性能表现 