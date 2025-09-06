// Package errors provides common error types that can be used with the gotrycatch library.
//
// This package includes pre-defined error types for common scenarios like validation,
// database operations, network operations, and business logic violations.
package errors

import "fmt"

// ValidationError represents an error that occurs during data validation.
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // Human-readable error message
	Code    int    // Error code for programmatic handling
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error [%d] on field '%s': %s", e.Code, e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string, code int) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	}
}

// DatabaseError represents an error that occurs during database operations.
type DatabaseError struct {
	Operation string // The database operation that failed (SELECT, INSERT, UPDATE, DELETE)
	Table     string // The table involved in the operation
	Cause     error  // The underlying error
}

func (e DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s on table '%s': %v", e.Operation, e.Table, e.Cause)
}

// NewDatabaseError creates a new DatabaseError.
func NewDatabaseError(operation, table string, cause error) DatabaseError {
	return DatabaseError{
		Operation: operation,
		Table:     table,
		Cause:     cause,
	}
}

// NetworkError represents an error that occurs during network operations.
type NetworkError struct {
	URL        string // The URL that was accessed
	StatusCode int    // HTTP status code (if applicable)
	Timeout    bool   // Whether the error was due to a timeout
}

func (e NetworkError) Error() string {
	if e.Timeout {
		return fmt.Sprintf("network timeout when accessing %s", e.URL)
	}
	return fmt.Sprintf("network error %d when accessing %s", e.StatusCode, e.URL)
}

// NewNetworkError creates a new NetworkError with a status code.
func NewNetworkError(url string, statusCode int) NetworkError {
	return NetworkError{
		URL:        url,
		StatusCode: statusCode,
		Timeout:    false,
	}
}

// NewNetworkTimeoutError creates a new NetworkError for timeout scenarios.
func NewNetworkTimeoutError(url string) NetworkError {
	return NetworkError{
		URL:     url,
		Timeout: true,
	}
}

// BusinessLogicError represents an error that occurs due to business rule violations.
type BusinessLogicError struct {
	Rule    string // The business rule that was violated
	Details string // Additional details about the violation
}

func (e BusinessLogicError) Error() string {
	return fmt.Sprintf("business rule violation: %s - %s", e.Rule, e.Details)
}

// NewBusinessLogicError creates a new BusinessLogicError.
func NewBusinessLogicError(rule, details string) BusinessLogicError {
	return BusinessLogicError{
		Rule:    rule,
		Details: details,
	}
}
