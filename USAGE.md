# GoTryCatch 使用示例

## 快速开始（v2 推荐：Run）

```go
package main

import (
    "fmt"

    "github.com/linkerlin/gotrycatch"
    "github.com/linkerlin/gotrycatch/errtypes"
)

func main() {
    // 一次调用：fn + 处理子句；未处理的 panic 以 error 返回
    err := gotrycatch.Run(func() {
        gotrycatch.Throw(errtypes.NewValidationError("email", "invalid format", 1001))
    },
        gotrycatch.OnAs(func(e errtypes.ValidationError) {
            fmt.Printf("Validation error: %s (field: %s, code: %d)\n", e.Message, e.Field, e.Code)
        }),
        gotrycatch.Cleanup(func() { fmt.Println("Cleanup done") }),
    )
    if err != nil {
        fmt.Println("unhandled:", err)
    }
}
```

`Run` 的三个保证：
- 未处理的 panic 一定以 `error` 返回——不可能静默吞错
- `Cleanup` 总是执行（LIFO），handler panic 也不会跳过
- 子句按声明顺序匹配，第一个命中者处理

## 穿透包装：OnAs

```go
// 业务代码用 %w 包装（Go 惯用）
err := fmt.Errorf("query failed: %w", errtypes.NewDatabaseError("SELECT", "users", cause))

// OnAs 能穿透包装链命中底层类型（errors.As 语义）
gotrycatch.Run(func() { throwIt() },
    gotrycatch.OnAs(func(e errtypes.DatabaseError) { retry() }),
)
```

值/指针形态各自匹配：`panic(NewDatabaseError(...))`（值）配 `OnAs(func(e DatabaseError))`；`panic(&dbErr)`（指针）配 `OnAs(func(e *DatabaseError))`。

## Run1 - 带返回值

```go
v, err := gotrycatch.Run1(func() int {
    return computeValue()
},
    gotrycatch.OnAs(func(e errtypes.RateLimitError) { wait(e.RetryAfter) }),
)
if err != nil {
    // v 是零值
}
```

## 经典链式 API（v1 兼容）

```go
// 基本 Try/Catch/Finally
tb := gotrycatch.Try(func() {
    gotrycatch.Throw(errtypes.NewValidationError("email", "invalid format", 1001))
})

tb = gotrycatch.Catch[errtypes.ValidationError](tb, func(err errtypes.ValidationError) {
    fmt.Printf("Validation error: %s (field: %s, code: %d)\n", err.Message, err.Field, err.Code)
})

tb = gotrycatch.CatchAs[errtypes.DatabaseError](tb, func(err errtypes.DatabaseError) {
    // errors.As 语义：穿透 %w 包装
})

tb = gotrycatch.CatchAny(tb, func(v interface{}) {
    fmt.Println("Unexpected:", v)
})

tb.Finally(func() {
    fmt.Println("Cleanup done")
})
```

## panic 到 error 的桥接

```go
// 经典链式也可以桥回惯用 error 世界
if err := gotrycatch.Try(func() { risky() }).Err(); err != nil {
    return fmt.Errorf("risky failed: %w", err)
}

// errors.Is/As 直接可用（error 型 panic 原样直通；其他值包装为 *PanicError）
var dbErr errtypes.DatabaseError
if errors.As(gotrycatch.Try(func() { panic(innerErr) }).Err(), &dbErr) {
    // ...
}
```

## TryWithResult - 带返回值

```go
tb := gotrycatch.TryWithResult(func() int {
    return computeValue()
})

// 成功回调
tb.OnSuccess(func(result int) {
    fmt.Println("Result:", result)
})

// 错误时取默认值
result := gotrycatch.TryWithResult(func() int {
    panic("boom")
}).OrElse(0)

result2 := gotrycatch.TryWithResult(func() int {
    panic("boom")
}).OrElseGet(func() int { return computeDefault() })
```

## 多种异常类型处理

```go
tb := gotrycatch.Try(func() {
    validateUser("", "test@example.com", 25)
    accessDatabase("delete_all")
})

tb = gotrycatch.Catch[errtypes.ValidationError](tb, func(err errtypes.ValidationError) {
    fmt.Printf("Validation: %s\n", err.Message)
})

tb = gotrycatch.Catch[errtypes.DatabaseError](tb, func(err errtypes.DatabaseError) {
    fmt.Printf("Database: %s on %s\n", err.Operation, err.Table)
})

tb = gotrycatch.CatchAny(tb, func(v interface{}) {
    fmt.Printf("Unexpected: %v\n", v)
})
```

