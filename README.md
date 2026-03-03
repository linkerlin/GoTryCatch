[English](#english) | [中文](#chinese)

<a id="english"></a>

# GoTryCatch

[![Version](https://img.shields.io/badge/version-1.3.0-blue.svg)](https://github.com/linkerlin/gotrycatch)
[![Go](https://img.shields.io/badge/go-%3E%3D1.18-green.svg)](https://golang.org)

A type-safe exception handling library based on Go generics that brings try-catch-like capabilities to Go.

## Features

- 🎯 **Type-safe**: Uses Go generics to ensure type-safe exception handling
- 🔗 **Partial chaining**: Supports chaining for `CatchAny` and `Finally`
- 🏷️ **Multiple error types**: Built-in common error types (validation, database, network, business logic, auth, config, rate limit)
- 📊 **Structured output**: All errors support `ToMap()` and `ToJSON()` for easy parsing by agents/logs
- 🔍 **Rich context**: Errors include file, line, function, timestamp, and stack trace
- 🔄 **Finally support**: Guarantees cleanup code execution
- 🎁 **TryWithResult**: Support for functions with return values and `OnSuccess`/`OnError`/`OrElse` patterns
- 🐛 **Debug mode**: Optional debug logging for type matching issues
- 📦 **Zero dependency**: Pure Go implementation, no external dependencies
- 🚀 **High performance**: Built on Go's panic/recover with minimal overhead

## Important Note

⚠️ **Chaining limitation**: Due to Go's limitation that methods cannot have generic type parameters, you cannot write `tb.Catch[ErrorType](handler)`. Use the functional form instead: `gotrycatch.Catch[ErrorType](tb, handler)`. `CatchAny` and `Finally` do support chaining.

## Installation

```bash
go get github.com/linkerlin/gotrycatch
```

## Quick Start

### Basic usage

```go
package main

import (
    "fmt"
    "github.com/linkerlin/gotrycatch"
    "github.com/linkerlin/gotrycatch/errors"
)

func main() {
    tb := gotrycatch.Try(func() {
        // Code that may panic
        gotrycatch.Throw(errors.NewValidationError("email", "invalid format", 1001))
    })

    tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
        fmt.Printf("Validation error: %s (field: %s, code: %d)\n", err.Message, err.Field, err.Code)
    })

    tb.Finally(func() {
        fmt.Println("Cleanup done")
    })
}
```

### TryWithResult - Functions with return values

```go
// Execute function that returns a value
tb := gotrycatch.TryWithResult(func() int {
    return computeValue()
})

// Success callback
tb.OnSuccess(func(result int) {
    fmt.Println("Result:", result)
})

// Error callback
tb.OnError(func(err interface{}) {
    fmt.Println("Error:", err)
})

// Get result with default value
result := tb.OrElse(0)

// Or lazy evaluation of default
result := tb.OrElseGet(func() int { return computeDefault() })
```

### Handling multiple error types

```go
tb := gotrycatch.Try(func() {
    processUserData()
})

tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    fmt.Printf("Validation failed: %s\n", err.Message)
})

tb = gotrycatch.Catch[errors.DatabaseError](tb, func(err errors.DatabaseError) {
    fmt.Printf("Database error: %s on table %s\n", err.Operation, err.Table)
})

tb = gotrycatch.Catch[errors.NetworkError](tb, func(err errors.NetworkError) {
    if err.Timeout {
        fmt.Printf("Network timeout: %s\n", err.URL)
    } else {
        fmt.Printf("Network error %d: %s\n", err.StatusCode, err.URL)
    }
})

tb = tb.CatchAny(func(err interface{}) {
    fmt.Printf("Unknown error: %v\n", err)
})

tb.Finally(func() {
    fmt.Println("Processing done")
})
```

### State query and debugging

```go
tb := gotrycatch.Try(func() {
    riskyOperation()
})

// Query state
if tb.HasError() {
    fmt.Printf("Error type: %s\n", tb.GetErrorType())
    fmt.Printf("Error value: %v\n", tb.GetError())
}

if !tb.IsHandled() {
    // Decide how to handle based on error type
    switch tb.GetErrorType() {
    case "errors.ValidationError":
        // Handle validation error
    default:
        tb = tb.CatchAny(func(err interface{}) {
            logUnknownError(err)
        })
    }
}

// Enable debug mode to trace type matching
gotrycatch.SetDebug(true)
```

### Structured error output (Agent-friendly)

```go
tb := gotrycatch.Catch[errors.BusinessLogicError](tb, func(err errors.BusinessLogicError) {
    // JSON output for logging/agents
    jsonData, _ := err.ToJSON()
    log.Printf("ERROR: %s", string(jsonData))
    // Output: {"type":"BusinessLogicError","rule":"inventory_check","details":"Out of stock","file":"main.go","line":42,"function":"processOrder","timestamp":"2024-01-15T10:30:00Z","stack":[...]}
})
```

### Assertion helpers

```go
// Assert condition, throw error if false
gotrycatch.Assert(value != "", errors.NewValidationError("value", "cannot be empty", 1001))

// Assert no error, wrap and throw if error exists
gotrycatch.AssertNoError(err, "database operation failed")
```

## Built-in Error Types

All error types include: `File`, `Line`, `Function`, `Timestamp`, `Stack`

| Type | Specific Fields | Constructor | Use Case |
|------|-----------------|-------------|----------|
| `ValidationError` | Field, Message, Code | `NewValidationError(field, message, code)` | Data validation errors |
| `DatabaseError` | Operation, Table, Cause | `NewDatabaseError(operation, table, cause)` | Database operation errors |
| `NetworkError` | URL, StatusCode, Timeout | `NewNetworkError(url, code)` | HTTP errors |
| `NetworkError` | URL, Timeout | `NewNetworkTimeoutError(url)` | Network timeouts |
| `BusinessLogicError` | Rule, Details | `NewBusinessLogicError(rule, details)` | Business rule violations |
| `ConfigError` | Key, Value, Reason | `NewConfigError(key, value, reason)` | Configuration errors |
| `AuthError` | Operation, User, Reason | `NewAuthError(operation, user, reason)` | Authentication/authorization errors |
| `RateLimitError` | Resource, Limit, Current, RetryAfter | `NewRateLimitError(resource, limit, current, retryAfter)` | Rate limiting errors |

### Error methods

```go
var err errors.ValidationError

err.Error()     // string - Full error description with location
err.ToMap()     // map[string]interface{} - Structured data
err.ToJSON()    // ([]byte, error) - JSON output
err.Unwrap()    // error - Underlying error (DatabaseError returns Cause)
err.Is(target)  // bool - Error matching
```

## API Reference

### Core Functions

| Function/Method | Signature | Description |
|-----------------|----------|-------------|
| `Try` | `func Try(fn func()) *TryBlock` | Execute function and capture any panic |
| `Catch[T]` | `func Catch[T any](tb *TryBlock, handler func(T)) *TryBlock` | Handle panics of type T |
| `CatchWithReturn[T]` | `func CatchWithReturn[T any](tb *TryBlock, handler func(T) interface{}) (interface{}, *TryBlock)` | Handle and return value |
| `CatchAny` | `func (tb *TryBlock) CatchAny(handler func(interface{})) *TryBlock` | Handle any unhandled panic |
| `Finally` | `func (tb *TryBlock) Finally(fn func())` | Execute cleanup code |

### TryBlock State Query

| Method | Return Type | Description |
|--------|-------------|-------------|
| `HasError()` | `bool` | Whether a panic was captured |
| `GetError()` | `interface{}` | Get the panic value |
| `GetErrorType()` | `string` | Get error type name (e.g., "errors.ValidationError") |
| `IsHandled()` | `bool` | Whether error was handled |
| `String()` | `string` | Friendly string representation |

### TryWithResult

| Function/Method | Signature | Description |
|-----------------|----------|-------------|
| `TryWithResult` | `func TryWithResult[T any](fn func() T) *TryBlockWithResult[T]` | Execute function with return value |
| `CatchWithResult` | `func CatchWithResult[T, E any](tb *TryBlockWithResult[T], handler func(E)) *TryBlockWithResult[T]` | Typed catch for TryWithResult |
| `CatchAnyWithResult` | `func CatchAnyWithResult[T any](tb *TryBlockWithResult[T], handler func(interface{})) *TryBlockWithResult[T]` | Catch any for TryWithResult |
| `GetResult()` | `T` | Get the result value |
| `OnSuccess` | `func (tb *TryBlockWithResult[T]) OnSuccess(fn func(T)) *TryBlockWithResult[T]` | Callback on success |
| `OnError` | `func (tb *TryBlockWithResult[T]) OnError(fn func(interface{})) *TryBlockWithResult[T]` | Callback on error |
| `OrElse` | `func (tb *TryBlockWithResult[T]) OrElse(defaultValue T) T` | Get result or default |
| `OrElseGet` | `func (tb *TryBlockWithResult[T]) OrElseGet(supplier func() T) T` | Get result or lazy default |

### Debug & Assertions

| Function | Signature | Description |
|----------|-----------|-------------|
| `SetDebug` | `func SetDebug(enabled bool)` | Enable/disable debug logging |
| `IsDebug` | `func IsDebug() bool` | Check debug mode status |
| `Throw` | `func Throw(err interface{})` | Throw an exception (panic) |
| `Assert` | `func Assert(condition bool, err interface{})` | Assert condition, throw if false |
| `AssertNoError` | `func AssertNoError(err error, msg string)` | Assert no error, throw with message if error |

## Best Practices

1. **Order Catch blocks by specificity**: Most specific types first, generic types later
2. **Always use Finally**: Ensure resource cleanup
3. **Use predefined error types**: Prefer structured errors over raw strings/numbers
4. **Use CatchAny as fallback**: Handle unexpected error types gracefully
5. **Enable debug mode for troubleshooting**: Use `SetDebug(true)` when type matching doesn't work as expected
6. **Use ToJSON for logging**: Structured output is easier to parse and analyze

## Performance

- Built on Go's panic/recover; cost occurs only when exceptions actually happen
- Near-zero overhead on the normal execution path
- Try-Catch blocks can be nested without significant impact

## Compatibility

- Requires Go 1.18+ (generics)
- Fully compatible with the standard library
- Can coexist with existing error-handling code
- Thread-safe for concurrent use

## FAQ

### Q: Why can't I use full method chaining?

A: Because methods cannot have generic type parameters in Go. So this is not supported:
```go
// ❌ Not supported
tb := gotrycatch.Try(func() { ... }).Catch[ErrorType](handler)
```

Use the functional form instead:
```go
// ✅ Correct
tb := gotrycatch.Try(func() { ... })
tb = gotrycatch.Catch[ErrorType](tb, handler)
```

But `CatchAny` and `Finally` support chaining:
```go
// ✅ Supported
tb.CatchAny(handler).Finally(cleanup)
```

### Q: How do I debug type matching issues?

A: Enable debug mode:
```go
gotrycatch.SetDebug(true)
// Output: [gotrycatch] Catch: type errors.ValidationError does not match target type int
// Output: [gotrycatch] Catch: type errors.ValidationError matched, calling handler
```

### Q: What happens to unhandled errors?

A: Unhandled errors are re-thrown after `Finally` executes. Always use `CatchAny` as a fallback if you don't want panics to propagate.

## Examples

See the `examples/` directory for more:

- Basic usage
- Handling multiple error types
- Nested exception handling
- TryWithResult patterns
- Structured error logging

### Run examples

```bash
# Quick demo
go run ./cmd/demo

# Full examples
go run ./examples

# Run tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

## License

MIT License

---

<a id="chinese"></a>

# GoTryCatch

一个基于 Go 泛型的类型安全异常处理库，为 Go 带来类似 try-catch 的异常处理能力。

## 特性

- 🎯 **类型安全**: 使用 Go 泛型确保异常处理的类型安全
- 🔗 **部分链式调用**: 支持 `CatchAny` 和 `Finally` 的链式调用
- 🏷️ **七种异常类型**: 内置常用异常类型（验证、数据库、网络、业务逻辑、认证、配置、限流）
- 📊 **结构化输出**: 所有错误支持 `ToMap()` 和 `ToJSON()`，便于 Agent 解析和日志记录
- 🔍 **丰富上下文**: 错误自动包含文件名、行号、函数名、时间戳和调用堆栈
- 🔄 **Finally 支持**: 保证清理代码的执行
- 🎁 **TryWithResult**: 支持带返回值的函数，提供 `OnSuccess`/`OnError`/`OrElse`/`OrElseGet` 模式
- 🐛 **调试模式**: 可选的调试日志，帮助排查类型匹配问题
- ⚡ **断言辅助**: `Assert` 和 `AssertNoError` 简化条件检查
- 📦 **零依赖**: 纯 Go 实现，无外部依赖
- 🚀 **高性能**: 基于 Go 的 panic/recover 机制，性能开销极小

## 重要说明

⚠️ **链式调用限制**: 由于 Go 语言的限制，方法不能有泛型类型参数，因此不能直接写 `tb.Catch[ErrorType](handler)`。需要使用函数式调用：`gotrycatch.Catch[ErrorType](tb, handler)`。但是 `CatchAny` 和 `Finally` 方法支持链式调用。

## 安装

```bash
go get github.com/linkerlin/gotrycatch
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/linkerlin/gotrycatch"
    "github.com/linkerlin/gotrycatch/errors"
)

func main() {
    tb := gotrycatch.Try(func() {
        // 可能会 panic 的代码
        gotrycatch.Throw(errors.NewValidationError("email", "格式无效", 1001))
    })

    tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
        fmt.Printf("验证错误: %s (字段: %s, 代码: %d)\n", err.Message, err.Field, err.Code)
    })

    tb.Finally(func() {
        fmt.Println("清理工作完成")
    })
}
```

### TryWithResult - 带返回值的函数

```go
// 执行带返回值的函数
tb := gotrycatch.TryWithResult(func() int {
    return computeValue()
})

// 成功回调
tb.OnSuccess(func(result int) {
    fmt.Println("结果:", result)
})

// 错误回调
tb.OnError(func(err interface{}) {
    fmt.Println("错误:", err)
})

// 获取结果，有错误时返回默认值
result := tb.OrElse(0)

// 或者延迟计算默认值
result := tb.OrElseGet(func() int { return computeDefault() })
```

### 多种异常类型处理

```go
tb := gotrycatch.Try(func() {
    processUserData()
})

tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    fmt.Printf("验证失败: %s\n", err.Message)
})

