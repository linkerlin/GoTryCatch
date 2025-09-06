package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/linkerlin/gotrycatch"
	trycatcherrors "github.com/linkerlin/gotrycatch/errors"
)

// ============= 模拟业务函数 =============
func validateUser(name, email string, age int) {
	if name == "" {
		gotrycatch.Throw(trycatcherrors.NewValidationError("name", "name cannot be empty", 1001))
	}

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

	num, err := strconv.Atoi(s)
	if err != nil {
		gotrycatch.Throw(err)
	}
	return num
}

// ============= Demo示例 =============

func demo1_BasicUsage() {
	fmt.Println("=== Demo 1: 基本用法 ===")

	tb := gotrycatch.Try(func() {
		validateUser("", "test@example.com", 25)
	})

	tb = gotrycatch.Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		fmt.Printf("✓ Caught ValidationError: %s\n", err.Error())
		fmt.Printf("  Field: %s, Code: %d\n", err.Field, err.Code)
	})

	tb.Finally(func() {
		fmt.Println("  Cleanup completed")
	})
}

func demo2_MultipleExceptionTypes() {
	fmt.Println("\n=== Demo 2: 多种异常类型处理 ===")

	scenarios := []struct {
		name string
		fn   func()
	}{
		{"ValidationError", func() { validateUser("", "test@example.com", 25) }},
		{"DatabaseError", func() { accessDatabase("delete_all") }},
		{"NetworkError", func() { callExternalAPI("http://timeout.com") }},
		{"BusinessLogicError", func() { processOrder("refund") }},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n--- Testing %s ---\n", scenario.name)

		tb := gotrycatch.Try(scenario.fn)

		tb = gotrycatch.Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
			fmt.Printf("✓ Validation issue: %s (Code: %d)\n", err.Message, err.Code)
		})

		tb = gotrycatch.Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
			fmt.Printf("✓ Database issue: %s on %s\n", err.Operation, err.Table)
		})

		tb = gotrycatch.Catch[trycatcherrors.NetworkError](tb, func(err trycatcherrors.NetworkError) {
			if err.Timeout {
				fmt.Printf("✓ Network timeout: %s\n", err.URL)
			} else {
				fmt.Printf("✓ Network error %d: %s\n", err.StatusCode, err.URL)
			}
		})

		tb = gotrycatch.Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
			fmt.Printf("✓ Business rule violation: %s\n", err.Rule)
		})

		tb.Finally(func() {
			fmt.Println("  Scenario completed")
		})
	}
}

func demo3_InterfaceAndBuiltinTypes() {
	fmt.Println("\n=== Demo 3: 接口类型和内置类型 ===")

	scenarios := []struct {
		name string
		fn   func()
	}{
		{"String error", func() { parseNumber("invalid") }},
		{"Standard error", func() { parseNumber("abc") }},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n--- Testing %s ---\n", scenario.name)

		tb := gotrycatch.Try(scenario.fn)

		tb = gotrycatch.Catch[string](tb, func(err string) {
			fmt.Printf("✓ String error: %s\n", err)
		})

		tb = gotrycatch.Catch[error](tb, func(err error) {
			fmt.Printf("✓ Standard error: %s\n", err.Error())
		})

		tb = tb.CatchAny(func(err interface{}) {
			fmt.Printf("✓ Unknown error: %v\n", err)
		})

		tb.Finally(func() {
			fmt.Println("  Parsing attempt completed")
		})
	}
}

func demo4_WithReturnValues() {
	fmt.Println("\n=== Demo 4: 带返回值的异常处理 ===")

	tb := gotrycatch.Try(func() {
		validateUser("John", "invalid", 25)
	})

	result, tb := gotrycatch.CatchWithReturn[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) interface{} {
		fmt.Printf("✓ Validation failed: %s\n", err.Message)
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"code":    err.Code,
		}
	})

	if result != nil {
		if resultMap, ok := result.(map[string]interface{}); ok {
			fmt.Printf("  Return value: %+v\n", resultMap)
		}
	}

	tb.Finally(func() {
		fmt.Println("  Demo 4 completed")
	})
}

func demo5_NestedTryCatch() {
	fmt.Println("\n=== Demo 5: 嵌套异常处理 ===")

	tb := gotrycatch.Try(func() {
		fmt.Println("  Outer try block started")

		innerTb := gotrycatch.Try(func() {
			fmt.Println("    Inner try block started")
			validateUser("", "test@example.com", 25)
		})

		innerTb = gotrycatch.Catch[trycatcherrors.ValidationError](innerTb, func(err trycatcherrors.ValidationError) {
			fmt.Printf("    ✓ Inner catch: %s\n", err.Message)
			// 在内层处理后，抛出新的异常到外层
			gotrycatch.Throw(trycatcherrors.NewBusinessLogicError("user_validation", "failed inner validation: "+err.Field))
		})

		innerTb.Finally(func() {
			fmt.Println("    Inner finally block")
		})

		fmt.Println("  This should not be reached")
	})

	tb = gotrycatch.Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
		fmt.Printf("  ✓ Outer catch: %s\n", err.Details)
	})

	tb.Finally(func() {
		fmt.Println("  Nested example completed")
	})
}

func demo6_RealWorldExample() {
	fmt.Println("\n=== Demo 6: 真实场景示例 ===")

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
			fmt.Printf("  ❌ Input validation failed: %s\n", err.Message)
		})

		tb = gotrycatch.Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
			fmt.Printf("  ❌ Database operation failed: %s\n", err.Cause.Error())
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
			fmt.Printf("  📝 Order processing completed for user: %s at %s\n",
				userID, time.Now().Format("15:04:05"))
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
	fmt.Println("基于泛型的 Go 异常处理库 Demo")
	fmt.Println("=====================================")

	demo1_BasicUsage()
	demo2_MultipleExceptionTypes()
	demo3_InterfaceAndBuiltinTypes()
	demo4_WithReturnValues()
	demo5_NestedTryCatch()
	demo6_RealWorldExample()

	fmt.Println("\n🎉 所有Demo执行完成！")
}
