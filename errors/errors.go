// Package errors provides common error types that can be used with the gotrycatch library.
//
// This package includes pre-defined error types for common scenarios like validation,
// database operations, network operations, and business logic violations.
//
// All error types support:
//   - Call stack tracing
//   - Error chains (Unwrap/Is/As)
//   - Structured output (ToMap/ToJSON)
//   - Timestamp recording
package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// ============================================
// Base error interfaces and utility functions
// ============================================

// stackInfo stores call stack information.
type stackInfo struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Func string `json:"func"`
}

// captureStack captures call stack information starting from the specified skip depth.
func captureStack(skip int) []stackInfo {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip+2, pcs[:])

	var stack []stackInfo
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		stack = append(stack, stackInfo{
			File: frame.File,
			Line: frame.Line,
			Func: frame.Function,
		})
		if !more {
			break
		}
	}
	return stack
}

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

// ============================================
// ValidationError - Data validation errors
// ============================================

// ValidationError represents an error that occurs during data validation.
// Fields:
//   - Field: the field name that failed validation
//   - Message: human-readable error message
//   - Code: error code for programmatic handling
//   - File, Line, Function: source location where the error was created
//   - Timestamp: when the error occurred
//   - Stack: call stack trace
type ValidationError struct {
	Field     string    `json:"field"`     // Field that failed validation
	Message   string    `json:"message"`   // Human-readable error message
	Code      int       `json:"code"`      // Error code for programmatic handling
	File      string    `json:"file"`      // Source file name
	Line      int       `json:"line"`      // Line number
	Function  string    `json:"function"`  // Function name
	Timestamp time.Time `json:"timestamp"` // When error occurred
	Stack     []string  `json:"stack"`     // Call stack trace
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error [%d] on field '%s': %s (at %s:%d)", e.Code, e.Field, e.Message, e.File, e.Line)
}

// Unwrap returns nil; ValidationError has no underlying error.
func (e ValidationError) Unwrap() error {
	return nil
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
	return map[string]interface{}{
		"type":      "ValidationError",
		"field":     e.Field,
		"message":   e.Message,
		"code":      e.Code,
		"file":      e.File,
		"line":      e.Line,
		"function":  e.Function,
		"timestamp": e.Timestamp.Format(time.RFC3339),
		"stack":     e.Stack,
	}
}

