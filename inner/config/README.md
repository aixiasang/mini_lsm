# ⚙️ 配置模块

配置模块提供了LSM-Tree数据库的全局配置管理，允许用户自定义各种性能和行为参数。

## 🧰 核心结构

```go
type Config struct {
    // 🗄️ SST相关配置
    BlockSize      int  // 数据块大小
    BlockRestartInterval int  // 块重启间隔
    
    // 🧠 内存表相关配置
    MemtableSize   int  // 内存表大小上限
    
    // 📒 WAL相关配置
    WalDir         string  // WAL目录
    WalSize        int64   // WAL大小上限
    AutoSync       bool    // 是否自动同步
    
    // 🔍 过滤器相关配置
    BloomBitsPerKey int  // 布隆过滤器每个键的位数
    
    // 🌲 LSM-Tree相关配置
    MaxLevels      int    // 最大层级数
    Level0Size     int    // Level 0大小上限
    MaxFileSize    int64  // 单个文件大小上限
    SizeRatio      int    // 层级大小比例
    
    // 📂 目录配置
    DBDir          string  // 数据库目录
    
    // 🧹 压缩相关配置
    Compaction     string  // 压缩策略
}
```

## 🚀 使用方法

### 🏭 创建默认配置

```go
// 创建默认配置
conf := config.DefaultConfig()
```

### 🔧 自定义配置

```go
// 创建自定义配置
conf := &config.Config{
    BlockSize: 4096,
    MemtableSize: 4 * 1024 * 1024,  // 4MB
    WalDir: "/data/wal",
    DBDir: "/data/db",
    MaxLevels: 7,
    // 其他配置...
}
```

## 📊 配置项说明

### 🗄️ SST文件配置

- **BlockSize**: 单个数据块大小（字节）
  - 较大的值减少索引大小，提高顺序读性能
  - 较小的值提高随机读性能
  - 推荐范围：1KB~16KB

- **BlockRestartInterval**: 块重启间隔（键值对数量）
  - 控制前缀压缩的粒度，影响文件大小和读取性能
  - 推荐范围：8~32

### 🧠 内存表配置

- **MemtableSize**: 内存表大小上限（字节）
  - 较大的值减少刷盘频率，提高写性能
  - 较小的值减少内存使用，降低故障恢复时间
  - 推荐范围：1MB~64MB

### 📒 WAL配置

- **WalDir**: WAL文件目录
  - 可以配置在独立的磁盘上以提高性能

- **WalSize**: 单个WAL文件大小上限（字节）
  - 较大的值减少WAL切换频率
  - 较小的值加快恢复速度
  - 推荐范围：1MB~64MB

- **AutoSync**: 是否自动同步WAL
  - true：每次写入后立即同步到磁盘，保证数据安全但降低性能
  - false：系统决定同步时机，提高性能但可能丢失最近的写入

### 🔍 过滤器配置

- **BloomBitsPerKey**: 布隆过滤器每个键的位数
  - 较大的值降低假阳性概率，但增加内存使用
  - 较小的值节省内存，但增加不必要的磁盘读取
  - 推荐范围：8~16

### 🌲 LSM-Tree配置

- **MaxLevels**: 最大层级数
  - 控制LSM-Tree的深度，影响空间放大和读性能
  - 推荐范围：5~7

- **Level0Size**: Level 0的大小上限（字节）
  - 控制Level 0到Level 1的合并触发阈值
  - 推荐范围：4MB~64MB

- **MaxFileSize**: 单个SST文件的大小上限（字节）
  - 控制单个SST文件的大小，影响压缩和查询性能
  - 推荐范围：8MB~64MB

- **SizeRatio**: 层级大小比例
  - 每一层相对于上一层的大小倍数
  - 推荐值：10

### 📂 目录配置

- **DBDir**: 数据库文件目录
  - 数据文件、元数据等存储位置

### 🧹 压缩配置

- **Compaction**: 压缩策略
  - "leveled": 分层压缩，适合点查询场景
  - "tiered": 分层压缩，适合批量写入场景
  - "hybrid": 混合策略，平衡读写性能

## 🔄 动态配置

某些配置项支持在运行时动态调整：

```go
// 修改内存表大小
db.SetConfig("MemtableSize", 8 * 1024 * 1024)  // 8MB

// 修改WAL同步策略
db.SetConfig("AutoSync", true)
```

## 🎯 最佳实践

- 💻 **开发环境**: 优先配置较小的文件大小和内存使用，加快调试周期
  ```go
  conf := &config.Config{
      MemtableSize: 1 * 1024 * 1024,  // 1MB
      WalSize: 1 * 1024 * 1024,       // 1MB
      MaxFileSize: 4 * 1024 * 1024,   // 4MB
      AutoSync: true,
  }
  ```

- 🖥️ **生产环境**: 根据硬件配置和工作负载调整参数
  ```go
  conf := &config.Config{
      MemtableSize: 32 * 1024 * 1024,  // 32MB
      WalSize: 16 * 1024 * 1024,       // 16MB
      MaxFileSize: 32 * 1024 * 1024,   // 32MB
      AutoSync: false,                  // 根据数据重要性决定
      BloomBitsPerKey: 10,
      SizeRatio: 10,
  }
  ```

- 📊 **读密集型场景**: 优化读取性能
  ```go
  conf := &config.Config{
      BlockSize: 2048,                 // 较小的块大小
      BloomBitsPerKey: 14,             // 较大的布隆过滤器
      Compaction: "leveled",           // 分层压缩
  }
  ```

- ✍️ **写密集型场景**: 优化写入性能
  ```go
  conf := &config.Config{
      MemtableSize: 64 * 1024 * 1024,  // 较大的内存表
      AutoSync: false,                 // 非强制同步
      Compaction: "tiered",            // 分级压缩
  }
  ``` 