tb = gotrycatch.Catch[errors.DatabaseError](tb, func(err errors.DatabaseError) {
    fmt.Printf("数据库错误: %s on table %s\n", err.Operation, err.Table)
})

tb = gotrycatch.Catch[errors.NetworkError](tb, func(err errors.NetworkError) {
    if err.Timeout {
        fmt.Printf("网络超时: %s\n", err.URL)
    } else {
        fmt.Printf("网络错误 %d: %s\n", err.StatusCode, err.URL)
    }
})

tb = tb.CatchAny(func(err interface{}) {
    fmt.Printf("未知错误: %v\n", err)
})

tb.Finally(func() {
    fmt.Println("处理完成")
})
```

### 状态查询和调试

```go
tb := gotrycatch.Try(func() {
    riskyOperation()
})

// 查询状态
if tb.HasError() {
    fmt.Printf("错误类型: %s\n", tb.GetErrorType())
    fmt.Printf("错误值: %v\n", tb.GetError())
}

if !tb.IsHandled() {
    // 根据错误类型决定处理方式
    switch tb.GetErrorType() {
    case "errors.ValidationError":
        // 处理验证错误
    default:
        tb = tb.CatchAny(func(err interface{}) {
            logUnknownError(err)
        })
    }
}

// 开启调试模式追踪类型匹配
gotrycatch.SetDebug(true)
```

### 结构化错误输出（Agent 友好）

```go
tb := gotrycatch.Catch[errors.BusinessLogicError](tb, func(err errors.BusinessLogicError) {
    // JSON 输出便于日志和 Agent 解析
    jsonData, _ := err.ToJSON()
    log.Printf("ERROR: %s", string(jsonData))
    // 输出: {"type":"BusinessLogicError","rule":"inventory_check","details":"库存不足","file":"main.go","line":42,"function":"processOrder","timestamp":"2024-01-15T10:30:00Z","stack":[...]}
})
```

### 断言辅助函数

```go
// 条件断言，false 时抛出错误
gotrycatch.Assert(value != "", errors.NewValidationError("value", "不能为空", 1001))

