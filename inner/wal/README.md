# 📒 预写日志 (WAL)

WAL(Write-Ahead Logging，预写日志)模块是LSM-Tree数据库的持久化和崩溃恢复基础。它确保所有数据修改在应用到主数据结构之前，先被记录到持久存储中，从而实现数据的耐久性和崩溃恢复能力。

## 🌟 核心功能

- **💾 持久化保证**：所有写操作在返回前先持久化到磁盘
- **🛠️ 崩溃恢复**：系统崩溃后可从WAL中恢复数据
- **⚛️ 原子性操作**：保证操作的原子性，要么全部成功，要么全部失败
- **🔄 事务支持**：为数据库的事务处理提供基础

## 📋 主要结构

```go
type Wal struct {
    conf   *config.Config // 配置
    fileId uint32         // 文件ID
    offset uint32         // 偏移量
    fp     *os.File       // 文件
    mu     sync.RWMutex   // 互斥锁
}
```

## 📜 WAL记录格式

WAL文件中的每条记录包含以下组成部分：

- **📌 记录类型**：标识记录的类型(普通/删除)
- **📏 键长度**：键的字节长度
- **📐 值长度**：值的字节长度
- **🔑 键内容**：实际的键数据
- **📝 值内容**：实际的值数据
- **🔒 CRC校验**：用于验证记录完整性的校验和

## 🛠️ 主要方法

### 🆕 创建新的WAL

```go
func NewWal(conf *config.Config, fileId uint32) (*Wal, error)
```

此方法创建一个新的WAL实例，包括打开或创建对应的日志文件。

### ✍️ 写入记录

```go
func (w *Wal) Write(key, value []byte) error
```

将键值对写入WAL文件，包括计算CRC校验和。如果配置了自动同步，则会立即调用`fsync`确保数据持久化。

### 📖 读取全部记录

```go
func (w *Wal) ReadAll(memTable memtable.MemTable) error
```

从WAL文件中读取所有记录并重建内存表，用于系统启动时的恢复过程。

### ⚙️ 管理方法

- **📊 Size()**：获取当前WAL文件大小
- **🔄 Sync()**：手动同步文件到磁盘
- **🔢 FileId()**：获取当前WAL文件ID
- **🚪 Close()**：关闭WAL文件
- **🗑️ Delete()**：删除WAL文件

## 🔰 使用示例

```go
// 创建WAL
wal, err := wal.NewWal(conf, fileId)
if err != nil {
    // 处理错误
}

// 写入记录
if err := wal.Write([]byte("key"), []byte("value")); err != nil {
    // 处理错误
}

// 检查大小并根据需要轮转
if wal.Size() > conf.WalSize {
    // 执行WAL轮转
}

// 关闭WAL
wal.Close()
```

## ⚡ 性能考虑

为了平衡性能和持久性，WAL模块提供了以下配置选项：

- **🔄 AutoSync**：是否在每次写入后自动同步到磁盘
- **📏 WalSize**：单个WAL文件的最大大小，超过此大小将触发轮转

## 🛟 恢复过程

系统启动时，将执行以下步骤恢复数据：

1. 🔍 扫描WAL目录，按顺序加载所有WAL文件
2. 📥 对每个WAL文件调用`ReadAll`方法重建内存表
3. �� 重建完成后，系统可以开始正常工作 