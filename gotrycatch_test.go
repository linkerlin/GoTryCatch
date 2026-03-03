package gotrycatch

import (
	"errors"
	"testing"

	trycatcherrors "github.com/linkerlin/gotrycatch/errors"
)

func TestTry_NoPanic(t *testing.T) {
	tb := Try(func() {
		// Normal execution, no panic
	})

	if tb.err != nil {
		t.Errorf("Expected no error, got %v", tb.err)
	}
	if tb.handled {
		t.Errorf("Expected handled to be false, got true")
	}
}

func TestTry_WithPanic(t *testing.T) {
	expectedErr := "test error"
	tb := Try(func() {
		panic(expectedErr)
	})

	if tb.err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, tb.err)
	}
	if tb.handled {
		t.Errorf("Expected handled to be false, got true")
	}
}

func TestCatch_MatchingType(t *testing.T) {
	expectedErr := "test string error"
	var caughtErr string
	var handlerCalled bool

	tb := Try(func() {
		panic(expectedErr)
	})

	tb = Catch[string](tb, func(err string) {
		caughtErr = err
		handlerCalled = true
	})

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
	if caughtErr != expectedErr {
		t.Errorf("Expected caught error %v, got %v", expectedErr, caughtErr)
	}
	if !tb.handled {
		t.Error("Expected handled to be true")
	}
}

func TestCatch_NonMatchingType(t *testing.T) {
	expectedErr := 123
	var handlerCalled bool

	tb := Try(func() {
		panic(expectedErr)
	})

	tb = Catch[string](tb, func(err string) {
		handlerCalled = true
	})

	if handlerCalled {
		t.Error("Expected handler not to be called")
	}
	if tb.handled {
		t.Error("Expected handled to be false")
	}
}

func TestCatch_MultipleHandlers(t *testing.T) {
	var stringHandlerCalled, intHandlerCalled bool

	tb := Try(func() {
		panic("test error")
	})

	tb = Catch[int](tb, func(err int) {
		intHandlerCalled = true
	})

	tb = Catch[string](tb, func(err string) {
		stringHandlerCalled = true
	})

	if intHandlerCalled {
		t.Error("Expected int handler not to be called")
	}
	if !stringHandlerCalled {
		t.Error("Expected string handler to be called")
	}
	if !tb.handled {
		t.Error("Expected handled to be true")
	}
}

func TestCatchWithReturn(t *testing.T) {
	expectedErr := "test error"
	expectedResult := map[string]string{"error": expectedErr}

	tb := Try(func() {
		panic(expectedErr)
	})

	result, tb := CatchWithReturn[string](tb, func(err string) interface{} {
		return expectedResult
	})

	if result == nil {
		t.Error("Expected result to be non-nil")
	}

	if resultMap, ok := result.(map[string]string); ok {
		if resultMap["error"] != expectedErr {
			t.Errorf("Expected result error %v, got %v", expectedErr, resultMap["error"])
		}
	} else {
		t.Error("Expected result to be map[string]string")
	}

	if !tb.handled {
		t.Error("Expected handled to be true")
	}
}

func TestCatchAny(t *testing.T) {
	expectedErr := "test error"
	var caughtErr interface{}
	var handlerCalled bool

	tb := Try(func() {
		panic(expectedErr)
	})

	tb = tb.CatchAny(func(err interface{}) {
		caughtErr = err
		handlerCalled = true
	})

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
	if caughtErr != expectedErr {
		t.Errorf("Expected caught error %v, got %v", expectedErr, caughtErr)
	}
	if !tb.handled {
		t.Error("Expected handled to be true")
	}
}

func TestFinally_WithHandledException(t *testing.T) {
	var finallyExecuted bool
	var handlerCalled bool

	tb := Try(func() {
		panic("test error")
	})

	tb = Catch[string](tb, func(err string) {
		handlerCalled = true
	})

	tb.Finally(func() {
		finallyExecuted = true
	})

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
	if !finallyExecuted {
		t.Error("Expected finally block to be executed")
	}
}

func TestFinally_WithUnhandledException(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be re-thrown")
		}
	}()

	var finallyExecuted bool

	tb := Try(func() {
		panic("test error")
	})

	tb.Finally(func() {
		finallyExecuted = true
	})

	if !finallyExecuted {
		t.Error("Expected finally block to be executed")
	}
}

func TestThrow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic")
		} else if r != "test error" {
			t.Errorf("Expected panic value 'test error', got %v", r)
		}
	}()

	Throw("test error")
}

func TestValidationError(t *testing.T) {
	err := trycatcherrors.NewValidationError("email", "invalid format", 1001)

	// 验证基本字段
	if err.Field != "email" {
		t.Errorf("Expected field 'email', got %v", err.Field)
	}
	if err.Code != 1001 {
		t.Errorf("Expected code 1001, got %v", err.Code)
	}
	if err.Message != "invalid format" {
		t.Errorf("Expected message 'invalid format', got %v", err.Message)
	}

	// 验证新增字段
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
	if err.Line == 0 {
		t.Error("Expected Line to be populated")
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be populated")
	}
	if len(err.Stack) == 0 {
		t.Error("Expected Stack to be populated")
	}

	// 验证 Error() 方法包含位置信息
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}

	// 验证 ToMap
	m := err.ToMap()
	if m["type"] != "ValidationError" {
		t.Errorf("Expected type 'ValidationError', got %v", m["type"])
	}

	// 验证 ToJSON
	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Errorf("ToJSON failed: %v", jsonErr)
	}
	if len(jsonBytes) == 0 {
		t.Error("Expected non-empty JSON")
	}
}

