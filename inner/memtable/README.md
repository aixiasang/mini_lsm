# 📝 内存表 (MemTable)

MemTable模块提供了LSM-Tree架构中的内存数据结构，用于临时存储写入的键值对，并在适当时机将数据刷新到磁盘。内存表是写操作的第一站，它将随机写入聚合为批量顺序写入，从而提高系统整体性能。

## ✨ 核心特性

- **⚡ 高效的插入和查询**：通过优化的数据结构提供高效操作
- **📊 有序数据组织**：自动保持键的有序排列，便于后续处理
- **🔒 并发安全**：支持并发访问的安全保障
- **🔧 可配置性**：支持多种内部数据结构实现

## 🧩 可用实现

MemTable模块提供了多种底层实现以适应不同场景：

1. **🌳 B树实现** - 提供良好的读写平衡
2. **🪜 跳表实现** - 针对频繁写入的场景优化

## 📋 接口定义

```go
type MemTable interface {
    // 插入键值对
    Put(key, value []byte) error
    
    // 获取键对应的值
    Get(key []byte) ([]byte, error)
    
    // 删除键值对
    Delete(key []byte) error
    
    // 遍历所有键值对
    ForEach(fn func(key, value []byte) bool) error
    
    // 不加锁遍历，用于批量导出等场景
    ForEachUnSafe(fn func(key, value []byte) bool) bool
    
    // 获取内存表大小
    Size() int
}
```

## 🔰 使用方法

### 🏭 创建内存表

```go
// 创建B树内存表
memTable := memtable.NewMemTable(memtable.MemTableTypeBTree, 16)

// 创建跳表内存表
memTable := memtable.NewMemTable(memtable.MemTableTypeSkipList, 16)
```

### 🛠️ 基本操作

```go
// 插入
memTable.Put([]byte("key1"), []byte("value1"))

// 查询
value, err := memTable.Get([]byte("key1"))

// 删除
memTable.Delete([]byte("key1"))

// 遍历
memTable.ForEach(func(key, value []byte) bool {
    fmt.Printf("Key: %s, Value: %s\n", key, value)
    return true // 继续遍历
})
```

## 🔍 实现细节

### 🌳 B树实现

基于Google的btree库，提供了良好的读写性能平衡。适合读写操作比例相近的应用场景。

```go
type BTreeMemTable struct {
    tree *btree.BTree
    size int
    mu   sync.RWMutex
}
```

### 🪜 跳表实现

基于跳表数据结构，在写入密集型场景下表现优异。特别适合追加写入模式。

```go
type SkipListMemTable struct {
    list *skiplist.SkipList
    size int
    mu   sync.RWMutex
}
```

## 💾 内存管理

MemTable会在内存中累积数据，直到触发以下条件之一：

1. 📈 达到配置的大小阈值
2. 📝 关联的WAL文件达到配置的大小阈值

当触发条件时，当前MemTable会被转换为不可变内存表(Immutable MemTable)，同时创建新的MemTable继续接收写入。不可变内存表随后会被后台压缩线程转换为磁盘上的SST文件。

## ⚡ 性能考虑

- **🧠 内存占用**：权衡内存使用与磁盘I/O频率
- **🔒 锁粒度**：B树实现使用全局锁，跳表实现可优化为更细粒度的锁
- **📦 序列化开销**：键值对的序列化和反序列化会影响性能 