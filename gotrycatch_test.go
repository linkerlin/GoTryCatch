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

	expected := "validation error [1001] on field 'email': invalid format"
	if err.Error() != expected {
		t.Errorf("Expected error message %v, got %v", expected, err.Error())
	}

	if err.Field != "email" {
		t.Errorf("Expected field 'email', got %v", err.Field)
	}
	if err.Code != 1001 {
		t.Errorf("Expected code 1001, got %v", err.Code)
	}
}

func TestDatabaseError(t *testing.T) {
	cause := errors.New("connection failed")
	err := trycatcherrors.NewDatabaseError("SELECT", "users", cause)

	expected := "database error during SELECT on table 'users': connection failed"
	if err.Error() != expected {
		t.Errorf("Expected error message %v, got %v", expected, err.Error())
	}

	if err.Operation != "SELECT" {
		t.Errorf("Expected operation 'SELECT', got %v", err.Operation)
	}
	if err.Table != "users" {
		t.Errorf("Expected table 'users', got %v", err.Table)
	}
	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNetworkError(t *testing.T) {
	err := trycatcherrors.NewNetworkError("http://example.com", 404)

	expected := "network error 404 when accessing http://example.com"
	if err.Error() != expected {
		t.Errorf("Expected error message %v, got %v", expected, err.Error())
	}

	if err.URL != "http://example.com" {
		t.Errorf("Expected URL 'http://example.com', got %v", err.URL)
	}
	if err.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %v", err.StatusCode)
	}
	if err.Timeout {
		t.Errorf("Expected timeout to be false, got true")
	}
}

func TestNetworkTimeoutError(t *testing.T) {
	err := trycatcherrors.NewNetworkTimeoutError("http://example.com")

	expected := "network timeout when accessing http://example.com"
	if err.Error() != expected {
		t.Errorf("Expected error message %v, got %v", expected, err.Error())
	}

	if !err.Timeout {
		t.Errorf("Expected timeout to be true, got false")
	}
}

func TestBusinessLogicError(t *testing.T) {
	err := trycatcherrors.NewBusinessLogicError("age_limit", "must be over 18")

	expected := "business rule violation: age_limit - must be over 18"
	if err.Error() != expected {
		t.Errorf("Expected error message %v, got %v", expected, err.Error())
	}

	if err.Rule != "age_limit" {
		t.Errorf("Expected rule 'age_limit', got %v", err.Rule)
	}
	if err.Details != "must be over 18" {
		t.Errorf("Expected details 'must be over 18', got %v", err.Details)
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
