// Package errtypes provides common error types that can be used with the gotrycatch library.
//
// This package includes pre-defined error types for common scenarios like validation,
// database operations, network operations, and business logic violations.
//
// All error types support:
//   - Call stack tracing
//   - Error chains (Unwrap/Is/As)
//   - Structured output (ToMap/ToJSON)
//   - Timestamp recording
//
// To define your own error type, embed BaseError, implement Error/Is/ToMap,
// and call newBase(1) in your constructor — see any type below for the pattern.
package errtypes

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// ============================================
// Base error infrastructure
// ============================================

// captureCaller captures the caller's file, line, and function name.
func captureCaller(skip int) (file string, line int, function string) {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown", 0, "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return file, line, "unknown"
	}
	return file, line, fn.Name()
}

// captureStack captures the call stack starting from the specified skip depth,
// formatted as "file:line function" strings.
func captureStack(skip int) []string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip+2, pcs[:])

	var stack []string
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return stack
}

// BaseError carries the location and stack information shared by all error
// types in this package. Embed it in your own error types to get File/Line/
// Function/Timestamp/Stack fields for free.
type BaseError struct {
	File      string    `json:"file"`      // Source file name
	Line      int       `json:"line"`      // Line number
	Function  string    `json:"function"`  // Function name
	Timestamp time.Time `json:"timestamp"` // When error occurred
	Stack     []string  `json:"stack"`     // Call stack trace
}

// newBase captures the caller context. skip follows runtime.Caller semantics:
// newBase(1) attributes the error to the caller of the constructor that calls newBase.
func newBase(skip int) BaseError {
	file, line, fn := captureCaller(skip + 1)
	return BaseError{
		File:      file,
		Line:      line,
		Function:  fn,
		Timestamp: time.Now(),
		Stack:     captureStack(skip + 1),
	}
}

// NewBase captures the caller's location and stack. Use it in custom error
// constructors outside this package so File/Line/Stack are attributed to the
// user's call site (the caller of your constructor):
//
//	type PaymentError struct {
//		errtypes.BaseError
//		OrderID string `json:"orderId"`
//	}
//
//	func NewPaymentError(orderID string) PaymentError {
//		return PaymentError{BaseError: errtypes.NewBase(), OrderID: orderID}
//	}
func NewBase() BaseError {
	return newBase(2)
}

// Unwrap returns nil; BaseError has no underlying error. Types with a cause
// (e.g. DatabaseError) shadow this with their own implementation.
func (e BaseError) Unwrap() error {
	return nil
}

// baseMap returns the map keys shared by all error types' ToMap output.
func (e BaseError) baseMap() map[string]interface{} {
	return map[string]interface{}{
		"file":      e.File,
		"line":      e.Line,
		"function":  e.Function,
		"timestamp": e.Timestamp.Format(time.RFC3339),
		"stack":     e.Stack,
	}
}

// ============================================
// ValidationError - Data validation errors
// ============================================

// ValidationError represents an error that occurs during data validation.
// Fields:
//   - Field: the field name that failed validation
//   - Message: human-readable error message
//   - Code: error code for programmatic handling
//   - BaseError: File, Line, Function, Timestamp, Stack
type ValidationError struct {
	BaseError
	Field   string `json:"field"`   // Field that failed validation
	Message string `json:"message"` // Human-readable error message
	Code    int    `json:"code"`    // Error code for programmatic handling
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error [%d] on field '%s': %s (at %s:%d)", e.Code, e.Field, e.Message, e.File, e.Line)
}

// Is returns true if the target error matches based on error code.
func (e ValidationError) Is(target error) bool {
	t, ok := target.(ValidationError)
	if !ok {
		return false
	}
	return e.Code != 0 && e.Code == t.Code
}

// ToMap returns structured error information for Agent parsing.
func (e ValidationError) ToMap() map[string]interface{} {
	m := e.baseMap()
	m["type"] = "ValidationError"
	m["field"] = e.Field
	m["message"] = e.Message
	m["code"] = e.Code
	return m
}

