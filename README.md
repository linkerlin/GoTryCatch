# GoTryCatch

一个基于 Go 泛型的类型安全异常处理库，提供类似于其他语言中 try-catch 语句的功能。

## 特性

- 🎯 **类型安全**: 使用 Go 泛型确保异常处理的类型安全
- 🔗 **链式调用**: 支持多个 catch 块的链式调用
- 🏷️ **多种异常类型**: 内置常用的异常类型（验证、数据库、网络、业务逻辑错误）
- 🔄 **Finally 支持**: 保证清理代码的执行
- 📦 **零依赖**: 纯 Go 实现，无外部依赖
- 🚀 **高性能**: 基于 Go 的 panic/recover 机制，性能开销极小

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
