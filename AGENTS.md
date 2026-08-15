尽可能用中文进行思考、推理和输出！

# AGENTS.md

帮助 AI Agent 在 GoTryCatch 仓库中高效工作的指南。

## 项目概述

GoTryCatch 是一个利用 Go 泛型实现类型安全异常处理的 Go 库。它通过封装 Go 内置的 panic/recover 机制，为 Go 带来了类似 try-catch 的异常处理能力。

**模块路径**: `github.com/linkerlin/gotrycatch`
**Go 版本**: 1.21+（泛型支持所需；`panic(nil)` 检测语义依赖 1.21+）
**版本**: 1.3.0

## 常用命令

### 运行测试
```bash
go test -v
```

### 运行示例
```bash
# 演示程序（11个详细Demo：基本用法/错误信息/结构化输出/TryWithResult/调试模式/断言/状态查询/错误链/真实场景/v2 Run惯用法）
go run ./cmd/demo
```

### 构建
```bash
go build ./...
```

## 项目结构

```
/
├── gotrycatch.go        # 主包 - Try/Catch/Finally 核心实现（blockCore 统一分发）
├── run.go               # v2 - Run/Run1 子句式 API + CatchAs + PanicError
├── gotrycatch_test.go   # 单元测试
├── run_test.go          # v2 API 测试
├── gotrycatch_bench_test.go # 基准测试
├── errtypes/
│   └── errors.go        # 预定义错误类型（v2 正名；BaseError 嵌入，含堆栈、位置、时间戳）
├── errors/
│   ├── errors.go        # 兼容别名层（类型别名 + 构造函数直接绑定，仅保证编译兼容）
│   └── forward_test.go  # 转发层验证测试
├── cmd/
│   └── demo/
│       └── main.go      # 演示程序（11个Demo）
├── README.md            # 双语文档（英文/中文）
├── USAGE.md             # 使用示例
├── TODO.md              # 改进计划
├── 演进方案.md           # 架构审阅与演进路线
├── 教程.md               # 中文教程（费曼笔法）
```

## 核心 API

### 基本 Try/Catch/Finally

```go
tb := gotrycatch.Try(func() {
    // 可能 panic 的代码
})
tb = gotrycatch.Catch[errtypes.ValidationError](tb, func(err errtypes.ValidationError) {
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
tb.GetErrorType()  // string - 获取错误类型名（如 "string", "errtypes.ValidationError"）
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

## 错误类型（errtypes 包）

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
err := errtypes.NewValidationError("email", "invalid", 1001)

err.Error()     // string - 错误描述（含位置信息）
err.ToMap()     // map[string]interface{} - 结构化数据
err.ToJSON()    // ([]byte, error) - JSON 格式
err.Unwrap()    // error - 底层错误（DatabaseError 支持）
err.Is(target)  // bool - 错误匹配判断
```

## v2 核心 API：Run 子句式（推荐）

```go
err := gotrycatch.Run(func() { queryUser(id) },
    gotrycatch.OnAs(func(e errtypes.DatabaseError) { retry(id) }),  // errors.As 穿透 %w 包装
    gotrycatch.On(func(e errtypes.RateLimitError) { wait(e.RetryAfter) }),
    gotrycatch.Any(func(v interface{}) { log(v) }),                 // 兜底
    gotrycatch.Cleanup(func() { conn.Close() }),                    // 总是执行，LIFO
)
if err != nil { /* 未处理 panic 以 error 返回；error 型 panic 直通，其余包装为 *PanicError */ }

v, err := gotrycatch.Run1(func() int { return compute() }, clauses...)  // 带返回值
```

- 实现在 `run.go`：`Clause`/`On`/`OnAs`/`Any`/`Cleanup`/`Run`/`Run1`/`CatchAs`/`CatchAsWithResult`/`PanicError`
- `TryBlock.Err()` / `TryBlockWithResult.Err()` 把捕获的 panic 桥接为 error
- `CanonicalErrorType()` 返回去指针短类型名（Agent 消费）
- 经典链式 API（Try/Catch/CatchAny/Finally）行为与 v1 完全一致，全量保留

## 方法链式调用规则（经典 API）

- `Catch[T]` 是**函数**，不是方法 —— 必须用 `gotrycatch.Catch[T](tb, handler)`
- `CatchAs[E]` 是**函数** —— `gotrycatch.CatchAs[E](tb, handler)`（errors.As 语义）
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
    if err, ok := tb.GetError().(errtypes.ValidationError); ok {
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
    panic(errtypes.NewValidationError("field", "msg", 1001))
})

if err, ok := tb.GetError().(errtypes.ValidationError); ok {
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

**库内新增类型** 3 步（BaseError 已提供 File/Line/Function/Timestamp/Stack 字段、Unwrap 和 ToMap 公共键）：
1. 在 `errtypes/errors.go` 中定义结构体：嵌入 `BaseError` + 专有字段，实现 `Error()`, `Is()`, `ToMap()`（先 `m := e.baseMap()` 再加 `type` 和专有键）、`ToJSON()`（一行 `json.Marshal(e.ToMap())`）
2. 构造函数调用 `newBase(1)`（自动捕获调用方位置与堆栈）
3. 在 `errtypes/errors_test.go` 添加测试；如需演示，更新 cmd/demo/main.go 和 README.md

**用户代码自定义类型**：嵌入 `errtypes.BaseError`，构造函数调 `errtypes.NewBase()`（导出版，归因于用户调用行），示例见 USAGE.md「自定义错误类型」。

注意：有底层错误的类型需自行实现 `Unwrap()`（覆盖 BaseError 的默认 nil 返回），参见 DatabaseError。新类型只加入 `errtypes` 包；`errors/` 兼容层冻结。

## 文档规范

- README.md 为双语文档（英文 + 中文，使用锚点导航）
- 保持两种语言内容同步
- 代码示例在两种语言中应保持一致
- 教程.md 使用费曼笔法，通俗易懂