## 状态查询

```go
tb := gotrycatch.Try(func() { panic("err") })

tb.HasError()                // true - 是否有错误
tb.GetError()                // "err" - 原始 panic 值
tb.Err()                     // *gotrycatch.PanicError - 桥接为 error
tb.GetErrorType()            // "string" - 类型名
tb.CanonicalErrorType()      // "string" - 去指针短类型名（Agent 友好）
tb.IsHandled()               // false - 是否已处理
tb.String()                  // 友好字符串
```

## 调试模式

```go
gotrycatch.SetDebug(true)  // 开启后输出类型匹配日志
gotrycatch.IsDebug()       // 查询状态
```

## 结构化错误输出

```go
err := errtypes.NewDatabaseError("SELECT", "users", cause)
m, _ := err.ToMap()    // map[string]interface{}
j, _ := err.ToJSON()   // {"type":"DatabaseError",...}
err.Stack              // []string - 调用堆栈（含 File/Line/Function/Timestamp）
```

## 断言辅助函数

```go
gotrycatch.Assert(condition, err)            // false 时抛出 err
gotrycatch.AssertNoError(err, "operation")   // err 非 nil 时抛出包装错误
```

## 内置错误类型（errtypes 包）

| 类型 | 构造函数 |
|------|----------|
| `ValidationError` | `NewValidationError(field, message, code)` |
| `DatabaseError` | `NewDatabaseError(operation, table, cause)` |
| `NetworkError` | `NewNetworkError(url, code)` / `NewNetworkTimeoutError(url)` |
| `BusinessLogicError` | `NewBusinessLogicError(rule, details)` |
| `ConfigError` | `NewConfigError(key, value, reason)` |
| `AuthError` | `NewAuthError(operation, user, reason)` |
| `RateLimitError` | `NewRateLimitError(resource, limit, current, retryAfter)` |

> v2 起包名改为 `errtypes`（避免与标准库 `errors` 冲突）。旧的
> `github.com/linkerlin/gotrycatch/errors` 路径仍是纯别名层，可继续编译，
> 但新代码请使用 `errtypes`。

## 自定义错误类型

嵌入 `BaseError` 即可获得 File/Line/Function/Timestamp/Stack 字段与 ToMap 公共键：

```go
type PaymentError struct {
    errtypes.BaseError
    OrderID string `json:"orderId"`
}

func (e PaymentError) Error() string {
    return fmt.Sprintf("payment failed for order %s (at %s:%d)", e.OrderID, e.File, e.Line)
}

func (e PaymentError) ToMap() map[string]interface{} {
    m := e.baseMap()
    m["type"] = "PaymentError"
    m["orderId"] = e.OrderID
    return m
}

func NewPaymentError(orderID string) PaymentError {
    return PaymentError{BaseError: errtypes.NewBase(), OrderID: orderID}
}
```

`errtypes.NewBase()` 自动捕获调用方位置与堆栈（指向用户调用行）。实现 `Error()`/`ToMap()` 后即可与内置类型同等使用；`ToMap` 可先调用 `baseMap()` 获取公共键。若需封装更深的构造函数链，可参考内置类型在包内直接使用 `newBase(skip)`。

## 运行示例

```bash
# 演示程序（11个Demo：基本用法/错误信息/结构化输出/TryWithResult/调试模式/断言/状态查询/错误链/真实场景/v2 Run）
go run ./cmd/demo

# 运行测试
go test -v ./...

# 查看覆盖率
go test -cover ./...

# 竞态检测
go test -race ./...
```

## 注意事项

- 同一 `TryBlock` 上多次 `Catch`：按声明顺序匹配，第一个命中者处理，后续跳过
- `Finally` 中 panic 会覆盖原始错误（与 Go 原生 defer 一致）
- `Try`/`Run` 不捕获子 goroutine 的 panic（recover 是 goroutine 局部的）
- `Catch[T]` 是精确类型断言；穿透 `%w` 包装请用 `CatchAs`/`OnAs`
