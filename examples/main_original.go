package main

import (
    "errors"
    "fmt"
    "strconv"
    "time"
)

// ============= 异常处理库实现 =============
type TryBlock struct {
    err     interface{}
    handled bool
}

func Try(fn func()) *TryBlock {
    tb := &TryBlock{}
    
    defer func() {
        if r := recover(); r != nil {
            tb.err = r
        }
    }()
    
    fn()
    return tb
}

// 简化版本：直接在 TryBlock 上添加泛型方法
func Catch[T any](tb *TryBlock, handler func(T)) *TryBlock {
    if tb == nil {
        return &TryBlock{}
    }
    
    if tb.err != nil && !tb.handled {
        if err, ok := tb.err.(T); ok {
            handler(err)
            tb.handled = true
        }
    }
    return tb
}

// 带返回值的版本
func CatchWithReturn[T any](tb *TryBlock, handler func(T) interface{}) (interface{}, *TryBlock) {
    if tb == nil {
        return nil, &TryBlock{}
    }
    
    if tb.err != nil && !tb.handled {
        if err, ok := tb.err.(T); ok {
            result := handler(err)
            tb.handled = true
            return result, tb
        }
    }
    return nil, tb
}

func (tb *TryBlock) CatchAny(handler func(interface{})) *TryBlock {
    if tb == nil {
        return &TryBlock{}
    }
    
    if tb.err != nil && !tb.handled {
        handler(tb.err)
        tb.handled = true
    }
    return tb
}

func (tb *TryBlock) Finally(fn func()) {
    if tb == nil {
        fn()
        return
    }
    
    defer fn()
    if tb.err != nil && !tb.handled {
        panic(tb.err) // 重新抛出未处理的异常
    }
}

func Throw(err interface{}) {
    panic(err)
}

// ============= 自定义异常类型 =============
type ValidationError struct {
    Field   string
    Message string
    Code    int
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error [%d] on field '%s': %s", e.Code, e.Field, e.Message)
}

type DatabaseError struct {
    Operation string
    Table     string
    Cause     error
}

func (e DatabaseError) Error() string {
    return fmt.Sprintf("database error during %s on table '%s': %v", e.Operation, e.Table, e.Cause)
}

type NetworkError struct {
    URL        string
    StatusCode int
    Timeout    bool
}

func (e NetworkError) Error() string {
    if e.Timeout {
        return fmt.Sprintf("network timeout when accessing %s", e.URL)
    }
    return fmt.Sprintf("network error %d when accessing %s", e.StatusCode, e.URL)
}

type BusinessLogicError struct {
    Rule    string
    Details string
}

func (e BusinessLogicError) Error() string {
    return fmt.Sprintf("business rule violation: %s - %s", e.Rule, e.Details)
}

// ============= 模拟业务函数 =============
func validateUser(name, email string, age int) {
    if name == "" {
        Throw(ValidationError{
            Field:   "name",
            Message: "name cannot be empty",
            Code:    1001,
        })
    }
    
    if age < 0 || age > 150 {
        Throw(ValidationError{
            Field:   "age", 
            Message: "age must be between 0 and 150",
            Code:    1002,
        })
    }
    
    if email == "invalid" {
        Throw(ValidationError{
            Field:   "email",
            Message: "invalid email format",
            Code:    1003,
        })
    }
}

func accessDatabase(operation string) {
    if operation == "delete_all" {
        Throw(DatabaseError{
            Operation: "DELETE",
            Table:     "users",
            Cause:     errors.New("permission denied"),
        })
    }
    
    if operation == "timeout" {
        Throw(DatabaseError{
            Operation: "SELECT",
            Table:     "orders",
            Cause:     errors.New("connection timeout"),
        })
    }
}

func callExternalAPI(url string) {
    if url == "http://timeout.com" {
        Throw(NetworkError{
            URL:     url,
            Timeout: true,
        })
    }
    
    if url == "http://notfound.com" {
        Throw(NetworkError{
            URL:        url,
            StatusCode: 404,
        })
    }
}

func processOrder(orderType string) {
    if orderType == "refund" {
        Throw(BusinessLogicError{
            Rule:    "refund_policy",
            Details: "refunds not allowed after 30 days",
        })
    }
}

func parseNumber(s string) int {
    if s == "invalid" {
        Throw("invalid number format")
    }
    
    num, err := strconv.Atoi(s)
    if err != nil {
        Throw(err)
    }
    return num
}

