[English](#english) | [中文](#chinese)

<a id="english"></a>

# GoTryCatch

A type-safe exception handling library based on Go generics that brings try-catch-like capabilities to Go.

## Features

- 🎯 Type-safe: Uses Go generics to ensure type-safe exception handling
- 🔗 Partial chaining: Supports chaining for `CatchAny` and `Finally`
- 🏷️ Multiple error kinds: Built-in common error types (validation, database, network, business logic)
- 🔄 Finally support: Guarantees cleanup code execution
- 📦 Zero dependency: Pure Go implementation, no external dependencies
- 🚀 High performance: Built on Go's panic/recover with minimal overhead

## Important Note

⚠️ Chaining limitation: Due to Go's limitation that methods cannot have generic type parameters, you cannot write `tb.Catch[ErrorType](handler)`. Use the functional form instead: `gotrycatch.Catch[ErrorType](tb, handler)`. `CatchAny` and `Finally` do support chaining.

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

### Handling multiple error types

```go
tb := gotrycatch.Try(func() {
    // Business logic
    processUserData()
})

tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    fmt.Printf("Validation failed: %s\n", err.Message)
})

tb = gotrycatch.Catch[errors.DatabaseError](tb, func(err errors.DatabaseError) {
    fmt.Printf("Database error: %s\n", err.Operation)
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

### Exception handling with return value

```go
tb := gotrycatch.Try(func() {
    validateUserInput(userData)
})

result, tb := gotrycatch.CatchWithReturn[errors.ValidationError](tb, func(err errors.ValidationError) interface{} {
    return map[string]interface{}{
        "success": false,
        "error":   err.Error(),
        "code":    err.Code,
    }
})

if result != nil {
    fmt.Printf("Result: %+v\n", result)
}
```

## Built-in Error Types

The library provides the following common error types:

### ValidationError

```go
err := errors.NewValidationError("email", "invalid email format", 1001)
```

### DatabaseError

```go
err := errors.NewDatabaseError("SELECT", "users", sqlErr)
```

### NetworkError

```go
// HTTP error
err := errors.NewNetworkError("http://api.example.com", 404)

// Timeout error
err := errors.NewNetworkTimeoutError("http://api.example.com")
```

### BusinessLogicError

```go
err := errors.NewBusinessLogicError("age_limit", "user must be at least 18 years old")
```

## API

### Core functions

#### `Try(fn func()) *TryBlock`
Executes the given function and captures any panic. Returns a `TryBlock` for subsequent handling.

#### `Catch[T any](tb *TryBlock, handler func(T)) *TryBlock`
Handles exceptions of type T. If the panic value can be converted to T, the handler is invoked.

#### `CatchWithReturn[T any](tb *TryBlock, handler func(T) interface{}) (interface{}, *TryBlock)`
Like `Catch`, but allows the handler to return a value.

#### `(*TryBlock) CatchAny(handler func(interface{})) *TryBlock`
Handles any unhandled exception regardless of its type.

#### `(*TryBlock) Finally(fn func())`
Executes cleanup code whether or not an exception occurred. If an exception remains unhandled, it is rethrown after the finally block.

#### `Throw(err interface{})`
Throws an exception (creates a panic).

## Best Practices

1. Order `Catch` blocks by specificity: most specific first, more general later
2. Always use `Finally` to ensure resource cleanup
3. Prefer predefined error types over raw strings or numbers
4. Avoid throwing in `Finally` to not mask the original exception

## Performance

- Built on Go's panic/recover; cost occurs only when exceptions actually happen
- Near-zero overhead on the normal execution path
- Try-Catch blocks can be nested without significant impact

## Compatibility

- Requires Go 1.18+ (generics)
- Fully compatible with the standard library
- Can coexist with existing error-handling code

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

### Q: How's the performance?

A: Exception handling relies on Go's panic/recover. Overhead is incurred only when an exception actually occurs. The normal path overhead is near zero.

## Examples

See the `examples/` directory for more:

- Basic usage
- Handling multiple error types
- Nested exception handling
- Real-world scenarios

### Run examples

```bash
# Quick demo
go run ./cmd/demo

# Full examples
go run ./examples

# Run tests
go test -v
```

## License

MIT License

A lib for using trycatch in Go!

---

<a id="chinese"></a>

# GoTryCatch

一个基于 Go 泛型的类型安全异常处理库，提供类似于其他语言中 try-catch 语句的功能。

## 特性

- 🎯 **类型安全**: 使用 Go 泛型确保异常处理的类型安全
- 🔗 **部分链式调用**: 支持 `CatchAny` 和 `Finally` 的链式调用
- 🏷️ **多种异常类型**: 内置常用的异常类型（验证、数据库、网络、业务逻辑错误）
- 🔄 **Finally 支持**: 保证清理代码的执行
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
        gotrycatch.Throw(errors.NewValidationError("email", "invalid format", 1001))
    })

    tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
        fmt.Printf("验证错误: %s (字段: %s, 代码: %d)\n", err.Message, err.Field, err.Code)
    })

    tb.Finally(func() {
        fmt.Println("清理工作完成")
    })
}
```

### 多种异常类型处理

```go
tb := gotrycatch.Try(func() {
    // 业务逻辑代码
    processUserData()
})

tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    fmt.Printf("验证失败: %s\n", err.Message)
})

tb = gotrycatch.Catch[errors.DatabaseError](tb, func(err errors.DatabaseError) {
    fmt.Printf("数据库错误: %s\n", err.Operation)
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

### 带返回值的异常处理

```go
tb := gotrycatch.Try(func() {
    validateUserInput(userData)
})

result, tb := gotrycatch.CatchWithReturn[errors.ValidationError](tb, func(err errors.ValidationError) interface{} {
    return map[string]interface{}{
        "success": false,
        "error":   err.Error(),
        "code":    err.Code,
    }
})

if result != nil {
    fmt.Printf("处理结果: %+v\n", result)
}
```

## 内置异常类型

库提供了以下常用的异常类型：

### ValidationError - 验证错误
```go
err := errors.NewValidationError("email", "邮箱格式无效", 1001)
```

### DatabaseError - 数据库错误
```go
err := errors.NewDatabaseError("SELECT", "users", sqlErr)
```

### NetworkError - 网络错误
```go
// HTTP 错误
err := errors.NewNetworkError("http://api.example.com", 404)

// 超时错误
err := errors.NewNetworkTimeoutError("http://api.example.com")
```

### BusinessLogicError - 业务逻辑错误
```go
err := errors.NewBusinessLogicError("age_limit", "用户必须年满18岁")
```

## API 文档

### 核心函数

#### `Try(fn func()) *TryBlock`
执行给定的函数并捕获任何 panic。返回一个 `TryBlock` 用于后续的异常处理。

#### `Catch[T any](tb *TryBlock, handler func(T)) *TryBlock`
处理指定类型 T 的异常。如果 panic 的值可以转换为类型 T，则调用处理函数。

#### `CatchWithReturn[T any](tb *TryBlock, handler func(T) interface{}) (interface{}, *TryBlock)`
类似于 `Catch`，但允许处理函数返回一个值。

#### `(*TryBlock) CatchAny(handler func(interface{})) *TryBlock`
处理任何未被处理的异常，无论类型如何。

#### `(*TryBlock) Finally(fn func())`
无论是否发生异常，都会执行的清理代码。如果有未处理的异常，会在 finally 块执行后重新抛出。

#### `Throw(err interface{})`
抛出一个异常（创建 panic）。

## 最佳实践

1. **按特定性排序 Catch 块**: 将最具体的异常类型放在前面，通用类型放在后面
2. **总是使用 Finally**: 确保资源清理代码被执行
3. **使用预定义异常类型**: 优先使用库提供的异常类型，而不是原始字符串或数字
4. **避免在 Finally 中抛出异常**: 这可能会掩盖原始异常

## 性能考虑

- 异常处理基于 Go 的 panic/recover 机制，只在实际发生异常时才有性能开销
- 正常执行路径的性能开销接近零
- Try-Catch 块可以嵌套使用，不会显著影响性能

## 兼容性

- 需要 Go 1.18+ （泛型支持）
- 与标准库完全兼容
- 可以与现有的错误处理代码共存

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

### Q: 性能如何？

A: 异常处理基于 Go 的 panic/recover 机制，只在实际发生异常时才有性能开销。正常执行路径的性能开销接近零。

## 示例

查看 `examples/` 目录获取更多详细示例，包括：
- 基本用法演示
- 多种异常类型处理
- 嵌套异常处理
- 真实场景应用示例

### 运行示例

```bash
# 快速演示
go run ./cmd/demo

# 完整示例
go run ./examples

# 运行测试
go test -v
```

## 许可证

MIT License
A lib for using trycatch in Go!
