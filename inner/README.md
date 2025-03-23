# 🌟 LSM-Tree 核心实现

inner包是LSM-Tree数据库的核心实现，包含了LSM-Tree数据结构和主要操作逻辑。

## 🧩 核心组件

### 🌲 LsmTree

`LsmTree`是整个存储引擎的核心类，它协调各个组件工作，提供键值存储的主要功能：

- ✍️ 写入操作：通过WAL和MemTable实现高效写入
- 🔍 读取操作：通过多层查询实现数据检索
- 🔄 压缩操作：通过后台压缩工作线程实现数据整理

### 📊 主要字段

```go
type LsmTree struct {
    conf           *config.Config    // 配置
    mutableIndex   memtable.MemTable // 内存表
    walId          uint32            // 写日志文件id
    curWal         *wal.Wal          // 当前写日志
    immutableIndex []*immutable      // 不可变索引
    compactCh      chan *immutable   // 压缩通道，用于异步传递不可变索引进行压缩
    stopCh         chan struct{}     // 停止信号通道
    nodes          [][]*sst.Node     // 节点
    seq            []atomic.Uint32   // 序列号
}
```

### 🧊 不可变索引

```go
type immutable struct {
    wal   *wal.Wal          // 关联的WAL
    index memtable.MemTable // 内存表
}
```

## 🔄 工作流程

1. **📥 写入操作**
   - 先写入WAL，确保数据持久性
   - 然后更新内存表MemTable
   - 当WAL大小达到阈值时，触发MemTable转换为不可变索引

2. **📤 读取操作**
   - 首先在活跃MemTable中查找
   - 然后按照从新到旧的顺序在不可变索引中查找
   - 最后在SST文件节点中查找

3. **⚙️ 异步压缩**
   - 不可变索引通过通道传递给压缩工作线程
   - 压缩工作线程将不可变索引转换为SST文件
   - 完成压缩后，关闭WAL并从immutableIndex中移除

## 🛠️ 主要方法

### 🏗️ 创建LSM-Tree

```go
func NewLsmTree(conf *config.Config) (*LsmTree, error)
```

### 🔑 基本操作

```go
func (t *LsmTree) Put(key, value []byte) error
func (t *LsmTree) Get(key []byte) ([]byte, error)
func (t *LsmTree) Delete(key []byte) error
```

### 🔧 内部操作

```go
func (t *LsmTree) rotateWal() error
func (t *LsmTree) doCompact(imm *immutable) error
func (t *LsmTree) compactWorker()
func (t *LsmTree) Close() error
```

## 📦 压缩机制

当内存表达到一定大小时，会触发WAL轮转，将当前内存表标记为不可变，创建新的内存表和WAL继续接收写入。不可变内存表会通过通道传递给压缩工作线程，由工作线程将其转换为SST文件。压缩过程是异步的，不会阻塞主线程的写入操作。

### ✨ 压缩特点

- **🧵 异步处理**：通过channel实现不阻塞主线程
- **🔄 自动触发**：基于WAL大小自动触发
- **📚 分层存储**：支持多层SST文件组织
- **📊 有序合并**：保持数据的有序性，优化范围查询 