// ToJSON returns JSON-formatted error information.
func (e ValidationError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewValidationError creates a new ValidationError with automatic stack capture.
func NewValidationError(field, message string, code int) ValidationError {
	return ValidationError{
		BaseError: newBase(1),
		Field:     field,
		Message:   message,
		Code:      code,
	}
}

// ============================================
// DatabaseError - Database operation errors
// ============================================

// DatabaseError represents an error that occurs during database operations.
// Fields:
//   - Operation: the database operation that failed (SELECT, INSERT, UPDATE, DELETE)
//   - Table: the table involved in the operation
//   - Cause: the underlying error
//   - BaseError: File, Line, Function, Timestamp, Stack
type DatabaseError struct {
	BaseError
	Operation string `json:"operation"` // Database operation (SELECT, INSERT, UPDATE, DELETE)
	Table     string `json:"table"`     // Table name involved
	Cause     error  `json:"cause"`     // Underlying error
}

func (e DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s on table '%s': %v (at %s:%d)", e.Operation, e.Table, e.Cause, e.File, e.Line)
}

// Unwrap returns the underlying cause error.
func (e DatabaseError) Unwrap() error {
	return e.Cause
}

// Is returns true if the target error matches based on operation and table.
func (e DatabaseError) Is(target error) bool {
	t, ok := target.(DatabaseError)
	if !ok {
		return false
	}
	return e.Operation == t.Operation && e.Table == t.Table
}

// ToMap returns structured error information.
func (e DatabaseError) ToMap() map[string]interface{} {
	causeStr := ""
	if e.Cause != nil {
		causeStr = e.Cause.Error()
	}
	m := e.baseMap()
	m["type"] = "DatabaseError"
	m["operation"] = e.Operation
	m["table"] = e.Table
	m["cause"] = causeStr
	return m
}

// ToJSON returns JSON-formatted error information.
func (e DatabaseError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewDatabaseError creates a new DatabaseError with automatic stack capture.
func NewDatabaseError(operation, table string, cause error) DatabaseError {
	return DatabaseError{
		BaseError: newBase(1),
		Operation: operation,
		Table:     table,
		Cause:     cause,
	}
}

// ============================================
// NetworkError - Network operation errors
// ============================================

// NetworkError represents an error that occurs during network operations.
// Fields:
//   - URL: the requested URL
//   - StatusCode: HTTP status code (if applicable)
//   - Timeout: whether the error was caused by a timeout
//   - BaseError: File, Line, Function, Timestamp, Stack
type NetworkError struct {
	BaseError
	URL        string `json:"url"`        // Requested URL
	StatusCode int    `json:"statusCode"` // HTTP status code (if applicable)
	Timeout    bool   `json:"timeout"`    // Whether caused by timeout
}

func (e NetworkError) Error() string {
	if e.Timeout {
		return fmt.Sprintf("network timeout when accessing %s (at %s:%d)", e.URL, e.File, e.Line)
	}
	return fmt.Sprintf("network error %d when accessing %s (at %s:%d)", e.StatusCode, e.URL, e.File, e.Line)
}

// Is returns true if the target error matches based on URL and timeout status.
func (e NetworkError) Is(target error) bool {
	t, ok := target.(NetworkError)
	if !ok {
		return false
	}
	return e.URL == t.URL && e.Timeout == t.Timeout
}

// ToMap returns structured error information.
func (e NetworkError) ToMap() map[string]interface{} {
	m := e.baseMap()
	m["type"] = "NetworkError"
	m["url"] = e.URL
	m["statusCode"] = e.StatusCode
	m["timeout"] = e.Timeout
	return m
}

// ToJSON returns JSON-formatted error information.
func (e NetworkError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewNetworkError creates a new NetworkError with a status code and automatic stack capture.
func NewNetworkError(url string, statusCode int) NetworkError {
	return NetworkError{
		BaseError:  newBase(1),
		URL:        url,
		StatusCode: statusCode,
	}
}

// NewNetworkTimeoutError creates a new NetworkError for timeout scenarios with automatic stack capture.
func NewNetworkTimeoutError(url string) NetworkError {
	return NetworkError{
		BaseError: newBase(1),
		URL:       url,
		Timeout:   true,
	}
}

// ============================================
// BusinessLogicError - Business rule violations
// ============================================

// BusinessLogicError represents an error that occurs due to business rule violations.
// Fields:
//   - Rule: the name of the violated business rule
//   - Details: detailed information about the violation
//   - BaseError: File, Line, Function, Timestamp, Stack
type BusinessLogicError struct {
	BaseError
	Rule    string `json:"rule"`    // Violated business rule name
	Details string `json:"details"` // Violation details
}

func (e BusinessLogicError) Error() string {
	return fmt.Sprintf("business rule violation: %s - %s (at %s:%d)", e.Rule, e.Details, e.File, e.Line)
}

// Is returns true if the target error matches based on rule name.
func (e BusinessLogicError) Is(target error) bool {
	t, ok := target.(BusinessLogicError)
	if !ok {
		return false
	}
	return e.Rule == t.Rule
}

// ToMap returns structured error information.
func (e BusinessLogicError) ToMap() map[string]interface{} {
	m := e.baseMap()
	m["type"] = "BusinessLogicError"
	m["rule"] = e.Rule
	m["details"] = e.Details
	return m
}

// ToJSON returns JSON-formatted error information.
func (e BusinessLogicError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewBusinessLogicError creates a new BusinessLogicError with automatic stack capture.
func NewBusinessLogicError(rule, details string) BusinessLogicError {
	return BusinessLogicError{
		BaseError: newBase(1),
		Rule:      rule,
		Details:   details,
	}
}

// ============================================
// ConfigError - Configuration errors
// ============================================

// ConfigError represents an error that occurs during configuration loading or parsing.
// Fields:
//   - Key: the configuration key name
//   - Value: the configuration value
//   - Reason: the error reason
//   - BaseError: File, Line, Function, Timestamp, Stack
type ConfigError struct {
	BaseError
	Key    string `json:"key"`    // Configuration key name
	Value  string `json:"value"`  // Configuration value
	Reason string `json:"reason"` // Error reason
}

func (e ConfigError) Error() string {
	return fmt.Sprintf("config error on key '%s': %s (value: %q, at %s:%d)", e.Key, e.Reason, e.Value, e.File, e.Line)
}

// Is returns true if the target error matches based on key name.
func (e ConfigError) Is(target error) bool {
	t, ok := target.(ConfigError)
	if !ok {
		return false
	}
	return e.Key == t.Key
}

// ToMap returns structured error information.
func (e ConfigError) ToMap() map[string]interface{} {
	m := e.baseMap()
	m["type"] = "ConfigError"
	m["key"] = e.Key
	m["value"] = e.Value
	m["reason"] = e.Reason
	return m
}

// ToJSON returns JSON-formatted error information.
func (e ConfigError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewConfigError creates a new ConfigError with automatic stack capture.
func NewConfigError(key, value, reason string) ConfigError {
	return ConfigError{
		BaseError: newBase(1),
		Key:       key,
		Value:     value,
		Reason:    reason,
	}
}

// ============================================
// AuthError - Authentication/Authorization errors
// ============================================

// AuthError represents an error that occurs during authentication or authorization.
// Fields:
//   - Operation: the auth operation type (login, token_verify, permission_check)
//   - User: the user identifier
//   - Reason: the error reason
//   - BaseError: File, Line, Function, Timestamp, Stack
type AuthError struct {
	BaseError
	Operation string `json:"operation"` // Auth operation type (login, token_verify, permission_check)
	User      string `json:"user"`      // User identifier
	Reason    string `json:"reason"`    // Error reason
}

func (e AuthError) Error() string {
	return fmt.Sprintf("auth error during %s for user '%s': %s (at %s:%d)", e.Operation, e.User, e.Reason, e.File, e.Line)
}

// Is returns true if the target error matches based on operation type.
func (e AuthError) Is(target error) bool {
	t, ok := target.(AuthError)
	if !ok {
		return false
	}
	return e.Operation == t.Operation
}

// ToMap returns structured error information.
func (e AuthError) ToMap() map[string]interface{} {
	m := e.baseMap()
	m["type"] = "AuthError"
	m["operation"] = e.Operation
	m["user"] = e.User
	m["reason"] = e.Reason
	return m
}

// ToJSON returns JSON-formatted error information.
func (e AuthError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewAuthError creates a new AuthError with automatic stack capture.
func NewAuthError(operation, user, reason string) AuthError {
	return AuthError{
		BaseError: newBase(1),
		Operation: operation,
		User:      user,
		Reason:    reason,
	}
}

// ============================================
// RateLimitError - Rate limiting errors
// ============================================

// RateLimitError represents an error that occurs when rate limiting is triggered.
// Fields:
//   - Resource: the rate-limited resource
//   - Limit: the rate limit threshold
//   - Current: the current count
//   - RetryAfter: seconds to wait before retrying
//   - BaseError: File, Line, Function, Timestamp, Stack
type RateLimitError struct {
	BaseError
	Resource   string `json:"resource"`   // Rate-limited resource
	Limit      int    `json:"limit"`      // Rate limit threshold
	Current    int    `json:"current"`    // Current count
	RetryAfter int    `json:"retryAfter"` // Seconds to wait before retry
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded on '%s': %d/%d, retry after %ds (at %s:%d)", e.Resource, e.Current, e.Limit, e.RetryAfter, e.File, e.Line)
}

// Is returns true if the target error matches based on resource name.
func (e RateLimitError) Is(target error) bool {
	t, ok := target.(RateLimitError)
	if !ok {
		return false
	}
	return e.Resource == t.Resource
}

// ToMap returns structured error information.
func (e RateLimitError) ToMap() map[string]interface{} {
	m := e.baseMap()
	m["type"] = "RateLimitError"
	m["resource"] = e.Resource
	m["limit"] = e.Limit
	m["current"] = e.Current
	m["retryAfter"] = e.RetryAfter
	return m
}

// ToJSON returns JSON-formatted error information.
func (e RateLimitError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewRateLimitError creates a new RateLimitError with automatic stack capture.
func NewRateLimitError(resource string, limit, current, retryAfter int) RateLimitError {
	return RateLimitError{
		BaseError:  newBase(1),
		Resource:   resource,
		Limit:      limit,
		Current:    current,
		RetryAfter: retryAfter,
	}
}
