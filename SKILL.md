# SKILL: GoTryCatch - Go 泛型异常处理库

## 技能描述

GoTryCatch 是一个为 Go 语言提供类型安全异常处理的库。它使用 Go 泛型实现了类似其他语言 try-catch-finally 的异常处理机制，基于 panic/recover 构建。

**核心价值：**
- 将 panic 转化为可控的异常处理流程
- 支持类型安全的错误捕获
- 提供丰富的错误上下文信息（堆栈、位置、时间戳）
- 支持结构化错误输出，便于 Agent 解析

---

## 快速开始

```go
import (
    "github.com/linkerlin/gotrycatch"
    "github.com/linkerlin/gotrycatch/errors"
)

// 基本用法
tb := gotrycatch.Try(func() {
    // 可能 panic 的代码
    gotrycatch.Throw(errors.NewValidationError("field", "invalid", 1001))
})

// 类型安全的错误捕获
tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    fmt.Printf("验证错误: %s (code: %d)\n", err.Message, err.Code)
})

// 清理代码
tb.Finally(func() {
    fmt.Println("清理完成")
})
```

---

## 核心 API

### Try / Catch / Finally

| 函数/方法 | 签名 | 说明 |
|-----------|------|------|
| `Try` | `func Try(fn func()) *TryBlock` | 执行函数并捕获 panic |
| `Catch[T]` | `func Catch[T any](tb *TryBlock, handler func(T)) *TryBlock` | 捕获指定类型的错误 |
| `CatchWithReturn[T]` | `func CatchWithReturn[T any](tb *TryBlock, handler func(T) interface{}) (interface{}, *TryBlock)` | 捕获并返回值 |
| `CatchAny` | `func (tb *TryBlock) CatchAny(handler func(interface{})) *TryBlock` | 捕获任意未处理的错误 |
| `Finally` | `func (tb *TryBlock) Finally(fn func())` | 执行清理代码 |

### TryBlock 状态查询

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `HasError()` | `bool` | 是否捕获了错误 |
| `GetError()` | `interface{}` | 获取错误值 |
| `GetErrorType()` | `string` | 获取错误类型名（如 "errors.ValidationError"） |
| `IsHandled()` | `bool` | 错误是否已被处理 |
| `String()` | `string` | 友好的字符串表示 |

### TryWithResult（支持返回值）

```go
tb := gotrycatch.TryWithResult(func() int {
    return computeValue()
})

// 成功回调
tb.OnSuccess(func(result int) {
    fmt.Println("结果:", result)
})

// 错误捕获
tb = gotrycatch.CatchWithResult[int, errors.ValidationError](tb, func(err errors.ValidationError) {
    // 处理错误
})

// 获取结果（有错误时返回默认值）
result := tb.OrElse(0)

// 延迟计算默认值
result := tb.OrElseGet(func() int { return computeDefault() })
```

### 断言辅助函数

```go
// 条件断言：condition 为 false 时抛出 err
gotrycatch.Assert(value != "", errors.NewValidationError("value", "不能为空", 1001))

// 错误断言：err 不为 nil 时抛出包装错误
gotrycatch.AssertNoError(err, "操作失败")
```

### 调试模式

```go
gotrycatch.SetDebug(true)  // 开启调试日志
gotrycatch.IsDebug()       // 查询调试状态
```

---

## 错误类型

所有错误类型都包含以下上下文字段：
- `File` (string) - 源文件名
- `Line` (int) - 行号
- `Function` (string) - 函数名
- `Timestamp` (time.Time) - 时间戳
- `Stack` ([]string) - 调用堆栈

### 内置错误类型

| 类型 | 专有字段 | 构造函数 | 用途 |
|------|----------|----------|------|
| `ValidationError` | Field, Message, Code | `NewValidationError(field, message, code)` | 数据验证错误 |
| `DatabaseError` | Operation, Table, Cause | `NewDatabaseError(operation, table, cause)` | 数据库操作错误 |
| `NetworkError` | URL, StatusCode, Timeout | `NewNetworkError(url, code)` | 网络请求错误 |
| `NetworkError` | URL, Timeout | `NewNetworkTimeoutError(url)` | 网络超时错误 |
| `BusinessLogicError` | Rule, Details | `NewBusinessLogicError(rule, details)` | 业务规则违规 |
| `ConfigError` | Key, Value, Reason | `NewConfigError(key, value, reason)` | 配置错误 |
| `AuthError` | Operation, User, Reason | `NewAuthError(operation, user, reason)` | 认证授权错误 |
| `RateLimitError` | Resource, Limit, Current, RetryAfter | `NewRateLimitError(resource, limit, current, retryAfter)` | 限流错误 |

### 错误类型通用方法

```go
var err errors.ValidationError

err.Error()     // string - 完整错误描述（含位置信息）
err.ToMap()     // map[string]interface{} - 结构化数据
err.ToJSON()    // ([]byte, error) - JSON 格式输出
err.Unwrap()    // error - 底层错误（DatabaseError 返回 Cause）
err.Is(target)  // bool - 错误匹配判断
```

---

## 使用模式

### 模式 1：基础异常处理

