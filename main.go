package main

import (
	"fmt"
	"github.com/linkerlin/gotrycatch"
	"github.com/linkerlin/gotrycatch/errors"
)

func main() {
	fmt.Println("GoTryCatch - Go 泛型异常处理库")
	fmt.Println("================================")
	
	// 简单示例
	tb := gotrycatch.Try(func() {
		fmt.Println("执行可能出错的代码...")
		gotrycatch.Throw(errors.NewValidationError("name", "姓名不能为空", 1001))
	})

	tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
		fmt.Printf("✓ 捕获到验证错误: %s (字段: %s, 代码: %d)\n", err.Message, err.Field, err.Code)
	})

	tb.Finally(func() {
		fmt.Println("✓ 清理工作完成")
	})

	fmt.Println("\n请查看 examples/ 目录获取更多详细示例")
	fmt.Println("运行示例: go run examples/main.go")
}