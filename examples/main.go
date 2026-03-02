package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/linkerlin/gotrycatch"
	trycatcherrors "github.com/linkerlin/gotrycatch/errors"
)

// ============= 模拟业务函数 =============
func validateUser(name, email string, age int) {
	gotrycatch.Assert(name != "", trycatcherrors.NewValidationError("name", "name cannot be empty", 1001))

	if age < 0 || age > 150 {
		gotrycatch.Throw(trycatcherrors.NewValidationError("age", "age must be between 0 and 150", 1002))
	}

	if email == "invalid" {
		gotrycatch.Throw(trycatcherrors.NewValidationError("email", "invalid email format", 1003))
	}
}

func accessDatabase(operation string) {
	if operation == "delete_all" {
		gotrycatch.Throw(trycatcherrors.NewDatabaseError("DELETE", "users", errors.New("permission denied")))
	}

	if operation == "timeout" {
		gotrycatch.Throw(trycatcherrors.NewDatabaseError("SELECT", "orders", errors.New("connection timeout")))
	}
}

func callExternalAPI(url string) {
	if url == "http://timeout.com" {
		gotrycatch.Throw(trycatcherrors.NewNetworkTimeoutError(url))
	}

	if url == "http://notfound.com" {
		gotrycatch.Throw(trycatcherrors.NewNetworkError(url, 404))
	}
}

func processOrder(orderType string) {
	if orderType == "refund" {
		gotrycatch.Throw(trycatcherrors.NewBusinessLogicError("refund_policy", "refunds not allowed after 30 days"))
	}
}

func parseNumber(s string) int {
	if s == "invalid" {
		gotrycatch.Throw("invalid number format")
	}
	return 42
}

func loadConfig(key string) string {
	if key == "invalid_key" {
		gotrycatch.Throw(trycatcherrors.NewConfigError(key, "", "key not found in config"))
	}
	return "config_value"
}

func authenticate(user, password string) {
	if password != "secret" {
		gotrycatch.Throw(trycatcherrors.NewAuthError("login", user, "invalid password"))
	}
}

func checkRateLimit(resource string, current int) {
	if current > 100 {
		gotrycatch.Throw(trycatcherrors.NewRateLimitError(resource, 100, current, 60))
	}
}

// ============= Demo示例 =============

func demo1_BasicUsage() {
	fmt.Println("=== Demo 1: 基本用法 ===")

	tb := gotrycatch.Try(func() {
		validateUser("", "test@example.com", 25)
	})

	// 新增：使用 HasError() 判断是否有错误
	if tb.HasError() {
		fmt.Printf("✓ 检测到错误: %s\n", tb.GetErrorType())
	}

	tb = gotrycatch.Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		fmt.Printf("✓ Caught ValidationError: %s\n", err.Message)
		fmt.Printf("  Field: %s, Code: %d\n", err.Field, err.Code)
		// 新增：显示调用位置
		fmt.Printf("  Location: %s:%d\n", err.File, err.Line)
		fmt.Printf("  Function: %s\n", err.Function)
	})

	tb.Finally(func() {
		fmt.Println("  Cleanup completed")
	})
}

func demo2_ErrorInfo() {
	fmt.Println("\n=== Demo 2: 错误信息显性化 ===")

	tb := gotrycatch.Try(func() {
		validateUser("", "test@example.com", 25)
	})

	tb = gotrycatch.Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		fmt.Printf("✓ 错误详情:\n")
		fmt.Printf("  类型: %T\n", err)
		fmt.Printf("  字段: %s\n", err.Field)
		fmt.Printf("  消息: %s\n", err.Message)
		fmt.Printf("  代码: %d\n", err.Code)
		fmt.Printf("  位置: %s:%d\n", err.File, err.Line)
		fmt.Printf("  函数: %s\n", err.Function)
		fmt.Printf("  时间: %s\n", err.Timestamp.Format(time.RFC3339))
		fmt.Printf("  堆栈深度: %d 层\n", len(err.Stack))
	})

	tb.Finally(func() {
		fmt.Println("  错误处理完成")
	})
}

func demo3_StructuredOutput() {
	fmt.Println("\n=== Demo 3: 结构化输出 (ToMap/ToJSON) ===")

	tb := gotrycatch.Try(func() {
		accessDatabase("delete_all")
	})

	tb = gotrycatch.Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		// 使用 ToMap 获取结构化数据
		m := err.ToMap()
		fmt.Printf("✓ ToMap 输出:\n")
		for k, v := range m {
			fmt.Printf("  %s: %v\n", k, v)
		}

		// 使用 ToJSON 获取 JSON 格式
		jsonBytes, _ := err.ToJSON()
		fmt.Printf("\n✓ ToJSON 输出:\n  %s\n", string(jsonBytes))
	})

	tb.Finally(func() {
		fmt.Println("  结构化输出演示完成")
	})
}

