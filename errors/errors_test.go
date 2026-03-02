package errors

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============================================
// Edge Case Tests for Error Types
// ============================================

func TestValidationError_EmptyFields(t *testing.T) {
	err := NewValidationError("", "", 0)

	if err.Field != "" {
		t.Errorf("Expected empty field, got %v", err.Field)
	}
	if err.Message != "" {
		t.Errorf("Expected empty message, got %v", err.Message)
	}
	if err.Code != 0 {
		t.Errorf("Expected code 0, got %v", err.Code)
	}

	// Should still have location info
	if err.File == "" {
		t.Error("Expected File to be populated even with empty fields")
	}
}

func TestValidationError_SpecialCharacters(t *testing.T) {
	specialChars := "field\x00with\x01null\x02bytes\n\t\"quotes\""
	err := NewValidationError(specialChars, "message with 中文 and emoji 🎉", 1001)

	// ToJSON should handle special characters
	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Errorf("ToJSON failed with special characters: %v", jsonErr)
	}

	// Verify JSON is valid
	var parsed map[string]interface{}
	if unmarshalErr := json.Unmarshal(jsonBytes, &parsed); unmarshalErr != nil {
		t.Errorf("Failed to parse JSON: %v", unmarshalErr)
	}

	// Check that field contains the special chars
	if field, ok := parsed["field"].(string); !ok || !strings.Contains(field, "null") {
		t.Errorf("Field not properly encoded in JSON: %v", parsed["field"])
	}
}

func TestValidationError_VeryLongMessage(t *testing.T) {
	longMessage := strings.Repeat("x", 10000)
	err := NewValidationError("field", longMessage, 1001)

	if err.Message != longMessage {
		t.Errorf("Expected long message to be preserved")
	}

	// ToJSON should handle long messages
	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Errorf("ToJSON failed with long message: %v", jsonErr)
	}
	if len(jsonBytes) < 10000 {
		t.Errorf("Expected JSON to contain full message, got length %d", len(jsonBytes))
	}
}

func TestDatabaseError_NilCause(t *testing.T) {
	err := NewDatabaseError("SELECT", "users", nil)

	if err.Cause != nil {
		t.Errorf("Expected nil cause, got %v", err.Cause)
	}

	// Unwrap should return nil
	if err.Unwrap() != nil {
		t.Error("Expected Unwrap to return nil for nil cause")
	}

	// ToMap should handle nil cause
	m := err.ToMap()
	if m["cause"] != "" {
		t.Errorf("Expected empty cause string in map, got %v", m["cause"])
	}

	// Error() should not panic
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}
}

func TestDatabaseError_DeepUnwrap(t *testing.T) {
	cause1 := errors.New("cause1")
	cause2 := &wrappedError{msg: "cause2", cause: cause1}
	cause3 := &wrappedError{msg: "cause3", cause: cause2}

	err := NewDatabaseError("INSERT", "table", cause3)

	// Unwrap returns the immediate cause
	unwrapped := err.Unwrap()
	if unwrapped != cause3 {
		t.Error("Expected Unwrap to return cause3")
	}
}

type wrappedError struct {
	msg   string
	cause error
}

func (e *wrappedError) Error() string   { return e.msg }
func (e *wrappedError) Unwrap() error { return e.cause }

func TestNetworkError_ZeroStatusCode(t *testing.T) {
	err := NewNetworkError("http://example.com", 0)

	if err.StatusCode != 0 {
		t.Errorf("Expected status code 0, got %v", err.StatusCode)
	}

	// ToMap should include zero status code
	m := err.ToMap()
	if m["statusCode"] != 0 {
		t.Errorf("Expected statusCode 0 in map, got %v", m["statusCode"])
	}
}

func TestNetworkError_EmptyURL(t *testing.T) {
	err := NewNetworkError("", 404)

	if err.URL != "" {
		t.Errorf("Expected empty URL, got %v", err.URL)
	}

	// Error() should still work
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}
}