// 错误断言，有错误时包装并抛出
gotrycatch.AssertNoError(err, "数据库操作失败")
```

## 内置异常类型

所有错误类型都包含：`File`、`Line`、`Function`、`Timestamp`、`Stack`

| 类型 | 专有字段 | 构造函数 | 用途 |
|------|----------|----------|------|
| `ValidationError` | Field, Message, Code | `NewValidationError(field, message, code)` | 数据验证错误 |
| `DatabaseError` | Operation, Table, Cause | `NewDatabaseError(operation, table, cause)` | 数据库操作错误 |
| `NetworkError` | URL, StatusCode, Timeout | `NewNetworkError(url, code)` | HTTP 错误 |
| `NetworkError` | URL, Timeout | `NewNetworkTimeoutError(url)` | 网络超时 |
| `BusinessLogicError` | Rule, Details | `NewBusinessLogicError(rule, details)` | 业务规则违规 |
| `ConfigError` | Key, Value, Reason | `NewConfigError(key, value, reason)` | 配置错误 |
| `AuthError` | Operation, User, Reason | `NewAuthError(operation, user, reason)` | 认证授权错误 |
| `RateLimitError` | Resource, Limit, Current, RetryAfter | `NewRateLimitError(resource, limit, current, retryAfter)` | 限流错误 |

### 错误方法

```go
var err errors.ValidationError