```go
tb := gotrycatch.Try(func() {
    processData(data)
})

tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    handleValidationError(err)
})

tb = gotrycatch.Catch[errors.DatabaseError](tb, func(err errors.DatabaseError) {
    handleDatabaseError(err)
})

tb = tb.CatchAny(func(err interface{}) {
    handleUnknownError(err)
})

tb.Finally(func() {
    cleanup()
})
```

### 模式 2：带返回值的处理

```go
result := gotrycatch.TryWithResult(func() string {
    return fetchData()
}).
CatchWithResult[string, errors.NetworkError](nil, func(err errors.NetworkError) {
    logNetworkError(err)
}).
OrElse("default_value")
```

### 模式 3：状态驱动的错误处理

```go
tb := gotrycatch.Try(func() {
    riskyOperation()
})

if !tb.HasError() {
    fmt.Println("操作成功")
    return
}

// 根据错误类型决定处理方式
switch tb.GetErrorType() {
case "errors.ValidationError":
    // 处理验证错误
case "errors.DatabaseError":
    // 处理数据库错误
default:
    // 未知错误，可能需要重新抛出
    tb = tb.CatchAny(func(err interface{}) {
        logUnknownError(err)
    })
}

tb.Finally(func() { cleanup() })
```

### 模式 4：结构化错误日志（Agent 友好）

```go
tb := gotrycatch.Try(func() {
    processOrder(order)
})

tb = gotrycatch.Catch[errors.BusinessLogicError](tb, func(err errors.BusinessLogicError) {
    // 结构化日志，便于 Agent 解析
    jsonData, _ := err.ToJSON()
    log.Printf("ERROR: %s", string(jsonData))
    // 输出示例:
    // {"type":"BusinessLogicError","rule":"inventory_check","details":"库存不足","file":"main.go","line":42,...}
})

tb.Finally(func() {})
```

### 模式 5：嵌套 Try-Catch

```go
tb := gotrycatch.Try(func() {
    // 外层操作

    innerTb := gotrycatch.Try(func() {
        // 内层操作
        riskyInnerOperation()
    })

    innerTb = gotrycatch.Catch[errors.ValidationError](innerTb, func(err errors.ValidationError) {
        // 内层错误处理，可能转换后重新抛出
        gotrycatch.Throw(errors.NewBusinessLogicError("inner_failed", err.Message))
    })

    innerTb.Finally(func() {
        // 内层清理
    })
})

tb = gotrycatch.Catch[errors.BusinessLogicError](tb, func(err errors.BusinessLogicError) {
    // 外层捕获内层转换后的错误
})
```

---

## 重要规则

### 链式调用限制

**Go 方法的类型参数限制：**
- ❌ `tb.Catch[ErrorType](handler)` - 方法不能有类型参数
- ✅ `gotrycatch.Catch[ErrorType](tb, handler)` - 使用函数形式
- ✅ `tb.CatchAny(handler).Finally(cleanup)` - 方法可以链式调用

### Finally 行为

1. **总是执行** - 无论是否发生 panic
2. **未处理错误会重新抛出** - 如果没有匹配的 Catch，Finally 执行后重新 panic
3. **执行顺序** - Finally 在可能的 re-panic 之前执行

### Catch 匹配规则

1. **类型精确匹配** - 必须是精确的类型，不支持继承
2. **首次匹配生效** - 第一个匹配的 Catch 会设置 `handled = true`
3. **后续 Catch 跳过** - `handled = true` 后的 Catch 不执行

### 错误链

- `DatabaseError` 支持 `Unwrap()` 返回底层 Cause
- 所有错误类型支持 `Is(target error) bool` 进行错误匹配

---

## Agent 使用指南

### 诊断错误

```go
// 开启调试模式追踪类型匹配
gotrycatch.SetDebug(true)

// 输出示例:
// [gotrycatch] Catch: type errors.ValidationError does not match target type int
// [gotrycatch] Catch: type errors.ValidationError matched, calling handler
```

### 解析错误信息

```go
if tb.HasError() {
    // 获取错误类型
    errorType := tb.GetErrorType()

    // 获取结构化数据
    if err, ok := tb.GetError().(errors.ValidationError); ok {
        data := err.ToMap()
        // data["type"] == "ValidationError"
        // data["field"] == "email"
        // data["code"] == 1001
        // data["file"] == "main.go"
        // data["line"] == 42
    }
}
```

### 错误类型判断流程

```
1. tb.GetErrorType() 获取类型名
2. 根据类型名选择对应的 Catch
3. 使用 ToMap()/ToJSON() 获取结构化数据
4. 根据 Code/Rule/Operation 等字段决定处理逻辑
```

---

## 常见陷阱

1. **忘记 Finally** - 未处理的错误会在 Finally 后 re-panic
2. **Catch 顺序错误** - 应该从具体到通用排列
3. **类型不匹配** - 使用 `CatchAny` 作为兜底
4. **忽略返回值** - TryWithResult 的结果需要使用 OrElse/OnSuccess 处理

---

## 文件结构

```
gotrycatch/
├── gotrycatch.go        # 核心 API: Try, Catch, Finally, TryWithResult
├── errors/
│   └── errors.go        # 错误类型定义
├── examples/main.go     # 完整示例
└── cmd/demo/main.go     # 快速演示
```

---

## 版本信息

- **Go 版本要求**: 1.18+（泛型支持）
- **模块路径**: `github.com/linkerlin/gotrycatch`
- **依赖**: 仅标准库
