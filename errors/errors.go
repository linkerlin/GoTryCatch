// Package errors is a compatibility alias for github.com/linkerlin/gotrycatch/errtypes.
//
// The package was renamed to errtypes in v2.0 to avoid clashing with the
// standard library's errors package. All types are exact aliases and all
// constructors are direct bindings, so existing imports keep compiling and
// behave identically — including File/Line/Stack attribution.
//
// New error types will only be added to errtypes. Prefer importing errtypes
// in new code.
package errors

import "github.com/linkerlin/gotrycatch/errtypes"

// BaseError carries the location and stack information shared by all error types.
type BaseError = errtypes.BaseError

// ValidationError represents a data validation error.
type ValidationError = errtypes.ValidationError

// DatabaseError represents a database operation error.
type DatabaseError = errtypes.DatabaseError

// NetworkError represents a network operation error.
type NetworkError = errtypes.NetworkError

// BusinessLogicError represents a business rule violation.
type BusinessLogicError = errtypes.BusinessLogicError

// ConfigError represents a configuration error.
type ConfigError = errtypes.ConfigError

// AuthError represents an authentication/authorization error.
type AuthError = errtypes.AuthError

// RateLimitError represents a rate limiting error.
type RateLimitError = errtypes.RateLimitError

// Constructors are bound directly (not wrapped), so runtime.Caller-based
// File/Line/Stack attribution points at the user's call site, exactly as
// calling errtypes.NewXxx yourself.

// NewValidationError creates a new ValidationError with automatic stack capture.
var NewValidationError = errtypes.NewValidationError

// NewDatabaseError creates a new DatabaseError with automatic stack capture.
var NewDatabaseError = errtypes.NewDatabaseError

// NewNetworkError creates a new NetworkError with a status code.
var NewNetworkError = errtypes.NewNetworkError

// NewNetworkTimeoutError creates a new NetworkError for timeout scenarios.
var NewNetworkTimeoutError = errtypes.NewNetworkTimeoutError

// NewBusinessLogicError creates a new BusinessLogicError.
var NewBusinessLogicError = errtypes.NewBusinessLogicError

// NewConfigError creates a new ConfigError.
var NewConfigError = errtypes.NewConfigError

// NewAuthError creates a new AuthError.
var NewAuthError = errtypes.NewAuthError

// NewRateLimitError creates a new RateLimitError.
var NewRateLimitError = errtypes.NewRateLimitError