func TestDatabaseError(t *testing.T) {
	cause := errors.New("connection failed")
	err := trycatcherrors.NewDatabaseError("SELECT", "users", cause)

	// 验证基本字段
	if err.Operation != "SELECT" {
		t.Errorf("Expected operation 'SELECT', got %v", err.Operation)
	}
	if err.Table != "users" {
		t.Errorf("Expected table 'users', got %v", err.Table)
	}
	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}

	// 验证新增字段
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
	if err.Line == 0 {
		t.Error("Expected Line to be populated")
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be populated")
	}

	// 验证 Unwrap
	if err.Unwrap() != cause {
		t.Errorf("Expected Unwrap to return cause")
	}

	// 验证 ToMap
	m := err.ToMap()
	if m["type"] != "DatabaseError" {
		t.Errorf("Expected type 'DatabaseError', got %v", m["type"])
	}
}

func TestNetworkError(t *testing.T) {
	err := trycatcherrors.NewNetworkError("http://example.com", 404)

	// 验证基本字段
	if err.URL != "http://example.com" {
		t.Errorf("Expected URL 'http://example.com', got %v", err.URL)
	}
	if err.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %v", err.StatusCode)
	}
	if err.Timeout {
		t.Errorf("Expected timeout to be false, got true")
	}

	// 验证新增字段
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be populated")
	}

	// 验证 ToMap
	m := err.ToMap()
	if m["statusCode"] != 404 {
		t.Errorf("Expected statusCode 404, got %v", m["statusCode"])
	}
}

func TestNetworkTimeoutError(t *testing.T) {
	err := trycatcherrors.NewNetworkTimeoutError("http://example.com")

	if !err.Timeout {
		t.Errorf("Expected timeout to be true, got false")
	}
	if err.URL != "http://example.com" {
		t.Errorf("Expected URL 'http://example.com', got %v", err.URL)
	}

	// 验证 Error() 包含 timeout 信息
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}
}

func TestBusinessLogicError(t *testing.T) {
	err := trycatcherrors.NewBusinessLogicError("age_limit", "must be over 18")

	// 验证基本字段
	if err.Rule != "age_limit" {
		t.Errorf("Expected rule 'age_limit', got %v", err.Rule)
	}
	if err.Details != "must be over 18" {
		t.Errorf("Expected details 'must be over 18', got %v", err.Details)
	}

	// 验证新增字段
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be populated")
	}

	// 验证 ToMap
	m := err.ToMap()
	if m["type"] != "BusinessLogicError" {
		t.Errorf("Expected type 'BusinessLogicError', got %v", m["type"])
	}
}

// ============================================
// 新增错误类型测试
// ============================================

func TestConfigError(t *testing.T) {
	err := trycatcherrors.NewConfigError("database.url", "invalid://url", "invalid URL format")

	// 验证基本字段
	if err.Key != "database.url" {
		t.Errorf("Expected key 'database.url', got %v", err.Key)
	}
	if err.Value != "invalid://url" {
		t.Errorf("Expected value 'invalid://url', got %v", err.Value)
	}
	if err.Reason != "invalid URL format" {
		t.Errorf("Expected reason 'invalid URL format', got %v", err.Reason)
	}

	// 验证新增字段
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be populated")
	}

	// 验证 Error() 方法
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}

	// 验证 ToMap 和 ToJSON
	m := err.ToMap()
	if m["type"] != "ConfigError" {
		t.Errorf("Expected type 'ConfigError', got %v", m["type"])
	}

	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Errorf("ToJSON failed: %v", jsonErr)
	}
	if len(jsonBytes) == 0 {
		t.Error("Expected non-empty JSON")
	}
}

func TestAuthError(t *testing.T) {
	err := trycatcherrors.NewAuthError("login", "user123", "invalid password")

	// 验证基本字段
	if err.Operation != "login" {
		t.Errorf("Expected operation 'login', got %v", err.Operation)
	}
	if err.User != "user123" {
		t.Errorf("Expected user 'user123', got %v", err.User)
	}
	if err.Reason != "invalid password" {
		t.Errorf("Expected reason 'invalid password', got %v", err.Reason)
	}

	// 验证新增字段
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be populated")
	}

	// 验证 ToMap
	m := err.ToMap()
	if m["type"] != "AuthError" {
		t.Errorf("Expected type 'AuthError', got %v", m["type"])
	}
}

func TestRateLimitError(t *testing.T) {
	err := trycatcherrors.NewRateLimitError("api/endpoint", 100, 105, 60)

	// 验证基本字段
	if err.Resource != "api/endpoint" {
		t.Errorf("Expected resource 'api/endpoint', got %v", err.Resource)
	}
	if err.Limit != 100 {
		t.Errorf("Expected limit 100, got %v", err.Limit)
	}
	if err.Current != 105 {
		t.Errorf("Expected current 105, got %v", err.Current)
	}
	if err.RetryAfter != 60 {
		t.Errorf("Expected retryAfter 60, got %v", err.RetryAfter)
	}

	// 验证新增字段
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be populated")
	}

	// 验证 ToMap
	m := err.ToMap()
	if m["type"] != "RateLimitError" {
		t.Errorf("Expected type 'RateLimitError', got %v", m["type"])
	}
}

