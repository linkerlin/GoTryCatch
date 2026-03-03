尽可能用中文进行思考、推理和输出！

# AGENTS.md

帮助 AI Agent 在 GoTryCatch 仓库中高效工作的指南。

## 项目概述

GoTryCatch 是一个利用 Go 泛型实现类型安全异常处理的 Go 库。它通过封装 Go 内置的 panic/recover 机制，为 Go 带来了类似 try-catch 的异常处理能力。

**模块路径**: `github.com/linkerlin/gotrycatch`
**Go 版本**: 1.18+（泛型支持所需）
**版本**: 1.2.0

## 常用命令

### 运行测试
```bash
go test -v
```

### 运行示例
```bash
# 快速演示（展示所有新功能）
go run ./cmd/demo

# 完整示例（10个详细Demo）
go run ./examples
```

### 构建
```bash
go build ./...
```

## 项目结构

```
/
├── gotrycatch.go        # 主包 - Try/Catch/Finally 核心实现
├── gotrycatch_test.go   # 单元测试
├── errors/
│   └── errors.go        # 预定义错误类型（含堆栈、位置、时间戳）
├── examples/
│   └── main.go          # 完整的使用示例（10个Demo）
├── cmd/
│   └── demo/
│       └── main.go      # 快速演示程序
├── README.md            # 双语文档（英文/中文）
├── USAGE.md             # 使用示例
├── TODO.md              # 改进计划
├── 教程.md               # 中文教程（费曼笔法）
```

## 核心 API

### 基本 Try/Catch/Finally

```go
tb := gotrycatch.Try(func() {
    // 可能 panic 的代码
})
tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
    // 处理特定类型错误
})
tb = tb.CatchAny(func(err interface{}) {
    // 兜底处理
})
tb.Finally(func() {
    // 清理工作
})
```

### TryBlock 状态查询

```go
tb := gotrycatch.Try(func() { panic("err") })

tb.HasError()      // bool - 是否有错误
tb.GetError()      // interface{} - 获取错误值
tb.GetErrorType()  // string - 获取错误类型名（如 "string", "errors.ValidationError"）
tb.IsHandled()     // bool - 错误是否已处理
tb.String()        // string - 友好的字符串表示
```

### TryWithResult - 支持返回值

```go
tb := gotrycatch.TryWithResult(func() int {
    return 42
})

tb.OnSuccess(func(result int) {
    fmt.Println("成功:", result)
})

tb = gotrycatch.CatchWithResult[int, string](tb, func(err string) {
    fmt.Println("错误:", err)
})

result := tb.OrElse(0)  // 有错误时返回默认值
result := tb.OrElseGet(func() int { return computeDefault() })  // 延迟计算默认值
```

### 调试模式

```go
gotrycatch.SetDebug(true)   // 开启调试，输出类型匹配日志
gotrycatch.IsDebug()        // 查询调试状态
```

### 断言辅助函数

```go
gotrycatch.Assert(condition, err)           // 条件为 false 时抛出 err
gotrycatch.AssertNoError(err, "operation")  // err 不为 nil 时抛出包装错误
```

## 错误类型（errors 包）

所有错误类型都包含以下增强字段：
- `File` - 源文件名
- `Line` - 行号
- `Function` - 函数名
- `Timestamp` - 时间戳
- `Stack` - 调用堆栈

### 内置错误类型

| 类型 | 专有字段 | 构造函数 |
|------|----------|----------|
| `ValidationError` | Field, Message, Code | `NewValidationError(field, message, code)` |
| `DatabaseError` | Operation, Table, Cause | `NewDatabaseError(operation, table, cause)` |
| `NetworkError` | URL, StatusCode, Timeout | `NewNetworkError(url, code)` / `NewNetworkTimeoutError(url)` |
| `BusinessLogicError` | Rule, Details | `NewBusinessLogicError(rule, details)` |
| `ConfigError` | Key, Value, Reason | `NewConfigError(key, value, reason)` |
| `AuthError` | Operation, User, Reason | `NewAuthError(operation, user, reason)` |
| `RateLimitError` | Resource, Limit, Current, RetryAfter | `NewRateLimitError(resource, limit, current, retryAfter)` |

