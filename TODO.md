# GoTryCatch 改进计划

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