func TestErrorIsMethod(t *testing.T) {
	// 测试 ValidationError 的 Is 方法
	err1 := trycatcherrors.NewValidationError("field1", "msg1", 1001)
	err2 := trycatcherrors.NewValidationError("field2", "msg2", 1001)
	err3 := trycatcherrors.NewValidationError("field3", "msg3", 1002)

	if !err1.Is(err2) {
		t.Error("Expected err1.Is(err2) to be true (same code)")
	}
	if err1.Is(err3) {
		t.Error("Expected err1.Is(err3) to be false (different code)")
	}
}

func TestErrorUnwrapMethod(t *testing.T) {
	// 测试 DatabaseError 的 Unwrap 方法
	cause := errors.New("underlying error")
	dbErr := trycatcherrors.NewDatabaseError("SELECT", "users", cause)

	unwrapped := dbErr.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected Unwrap to return cause, got %v", unwrapped)
	}

	// 测试没有底层错误的类型
	valErr := trycatcherrors.NewValidationError("field", "msg", 1001)
	if valErr.Unwrap() != nil {
		t.Error("Expected ValidationError.Unwrap() to return nil")
	}
}

// Integration test that demonstrates complete workflow
func TestIntegration_CompleteWorkflow(t *testing.T) {
	var steps []string

	tb := Try(func() {
		steps = append(steps, "try_block_executed")
		panic(trycatcherrors.NewValidationError("name", "required", 1001))
	})

	tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		steps = append(steps, "database_handler_called")
	})

	tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		steps = append(steps, "validation_handler_called")
		if err.Field != "name" || err.Code != 1001 {
			t.Errorf("Unexpected validation error: %+v", err)
		}
	})

	tb.Finally(func() {
		steps = append(steps, "finally_executed")
	})

	expectedSteps := []string{
		"try_block_executed",
		"validation_handler_called",
		"finally_executed",
	}

	if len(steps) != len(expectedSteps) {
		t.Fatalf("Expected %d steps, got %d: %v", len(expectedSteps), len(steps), steps)
	}

	for i, expected := range expectedSteps {
		if steps[i] != expected {
			t.Errorf("Step %d: expected %v, got %v", i, expected, steps[i])
		}
	}
}

// ============================================
// 第一阶段新增方法测试
// ============================================

func TestGetError(t *testing.T) {
	// 无错误情况
	tb := Try(func() {})
	if tb.GetError() != nil {
		t.Errorf("Expected nil error, got %v", tb.GetError())
	}

	// 有错误情况
	tb = Try(func() {
		panic("test error")
	})
	if tb.GetError() != "test error" {
		t.Errorf("Expected 'test error', got %v", tb.GetError())
	}

	// nil TryBlock
	var nilTb *TryBlock
	if nilTb.GetError() != nil {
		t.Errorf("Expected nil for nil TryBlock, got %v", nilTb.GetError())
	}
}

func TestHasError(t *testing.T) {
	// 无错误
	tb := Try(func() {})
	if tb.HasError() {
		t.Error("Expected HasError to be false")
	}

	// 有错误
	tb = Try(func() {
		panic("error")
	})
	if !tb.HasError() {
		t.Error("Expected HasError to be true")
	}

	// nil TryBlock
	var nilTb *TryBlock
	if nilTb.HasError() {
		t.Error("Expected HasError to be false for nil TryBlock")
	}
}

func TestIsHandled(t *testing.T) {
	// 未处理
	tb := Try(func() {
		panic("error")
	})
	if tb.IsHandled() {
		t.Error("Expected IsHandled to be false before Catch")
	}

	// 已处理
	tb = Catch[string](tb, func(err string) {})
	if !tb.IsHandled() {
		t.Error("Expected IsHandled to be true after Catch")
	}

	// nil TryBlock
	var nilTb *TryBlock
	if nilTb.IsHandled() {
		t.Error("Expected IsHandled to be false for nil TryBlock")
	}
}

func TestString(t *testing.T) {
	// 无错误
	tb := Try(func() {})
	expected := "TryBlock{err: nil, handled: false}"
	if tb.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, tb.String())
	}

	// 有错误
	tb = Try(func() {
		panic("test error")
	})
	str := tb.String()
	if str == "" || str == "TryBlock{nil}" {
		t.Errorf("Unexpected String output: %s", str)
	}
	// 应包含错误类型和值
	if str != "TryBlock{err: string(test error), handled: false}" {
		t.Errorf("Expected specific format, got '%s'", str)
	}

	// nil TryBlock
	var nilTb *TryBlock
	if nilTb.String() != "TryBlock{nil}" {
		t.Errorf("Expected 'TryBlock{nil}', got '%s'", nilTb.String())
	}
}

func TestGetErrorType(t *testing.T) {
	// 无错误
	tb := Try(func() {})
	if tb.GetErrorType() != "" {
		t.Errorf("Expected empty string, got '%s'", tb.GetErrorType())
	}

	// string 类型错误
	tb = Try(func() {
		panic("string error")
	})
	if tb.GetErrorType() != "string" {
		t.Errorf("Expected 'string', got '%s'", tb.GetErrorType())
	}

	// 自定义类型错误
	tb = Try(func() {
		panic(trycatcherrors.NewValidationError("field", "msg", 1001))
	})
	expectedType := "errors.ValidationError"
	if tb.GetErrorType() != expectedType {
		t.Errorf("Expected '%s', got '%s'", expectedType, tb.GetErrorType())
	}

	// nil TryBlock
	var nilTb *TryBlock
	if nilTb.GetErrorType() != "" {
		t.Errorf("Expected empty string for nil TryBlock, got '%s'", nilTb.GetErrorType())
	}
}