### 错误类型方法

```go
err := errors.NewValidationError("email", "invalid", 1001)

err.Error()     // string - 错误描述（含位置信息）
err.ToMap()     // map[string]interface{} - 结构化数据
err.ToJSON()    // ([]byte, error) - JSON 格式
err.Unwrap()    // error - 底层错误（DatabaseError 支持）
err.Is(target)  // bool - 错误匹配判断
```

## 方法链式调用规则

- `Catch[T]` 是**函数**，不是方法 —— 必须用 `gotrycatch.Catch[T](tb, handler)`
- `CatchWithResult[T, E]` 是**函数**，不是方法 —— 必须用 `gotrycatch.CatchWithResult[T, E](tb, handler)`
- `CatchAnyWithResult` 是**函数** —— 必须用 `gotrycatch.CatchAnyWithResult(tb, handler)`
- `CatchAny` 是**方法** —— 可链式调用：`tb.CatchAny(handler)`
- `Finally` 是**方法** —— 可链式调用：`tb.CatchAny(handler).Finally(cleanup)`

```go
// ❌ 错误：方法不能有类型参数
// tb.Catch[ErrorType](handler)

// ✅ 正确：使用函数形式
tb = gotrycatch.Catch[ErrorType](tb, handler)
```

## Agent 排错指南

### 获取错误详情

```go
tb := gotrycatch.Try(func() { ... })

if tb.HasError() {
    // 1. 获取错误类型
    errorType := tb.GetErrorType()

    // 2. 根据类型获取结构化信息
    if err, ok := tb.GetError().(errors.ValidationError); ok {
        jsonData, _ := err.ToJSON()
        fmt.Println(string(jsonData))
        // 输出: {"type":"ValidationError","field":"...","code":...,"file":"...","line":...}
    }
}
```

### 开启调试追踪类型匹配

```go
gotrycatch.SetDebug(true)
// 类型不匹配时会输出: [gotrycatch] Catch: type string does not match target type int
```

### 解析错误堆栈

```go
tb := gotrycatch.Try(func() {
    panic(errors.NewValidationError("field", "msg", 1001))
})

if err, ok := tb.GetError().(errors.ValidationError); ok {
    for i, frame := range err.Stack {
        fmt.Printf("%d: %s\n", i, frame)
    }
}
```

## 测试覆盖

运行 `go test -v` 可验证所有功能：
- 基本功能：Try, Catch, CatchAny, Finally, Throw
- 状态查询：GetError, HasError, IsHandled, String, GetErrorType
- 调试模式：SetDebug, IsDebug
- 断言函数：Assert, AssertNoError
- TryWithResult：完整流程测试（OnSuccess, OnError, OrElse, OrElseGet）
- CatchWithResult, CatchAnyWithResult
- 错误类型：所有7种类型的创建、ToMap、ToJSON、Unwrap、Is

## 扩展错误类型

添加新错误类型的步骤：
1. 在 `errors/errors.go` 中添加结构体（包含 File, Line, Function, Timestamp, Stack 字段）
2. 实现 `Error() string` 方法
3. 实现 `Unwrap()`, `Is()`, `ToMap()`, `ToJSON()` 方法
4. 添加 `NewXxxError()` 构造函数（调用 `captureCaller(1)` 和 `captureStack(1)`）
5. 在 `gotrycatch_test.go` 中添加测试
6. 更新 examples/main.go 添加使用示例
7. 更新 README.md 和 教程.md

## 文档规范

- README.md 为双语文档（英文 + 中文，使用锚点导航）
- 保持两种语言内容同步
- 代码示例在两种语言中应保持一致
- 教程.md 使用费曼笔法，通俗易懂
