# ⚠️ 错误处理模块

错误处理模块定义了LSM-Tree数据库中使用的标准错误类型，提供了统一的错误处理机制。

## 🔍 错误类型

```go
var (
    // 🚫 通用错误
    ErrNotFound     = errors.New("not found")           // 未找到
    ErrKeyEmpty     = errors.New("key is empty")        // 键为空
    ErrKeyNotFound  = errors.New("key not found")       // 键不存在
    ErrKeyTooLarge  = errors.New("key too large")       // 键过长
    ErrValueTooLarge = errors.New("value too large")    // 值过长
    
    // 📁 文件相关错误
    ErrFileNotExist  = errors.New("file not exist")     // 文件不存在
    ErrFileCorrupted = errors.New("file corrupted")     // 文件损坏
    ErrCrcMismatch   = errors.New("crc mismatch")       // CRC校验不匹配
    
    // 📊 数据结构错误
    ErrTableFull     = errors.New("table is full")      // 表已满
    ErrIterInvalid   = errors.New("iterator invalid")   // 迭代器无效
    
    // 🔒 并发控制错误
    ErrDBClosed      = errors.New("database closed")    // 数据库已关闭
    ErrWriteConflict = errors.New("write conflict")     // 写入冲突
    
    // 🔧 配置错误
    ErrInvalidConfig = errors.New("invalid config")     // 无效配置
    
    // 🧠 内存错误
    ErrOutOfMemory   = errors.New("out of memory")      // 内存不足
)
```

## 🚀 使用方法

### 🔄 返回标准错误

```go
func Get(key []byte) ([]byte, error) {
    if len(key) == 0 {
        return nil, myerror.ErrKeyEmpty
    }
    
    // 键不存在的情况
    if notFound {
        return nil, myerror.ErrKeyNotFound
    }
    
    // 文件损坏的情况
    if corrupted {
        return nil, myerror.ErrFileCorrupted
    }
    
    return value, nil
}
```

### 🔍 错误检查

```go
value, err := db.Get(key)
if err != nil {
    if errors.Is(err, myerror.ErrKeyNotFound) {
        // 处理键不存在的情况
    } else if errors.Is(err, myerror.ErrFileCorrupted) {
        // 处理文件损坏的情况
    } else {
        // 处理其他错误
    }
}
```

### 🧩 错误包装

```go
func ReadFromFile(path string) ([]byte, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, myerror.ErrFileNotExist
        }
        return nil, fmt.Errorf("read file error: %w", err)
    }
    return data, nil
}
```

## 💡 错误处理最佳实践

### 1. 🔄 使用标准错误

始终使用模块定义的标准错误，而不是创建新的错误实例。这样可以通过`errors.Is`进行准确的错误比较。

```go
// 正确 ✅
return nil, myerror.ErrKeyNotFound

// 错误 ❌
return nil, errors.New("key not found")
```

### 2. 📋 提供上下文信息

包装错误以提供更丰富的上下文信息，但保留原始错误类型。

```go
// 包含上下文信息
return fmt.Errorf("failed to read key %q: %w", key, myerror.ErrFileCorrupted)
```

### 3. 🔍 合适的错误粒度

选择合适的错误粒度，既不过于宽泛也不过于具体。

```go
// 过于宽泛 ❌
return nil, errors.New("operation failed")

// 过于具体 ❌
return nil, errors.New("b-tree node with id 42 at level 3 has invalid child pointer")

// 合适粒度 ✅
return nil, myerror.ErrFileCorrupted
```

### 4. 📊 错误分类

根据错误的性质进行分类，使处理逻辑更清晰。

```go
func handleError(err error) {
    switch {
    case isTemporaryError(err):
        // 可以重试的临时错误
        retry()
    case isUserError(err):
        // 由用户输入导致的错误
        reportToUser(err)
    case isSystemError(err):
        // 系统级错误
        logAndAlert(err)
    default:
        // 未知错误
        panic(err)
    }
}

// 判断是否为临时错误
func isTemporaryError(err error) bool {
    return errors.Is(err, myerror.ErrWriteConflict)
}

// 判断是否为用户错误
func isUserError(err error) bool {
    return errors.Is(err, myerror.ErrKeyEmpty) ||
           errors.Is(err, myerror.ErrKeyTooLarge) ||
           errors.Is(err, myerror.ErrValueTooLarge)
}

// 判断是否为系统错误
func isSystemError(err error) bool {
    return errors.Is(err, myerror.ErrFileCorrupted) ||
           errors.Is(err, myerror.ErrOutOfMemory) ||
           errors.Is(err, myerror.ErrCrcMismatch)
}
```

### 5. 🛡️ 错误恢复策略

为不同类型的错误制定恢复策略。

```go
func get(key []byte) ([]byte, error) {
    for retries := 0; retries < 3; retries++ {
        value, err := tryGet(key)
        if err == nil {
            return value, nil
        }
        
        // 临时错误可以重试
        if errors.Is(err, myerror.ErrWriteConflict) {
            time.Sleep(100 * time.Millisecond)
            continue
        }
        
        // 永久性错误直接返回
        return nil, err
    }
    
    return nil, fmt.Errorf("failed after multiple retries")
}
```

## 🔄 与其他模块的集成

错误处理模块与其他所有模块紧密集成，提供了统一的错误类型和处理机制。

- 🗄️ **SST模块**: 使用文件相关错误类型报告SST文件访问和解析问题
- 📒 **WAL模块**: 使用CRC校验错误报告日志损坏情况
- 🧠 **MemTable模块**: 使用数据结构错误报告内存表操作异常
- 🌲 **LSM-Tree模块**: 集成所有错误类型，提供统一的错误处理入口 