func TestSetDebug(t *testing.T) {
	// 测试调试模式开关
	originalDebug := IsDebug()

	SetDebug(true)
	if !IsDebug() {
		t.Error("Expected debug mode to be true")
	}

	SetDebug(false)
	if IsDebug() {
		t.Error("Expected debug mode to be false")
	}

	// 恢复原状态
	SetDebug(originalDebug)
}

func TestAssert(t *testing.T) {
	// 条件为 true，不应 panic
	Assert(true, "should not panic")

	// 条件为 false，应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from Assert")
		} else if r != "assertion failed" {
			t.Errorf("Expected 'assertion failed', got %v", r)
		}
	}()

	Assert(false, "assertion failed")
}

func TestAssertNoError(t *testing.T) {
	// err 为 nil，不应 panic
	AssertNoError(nil, "no error")

	// err 不为 nil，应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from AssertNoError")
		} else {
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type, got %T", r)
			}
			if err.Error() != "operation failed: test error" {
				t.Errorf("Unexpected error message: %v", err)
			}
		}
	}()

	AssertNoError(errors.New("test error"), "operation failed")
}

// ============================================
// 第三阶段：TryWithResult 测试
// ============================================

func TestTryWithResult_NoPanic(t *testing.T) {
	tb := TryWithResult(func() int {
		return 42
	})

	if tb.HasError() {
		t.Errorf("Expected no error, got %v", tb.GetError())
	}
	if tb.GetResult() != 42 {
		t.Errorf("Expected result 42, got %v", tb.GetResult())
	}
	if tb.IsHandled() {
		t.Error("Expected IsHandled to be false")
	}
}

func TestTryWithResult_WithPanic(t *testing.T) {
	tb := TryWithResult(func() int {
		panic("something went wrong")
	})

	if !tb.HasError() {
		t.Error("Expected HasError to be true")
	}
	if tb.GetError() != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got %v", tb.GetError())
	}
}

func TestTryWithResult_Catch(t *testing.T) {
	tb := TryWithResult(func() int {
		panic(trycatcherrors.NewValidationError("field", "invalid", 1001))
	})

	tb = CatchWithResult[int, trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		// 处理错误
	})

	if !tb.IsHandled() {
		t.Error("Expected IsHandled to be true after catch")
	}
}

func TestTryWithResult_Finally_Success(t *testing.T) {
	var finallyCalled bool

	tb := TryWithResult(func() int {
		return 100
	})

	result := tb.Finally(func() {
		finallyCalled = true
	})

	if !finallyCalled {
		t.Error("Expected Finally to be called")
	}
	if result != 100 {
		t.Errorf("Expected result 100, got %v", result)
	}
}

func TestTryWithResult_Finally_WithHandledError(t *testing.T) {
	var finallyCalled bool

	tb := TryWithResult(func() int {
		panic("error")
	})

	tb = CatchWithResult[int, string](tb, func(err string) {
		// 处理错误
	})

	result := tb.Finally(func() {
		finallyCalled = true
	})

	if !finallyCalled {
		t.Error("Expected Finally to be called")
	}
	// 有错误时返回零值
	if result != 0 {
		t.Errorf("Expected zero result, got %v", result)
	}
}

func TestTryWithResult_OnSuccess(t *testing.T) {
	var successCalled bool
	var receivedResult int

	tb := TryWithResult(func() int {
		return 42
	})

	tb.OnSuccess(func(result int) {
		successCalled = true
		receivedResult = result
	})

	if !successCalled {
		t.Error("Expected OnSuccess to be called")
	}
	if receivedResult != 42 {
		t.Errorf("Expected result 42, got %v", receivedResult)
	}
}

func TestTryWithResult_OnSuccess_NotCalledOnError(t *testing.T) {
	var successCalled bool

	tb := TryWithResult(func() int {
		panic("error")
	})

	tb.OnSuccess(func(result int) {
		successCalled = true
	})

	if successCalled {
		t.Error("Expected OnSuccess not to be called on error")
	}
}

func TestTryWithResult_OnError(t *testing.T) {
	var errorCalled bool
	var receivedError interface{}

	tb := TryWithResult(func() int {
		panic("test error")
	})

	tb.OnError(func(err interface{}) {
		errorCalled = true
		receivedError = err
	})

	if !errorCalled {
		t.Error("Expected OnError to be called")
	}
	if receivedError != "test error" {
		t.Errorf("Expected error 'test error', got %v", receivedError)
	}
	if !tb.IsHandled() {
		t.Error("Expected error to be handled after OnError")
	}
}

func TestTryWithResult_OrElse(t *testing.T) {
	// 成功情况 - 返回实际结果
	tb1 := TryWithResult(func() string {
		return "success"
	})
	result1 := tb1.OrElse("default")
	if result1 != "success" {
		t.Errorf("Expected 'success', got %v", result1)
	}

	// 失败情况 - 返回默认值
	tb2 := TryWithResult(func() string {
		panic("error")
	})
	result2 := tb2.OrElse("default")
	if result2 != "default" {
		t.Errorf("Expected 'default', got %v", result2)
	}
}

func TestTryWithResult_OrElseGet(t *testing.T) {
	// 成功情况
	tb1 := TryWithResult(func() int {
		return 10
	})
	result1 := tb1.OrElseGet(func() int { return 0 })
	if result1 != 10 {
		t.Errorf("Expected 10, got %v", result1)
	}

	// 失败情况
	tb2 := TryWithResult(func() int {
		panic("error")
	})
	result2 := tb2.OrElseGet(func() int { return 99 })
	if result2 != 99 {
		t.Errorf("Expected 99, got %v", result2)
	}
}