func demo4_TryWithResult() {
	fmt.Println("\n=== Demo 4: TryWithResult - 支持返回值 ===")

	// 成功情况
	tb1 := gotrycatch.TryWithResult(func() int {
		return parseNumber("valid")
	})

	tb1.OnSuccess(func(result int) {
		fmt.Printf("✓ 成功获取结果: %d\n", result)
	})

	result1 := tb1.OrElse(0)
	fmt.Printf("  OrElse 结果: %d\n", result1)

	// 失败情况
	tb2 := gotrycatch.TryWithResult(func() int {
		return parseNumber("invalid")
	})

	tb2 = gotrycatch.CatchWithResult[int, string](tb2, func(err string) {
		fmt.Printf("✓ 捕获错误: %s\n", err)
	})

	result2 := tb2.OrElse(-1)
	fmt.Printf("  OrElse 默认值: %d\n", result2)

	// 使用 OrElseGet
	tb3 := gotrycatch.TryWithResult(func() string {
		gotrycatch.Throw("error")
		return ""
	})
	result3 := tb3.OrElseGet(func() string { return "default_value" })
	fmt.Printf("  OrElseGet 默认值: %s\n", result3)
}

func demo5_NewErrorTypes() {
	fmt.Println("\n=== Demo 5: 新增错误类型 ===")

	// ConfigError
	fmt.Println("\n--- ConfigError ---")
	tb := gotrycatch.Try(func() {
		loadConfig("invalid_key")
	})
	tb = gotrycatch.Catch[trycatcherrors.ConfigError](tb, func(err trycatcherrors.ConfigError) {
		fmt.Printf("✓ ConfigError: key=%s, reason=%s\n", err.Key, err.Reason)
	})
	tb.Finally(func() {})

	// AuthError
	fmt.Println("\n--- AuthError ---")
	tb = gotrycatch.Try(func() {
		authenticate("admin", "wrong_password")
	})
	tb = gotrycatch.Catch[trycatcherrors.AuthError](tb, func(err trycatcherrors.AuthError) {
		fmt.Printf("✓ AuthError: operation=%s, user=%s, reason=%s\n", err.Operation, err.User, err.Reason)
	})
	tb.Finally(func() {})

	// RateLimitError
	fmt.Println("\n--- RateLimitError ---")
	tb = gotrycatch.Try(func() {
		checkRateLimit("/api/data", 150)
	})
	tb = gotrycatch.Catch[trycatcherrors.RateLimitError](tb, func(err trycatcherrors.RateLimitError) {
		fmt.Printf("✓ RateLimitError: resource=%s, limit=%d, current=%d, retryAfter=%ds\n",
			err.Resource, err.Limit, err.Current, err.RetryAfter)
	})
	tb.Finally(func() {})
}

func demo6_DebugMode() {
	fmt.Println("\n=== Demo 6: 调试模式 ===")

	// 开启调试模式
	gotrycatch.SetDebug(true)
	fmt.Printf("调试模式: %v\n", gotrycatch.IsDebug())

	tb := gotrycatch.Try(func() {
		validateUser("", "test@example.com", 25)
	})

	tb = gotrycatch.Catch[int](tb, func(err int) {
		fmt.Printf("不会执行，类型不匹配\n")
	})

	tb = gotrycatch.Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		fmt.Printf("✓ 捕获到 ValidationError\n")
	})

	tb.Finally(func() {
		fmt.Println("  调试模式演示完成")
	})

	// 关闭调试模式
	gotrycatch.SetDebug(false)
}

func demo7_AssertHelpers() {
	fmt.Println("\n=== Demo 7: 断言辅助函数 ===")

	// Assert - 条件断言
	fmt.Println("\n--- Assert ---")
	tb := gotrycatch.Try(func() {
		value := ""
		gotrycatch.Assert(value != "", trycatcherrors.NewValidationError("value", "cannot be empty", 2001))
		fmt.Println("这行不会执行")
	})
	tb = gotrycatch.Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		fmt.Printf("✓ Assert 触发错误: %s\n", err.Message)
	})
	tb.Finally(func() {})

	// AssertNoError - 错误断言
	fmt.Println("\n--- AssertNoError ---")
	tb = gotrycatch.Try(func() {
		_, err := json.Marshal(make(chan int))
		gotrycatch.AssertNoError(err, "JSON marshal failed")
	})
	tb = gotrycatch.Catch[error](tb, func(err error) {
		fmt.Printf("✓ AssertNoError 触发错误: %v\n", err)
	})
	tb.Finally(func() {})
}

func demo8_TryBlockState() {
	fmt.Println("\n=== Demo 8: TryBlock 状态查询 ===")

	// 无错误状态
	tb1 := gotrycatch.Try(func() {})
	fmt.Printf("无错误: HasError=%v, IsHandled=%v\n", tb1.HasError(), tb1.IsHandled())
	fmt.Printf("String: %s\n", tb1.String())

	// 有错误未处理
	tb2 := gotrycatch.Try(func() {
		panic("test error")
	})
	fmt.Printf("\n有错误未处理: HasError=%v, IsHandled=%v\n", tb2.HasError(), tb2.IsHandled())
	fmt.Printf("GetErrorType: %s\n", tb2.GetErrorType())
	fmt.Printf("String: %s\n", tb2.String())

	// 有错误已处理
	tb2 = gotrycatch.Catch[string](tb2, func(err string) {})
	fmt.Printf("\n有错误已处理: HasError=%v, IsHandled=%v\n", tb2.HasError(), tb2.IsHandled())
}

