# 📚 排序字符串表 (SST)

SST(Sorted String Table，排序字符串表)是LSM-Tree数据库中的核心磁盘数据结构，它以不可变、有序的形式存储键值对数据。SST文件提供了高效的读取和范围查询能力，同时支持增量合并以优化存储空间。

## 🧩 核心组件

### 1. ✍️ SSTWriter

`SSTWriter`用于创建新的SST文件，它将内存中的有序键值对写入磁盘，并生成必要的索引和元数据。

```go
type SSTWriter struct {
    conf       *config.Config // 配置
    path       string         // 文件路径
    fp         *os.File       // 文件指针
    dataBlocks []*dataBlock   // 数据块
    curBlock   *dataBlock     // 当前数据块
    filter     filter.Filter  // 布隆过滤器
}
```

### 2. 📖 SSTReader

`SSTReader`用于读取现有的SST文件，支持单键查询和范围扫描操作。

```go
type SSTReader struct {
    conf   *config.Config   // 配置
    path   string           // 文件路径
    fp     *os.File         // 文件指针
    footer *Footer          // 文件尾部元数据
    index  []*IndexEntry    // 索引项
    filter filter.Filter    // 布隆过滤器
    size   int64            // 文件大小
}
```

### 3. 🔗 Node

`Node`封装了SST文件的读取操作，作为LSM-Tree各层级数据节点的基本单元。

```go
type Node struct {
    conf     *config.Config // 配置
    path     string         // 文件路径
    level    int            // 层级
    id       int32          // 节点ID
    minKey   []byte         // 最小键
    maxKey   []byte         // 最大键
    reader   *SSTReader     // SST读取器
    dataSize int64          // 数据大小
}
```

## 📄 文件格式

SST文件由以下几个部分组成：

```
+----------------+----------------+----------------+----------------+
|    数据部分     |    索引部分     |    过滤器部分   |     文件尾      |
+----------------+----------------+----------------+----------------+
```

### 📊 数据部分

数据部分由多个数据块组成，每个数据块包含有序的键值对。

数据块格式：
```
+----------------+----------------+----------------+----------------+
|  键值对数量(4B)  |     键1长度     |     值1长度     |     键1数据     |
+----------------+----------------+----------------+----------------+
|     值1数据     |     键2长度     |     值2长度     |     键2数据     |
+----------------+----------------+----------------+----------------+
|      ...       |      ...       |      ...       |      ...       |
+----------------+----------------+----------------+----------------+
```

### 🔍 索引部分

索引部分包含指向各个数据块的索引项，用于快速定位数据。

索引项格式：
```
+----------------+----------------+----------------+----------------+
|    键长度(4B)   |     键数据     |    偏移量(8B)   |    大小(4B)     |
+----------------+----------------+----------------+----------------+
```

### 🔬 过滤器部分

存储布隆过滤器数据，用于快速判断键是否可能存在于文件中。

### 📝 文件尾

包含各部分的元数据信息，如偏移量、大小等。

```
+----------------+----------------+----------------+----------------+
| 索引偏移量(8B)  | 索引大小(4B)   | 过滤器偏移量(8B) | 过滤器大小(4B)  |
+----------------+----------------+----------------+----------------+
|  魔数(4B)      |
+----------------+
```

## 🔧 主要功能

### 📝 创建SST文件

```go
writer, err := sst.NewSSTWriter(conf, filePath)
if err != nil {
    return err
}

// 添加键值对
writer.Add([]byte("key1"), []byte("value1"))
writer.Add([]byte("key2"), []byte("value2"))

// 刷新并关闭
writer.Flush()
writer.Close()
```

### 📖 读取SST文件

```go
reader, err := sst.NewSSTReader(conf, filePath)
if err != nil {
    return err
}

// 获取特定键的值
value, err := reader.Get([]byte("key1"))

// 创建迭代器进行范围查询
iter := reader.Iterator()
for iter.SeekToFirst(); iter.Valid(); iter.Next() {
    key := iter.Key()
    value := iter.Value()
    // 处理键值对
}
```

### 🏗️ 创建节点

```go
reader, _ := sst.NewSSTReader(conf, filePath)
node, err := sst.NewNode(conf, filePath, level, id, reader)
if err != nil {
    return err
}

// 使用节点
value, err := node.Get([]byte("key1"))
```

## ⚡ 性能优化

### 🧩 数据块分割与合并

- 📏 数据块大小配置可调整，平衡查询性能和空间利用率
- 🧮 较小的块有利于随机读取，较大的块有利于顺序扫描

### 🔍 布隆过滤器

- 🚀 每个SST文件包含一个布隆过滤器，快速过滤不存在的键
- ⚖️ 过滤器参数可配置，权衡空间使用和假阳性率

### 🔎 二分查找

- 🔍 索引部分使用二分查找快速定位数据块
- 🔬 数据块内使用二分查找定位具体键值对

## 🔗 与其他模块集成

- 📒 与WAL模块协同工作，确保数据一致性
- 🔍 与过滤器模块集成，提供高效查询
- 🔄 与压缩模块交互，实现分层存储和文件合并 