func TestTryWithResult_String(t *testing.T) {
	// 无错误
	tb1 := TryWithResult(func() int { return 42 })
	str1 := tb1.String()
	if str1 == "" {
		t.Error("Expected non-empty string")
	}

	// 有错误
	tb2 := TryWithResult(func() int { panic("err") })
	str2 := tb2.String()
	if str2 == "" {
		t.Error("Expected non-empty string")
	}

	// nil
	var nilTb *TryBlockWithResult[int]
	if nilTb.String() != "TryBlockWithResult{nil}" {
		t.Errorf("Unexpected string for nil: %s", nilTb.String())
	}
}

// ============================================
// 边界情况测试 (Edge Case Tests)
// ============================================

func TestCatch_NilHandler(t *testing.T) {
	// Catch with nil handler should not panic, just return the TryBlock
	tb := Try(func() {
		panic("test error")
	})

	// Passing nil handler should not crash
	tb = Catch[string](tb, nil)

	// Error should still be present but not handled
	if tb.GetError() != "test error" {
		t.Errorf("Expected error 'test error', got %v", tb.GetError())
	}
	if tb.IsHandled() {
		t.Error("Expected IsHandled to be false with nil handler")
	}
}

func TestCatchAny_NilHandler(t *testing.T) {
	tb := Try(func() {
		panic("test error")
	})

	// CatchAny with nil handler
	tb = tb.CatchAny(nil)

	if tb.GetError() != "test error" {
		t.Errorf("Expected error 'test error', got %v", tb.GetError())
	}
	if tb.IsHandled() {
		t.Error("Expected IsHandled to be false with nil handler")
	}
}

func TestFinally_NilHandler(t *testing.T) {
	tb := Try(func() {
		panic("test error")
	})

	tb = Catch[string](tb, func(err string) {})

	// Finally with nil handler should not panic
	tb.Finally(nil)

	// Should still be handled
	if !tb.IsHandled() {
		t.Error("Expected IsHandled to be true")
	}
}

func TestFinally_PanicInHandler(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from Finally handler to propagate")
		} else if r != "finally panic" {
			t.Errorf("Expected 'finally panic', got %v", r)
		}
	}()

	tb := Try(func() {
		// No panic
	})

	tb.Finally(func() {
		panic("finally panic")
	})
}

func TestCatch_PanicInHandler(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from Catch handler to propagate")
		} else if r != "handler panic" {
			t.Errorf("Expected 'handler panic', got %v", r)
		}
	}()

	tb := Try(func() {
		panic("original error")
	})

	Catch[string](tb, func(err string) {
		panic("handler panic")
	})
}

func TestFinally_CalledAfterCatchPanic(t *testing.T) {
	// When Catch handler panics, Finally is NOT called because panic propagates immediately
	// before Finally can be invoked. This test verifies that behavior.
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to propagate")
		} else if r != "catch panic" {
			t.Errorf("Expected 'catch panic', got %v", r)
		}
	}()

	tb := Try(func() {
		panic("test error")
	})

	// This panic propagates before Finally is called
	Catch[string](tb, func(err string) {
		panic("catch panic")
	})

	// This line is never reached because of the panic above
	t.Error("Should not reach this line")
}

func TestNestedTryCatch(t *testing.T) {
	var outerHandlerCalled, innerHandlerCalled bool

	tb := Try(func() {
		// Inner try-catch
		innerTb := Try(func() {
			panic("inner error")
		})
		innerTb = Catch[string](innerTb, func(err string) {
			innerHandlerCalled = true
			if err != "inner error" {
				t.Errorf("Unexpected inner error: %v", err)
			}
		})
		innerTb.Finally(func() {})

		// Now cause outer panic
		panic("outer error")
	})

	tb = Catch[string](tb, func(err string) {
		outerHandlerCalled = true
		if err != "outer error" {
			t.Errorf("Unexpected outer error: %v", err)
		}
	})

	if !innerHandlerCalled {
		t.Error("Expected inner handler to be called")
	}
	if !outerHandlerCalled {
		t.Error("Expected outer handler to be called")
	}
}

func TestEmptyStringPanic(t *testing.T) {
	tb := Try(func() {
		panic("")
	})

	if tb.GetError() != "" {
		t.Errorf("Expected empty string error, got %v", tb.GetError())
	}
	if !tb.HasError() {
		t.Error("Expected HasError to be true even for empty string")
	}

	// Should still be able to catch empty string
	var caught bool
	tb = Catch[string](tb, func(err string) {
		caught = true
		if err != "" {
			t.Errorf("Expected empty string, got %v", err)
		}
	})

	if !caught {
		t.Error("Expected Catch handler to be called for empty string")
	}
}

func TestNilPanicValue(t *testing.T) {
	// Note: panic(nil) in Go has special behavior - recover() returns a runtime.nilPanicError
	// We still detect it as having an error (HasError = true)
	tb := Try(func() {
		panic(nil)
	})

	// HasError should be true even for nil panic
	if !tb.HasError() {
		t.Error("Expected HasError to be true even for nil panic value")
	}

	// The error value will be a special runtime type, not nil itself
	if tb.GetError() == nil {
		t.Error("Expected GetError to return non-nil (runtime.nilPanicError)")
	}
}