err.Error()     // string - 完整错误描述（含位置信息）
err.ToMap()     // map[string]interface{} - 结构化数据
err.ToJSON()    // ([]byte, error) - JSON 输出
err.Unwrap()    // error - 底层错误（DatabaseError 返回 Cause）
err.Is(target)  // bool - 错误匹配
```

## API 文档

### 核心函数

| 函数/方法 | 签名 | 说明 |
|-----------|------|------|
| `Try` | `func Try(fn func()) *TryBlock` | 执行函数并捕获任何 panic |
| `Catch[T]` | `func Catch[T any](tb *TryBlock, handler func(T)) *TryBlock` | 处理指定类型 T 的异常 |
| `CatchWithReturn[T]` | `func CatchWithReturn[T any](tb *TryBlock, handler func(T) interface{}) (interface{}, *TryBlock)` | 处理并返回值 |
| `CatchAny` | `func (tb *TryBlock) CatchAny(handler func(interface{})) *TryBlock` | 处理任何未处理的异常 |
| `Finally` | `func (tb *TryBlock) Finally(fn func())` | 执行清理代码 |

### TryBlock 状态查询

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `HasError()` | `bool` | 是否捕获了错误 |
| `GetError()` | `interface{}` | 获取错误值 |
| `GetErrorType()` | `string` | 获取错误类型名（如 "errors.ValidationError"） |
| `IsHandled()` | `bool` | 错误是否已被处理 |
| `String()` | `string` | 友好的字符串表示 |

### TryWithResult

| 函数/方法 | 签名 | 说明 |
|-----------|------|------|
| `TryWithResult` | `func TryWithResult[T any](fn func() T) *TryBlockWithResult[T]` | 执行带返回值的函数 |
| `CatchWithResult` | `func CatchWithResult[T, E any](tb *TryBlockWithResult[T], handler func(E)) *TryBlockWithResult[T]` | TryWithResult 的类型捕获 |
| `CatchAnyWithResult` | `func CatchAnyWithResult[T any](tb *TryBlockWithResult[T], handler func(interface{})) *TryBlockWithResult[T]` | TryWithResult 的任意捕获 |
| `GetResult()` | `T` | 获取结果值 |
| `OnSuccess` | `func (tb *TryBlockWithResult[T]) OnSuccess(fn func(T)) *TryBlockWithResult[T]` | 成功时回调 |
| `OnError` | `func (tb *TryBlockWithResult[T]) OnError(fn func(interface{})) *TryBlockWithResult[T]` | 错误时回调 |
| `OrElse` | `func (tb *TryBlockWithResult[T]) OrElse(defaultValue T) T` | 获取结果或默认值 |
| `OrElseGet` | `func (tb *TryBlockWithResult[T]) OrElseGet(supplier func() T) T` | 获取结果或延迟计算默认值 |

### 调试与断言

| 函数 | 签名 | 说明 |
|------|------|------|
| `SetDebug` | `func SetDebug(enabled bool)` | 开启/关闭调试日志 |
| `IsDebug` | `func IsDebug() bool` | 查询调试模式状态 |
| `Throw` | `func Throw(err interface{})` | 抛出异常（panic） |
| `Assert` | `func Assert(condition bool, err interface{})` | 条件断言，false 时抛出 |
| `AssertNoError` | `func AssertNoError(err error, msg string)` | 错误断言，有错误时抛出 |

## 最佳实践

1. **按特定性排序 Catch 块**: 将最具体的异常类型放在前面，通用类型放在后面
2. **总是使用 Finally**: 确保资源清理代码被执行
3. **使用预定义异常类型**: 优先使用库提供的异常类型，而不是原始字符串或数字
4. **使用 CatchAny 作为兜底**: 优雅地处理未预期的错误类型
5. **调试时开启调试模式**: 当类型匹配不符合预期时，使用 `SetDebug(true)` 排查
6. **日志使用 ToJSON**: 结构化输出更易于解析和分析

## 性能考虑

- 异常处理基于 Go 的 panic/recover 机制，只在实际发生异常时才有性能开销
- 正常执行路径的性能开销接近零
- Try-Catch 块可以嵌套使用，不会显著影响性能

## 兼容性

- 需要 Go 1.18+ （泛型支持）
- 与标准库完全兼容
- 可以与现有的错误处理代码共存
- 支持并发安全使用

## 常见问题 (FAQ)

### Q: 为什么不能使用完全的链式调用？

A: 由于 Go 语言的限制，方法不能有泛型类型参数。因此不能写：

```go
// ❌ 这样写是不支持的
tb := gotrycatch.Try(func() { ... }).Catch[ErrorType](handler)
```

只能使用函数式调用：

```go
// ✅ 正确的写法
tb := gotrycatch.Try(func() { ... })
tb = gotrycatch.Catch[ErrorType](tb, handler)
```

但是 `CatchAny` 和 `Finally` 方法支持链式调用：

```go
// ✅ 这样是可以的
tb.CatchAny(handler).Finally(cleanup)
```

### Q: 如何调试类型匹配问题？

A: 开启调试模式：
```go
gotrycatch.SetDebug(true)
// 输出: [gotrycatch] Catch: type errors.ValidationError does not match target type int
// 输出: [gotrycatch] Catch: type errors.ValidationError matched, calling handler
```

### Q: 未处理的错误会怎样？

A: 未处理的错误会在 `Finally` 执行后重新抛出。如果不想让 panic 传播，请使用 `CatchAny` 作为兜底。

## 示例

查看 `examples/` 目录获取更多详细示例，包括：
- 基本用法演示
- 多种异常类型处理
- 嵌套异常处理
- TryWithResult 模式
- 结构化错误日志

### 运行示例

```bash
# 快速演示
go run ./cmd/demo

# 完整示例
go run ./examples

# 运行测试
go test -v ./...

# 查看覆盖率
go test -cover ./...

# 竞态检测
go test -race ./...
```

## 许可证

MIT License
