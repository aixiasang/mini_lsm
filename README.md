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

## 🚀 快速开始

### 📋 先决条件

- Go 1.16+
- Git

### 🔧 安装

```bash
# 克隆仓库
git clone https://github.com/yourusername/golsm.git
cd golsm

# 构建项目
go build -o golsm ./cmd/main.go
```

### 📝 基本使用

```go
package main

import (
    "fmt"
    "github.com/yourusername/golsm/inner"
    "github.com/yourusername/golsm/inner/config"
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

### 🌲 LSM-Tree

LSM-Tree是一种写优化的数据结构，通过将随机写转换为顺序写来提高写入性能。核心包括内存表、排序字符串表和预写日志。

### 📝 WAL (预写日志)

WAL确保数据持久性，所有写操作先写入WAL再修改内存，提供崩溃恢复能力。

### 🧠 MemTable (内存表)

MemTable是内存中的数据结构，支持高效的插入和查询。当内存表达到阈值后，会被转换为不可变内存表，并创建新的SST文件。

### 📑 SST (排序字符串表)

SST是磁盘上的不可变数据结构，包含排序的键值对。多级SST结构允许增量合并，减少写入放大。

### 🔍 BloomFilter (布隆过滤器)

布隆过滤器用于快速判断键是否可能存在，减少不必要的磁盘访问。

## 🛠️ 高级配置

```go
// 创建自定义配置
conf := &config.Config{
    // 数据目录
    DBDir:       "./data",
    WalDir:      "./data/wal",
    
    // 内存表大小限制（内存使用）
    MemtableSize: 32 * 1024 * 1024,  // 32MB
    
    // SST文件参数
    BlockSize:   4 * 1024,  // 4KB 
    MaxFileSize: 64 * 1024 * 1024,  // 64MB
    
    // LSM-Tree配置
    MaxLevels:   7,
    SizeRatio:   10,
    
    // 预写日志配置
    WalSize:     16 * 1024 * 1024,  // 16MB
    AutoSync:    true,
    
    // 过滤器配置
    BloomBitsPerKey: 10,
    
    // 压缩策略
    Compaction:  "leveled",
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
    conf := &config.Config{
        DataDir: "./data",
        WalDir: "wal",
        WalSize: 4 * 1024 * 1024, // 4MB
        AutoSync: true,
    }
    
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

## 🚀 性能特点

- ✍️ 写入优先 - 针对高吞吐量写入场景优化
- 📊 分层存储 - 随着数据量增长，性能保持稳定
- 🔄 压缩与合并 - 自动管理数据分布，优化存储空间 