func demo9_ErrorChain() {
	fmt.Println("\n=== Demo 9: 错误链支持 ===")

	cause := errors.New("underlying connection error")
	dbErr := trycatcherrors.NewDatabaseError("SELECT", "users", cause)

	fmt.Printf("DatabaseError: %s\n", dbErr.Error())
	fmt.Printf("Unwrap: %v\n", dbErr.Unwrap())

	// 使用 errors.Is 检查
	if dbErr.Unwrap() == cause {
		fmt.Println("✓ Unwrap 返回了原始错误")
	}

	// 使用 errors.As 提取
	var extracted trycatcherrors.DatabaseError
	if err, ok := interface{}(dbErr).(trycatcherrors.DatabaseError); ok {
		extracted = err
		fmt.Printf("✓ 提取的 Operation: %s, Table: %s\n", extracted.Operation, extracted.Table)
	}
}

func demo10_RealWorldExample() {
	fmt.Println("\n=== Demo 10: 真实场景示例 ===")

	processUserOrder := func(userID, orderData string) {
		tb := gotrycatch.Try(func() {
			// 步骤1：验证用户输入
			if userID == "" {
				gotrycatch.Throw(trycatcherrors.NewValidationError("userID", "user ID is required", 2001))
			}

			// 步骤2：访问数据库
			if userID == "blocked_user" {
				gotrycatch.Throw(trycatcherrors.NewDatabaseError("SELECT", "users", errors.New("user account blocked")))
			}

			// 步骤3：调用外部支付API
			if orderData == "payment_failed" {
				gotrycatch.Throw(trycatcherrors.NewNetworkError("https://payment.api.com", 402))
			}

			// 步骤4：业务逻辑检查
			if orderData == "insufficient_stock" {
				gotrycatch.Throw(trycatcherrors.NewBusinessLogicError("inventory_check", "requested quantity exceeds available stock"))
			}

			fmt.Printf("  ✓ Order processed successfully for user: %s\n", userID)
		})

		tb = gotrycatch.Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
			fmt.Printf("  ❌ Input validation failed: %s (code: %d)\n", err.Message, err.Code)
		})

		tb = gotrycatch.Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
			fmt.Printf("  ❌ Database operation failed: %s on %s\n", err.Operation, err.Table)
		})

		tb = gotrycatch.Catch[trycatcherrors.NetworkError](tb, func(err trycatcherrors.NetworkError) {
			fmt.Printf("  ❌ External service error: %d from %s\n", err.StatusCode, err.URL)
		})

		tb = gotrycatch.Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
			fmt.Printf("  ❌ Business rule violation: %s\n", err.Details)
		})

		tb = tb.CatchAny(func(err interface{}) {
			fmt.Printf("  ❌ Unexpected error: %v\n", err)
		})

		tb.Finally(func() {
			fmt.Printf("  📝 Order processing completed at %s\n", time.Now().Format("15:04:05"))
		})
	}

	// 测试不同的场景
	testCases := []struct {
		userID    string
		orderData string
		desc      string
	}{
		{"user123", "valid_order", "成功案例"},
		{"", "valid_order", "验证错误"},
		{"blocked_user", "valid_order", "数据库错误"},
		{"user123", "payment_failed", "网络错误"},
		{"user123", "insufficient_stock", "业务逻辑错误"},
	}

	for i, tc := range testCases {
		fmt.Printf("\n--- Test Case %d: %s ---\n", i+1, tc.desc)
		processUserOrder(tc.userID, tc.orderData)
	}
}

func main() {
	fmt.Println("GoTryCatch v2.0 - 增强版异常处理库 Demo")
	fmt.Println("=========================================")

	demo1_BasicUsage()
	demo2_ErrorInfo()
	demo3_StructuredOutput()
	demo4_TryWithResult()
	demo5_NewErrorTypes()
	demo6_DebugMode()
	demo7_AssertHelpers()
	demo8_TryBlockState()
	demo9_ErrorChain()
	demo10_RealWorldExample()

	fmt.Println("\n🎉 所有Demo执行完成！")
	fmt.Println("\n新增功能摘要:")
	fmt.Println("  1. TryBlock 状态查询: GetError(), HasError(), IsHandled(), String(), GetErrorType()")
	fmt.Println("  2. 调试模式: SetDebug(true), IsDebug()")
	fmt.Println("  3. 错误信息增强: Stack, File, Line, Function, Timestamp")
	fmt.Println("  4. 结构化输出: ToMap(), ToJSON()")
	fmt.Println("  5. 错误链支持: Unwrap(), Is()")
	fmt.Println("  6. TryWithResult: 支持返回值, OnSuccess, OnError, OrElse, OrElseGet")
	fmt.Println("  7. 断言辅助: Assert(), AssertNoError()")
	fmt.Println("  8. 新增错误类型: ConfigError, AuthError, RateLimitError")
}