// ============= Demo示例 =============

func demo1_BasicUsage() {
    fmt.Println("=== Demo 1: 基本用法 ===")
    
    tb := Try(func() {
        validateUser("", "test@example.com", 25)
    })
    
    tb = Catch[ValidationError](tb, func(err ValidationError) {
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
        
        tb := Try(scenario.fn)
        
        tb = Catch[ValidationError](tb, func(err ValidationError) {
            fmt.Printf("✓ Validation issue: %s (Code: %d)\n", err.Message, err.Code)
        })
        
        tb = Catch[DatabaseError](tb, func(err DatabaseError) {
            fmt.Printf("✓ Database issue: %s on %s\n", err.Operation, err.Table)
        })
        
        tb = Catch[NetworkError](tb, func(err NetworkError) {
            if err.Timeout {
                fmt.Printf("✓ Network timeout: %s\n", err.URL)
            } else {
                fmt.Printf("✓ Network error %d: %s\n", err.StatusCode, err.URL)
            }
        })
        
        tb = Catch[BusinessLogicError](tb, func(err BusinessLogicError) {
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
        
        tb := Try(scenario.fn)
        
        tb = Catch[string](tb, func(err string) {
            fmt.Printf("✓ String error: %s\n", err)
        })
        
        tb = Catch[error](tb, func(err error) {
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
    
    tb := Try(func() {
        validateUser("John", "invalid", 25)
    })
    
    result, tb := CatchWithReturn[ValidationError](tb, func(err ValidationError) interface{} {
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
    
    tb := Try(func() {
        fmt.Println("  Outer try block started")
        
        innerTb := Try(func() {
            fmt.Println("    Inner try block started")
            validateUser("", "test@example.com", 25)
        })
        
        innerTb = Catch[ValidationError](innerTb, func(err ValidationError) {
            fmt.Printf("    ✓ Inner catch: %s\n", err.Message)
            // 在内层处理后，抛出新的异常到外层
            Throw(BusinessLogicError{
                Rule:    "user_validation",
                Details: "failed inner validation: " + err.Field,
            })
        })
        
        innerTb.Finally(func() {
            fmt.Println("    Inner finally block")
        })
        
        fmt.Println("  This should not be reached")
    })
    
    tb = Catch[BusinessLogicError](tb, func(err BusinessLogicError) {
        fmt.Printf("  ✓ Outer catch: %s\n", err.Details)
    })
    
    tb.Finally(func() {
        fmt.Println("  Nested example completed")
    })
}

func demo6_RealWorldExample() {
    fmt.Println("\n=== Demo 6: 真实场景示例 ===")
    
    processUserOrder := func(userID, orderData string) {
        tb := Try(func() {
            // 步骤1：验证用户输入
            if userID == "" {
                Throw(ValidationError{
                    Field:   "userID",
                    Message: "user ID is required",
                    Code:    2001,
                })
            }
            
            // 步骤2：访问数据库
            if userID == "blocked_user" {
                Throw(DatabaseError{
                    Operation: "SELECT",
                    Table:     "users",
                    Cause:     errors.New("user account blocked"),
                })
            }
            
            // 步骤3：调用外部支付API
            if orderData == "payment_failed" {
                Throw(NetworkError{
                    URL:        "https://payment.api.com",
                    StatusCode: 402,
                })
            }
            
            // 步骤4：业务逻辑检查
            if orderData == "insufficient_stock" {
                Throw(BusinessLogicError{
                    Rule:    "inventory_check",
                    Details: "requested quantity exceeds available stock",
                })
            }
            
            fmt.Printf("  ✓ Order processed successfully for user: %s\n", userID)
        })
        
        tb = Catch[ValidationError](tb, func(err ValidationError) {
            fmt.Printf("  ❌ Input validation failed: %s\n", err.Message)
        })
        
        tb = Catch[DatabaseError](tb, func(err DatabaseError) {
            fmt.Printf("  ❌ Database operation failed: %s\n", err.Cause.Error())
        })
        
        tb = Catch[NetworkError](tb, func(err NetworkError) {
            fmt.Printf("  ❌ External service error: %d from %s\n", err.StatusCode, err.URL)
        })
        
        tb = Catch[BusinessLogicError](tb, func(err BusinessLogicError) {
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