func TestIntegerPanicValue(t *testing.T) {
	tb := Try(func() {
		panic(42)
	})

	if tb.GetError() != 42 {
		t.Errorf("Expected error 42, got %v", tb.GetError())
	}

	var caught bool
	tb = Catch[int](tb, func(err int) {
		caught = true
		if err != 42 {
			t.Errorf("Expected 42, got %v", err)
		}
	})

	if !caught {
		t.Error("Expected int Catch handler to be called")
	}
}

func TestStructPanicValue(t *testing.T) {
	type CustomError struct {
		Code    int
		Message string
	}

	tb := Try(func() {
		panic(CustomError{Code: 500, Message: "internal error"})
	})

	var caught bool
	tb = Catch[CustomError](tb, func(err CustomError) {
		caught = true
		if err.Code != 500 || err.Message != "internal error" {
			t.Errorf("Unexpected error: %+v", err)
		}
	})

	if !caught {
		t.Error("Expected CustomError Catch handler to be called")
	}
}

func TestMultipleCatchAnyCalls(t *testing.T) {
	var firstCallCount, secondCallCount int

	tb := Try(func() {
		panic("error")
	})

	// First CatchAny should handle and mark as handled
	tb = tb.CatchAny(func(err interface{}) {
		firstCallCount++
	})

	// Second CatchAny should NOT be called since already handled
	tb = tb.CatchAny(func(err interface{}) {
		secondCallCount++
	})

	if firstCallCount != 1 {
		t.Errorf("Expected first handler called once, got %d", firstCallCount)
	}
	if secondCallCount != 0 {
		t.Errorf("Expected second handler not called, got %d", secondCallCount)
	}
}

func TestCatch_OrderMatters(t *testing.T) {
	var stringHandlerCalled, anyHandlerCalled bool

	tb := Try(func() {
		panic("string error")
	})

	// First try to catch as int (won't match)
	tb = Catch[int](tb, func(err int) {
		// Should not be called
	})

	// Then catch as string (will match)
	tb = Catch[string](tb, func(err string) {
		stringHandlerCalled = true
	})

	// CatchAny after match should not be called
	tb = tb.CatchAny(func(err interface{}) {
		anyHandlerCalled = true
	})

	if !stringHandlerCalled {
		t.Error("Expected string handler to be called")
	}
	if anyHandlerCalled {
		t.Error("Expected CatchAny not to be called after string handler matched")
	}
}

func TestCatchWithReturn_WithPanic(t *testing.T) {
	// When CatchWithReturn handler panics, the panic should propagate
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to propagate from handler")
		} else if r != "handler panic" {
			t.Errorf("Expected 'handler panic', got %v", r)
		}
	}()

	tb := Try(func() {
		panic("test error")
	})

	result, _ := CatchWithReturn[string](tb, func(err string) interface{} {
		panic("handler panic")
	})

	// Should not reach here
	_ = result
	t.Error("Should not reach this line")
}

func TestTryWithResult_NilPanic(t *testing.T) {
	tb := TryWithResult(func() int {
		panic(nil)
	})

	if !tb.HasError() {
		t.Error("Expected HasError to be true for nil panic")
	}
}

func TestTryWithResult_EmptyResult(t *testing.T) {
	tb := TryWithResult(func() string {
		return "" // Empty but valid result
	})

	if tb.HasError() {
		t.Error("Expected no error for empty string result")
	}
	if tb.GetResult() != "" {
		t.Errorf("Expected empty string result, got %v", tb.GetResult())
	}
}

func TestTryWithResult_OrElseGet_WithPanic(t *testing.T) {
	tb := TryWithResult(func() int {
		panic("error")
	})

	// OrElseGet should provide the default
	result := tb.OrElseGet(func() int {
		return 100
	})

	if result != 100 {
		t.Errorf("Expected 100, got %v", result)
	}
}

