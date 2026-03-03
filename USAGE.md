# GoTryCatch 使用示例

## 快速开始

```go
package main

import (
    "fmt"
    "github.com/linkerlin/gotrycatch"
    "github.com/linkerlin/gotrycatch/errors"
)

func main() {
    // 基本的 Try/Catch/Finally
    tb := gotrycatch.Try(func() {
        gotrycatch.Throw(errors.NewValidationError("email", "invalid format", 1001))
    })

    tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
        fmt.Printf("Validation error: %s (field: %s, code: %d)\n", err.Message, err.Field, err.Code)
    })

    tb.Finally(func() {
        fmt.Println("Cleanup completed")
    })
}
```

## TryWithResult - 带返回值

```go
// 执行带返回值的函数
tb := gotrycatch.TryWithResult(func() int {
    return computeValue()
})

// 成功回调
tb.OnSuccess(func(result int) {
    fmt.Println("Result:", result)
})

// 错误回调
tb.OnError(func(err interface{}) {
    fmt.Println("Error:", err)
})

// 获取结果，有错误时返回默认值
result := tb.OrElse(0)

// 或者延迟计算默认值
result := tb.OrElseGet(func() int { return computeDefault() })
```

## 多种异常类型处理

```go
tb := gotrycatch.Try(func() {
    processUserData()
})

// 按特定性排序：具体类型在前
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

tb = gotrycatch.Catch[errors.BusinessLogicError](tb, func(err errors.BusinessLogicError) {
    fmt.Printf("Business rule violation: %s - %s\n", err.Rule, err.Details)
})

tb = gotrycatch.Catch[errors.ConfigError](tb, func(err errors.ConfigError) {
    fmt.Printf("Config error on '%s': %s\n", err.Key, err.Reason)
})

tb = gotrycatch.Catch[errors.AuthError](tb, func(err errors.AuthError) {
    fmt.Printf("Auth error during %s for user '%s': %s\n", err.Operation, err.User, err.Reason)
})

tb = gotrycatch.Catch[errors.RateLimitError](tb, func(err errors.RateLimitError) {
    fmt.Printf("Rate limit exceeded on '%s': %d/%d, retry after %ds\n", 
        err.Resource, err.Current, err.Limit, err.RetryAfter)
})

// CatchAny 作为兜底
tb = tb.CatchAny(func(err interface{}) {
    fmt.Printf("Unknown error: %v\n", err)
})

// Finally 保证清理代码执行
tb.Finally(func() {
    fmt.Println("Processing done")
})
```

## 状态查询

```go
tb := gotrycatch.Try(func() {
    riskyOperation()
})

// 查询状态
if tb.HasError() {
    fmt.Printf("Error type: %s\n", tb.GetErrorType())
    fmt.Printf("Error value: %v\n", tb.GetError())
}

if !tb.IsHandled() {
    // 根据错误类型决定处理方式
    switch tb.GetErrorType() {
    case "errors.ValidationError":
        // 处理验证错误
    case "errors.DatabaseError":
        // 处理数据库错误
    default:
        tb = tb.CatchAny(func(err interface{}) {
            logUnknownError(err)
        })
    }
}

// 友好的字符串表示
fmt.Println(tb.String())
```

## 调试模式

```go
// 开启调试，输出类型匹配日志
gotrycatch.SetDebug(true)

tb := gotrycatch.Try(func() {
    panic("string error")
})

tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    fmt.Println("Caught validation error")
})

// 控制台输出:
// [gotrycatch] Catch: type string does not match target type errors.ValidationError

// 查询调试状态
if gotrycatch.IsDebug() {
    fmt.Println("Debug mode is on")
}
```

## 结构化错误输出

```go
tb := gotrycatch.Catch[errors.BusinessLogicError](tb, func(err errors.BusinessLogicError) {
    // JSON 输出便于日志和 Agent 解析
    jsonData, _ := err.ToJSON()
    log.Printf("ERROR: %s", string(jsonData))
    // 输出: {"type":"BusinessLogicError","rule":"inventory_check","details":"Out of stock","file":"main.go","line":42,"function":"processOrder","timestamp":"2024-01-15T10:30:00Z","stack":[...]}

    // Map 输出
    m := err.ToMap()
    fmt.Printf("Rule: %s, Details: %s\n", m["rule"], m["details"])
})
```

## 断言辅助函数

```go
// Assert: 条件为 false 时抛出错误
gotrycatch.Assert(value != "", errors.NewValidationError("value", "cannot be empty", 1001))

// AssertNoError: err 不为 nil 时抛出包装错误
result, err := someOperation()
gotrycatch.AssertNoError(err, "operation failed")
```

## TryWithResult 完整示例

```go
func divideAndLog(a, b int) int {
    tb := gotrycatch.TryWithResult(func() int {
        gotrycatch.Assert(b != 0,
            errors.NewValidationError("divisor", "cannot be zero", 1001))
        return a / b
    })

    tb = gotrycatch.CatchWithResult[int, errors.ValidationError](tb, func(err errors.ValidationError) {
        log.Printf("Validation error: %s", err.Message)
    })

    tb = gotrycatch.CatchAnyWithResult(tb, func(err interface{}) {
        log.Printf("Unknown error: %v", err)
    })

    return tb.OrElse(0)
}

// 使用 OrElseGet 延迟计算默认值
func computeWithDefault() int {
    tb := gotrycatch.TryWithResult(func() int {
        return expensiveComputation()
    })

    return tb.OrElseGet(func() int {
        // 只有出错时才计算默认值
        return computeFallbackValue()
    })
}
```

## 内置错误类型

| 类型 | 构造函数 | 用途 |
|------|----------|------|
| `ValidationError` | `NewValidationError(field, message, code)` | 数据验证错误 |
| `DatabaseError` | `NewDatabaseError(operation, table, cause)` | 数据库操作错误 |
| `NetworkError` | `NewNetworkError(url, code)` | HTTP 错误 |
| `NetworkError` | `NewNetworkTimeoutError(url)` | 网络超时 |
| `BusinessLogicError` | `NewBusinessLogicError(rule, details)` | 业务规则违规 |
| `ConfigError` | `NewConfigError(key, value, reason)` | 配置错误 |
| `AuthError` | `NewAuthError(operation, user, reason)` | 认证授权错误 |
| `RateLimitError` | `NewRateLimitError(resource, limit, current, retryAfter)` | 限流错误 |

所有错误类型都包含：`File`, `Line`, `Function`, `Timestamp`, `Stack`

## 运行示例

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

## 注意事项

1. **Catch 是函数，不是方法**：由于 Go 的限制，`Catch[T]` 必须写成 `gotrycatch.Catch[T](tb, handler)`
2. **CatchAny 和 Finally 是方法**：可以链式调用 `tb.CatchAny(...).Finally(...)`
3. **未处理的错误会重新抛出**：在 Finally 执行后
4. **Catch 顺序很重要**：具体类型在前，CatchAny 在最后
