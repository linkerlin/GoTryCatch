package main

import (
	"fmt"
	"github.com/linkerlin/gotrycatch"
	"github.com/linkerlin/gotrycatch/errors"
)

func main() {
	fmt.Println("GoTryCatch v2.0 - Go 泛型异常处理库")
	fmt.Println("====================================")

	// 测试 1: 简单字符串 panic + 状态查询
	fmt.Println("\n--- 测试 1: 字符串错误 + 状态查询 ---")
	tb1 := gotrycatch.Try(func() {
		fmt.Println("抛出字符串错误...")
		gotrycatch.Throw("simple string error")
	})

	// 新增：状态查询方法
	fmt.Printf("HasError: %v\n", tb1.HasError())
	fmt.Printf("GetErrorType: %s\n", tb1.GetErrorType())
	fmt.Printf("IsHandled: %v\n", tb1.IsHandled())
	fmt.Printf("String: %s\n", tb1.String())

	tb1 = gotrycatch.Catch[string](tb1, func(err string) {
		fmt.Printf("✓ 捕获到字符串错误: %s\n", err)
	})

	fmt.Printf("处理后 IsHandled: %v\n", tb1.IsHandled())

	tb1.Finally(func() {
		fmt.Println("✓ 字符串错误处理完成")
	})

	// 测试 2: ValidationError（展示错误信息增强）
	fmt.Println("\n--- 测试 2: ValidationError（错误信息增强）---")
	tb2 := gotrycatch.Try(func() {
		fmt.Println("抛出验证错误...")
		gotrycatch.Throw(errors.NewValidationError("name", "姓名不能为空", 1001))
	})

	tb2 = gotrycatch.Catch[errors.ValidationError](tb2, func(err errors.ValidationError) {
		fmt.Printf("✓ 捕获到验证错误:\n")
		fmt.Printf("  Message: %s\n", err.Message)
		fmt.Printf("  Field: %s\n", err.Field)
		fmt.Printf("  Code: %d\n", err.Code)
		fmt.Printf("  File: %s\n", err.File)
		fmt.Printf("  Line: %d\n", err.Line)
		fmt.Printf("  Function: %s\n", err.Function)
		fmt.Printf("  Timestamp: %v\n", err.Timestamp)
		fmt.Printf("  Stack depth: %d\n", len(err.Stack))

		// 结构化输出
		m := err.ToMap()
		fmt.Printf("\n  ToMap: type=%s\n", m["type"])

		jsonBytes, _ := err.ToJSON()
		fmt.Printf("  ToJSON: %s\n", string(jsonBytes))
	})

	tb2.Finally(func() {
		fmt.Println("✓ 验证错误处理完成")
	})

	// 测试 3: TryWithResult（支持返回值）
	fmt.Println("\n--- 测试 3: TryWithResult（支持返回值）---")
	tb3 := gotrycatch.TryWithResult(func() int {
		return 42
	})

	tb3.OnSuccess(func(result int) {
		fmt.Printf("✓ 成功获取结果: %d\n", result)
	})

	result := tb3.OrElse(0)
	fmt.Printf("  最终结果: %d\n", result)

	// 失败情况
	tb4 := gotrycatch.TryWithResult(func() string {
		gotrycatch.Throw("获取失败")
		return ""
	})

	tb4 = gotrycatch.CatchWithResult[string, string](tb4, func(err string) {
		fmt.Printf("✓ 捕获错误: %s\n", err)
	})

	result2 := tb4.OrElse("默认值")
	fmt.Printf("  使用默认值: %s\n", result2)

	// 测试 4: 调试模式
	fmt.Println("\n--- 测试 4: 调试模式 ---")
	fmt.Printf("当前调试模式: %v\n", gotrycatch.IsDebug())
	fmt.Println("开启调试模式...")
	gotrycatch.SetDebug(true)

	tb5 := gotrycatch.Try(func() {
		gotrycatch.Throw(errors.NewValidationError("debug_test", "调试测试", 9999))
	})

	tb5 = gotrycatch.Catch[int](tb5, func(err int) {
		// 类型不匹配，调试模式下会输出日志
	})

	tb5 = gotrycatch.Catch[errors.ValidationError](tb5, func(err errors.ValidationError) {
		fmt.Printf("✓ 调试模式捕获成功\n")
	})

	tb5.Finally(func() {})

	gotrycatch.SetDebug(false)
	fmt.Println("调试模式已关闭")

	// 测试 5: 断言辅助函数
	fmt.Println("\n--- 测试 5: 断言辅助函数 ---")
	tb6 := gotrycatch.Try(func() {
		value := ""
		gotrycatch.Assert(value != "", errors.NewValidationError("value", "不能为空", 3001))
	})

	tb6 = gotrycatch.Catch[errors.ValidationError](tb6, func(err errors.ValidationError) {
		fmt.Printf("✓ Assert 触发: %s\n", err.Message)
	})

	tb6.Finally(func() {})

	// 测试 6: 新增错误类型
	fmt.Println("\n--- 测试 6: 新增错误类型 ---")

	// ConfigError
	tb7 := gotrycatch.Try(func() {
		gotrycatch.Throw(errors.NewConfigError("db.url", "", "配置项缺失"))
	})
	tb7 = gotrycatch.Catch[errors.ConfigError](tb7, func(err errors.ConfigError) {
		fmt.Printf("✓ ConfigError: key=%s, reason=%s\n", err.Key, err.Reason)
	})

	// AuthError
	tb8 := gotrycatch.Try(func() {
		gotrycatch.Throw(errors.NewAuthError("login", "admin", "密码错误"))
	})
	tb8 = gotrycatch.Catch[errors.AuthError](tb8, func(err errors.AuthError) {
		fmt.Printf("✓ AuthError: operation=%s, user=%s\n", err.Operation, err.User)
	})

	// RateLimitError
	tb9 := gotrycatch.Try(func() {
		gotrycatch.Throw(errors.NewRateLimitError("/api/data", 100, 150, 60))
	})
	tb9 = gotrycatch.Catch[errors.RateLimitError](tb9, func(err errors.RateLimitError) {
		fmt.Printf("✓ RateLimitError: resource=%s, current=%d, limit=%d\n", err.Resource, err.Current, err.Limit)
	})

	fmt.Println("\n请查看 examples/ 目录获取更多详细示例")
	fmt.Println("运行完整示例: go run examples/main.go")

	fmt.Println("\n✅ 所有测试完成！")
}