func TestConcurrentTryCatch(t *testing.T) {
	const goroutines = 10
	const iterations = 100

	done := make(chan bool, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < iterations; i++ {
				tb := Try(func() {
					if i%2 == 0 {
						panic("error")
					}
				})

				tb = Catch[string](tb, func(err string) {
					if err != "error" {
						t.Errorf("Goroutine %d: unexpected error: %v", id, err)
					}
				})

				tb.Finally(func() {})
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for g := 0; g < goroutines; g++ {
		<-done
	}
}

func TestTryWithResultConcurrent(t *testing.T) {
	const goroutines = 10
	const iterations = 100

	done := make(chan bool, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < iterations; i++ {
				tb := TryWithResult(func() int {
					if i%2 == 0 {
						panic("error")
					}
					return i
				})

				tb = CatchWithResult[int, string](tb, func(err string) {})

				result := tb.OrElse(0)
				_ = result
			}
			done <- true
		}(g)
	}

	for g := 0; g < goroutines; g++ {
		<-done
	}
}

func TestDebugMode_DoesNotAffectBehavior(t *testing.T) {
	originalDebug := IsDebug()
	defer SetDebug(originalDebug)

	// Enable debug mode
	SetDebug(true)

	var handlerCalled bool
	tb := Try(func() {
		panic("test")
	})
	tb = Catch[string](tb, func(err string) {
		handlerCalled = true
	})

	// Behavior should be the same regardless of debug mode
	if !handlerCalled {
		t.Error("Expected handler to be called even in debug mode")
	}
	if tb.GetError() != "test" {
		t.Errorf("Expected error 'test', got %v", tb.GetError())
	}
}

func TestNilTryBlock_MethodCalls(t *testing.T) {
	var nilTb *TryBlock

	// All methods should handle nil receiver gracefully
	if nilTb.GetError() != nil {
		t.Error("Expected nil error for nil TryBlock")
	}
	if nilTb.HasError() {
		t.Error("Expected HasError false for nil TryBlock")
	}
	if nilTb.IsHandled() {
		t.Error("Expected IsHandled false for nil TryBlock")
	}
	if nilTb.GetErrorType() != "" {
		t.Error("Expected empty error type for nil TryBlock")
	}
	if nilTb.String() != "TryBlock{nil}" {
		t.Errorf("Unexpected String() for nil TryBlock: %s", nilTb.String())
	}

	// CatchAny returns a new empty TryBlock when receiver is nil (not the same nil pointer)
	result := nilTb.CatchAny(func(err interface{}) {})
	if result == nil {
		t.Error("Expected CatchAny to return non-nil TryBlock for nil receiver")
	}
}

func TestNilTryBlockWithResult_MethodCalls(t *testing.T) {
	var nilTb *TryBlockWithResult[int]

	if nilTb.GetError() != nil {
		t.Error("Expected nil error for nil TryBlockWithResult")
	}
	if nilTb.HasError() {
		t.Error("Expected HasError false for nil TryBlockWithResult")
	}
	if nilTb.IsHandled() {
		t.Error("Expected IsHandled false for nil TryBlockWithResult")
	}
	if nilTb.GetResult() != 0 {
		t.Errorf("Expected zero result for nil TryBlockWithResult, got %v", nilTb.GetResult())
	}
	if nilTb.String() != "TryBlockWithResult{nil}" {
		t.Errorf("Unexpected String() for nil TryBlockWithResult: %s", nilTb.String())
	}
}

func TestErrorInterface_PanicValue(t *testing.T) {
	// Test panic with error interface value
	err := errors.New("standard error")
	tb := Try(func() {
		panic(err)
	})

	if tb.GetError() != err {
		t.Errorf("Expected error %v, got %v", err, tb.GetError())
	}

	var caught bool
	tb = Catch[error](tb, func(e error) {
		caught = true
		if e != err {
			t.Errorf("Expected %v, got %v", err, e)
		}
	})

	if !caught {
		t.Error("Expected error Catch handler to be called")
	}
}

// ============================================
// Coverage Improvement Tests
// ============================================

func TestCatchAnyWithResult_BasicUsage(t *testing.T) {
	var caught bool
	var caughtErr interface{}

	tb := TryWithResult(func() int {
		panic("any error")
	})

	tb = CatchAnyWithResult(tb, func(err interface{}) {
		caught = true
		caughtErr = err
	})

	if !caught {
		t.Error("Expected CatchAnyWithResult handler to be called")
	}
	if caughtErr != "any error" {
		t.Errorf("Expected 'any error', got %v", caughtErr)
	}
	if !tb.IsHandled() {
		t.Error("Expected error to be handled")
	}
}

func TestCatchAnyWithResult_NilTryBlock(t *testing.T) {
	var nilTb *TryBlockWithResult[int]

	result := CatchAnyWithResult(nilTb, func(err interface{}) {
		t.Error("Handler should not be called for nil TryBlock")
	})

	if result == nil {
		t.Error("Expected non-nil TryBlockWithResult for nil input")
	}
}

func TestCatchAnyWithResult_NilHandler(t *testing.T) {
	tb := TryWithResult(func() int {
		panic("error")
	})

	result := CatchAnyWithResult[int](tb, nil)
	if result != tb {
		t.Error("Expected same TryBlock when handler is nil")
	}
}

func TestCatchAnyWithResult_AlreadyHandled(t *testing.T) {
	var catchAnyCalled bool

	tb := TryWithResult(func() int {
		panic("error")
	})

	tb = CatchWithResult[int, string](tb, func(err string) {
		// Handle error
	})

	tb = CatchAnyWithResult(tb, func(err interface{}) {
		catchAnyCalled = true
	})

	if catchAnyCalled {
		t.Error("CatchAnyWithResult should not be called when error already handled")
	}
}

func TestCatchAnyWithResult_NoError(t *testing.T) {
	var handlerCalled bool

	tb := TryWithResult(func() int {
		return 42
	})

	tb = CatchAnyWithResult(tb, func(err interface{}) {
		handlerCalled = true
	})

	if handlerCalled {
		t.Error("Handler should not be called when no error")
	}
}

func TestTryBlockWithResult_Finally_NilHandler(t *testing.T) {
	tb := TryWithResult(func() int {
		return 42
	})

	result := tb.Finally(nil)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}
}

func TestTryBlockWithResult_Finally_NilTryBlock(t *testing.T) {
	var nilTb *TryBlockWithResult[int]

	var finallyCalled bool
	result := nilTb.Finally(func() {
		finallyCalled = true
	})

	if !finallyCalled {
		t.Error("Finally should be called even for nil TryBlock")
	}
	if result != 0 {
		t.Errorf("Expected zero value 0, got %d", result)
	}
}

func TestTryBlockWithResult_Finally_UnhandledErrorRethrow(t *testing.T) {
	defer func() {
		if r := recover(); r != r {
			t.Errorf("Expected panic with 'unhandled', got %v", r)
		}
	}()

	tb := TryWithResult(func() int {
		panic("unhandled")
	})

	tb.Finally(func() {
		// Cleanup
	})
}

func TestCatchWithResult_NilTryBlock(t *testing.T) {
	var nilTb *TryBlockWithResult[int]

	result := CatchWithResult[int, string](nilTb, func(err string) {
		t.Error("Handler should not be called for nil TryBlock")
	})

	if result == nil {
		t.Error("Expected non-nil TryBlockWithResult for nil input")
	}
}

