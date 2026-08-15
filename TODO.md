# GoTryCatch 改进计划

> v1.4 起的改进以《演进方案.md》为准，本文件保留历史记录并追踪新阶段。

## 第七阶段：v2.0 语义升级（易用性 + Go 惯用法）✅

- [x] `Run`/`Run1` 子句式 API（run.go）：未处理 panic 以 error 返回——结构性消灭吞错
- [x] `On`/`OnAs`/`Any`/`Cleanup` 子句构造器；Cleanup 总是执行（LIFO），handler panic 也不跳过
- [x] `PanicError`：非 error panic 的 error 包装（含 Stack() 懒格式化）；error 型 panic 直通，errors.Is/As 全链路可用
- [x] `CatchAs`/`CatchAsWithResult`：errors.As 语义穿透 `%w` 包装（值/指针形态各自匹配）
- [x] `TryBlock.Err()`/`TryBlockWithResult.Err()`：panic→error 桥接
- [x] `CanonicalErrorType()`：去指针短类型名（Agent 消费）
- [x] Try/TryWithResult 恢复时捕获调用栈 pcs（PanicError.Stack 消费）
- [x] 删除 `CatchWithReturn`（用 TryWithResult + CatchWithResult + OrElse 替代）
- [x] `errtypes` 正名包；旧 `errors/` 变纯别名层（类型别名 + 构造函数 var 绑定，位置归因与直连一致），forward_test.go 验证
- [x] run_test.go 28 个新测试（穿透/Cleanup 三态/handler panic/嵌套/并发）
- [x] Demo11 演示 v2 Run API；README 双语 v2 章节 + 迁移指南；AGENTS.md 同步
- [x] Version → 2.0.0

---

## 第六阶段：v1.5 内部去重与结构收敛 ✅

- [x] 主包 `blockCore` 统一分发核心：`dispatch`/`catchAny`/`rethrow`，两套 Try/Catch/Finally 委托同一实现（零行为变更，现有测试零修改全绿）
- [x] errors 包 `BaseError` 嵌入：7 类型共用 File/Line/Function/Timestamp/Stack + `newBase(1)`，errors.go 673→443 行；主包 gotrycatch.go 425→394 行
- [x] 堆栈归因与原版逐字节一致（git worktree 实测比对 File/Line/Stack0/栈深）
- [x] 基准无回退反而变快：panic+Catch 496→417ns（mismatch 日志惰性求值），分配数不变
- [x] 演示入口收敛：examples/ 10 个 Demo 并入 cmd/demo，删除 examples/，全文档引用同步
- [x] 测试样板收敛：`assertBaseFields` helper 替代 7 处重复断言块
- [x] AGENTS.md 扩展错误类型步骤 7 步→3 步

---

## 第五阶段：v1.4 正确性修复与工程化 ✅

- [x] 修复 `SetDebug` 数据竞争（`atomic.Bool`）+ 并发测试（`-race` 通过）
- [x] 统一 Go 版本承诺：go.mod 降为 `go 1.21`，README/AGENTS.md 同步
  （不改 1.18：`TestNilPanicValue` 依赖 Go 1.21+ 的 `panic(nil)` 语义，声明 1.18 是虚假承诺）
- [x] `OrElseGet(nil)` 返回零值而非 panic
- [x] 修复测试假断言 `r != r`
- [x] README 双语新增"语义边界"（handler panic 跳过 Finally / Finally panic 覆盖原错 / 跨 goroutine 不可捕获 / Catch 精确断言）
- [x] 新增 CI（`.github/workflows/ci.yml`：race + vet + lint + 1.21/1.25 矩阵）
- [x] 新增基准测试（`gotrycatch_bench_test.go`），实测数据写入 README 性能一节
- [x] 修复 `errors/errors_test.go` 既有 gofmt 问题

---

## 第一阶段：TryBlock 易用性增强 ✅