func TestNetworkTimeoutError_EmptyURL(t *testing.T) {
	err := NewNetworkTimeoutError("")

	if !err.Timeout {
		t.Error("Expected Timeout to be true")
	}
	if err.URL != "" {
		t.Errorf("Expected empty URL, got %v", err.URL)
	}
}

func TestBusinessLogicError_EmptyFields(t *testing.T) {
	err := NewBusinessLogicError("", "")

	if err.Rule != "" || err.Details != "" {
		t.Error("Expected empty fields")
	}

	// Should still have location info
	if err.File == "" {
		t.Error("Expected File to be populated")
	}
}

func TestConfigError_EmptyFields(t *testing.T) {
	err := NewConfigError("", "", "")

	if err.Key != "" || err.Value != "" || err.Reason != "" {
		t.Error("Expected empty fields")
	}

	// ToMap should work
	m := err.ToMap()
	if m["type"] != "ConfigError" {
		t.Errorf("Expected type 'ConfigError', got %v", m["type"])
	}
}

func TestAuthError_EmptyFields(t *testing.T) {
	err := NewAuthError("", "", "")

	if err.Operation != "" || err.User != "" || err.Reason != "" {
		t.Error("Expected empty fields")
	}
}

func TestRateLimitError_ZeroValues(t *testing.T) {
	err := NewRateLimitError("", 0, 0, 0)

	if err.Resource != "" {
		t.Errorf("Expected empty resource, got %v", err.Resource)
	}
	if err.Limit != 0 || err.Current != 0 || err.RetryAfter != 0 {
		t.Error("Expected zero values for limit/current/retryAfter")
	}
}

func TestRateLimitError_ExceededLimit(t *testing.T) {
	err := NewRateLimitError("api", 100, 150, 60)

	if err.Current <= err.Limit {
		t.Error("Expected current > limit for exceeded scenario")
	}

	// Error message should reflect the exceeded state
	errStr := err.Error()
	if !strings.Contains(errStr, "150/100") {
		t.Errorf("Error message should show current/limit: %s", errStr)
	}
}

func TestErrorIs_DifferentTypes(t *testing.T) {
	valErr := NewValidationError("field", "msg", 1001)
	dbErr := NewDatabaseError("SELECT", "table", nil)

	// Is should return false for different types
	if valErr.Is(dbErr) {
		t.Error("Expected Is to return false for different types")
	}
	if dbErr.Is(valErr) {
		t.Error("Expected Is to return false for different types")
	}
}

func TestErrorIs_SameCodeValidation(t *testing.T) {
	err1 := NewValidationError("field1", "msg1", 1001)
	err2 := NewValidationError("field2", "msg2", 1001)

	// Same code should match
	if !err1.Is(err2) {
		t.Error("Expected Is to return true for same error code")
	}
}

func TestErrorIs_ZeroCodeValidation(t *testing.T) {
	err1 := NewValidationError("field1", "msg1", 0)
	err2 := NewValidationError("field2", "msg2", 0)

	// Zero code should not match (per implementation)
	if err1.Is(err2) {
		t.Error("Expected Is to return false for zero error codes")
	}
}

func TestToMap_AllFields(t *testing.T) {
	err := NewValidationError("email", "invalid format", 1001)

	m := err.ToMap()

	// Check all expected fields exist
	expectedFields := []string{"type", "field", "message", "code", "file", "line", "function", "timestamp", "stack"}
	for _, field := range expectedFields {
		if _, ok := m[field]; !ok {
			t.Errorf("Expected field '%s' in ToMap output", field)
		}
	}

	// Verify type is correct
	if m["type"] != "ValidationError" {
		t.Errorf("Expected type 'ValidationError', got %v", m["type"])
	}
}

