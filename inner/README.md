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

## 🔄 工作流程

1. **📥 写入操作**
   - 先写入WAL，确保数据持久性
   - 然后更新内存表MemTable
   - 当WAL大小达到阈值时，触发MemTable转换为不可变索引

2. **📤 读取操作**
   - 首先在活跃MemTable中查找
   - 然后按照从新到旧的顺序在不可变索引中查找
   - 最后在SST文件节点中按层查找

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

### 📥 加载操作

```go
func (t *LsmTree) load() error
func (t *LsmTree) loadSST() error
func (t *LsmTree) loadWAL() error
```

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

## 🔧 配置选项

LSM-Tree支持以下配置选项：

```go
type Config struct {
    DataDir             string                                   // 数据目录
    WalDir              string                                   // WAL目录
    SSTDir              string                                   // SST目录
    AutoSync            bool                                     // 是否自动同步
    BlockSize           int64                                    // 块大小
    WalSize             uint32                                   // WAL大小
    MemTableType        MemTableType                             // 内存表类型
    MemTableDegree      int                                      // 内存表度
    LevelSize           int                                      // 层级大小
    FilterConstructor   func(m uint64, k uint) filter.Filter     // 过滤器构造函数
    MemTableConstructor func(...) memtable.MemTable              // 内存表构造函数
    IsDebug             bool                                     // 是否调试
}
```

## 📚 SST文件结构

SST文件由以下组件组成：

- **数据块**：存储实际的键值对
- **索引块**：存储数据块的索引信息
- **元数据块**：存储文件的元数据
- **过滤器块**：存储布隆过滤器等数据结构，加速查找

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

## 🚀 未来规划

1. **✨ 实现层次合并**：基于大小和层级触发SST文件之间的合并，优化存储效率
2. **✨ 添加范围查询支持**：实现高效的范围查询功能
3. **✨ 提供迭代器接口**：用于高效遍历数据
4. **✨ 优化读取性能**：通过缓存、索引优化等手段提升读取性能
5. **✨ 增强并发控制**：优化多线程下的性能表现 