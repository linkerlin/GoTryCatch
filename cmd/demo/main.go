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

	// 测试 2: ValidationError（展示伪链式调用）
	fmt.Println("\n--- 测试 2: ValidationError（伪链式调用风格）---")
	func() {
		tb := gotrycatch.Try(func() {
			fmt.Println("抛出验证错误...")
			gotrycatch.Throw(errors.NewValidationError("name", "姓名不能为空", 1001))
		})
		
		tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
			fmt.Printf("✓ 捕获到验证错误: %s (字段: %s, 代码: %d)\n", err.Message, err.Field, err.Code)
		})
		
		tb = tb.CatchAny(func(err interface{}) {
			fmt.Printf("✓ CatchAny 捕获到错误: %v (类型: %T)\n", err, err)
		})
		
		tb.Finally(func() {
			fmt.Println("✓ 验证错误处理完成")
		})
	}()

	// 测试 3: 为什么不能真正链式调用的说明
	fmt.Println("\n--- 测试 3: Go 泛型限制说明 ---")
	fmt.Println("⚠️  Go 方法不能有泛型类型参数，所以不能写：")
	fmt.Println("   tb.Catch[ErrorType](handler)  // ❌ 不支持")
	fmt.Println("✅ 只能使用函数式调用：")
	fmt.Println("   gotrycatch.Catch[ErrorType](tb, handler)  // ✅ 支持")
	fmt.Println("✅ 但 CatchAny 和 Finally 是方法，可以链式调用")

	fmt.Println("\n请查看 examples/ 目录获取更多详细示例")
	fmt.Println("运行示例: go run examples/main.go")
}