func TestToJSON_ValidJSON(t *testing.T) {
	err := NewValidationError("field", "message", 1001)

	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("ToJSON failed: %v", jsonErr)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if unmarshalErr := json.Unmarshal(jsonBytes, &parsed); unmarshalErr != nil {
		t.Errorf("Failed to unmarshal JSON: %v", unmarshalErr)
	}
}

func TestToJSON_UnicodeSupport(t *testing.T) {
	err := NewValidationError("字段", "错误消息：包含中文和emoji 🔥💯", 999)

	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("ToJSON failed: %v", jsonErr)
	}

	var parsed map[string]interface{}
	if unmarshalErr := json.Unmarshal(jsonBytes, &parsed); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	// Verify Chinese characters are preserved
	if field, ok := parsed["field"].(string); !ok || field != "字段" {
		t.Errorf("Field not properly encoded: %v", parsed["field"])
	}
	if msg, ok := parsed["message"].(string); !ok || !strings.Contains(msg, "emoji") {
		t.Errorf("Message not properly encoded: %v", parsed["message"])
	}
}

func TestStackCapture(t *testing.T) {
	err := NewValidationError("field", "msg", 1001)

	// Stack should not be empty
	if len(err.Stack) == 0 {
		t.Error("Expected non-empty stack trace")
	}

	// Stack should contain function names
	foundTestFunc := false
	for _, frame := range err.Stack {
		if strings.Contains(frame, "TestStackCapture") {
			foundTestFunc = true
			break
		}
	}
	if !foundTestFunc {
		t.Error("Expected stack to contain test function name")
	}
}

func TestTimestamp_Recent(t *testing.T) {
	before := time.Now()
	err := NewValidationError("field", "msg", 1001)
	after := time.Now()

	// Timestamp should be between before and after
	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Errorf("Timestamp %v not in expected range [%v, %v]", err.Timestamp, before, after)
	}
}

func TestConcurrentErrorCreation(t *testing.T) {
	const goroutines = 100
	var wg sync.WaitGroup
	errs := make(chan ValidationError, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := NewValidationError("field", "message", id)
			errs <- err
		}(i)
	}

	wg.Wait()
	close(errs)

	count := 0
	for err := range errs {
		count++
		if err.File == "" {
			t.Error("Expected File to be populated in concurrent creation")
		}
	}

	if count != goroutines {
		t.Errorf("Expected %d errors, got %d", goroutines, count)
	}
}

func TestConcurrentToJSON(t *testing.T) {
	err := NewValidationError("field", "message", 1001)

	const goroutines = 100
	var wg sync.WaitGroup
	results := make(chan []byte, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jsonBytes, _ := err.ToJSON()
			results <- jsonBytes
		}()
	}

	wg.Wait()
	close(results)

	for jsonBytes := range results {
		if len(jsonBytes) == 0 {
			t.Error("Expected non-empty JSON output")
		}
	}
}

func TestErrorString_Format(t *testing.T) {
	// Test that Error() strings contain key information
	testCases := []struct {
		name     string
		errStr   string
		contains string
	}{
		{"ValidationError", NewValidationError("email", "invalid", 1001).Error(), "email"},
		{"DatabaseError", NewDatabaseError("SELECT", "users", nil).Error(), "users"},
		{"NetworkError", NewNetworkError("http://example.com", 404).Error(), "404"},
		{"NetworkTimeoutError", NewNetworkTimeoutError("http://example.com").Error(), "timeout"},
		{"BusinessLogicError", NewBusinessLogicError("rule1", "details").Error(), "rule1"},
		{"ConfigError", NewConfigError("key", "value", "reason").Error(), "key"},
		{"AuthError", NewAuthError("login", "user", "reason").Error(), "login"},
		{"RateLimitError", NewRateLimitError("api", 100, 150, 60).Error(), "api"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.errStr, tc.contains) {
				t.Errorf("Error string '%s' should contain '%s'", tc.errStr, tc.contains)
			}
		})
	}
}