func TestCatchWithResult_NilHandler(t *testing.T) {
	tb := TryWithResult(func() int {
		panic("error")
	})

	result := CatchWithResult[int, string](tb, nil)
	if result != tb {
		t.Error("Expected same TryBlock when handler is nil")
	}
}

func TestCatchWithResult_AlreadyHandled(t *testing.T) {
	var secondHandlerCalled bool

	tb := TryWithResult(func() int {
		panic("error")
	})

	tb = CatchWithResult[int, string](tb, func(err string) {
		// First handler
	})

	tb = CatchWithResult[int, string](tb, func(err string) {
		secondHandlerCalled = true
	})

	if secondHandlerCalled {
		t.Error("Second handler should not be called when error already handled")
	}
}

func TestCatchWithResult_NonMatchingType(t *testing.T) {
	var handlerCalled bool

	tb := TryWithResult(func() int {
		panic(123) // int panic, not string
	})

	tb = CatchWithResult[int, string](tb, func(err string) {
		handlerCalled = true
	})

	if handlerCalled {
		t.Error("Handler should not be called for non-matching type")
	}
	if tb.IsHandled() {
		t.Error("Error should not be marked as handled")
	}
}

func TestCatchWithReturn_BasicUsage(t *testing.T) {
	tb := Try(func() {
		panic("error")
	})

	result, tb := CatchWithReturn(tb, func(err string) interface{} {
		return "recovered: " + err
	})

	if result != "recovered: error" {
		t.Errorf("Expected 'recovered: error', got %v", result)
	}
	if !tb.IsHandled() {
		t.Error("Expected error to be handled")
	}
}

func TestCatchWithReturn_NilTryBlock(t *testing.T) {
	var nilTb *TryBlock

	result, tb := CatchWithReturn(nilTb, func(err string) interface{} {
		return "should not be called"
	})

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if tb == nil {
		t.Error("Expected non-nil TryBlock for nil input")
	}
}

func TestCatchWithReturn_NilHandler(t *testing.T) {
	tb := Try(func() {
		panic("error")
	})

	result, returnedTb := CatchWithReturn[string](tb, nil)
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if returnedTb != tb {
		t.Error("Expected same TryBlock when handler is nil")
	}
}

func TestCatchWithReturn_NonMatchingType(t *testing.T) {
	tb := Try(func() {
		panic(123) // int panic, not string
	})

	result, tb := CatchWithReturn(tb, func(err string) interface{} {
		return "should not be called"
	})

	if result != nil {
		t.Errorf("Expected nil result for non-matching type, got %v", result)
	}
	if tb.IsHandled() {
		t.Error("Error should not be marked as handled")
	}
}

func TestCatchWithReturn_NoError(t *testing.T) {
	tb := Try(func() {
		// No panic
	})

	result, _ := CatchWithReturn(tb, func(err string) interface{} {
		return "should not be called"
	})

	if result != nil {
		t.Errorf("Expected nil result when no error, got %v", result)
	}
}

func TestCatchWithReturn_AlreadyHandled(t *testing.T) {
	tb := Try(func() {
		panic("error")
	})

	tb = Catch[string](tb, func(err string) {
		// First handler
	})

	result, _ := CatchWithReturn(tb, func(err string) interface{} {
		return "should not be called"
	})

	if result != nil {
		t.Errorf("Expected nil result when already handled, got %v", result)
	}
}

func TestCatch_NilTryBlock(t *testing.T) {
	var nilTb *TryBlock

	result := Catch[string](nilTb, func(err string) {
		t.Error("Handler should not be called for nil TryBlock")
	})

	if result == nil {
		t.Error("Expected non-nil TryBlock for nil input")
	}
}

func TestCatch_NilHandler_EdgeCase(t *testing.T) {
	tb := Try(func() {
		panic("error")
	})

	result := Catch[string](tb, nil)
	if result != tb {
		t.Error("Expected same TryBlock when handler is nil")
	}
}

func TestCatch_AlreadyHandled(t *testing.T) {
	var secondHandlerCalled bool

	tb := Try(func() {
		panic("error")
	})

	tb = Catch[string](tb, func(err string) {
		// First handler
	})

	tb = Catch[string](tb, func(err string) {
		secondHandlerCalled = true
	})

	if secondHandlerCalled {
		t.Error("Second handler should not be called when error already handled")
	}
}

func TestCatch_NoError(t *testing.T) {
	var handlerCalled bool

	tb := Try(func() {
		// No panic
	})

	tb = Catch[string](tb, func(err string) {
		handlerCalled = true
	})

	if handlerCalled {
		t.Error("Handler should not be called when no error")
	}
}

func TestFinally_NoPanicNoError(t *testing.T) {
	var finallyCalled bool

	tb := Try(func() {
		// No panic
	})

	tb.Finally(func() {
		finallyCalled = true
	})

	if !finallyCalled {
		t.Error("Finally should be called")
	}
}

func TestCatchWithReturn_TypeMatch(t *testing.T) {
	tb := Try(func() {
		panic("string error")
	})

	result, tb := CatchWithReturn(tb, func(err string) interface{} {
		return "handled: " + err
	})

	if result != "handled: string error" {
		t.Errorf("Expected 'handled: string error', got %v", result)
	}
	if !tb.IsHandled() {
		t.Error("Expected error to be handled")
	}
}

func TestVersion_Exists(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Version != "1.3.0" {
		t.Logf("Version is %s", Version)
	}
}