### 1.1 状态查询方法
- [x] 添加 `GetError()` 方法 - 获取捕获的错误
- [x] 添加 `HasError()` 方法 - 判断是否有错误
- [x] 添加 `IsHandled()` 方法 - 判断错误是否已处理
- [x] 实现 `String()` 方法 - 友好的字符串表示
- [x] 添加 `GetErrorType()` 方法 - 获取错误类型名称

### 1.2 调试支持
- [x] 添加 `SetDebug(bool)` 全局开关
- [x] 添加 `IsDebug()` 查询方法
- [x] Catch 类型不匹配时输出调试信息
- [x] Finally 重新抛出时输出调试信息

### 1.3 断言辅助函数
- [x] 添加 `Assert(condition bool, err interface{})` 函数
- [x] 添加 `AssertNoError(err error, msg string)` 函数

### 1.4 测试更新
- [x] 为新增方法添加单元测试

---

## 第二阶段：errors 包错误显性化 ✅

### 2.1 基础错误信息增强
- [x] 添加 `Stack` 字段 - 调用堆栈
- [x] 添加 `File` 字段 - 源文件名
- [x] 添加 `Line` 字段 - 行号
- [x] 添加 `Timestamp` 字段 - 时间戳
- [x] 添加 `Function` 字段 - 函数名

### 2.2 错误链支持
- [x] 实现 `Unwrap()` 方法
- [x] 实现 `Is(error) bool` 方法

### 2.3 结构化输出
- [x] 添加 `ToMap() map[string]interface{}` 方法
- [x] 添加 `ToJSON() ([]byte, error)` 方法
- [x] 更新所有错误类型

### 2.4 新增错误类型
- [x] 添加 `ConfigError` - 配置错误
- [x] 添加 `AuthError` - 认证错误
- [x] 添加 `RateLimitError` - 限流错误

### 2.5 测试更新
- [x] 为新增功能添加单元测试

---

## 第三阶段：新增便捷功能 ✅

### 3.1 TryWithResult
- [x] 添加 `TryWithResult[T](func() T)` 支持返回值
- [x] 添加 `CatchWithResult[T, E]` 方法
- [x] 添加 `CatchAnyWithResult[T]` 方法
- [x] 添加 `OnSuccess(func(T))` 方法
- [x] 添加 `OnError(func(interface{}))` 方法
- [x] 添加 `OrElse(defaultValue T) T` 方法
- [x] 添加 `OrElseGet(supplier func() T) T` 方法

### 3.2 测试更新
- [x] 为新增功能添加单元测试

---

## 第四阶段：文档和示例更新 ✅

### 4.1 文档更新
- [x] 更新 AGENTS.md

### 4.2 示例更新
- [x] 更新 examples/main.go（10个详细Demo）
- [x] 更新 cmd/demo/main.go（快速演示）

---

## 完成摘要

所有四个阶段的改进已完成！

### 新增功能列表

**TryBlock 状态查询：**
- `GetError()` - 获取错误值
- `HasError()` - 判断是否有错误
- `IsHandled()` - 判断错误是否已处理
- `String()` - 友好的字符串表示
- `GetErrorType()` - 获取错误类型名称

**调试支持：**
- `SetDebug(bool)` - 开启/关闭调试模式
- `IsDebug()` - 查询调试状态

**断言辅助：**
- `Assert(condition, err)` - 条件断言
- `AssertNoError(err, msg)` - 错误断言

**TryWithResult：**
- `TryWithResult[T](func() T)` - 支持返回值
- `CatchWithResult[T, E]` - 带返回值的 Catch
- `OnSuccess(func(T))` - 成功回调
- `OnError(func(interface{}))` - 错误回调
- `OrElse(defaultValue)` - 默认值
- `OrElseGet(supplier)` - 延迟计算默认值

**错误信息增强：**
- 所有错误类型新增：`File`, `Line`, `Function`, `Timestamp`, `Stack`
- 新增方法：`ToMap()`, `ToJSON()`, `Unwrap()`, `Is()`

**新增错误类型：**
- `ConfigError` - 配置错误
- `AuthError` - 认证错误
- `RateLimitError` - 限流错误
