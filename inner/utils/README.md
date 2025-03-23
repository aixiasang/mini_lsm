# 🛠️ 工具模块

工具模块提供了LSM-Tree数据库中使用的各种通用功能和辅助工具，为其他模块提供基础服务支持。

## 📦 主要组件

### 1. 🧮 CRC校验

```go
// 计算数据的CRC32校验和
func CRC32(data []byte) uint32

// 验证数据的CRC32校验和
func VerifyCRC32(data []byte, expected uint32) bool
```

### 2. ⏱️ 时间工具

```go
// 获取当前时间戳（纳秒精度）
func NowNano() int64

// 获取格式化的当前时间字符串
func TimeString() string
```

### 3. 📁 文件操作

```go
// 确保目录存在，如不存在则创建
func EnsureDir(dir string) error

// 获取目录中的所有文件（可选按后缀过滤）
func ListFiles(dir, suffix string) ([]string, error)

// 安全地原子写入文件
func AtomicWrite(filename string, data []byte) error
```

### 4. 🔄 字节操作

```go
// 整数与字节数组的转换
func EncodeUint32(n uint32) []byte
func DecodeUint32(b []byte) uint32
func EncodeUint64(n uint64) []byte
func DecodeUint64(b []byte) uint64

// 字节数组比较
func CompareBytes(a, b []byte) int
```

### 5. 🔒 同步工具

```go
// 读写锁包装器，支持超时和统计
type RWMutex struct {
    // ...
}

// 带有超时功能的锁
func (rw *RWMutex) LockWithTimeout(timeout time.Duration) bool
func (rw *RWMutex) UnlockWithTimeout(timeout time.Duration) bool
```

### 6. 📊 度量统计

```go
// 操作计数器
type Counter struct {
    // ...
}

// 延迟统计
type Latency struct {
    // ...
}

// 吞吐量计算
type Throughput struct {
    // ...
}
```

## 🚀 使用示例

### 🧮 CRC校验示例

```go
// 计算数据的CRC32校验和
data := []byte("hello world")
checksum := utils.CRC32(data)

// 存储数据和校验和
record := append(data, utils.EncodeUint32(checksum)...)

// 验证数据完整性
storedData := record[:len(record)-4]
storedChecksum := utils.DecodeUint32(record[len(record)-4:])
if !utils.VerifyCRC32(storedData, storedChecksum) {
    // 数据损坏
    return myerror.ErrCrcMismatch
}
```

### 📁 文件操作示例

```go
// 确保目录存在
if err := utils.EnsureDir("./data"); err != nil {
    log.Fatalf("无法创建数据目录: %v", err)
}

// 列出所有WAL文件
walFiles, err := utils.ListFiles("./data", ".wal")
if err != nil {
    log.Fatalf("无法列出WAL文件: %v", err)
}

// 安全地写入数据
data := []byte("important data")
if err := utils.AtomicWrite("./data/metadata.json", data); err != nil {
    log.Fatalf("写入元数据失败: %v", err)
}
```

### 🔄 字节操作示例

```go
// 编码操作
fileSize := uint64(1024 * 1024)
encodedSize := utils.EncodeUint64(fileSize)

// 解码操作
decodedSize := utils.DecodeUint64(encodedSize)

// 比较键
key1 := []byte("apple")
key2 := []byte("banana")
if utils.CompareBytes(key1, key2) < 0 {
    // key1 < key2
}
```

### 🔒 同步工具示例

```go
var rwLock utils.RWMutex

// 读操作
func read() {
    rwLock.RLock()
    defer rwLock.RUnlock()
    // 读取共享资源
}

// 写操作（带超时）
func writeWithTimeout() bool {
    if !rwLock.LockWithTimeout(100 * time.Millisecond) {
        // 获取锁超时
        return false
    }
    defer rwLock.Unlock()
    // 修改共享资源
    return true
}
```

### 📊 度量统计示例

```go
// 初始化计数器
readCounter := &utils.Counter{}
writeCounter := &utils.Counter{}

// 记录操作
readCounter.Inc()  // 读操作计数
writeCounter.Add(10)  // 批量写入计数

// 初始化延迟统计
getLatency := &utils.Latency{}

// 记录操作延迟
start := time.Now()
// ... 执行Get操作 ...
getLatency.Record(time.Since(start))

// 获取统计信息
avgLatency := getLatency.Average()
maxLatency := getLatency.Max()
p99Latency := getLatency.Percentile(99)
```

## 💎 设计原则

### 1. 🎯 通用性与可重用性

工具模块中的功能设计为通用组件，可在数据库的不同部分重复使用，减少代码重复。

### 2. 🧩 单一职责

每个工具函数或结构体专注于解决一个特定问题，符合单一职责原则。

### 3. ⚡ 性能优先

工具函数经过优化以确保高性能，因为它们往往在关键路径上被频繁调用。

### 4. 🛡️ 错误处理

所有可能失败的操作都返回明确的错误信息，便于上层模块进行适当处理。

### 5. 📏 零依赖

工具模块尽量避免外部依赖，仅使用Go标准库，确保可移植性和稳定性。

## 🔗 与其他模块的关系

工具模块是基础设施，被其他所有模块使用：

- 📒 **WAL模块**: 使用CRC校验和文件操作函数
- 🗄️ **SST模块**: 使用字节操作和文件处理功能
- 🧠 **MemTable模块**: 使用字节比较和同步工具
- 🌲 **LSM-Tree模块**: 集成所有工具功能 