// ToJSON returns JSON-formatted error information.
func (e ValidationError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewValidationError creates a new ValidationError with automatic stack capture.
func NewValidationError(field, message string, code int) ValidationError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return ValidationError{
		Field:     field,
		Message:   message,
		Code:      code,
		File:      file,
		Line:      line,
		Function:  fn,
		Timestamp: time.Now(),
		Stack:     stackStrs,
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
//   - File, Line, Function: source location
//   - Timestamp: when error occurred
//   - Stack: call stack trace
type DatabaseError struct {
	Operation string    `json:"operation"` // Database operation (SELECT, INSERT, UPDATE, DELETE)
	Table     string    `json:"table"`     // Table name involved
	Cause     error     `json:"cause"`     // Underlying error
	File      string    `json:"file"`      // Source file name
	Line      int       `json:"line"`      // Line number
	Function  string    `json:"function"`  // Function name
	Timestamp time.Time `json:"timestamp"` // When error occurred
	Stack     []string  `json:"stack"`     // Call stack trace
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
	return map[string]interface{}{
		"type":      "DatabaseError",
		"operation": e.Operation,
		"table":     e.Table,
		"cause":     causeStr,
		"file":      e.File,
		"line":      e.Line,
		"function":  e.Function,
		"timestamp": e.Timestamp.Format(time.RFC3339),
		"stack":     e.Stack,
	}
}

// ToJSON returns JSON-formatted error information.
func (e DatabaseError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewDatabaseError creates a new DatabaseError with automatic stack capture.
func NewDatabaseError(operation, table string, cause error) DatabaseError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return DatabaseError{
		Operation: operation,
		Table:     table,
		Cause:     cause,
		File:      file,
		Line:      line,
		Function:  fn,
		Timestamp: time.Now(),
		Stack:     stackStrs,
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
//   - File, Line, Function: source location
//   - Timestamp: when error occurred
//   - Stack: call stack trace
type NetworkError struct {
	URL        string    `json:"url"`        // Requested URL
	StatusCode int       `json:"statusCode"` // HTTP status code (if applicable)
	Timeout    bool      `json:"timeout"`    // Whether caused by timeout
	File       string    `json:"file"`       // Source file name
	Line       int       `json:"line"`       // Line number
	Function   string    `json:"function"`   // Function name
	Timestamp  time.Time `json:"timestamp"`  // When error occurred
	Stack      []string  `json:"stack"`      // Call stack trace
}

func (e NetworkError) Error() string {
	if e.Timeout {
		return fmt.Sprintf("network timeout when accessing %s (at %s:%d)", e.URL, e.File, e.Line)
	}
	return fmt.Sprintf("network error %d when accessing %s (at %s:%d)", e.StatusCode, e.URL, e.File, e.Line)
}

// Unwrap returns nil.
func (e NetworkError) Unwrap() error {
	return nil
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
	return map[string]interface{}{
		"type":       "NetworkError",
		"url":        e.URL,
		"statusCode": e.StatusCode,
		"timeout":    e.Timeout,
		"file":       e.File,
		"line":       e.Line,
		"function":   e.Function,
		"timestamp":  e.Timestamp.Format(time.RFC3339),
		"stack":      e.Stack,
	}
}

// ToJSON returns JSON-formatted error information.
func (e NetworkError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewNetworkError creates a new NetworkError with a status code and automatic stack capture.
func NewNetworkError(url string, statusCode int) NetworkError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return NetworkError{
		URL:        url,
		StatusCode: statusCode,
		Timeout:    false,
		File:       file,
		Line:       line,
		Function:   fn,
		Timestamp:  time.Now(),
		Stack:      stackStrs,
	}
}

// NewNetworkTimeoutError creates a new NetworkError for timeout scenarios with automatic stack capture.
func NewNetworkTimeoutError(url string) NetworkError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return NetworkError{
		URL:       url,
		Timeout:   true,
		File:      file,
		Line:      line,
		Function:  fn,
		Timestamp: time.Now(),
		Stack:     stackStrs,
	}
}

// ============================================
// BusinessLogicError - Business rule violations
// ============================================

// BusinessLogicError represents an error that occurs due to business rule violations.
// Fields:
//   - Rule: the name of the violated business rule
//   - Details: detailed information about the violation
//   - File, Line, Function: source location
//   - Timestamp: when error occurred
//   - Stack: call stack trace
type BusinessLogicError struct {
	Rule      string    `json:"rule"`      // Violated business rule name
	Details   string    `json:"details"`   // Violation details
	File      string    `json:"file"`      // Source file name
	Line      int       `json:"line"`      // Line number
	Function  string    `json:"function"`  // Function name
	Timestamp time.Time `json:"timestamp"` // When error occurred
	Stack     []string  `json:"stack"`     // Call stack trace
}

func (e BusinessLogicError) Error() string {
	return fmt.Sprintf("business rule violation: %s - %s (at %s:%d)", e.Rule, e.Details, e.File, e.Line)
}

// Unwrap returns nil.
func (e BusinessLogicError) Unwrap() error {
	return nil
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
	return map[string]interface{}{
		"type":      "BusinessLogicError",
		"rule":      e.Rule,
		"details":   e.Details,
		"file":      e.File,
		"line":      e.Line,
		"function":  e.Function,
		"timestamp": e.Timestamp.Format(time.RFC3339),
		"stack":     e.Stack,
	}
}

// ToJSON returns JSON-formatted error information.
func (e BusinessLogicError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewBusinessLogicError creates a new BusinessLogicError with automatic stack capture.
func NewBusinessLogicError(rule, details string) BusinessLogicError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return BusinessLogicError{
		Rule:      rule,
		Details:   details,
		File:      file,
		Line:      line,
		Function:  fn,
		Timestamp: time.Now(),
		Stack:     stackStrs,
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
//   - File, Line, Function: source location
//   - Timestamp: when error occurred
//   - Stack: call stack trace
type ConfigError struct {
	Key       string    `json:"key"`       // Configuration key name
	Value     string    `json:"value"`     // Configuration value
	Reason    string    `json:"reason"`    // Error reason
	File      string    `json:"file"`      // Source file name
	Line      int       `json:"line"`      // Line number
	Function  string    `json:"function"`  // Function name
	Timestamp time.Time `json:"timestamp"` // When error occurred
	Stack     []string  `json:"stack"`     // Call stack trace
}

func (e ConfigError) Error() string {
	return fmt.Sprintf("config error on key '%s': %s (value: %q, at %s:%d)", e.Key, e.Reason, e.Value, e.File, e.Line)
}

// Unwrap returns nil.
func (e ConfigError) Unwrap() error {
	return nil
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
	return map[string]interface{}{
		"type":      "ConfigError",
		"key":       e.Key,
		"value":     e.Value,
		"reason":    e.Reason,
		"file":      e.File,
		"line":      e.Line,
		"function":  e.Function,
		"timestamp": e.Timestamp.Format(time.RFC3339),
		"stack":     e.Stack,
	}
}

// ToJSON returns JSON-formatted error information.
func (e ConfigError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewConfigError creates a new ConfigError with automatic stack capture.
func NewConfigError(key, value, reason string) ConfigError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return ConfigError{
		Key:       key,
		Value:     value,
		Reason:    reason,
		File:      file,
		Line:      line,
		Function:  fn,
		Timestamp: time.Now(),
		Stack:     stackStrs,
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
//   - File, Line, Function: source location
//   - Timestamp: when error occurred
//   - Stack: call stack trace
type AuthError struct {
	Operation string    `json:"operation"` // Auth operation type (login, token_verify, permission_check)
	User      string    `json:"user"`      // User identifier
	Reason    string    `json:"reason"`    // Error reason
	File      string    `json:"file"`      // Source file name
	Line      int       `json:"line"`      // Line number
	Function  string    `json:"function"`  // Function name
	Timestamp time.Time `json:"timestamp"` // When error occurred
	Stack     []string  `json:"stack"`     // Call stack trace
}

func (e AuthError) Error() string {
	return fmt.Sprintf("auth error during %s for user '%s': %s (at %s:%d)", e.Operation, e.User, e.Reason, e.File, e.Line)
}

// Unwrap returns nil.
func (e AuthError) Unwrap() error {
	return nil
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
	return map[string]interface{}{
		"type":      "AuthError",
		"operation": e.Operation,
		"user":      e.User,
		"reason":    e.Reason,
		"file":      e.File,
		"line":      e.Line,
		"function":  e.Function,
		"timestamp": e.Timestamp.Format(time.RFC3339),
		"stack":     e.Stack,
	}
}

// ToJSON returns JSON-formatted error information.
func (e AuthError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewAuthError creates a new AuthError with automatic stack capture.
func NewAuthError(operation, user, reason string) AuthError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return AuthError{
		Operation: operation,
		User:      user,
		Reason:    reason,
		File:      file,
		Line:      line,
		Function:  fn,
		Timestamp: time.Now(),
		Stack:     stackStrs,
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
//   - File, Line, Function: source location
//   - Timestamp: when error occurred
//   - Stack: call stack trace
type RateLimitError struct {
	Resource   string    `json:"resource"`   // Rate-limited resource
	Limit      int       `json:"limit"`      // Rate limit threshold
	Current    int       `json:"current"`    // Current count
	RetryAfter int       `json:"retryAfter"` // Seconds to wait before retry
	File       string    `json:"file"`       // Source file name
	Line       int       `json:"line"`       // Line number
	Function   string    `json:"function"`   // Function name
	Timestamp  time.Time `json:"timestamp"`  // When error occurred
	Stack      []string  `json:"stack"`      // Call stack trace
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded on '%s': %d/%d, retry after %ds (at %s:%d)", e.Resource, e.Current, e.Limit, e.RetryAfter, e.File, e.Line)
}

// Unwrap returns nil.
func (e RateLimitError) Unwrap() error {
	return nil
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
	return map[string]interface{}{
		"type":       "RateLimitError",
		"resource":   e.Resource,
		"limit":      e.Limit,
		"current":    e.Current,
		"retryAfter": e.RetryAfter,
		"file":       e.File,
		"line":       e.Line,
		"function":   e.Function,
		"timestamp":  e.Timestamp.Format(time.RFC3339),
		"stack":      e.Stack,
	}
}

// ToJSON returns JSON-formatted error information.
func (e RateLimitError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewRateLimitError creates a new RateLimitError with automatic stack capture.
func NewRateLimitError(resource string, limit, current, retryAfter int) RateLimitError {
	file, line, fn := captureCaller(1)
	stack := captureStack(1)

	var stackStrs []string
	for _, s := range stack {
		stackStrs = append(stackStrs, fmt.Sprintf("%s:%d %s", s.File, s.Line, s.Func))
	}

	return RateLimitError{
		Resource:   resource,
		Limit:      limit,
		Current:    current,
		RetryAfter: retryAfter,
		File:       file,
		Line:       line,
		Function:   fn,
		Timestamp:  time.Now(),
		Stack:      stackStrs,
	}
}
