# 🔍 布隆过滤器 (BloomFilter)

布隆过滤器是LSM-Tree数据库中用于优化读取操作的概率数据结构。它能够快速判断一个元素是否可能存在于集合中，并具有空间效率高的特点。注意，布隆过滤器可能会产生假阳性（误报），但不会产生假阴性（漏报）。

## ✨ 特点

- **🗜️ 空间效率**：相比于传统的集合数据结构，布隆过滤器占用空间更小
- **⚡ 查询时间复杂度O(k)**：k为哈希函数数量，与集合元素数量无关
- **🔒 无法删除元素**：一旦元素被添加，无法从布隆过滤器中删除
- **📊 可调节的错误率**：可以通过配置控制假阳性率

## 📋 核心接口

```go
type Filter interface {
    // 添加一个键到过滤器
    Add(key []byte)
    
    // 检查一个键是否可能存在于过滤器中
    Contains(key []byte) bool
    
    // 计算当前的假阳性率
    FalsePositiveRate() float64
    
    // 重置过滤器
    Reset()
    
    // 将过滤器序列化为字节数组
    Save() []byte
    
    // 从字节数组加载过滤器
    Load(data []byte) error
}
```

## 🧩 实现结构

```go
type BloomFilter struct {
    m     uint64   // 位数组大小
    k     uint     // 哈希函数数量
    bits  []uint64 // 位数组，每64位为一组
    n     uint64   // 已添加元素数量
    seeds []uint32 // 哈希函数种子，保证持久性
}
```

## 🏗️ 创建布隆过滤器

两种创建方式：

### 1. 🔢 指定位数组大小和哈希函数数量

```go
filter := filter.NewBloomFilter(1024, 3)
```

### 2. 📊 指定预期元素数量和目标假阳性率

```go
filter := filter.NewBloomFilterWithParams(10000, 0.01)
```

## 🔰 基本使用

```go
// 添加元素
filter.Add([]byte("key1"))
filter.Add([]byte("key2"))

// 检查元素是否可能存在
existsKey1 := filter.Contains([]byte("key1"))     // true
existsKey3 := filter.Contains([]byte("key3"))     // false（除非假阳性）

// 查看当前假阳性率
fpRate := filter.FalsePositiveRate()

// 重置过滤器
filter.Reset()

// 序列化与反序列化
serialized := filter.Save()
newFilter := filter.NewBloomFilter(0, 0)
newFilter.Load(serialized)
```

## ⚙️ 工作原理

1. **➕ 添加元素**：计算k个哈希值，将位数组中对应位置设为1
2. **🔍 查询元素**：计算k个哈希值，检查位数组对应位置是否都为1
   - 如果任一位为0，则元素肯定不存在
   - 如果所有位都为1，则元素可能存在（有一定概率是假阳性）

## 🧮 哈希函数

布隆过滤器使用MurmurHash3算法生成哈希值，通过不同的种子生成多个独立的哈希函数。种子值是固定的，以确保序列化和反序列化的一致性。

## ⚡ 性能优化

- **⚖️ 位数组大小(m)与哈希函数数量(k)的平衡**：m和k的选择直接影响假阳性率和性能
- **📦 批量操作**：对于批量添加场景，可以先进行添加再保存，避免频繁序列化
- **💾 内存布局**：使用[]uint64存储位数组，提高位操作效率

## 🔗 与SST文件集成

每个SST文件包含一个布隆过滤器，用于快速判断键是否可能存在于文件中，从而避免不必要的磁盘读取。这显著提高了读取性能，特别是对于不存在的键的查询。 