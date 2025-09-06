package main

import (
	"fmt"
	"github.com/linkerlin/gotrycatch"
	"github.com/linkerlin/gotrycatch/errors"
)

func main() {
	fmt.Println("GoTryCatch - Go 泛型异常处理库")
	fmt.Println("================================")
	
	// 测试 1: 简单字符串 panic
	fmt.Println("\n--- 测试 1: 字符串错误 ---")
	tb1 := gotrycatch.Try(func() {
		fmt.Println("抛出字符串错误...")
		gotrycatch.Throw("simple string error")
	})
	
	tb1 = gotrycatch.Catch[string](tb1, func(err string) {
		fmt.Printf("✓ 捕获到字符串错误: %s\n", err)
	})
	
	tb1.Finally(func() {
		fmt.Println("✓ 字符串错误处理完成")
	})

	// 测试 2: ValidationError
	fmt.Println("\n--- 测试 2: ValidationError ---")
	tb2 := gotrycatch.Try(func() {
		fmt.Println("抛出验证错误...")
		gotrycatch.Throw(errors.NewValidationError("name", "姓名不能为空", 1001))
	})

	tb2 = gotrycatch.Catch[errors.ValidationError](tb2, func(err errors.ValidationError) {
		fmt.Printf("✓ 捕获到验证错误: %s (字段: %s, 代码: %d)\n", err.Message, err.Field, err.Code)
	})

	tb2 = tb2.CatchAny(func(err interface{}) {
		fmt.Printf("✓ CatchAny 捕获到错误: %v (类型: %T)\n", err, err)
	})

	tb2.Finally(func() {
		fmt.Println("✓ 验证错误处理完成")
	})

	fmt.Println("\n请查看 examples/ 目录获取更多详细示例")
	fmt.Println("运行示例: go run examples/main.go")
}