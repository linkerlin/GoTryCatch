// Package gotrycatch provides a generic try-catch exception handling mechanism for Go.
//
// This package implements a type-safe exception handling system using Go generics,
// allowing developers to write more structured error handling code similar to
// try-catch blocks in other languages.
//
// Basic usage:
//
//	import "github.com/linkerlin/gotrycatch"
//
//	tb := gotrycatch.Try(func() {
//		// Code that might panic
//		panic("something went wrong")
//	})
//
//	tb = gotrycatch.Catch[string](tb, func(err string) {
//		fmt.Printf("Caught string error: %s\n", err)
//	})
//
//	tb.Finally(func() {
//		fmt.Println("Cleanup code")
//	})
package gotrycatch

import (
	"fmt"
	"log"
	"os"
)

// Version is the current version of the gotrycatch library.
const Version = "1.0.0"

// Global debug mode flag
var debugMode = false
var debugLogger = log.New(os.Stderr, "[gotrycatch] ", log.LstdFlags)

// SetDebug enables or disables debug mode. When enabled, type matching and exception handling details are logged.
func SetDebug(enabled bool) {
	debugMode = enabled
}

// IsDebug returns whether debug mode is currently enabled.
func IsDebug() bool {
	return debugMode
}

// debugLog outputs debug messages when debug mode is enabled.
func debugLog(format string, args ...interface{}) {
	if debugMode {
		debugLogger.Printf(format, args...)
	}
}

// TryBlock represents a try block that can catch and handle panics
type TryBlock struct {
	err     interface{}
	handled bool
}

// GetError returns the captured error, or nil if no error occurred.
// Returns nil if the TryBlock itself is nil.
func (tb *TryBlock) GetError() interface{} {
	if tb == nil {
		return nil
	}
	return tb.err
}

// HasError returns true if a panic was captured.
// Returns false if the TryBlock itself is nil.
func (tb *TryBlock) HasError() bool {
	if tb == nil {
		return false
	}
	return tb.err != nil
}

// IsHandled returns true if the error has been handled by a Catch handler.
// Returns false if the TryBlock itself is nil.
func (tb *TryBlock) IsHandled() bool {
	if tb == nil {
		return false
	}
	return tb.handled
}

// String returns a friendly string representation of the TryBlock for debugging and logging.
func (tb *TryBlock) String() string {
	if tb == nil {
		return "TryBlock{nil}"
	}
	if tb.err == nil {
		return "TryBlock{err: nil, handled: false}"
	}
	return fmt.Sprintf("TryBlock{err: %T(%v), handled: %v}", tb.err, tb.err, tb.handled)
}

// GetErrorType returns the type name of the captured error (e.g., "errors.ValidationError").
// Returns an empty string if the TryBlock is nil or no error was captured.
// Useful for Agent-based error type determination.
func (tb *TryBlock) GetErrorType() string {
	if tb == nil || tb.err == nil {
		return ""
	}
	return fmt.Sprintf("%T", tb.err)
}

// Try executes the given function and captures any panic that occurs.
// It returns a TryBlock that can be used with Catch and Finally methods.
func Try(fn func()) *TryBlock {
	tb := &TryBlock{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				tb.err = r
				debugLog("Try: captured panic of type %T: %v", r, r)
			}
		}()

		fn()
	}()

	return tb
}

// Catch handles panics of the specified type T.
// If the panic value can be cast to type T, the handler function is called.
// Returns the same TryBlock to allow chaining multiple Catch calls.
func Catch[T any](tb *TryBlock, handler func(T)) *TryBlock {
	if tb == nil {
		debugLog("Catch: TryBlock is nil, returning empty TryBlock")
		return &TryBlock{}
	}

	if handler == nil {
		debugLog("Catch: handler is nil, returning TryBlock unchanged")
		return tb
	}

	if tb.err != nil && !tb.handled {
		if err, ok := tb.err.(T); ok {
			debugLog("Catch: type %T matched, calling handler", tb.err)
			handler(err)
			tb.handled = true
		} else {
			debugLog("Catch: type %T does not match target type %T", tb.err, *new(T))
		}
	}
	return tb
}

// CatchWithReturn handles panics of the specified type T and allows the handler to return a value.
// If the panic value can be cast to type T, the handler function is called and its return value
// is returned along with the TryBlock.
func CatchWithReturn[T any](tb *TryBlock, handler func(T) interface{}) (interface{}, *TryBlock) {
	if tb == nil {
		debugLog("CatchWithReturn: TryBlock is nil, returning empty TryBlock")
		return nil, &TryBlock{}
	}

	if handler == nil {
		debugLog("CatchWithReturn: handler is nil, returning TryBlock unchanged")
		return nil, tb
	}

	if tb.err != nil && !tb.handled {
		if err, ok := tb.err.(T); ok {
			debugLog("CatchWithReturn: type %T matched, calling handler", tb.err)
			result := handler(err)
			tb.handled = true
			return result, tb
		} else {
			debugLog("CatchWithReturn: type %T does not match target type %T", tb.err, *new(T))
		}
	}
	return nil, tb
}

// CatchAny handles any unhandled panic, regardless of type.
// This method should typically be called last in a chain of Catch calls.
func (tb *TryBlock) CatchAny(handler func(interface{})) *TryBlock {
	if tb == nil {
		debugLog("CatchAny: TryBlock is nil, returning empty TryBlock")
		return &TryBlock{}
	}

	if handler == nil {
		debugLog("CatchAny: handler is nil, returning TryBlock unchanged")
		return tb
	}

	if tb.err != nil && !tb.handled {
		debugLog("CatchAny: handling error of type %T", tb.err)
		handler(tb.err)
		tb.handled = true
	}
	return tb
}

// Finally executes the given function regardless of whether a panic occurred.
// If there was an unhandled panic, it will be re-thrown after the finally block executes.
func (tb *TryBlock) Finally(fn func()) {
	if fn == nil {
		debugLog("Finally: handler is nil, returning without action")
		return
	}

	if tb == nil {
		fn()
		return
	}

	defer fn()
	if tb.err != nil && !tb.handled {
		debugLog("Finally: re-throwing unhandled error of type %T: %v", tb.err, tb.err)
		panic(tb.err) // Re-throw unhandled exception
	}
}

// Throw creates a panic with the given value.
// This is a convenience function to make code more readable.
func Throw(err interface{}) {
	panic(err)
}

// Assert throws the given error if the condition is false.
func Assert(condition bool, err interface{}) {
	if !condition {
		Throw(err)
	}
}

// AssertNoError throws a wrapped error if err is not nil.
// The error message includes the provided msg prefix.
func AssertNoError(err error, msg string) {
	if err != nil {
		Throw(fmt.Errorf("%s: %w", msg, err))
	}
}

// ============================================
// TryWithResult - Try with return value support
// ============================================

// TryBlockWithResult represents a try block that captures both a return value and any panic.
type TryBlockWithResult[T any] struct {
	result  T
	err     interface{}
	handled bool
}

// GetResult returns the result of the executed function.
// Returns the zero value of T if the TryBlockWithResult is nil.
func (tb *TryBlockWithResult[T]) GetResult() T {
	if tb == nil {
		var zero T
		return zero
	}
	return tb.result
}

// GetError returns the captured error, or nil if no error occurred.
// Returns nil if the TryBlockWithResult itself is nil.
func (tb *TryBlockWithResult[T]) GetError() interface{} {
	if tb == nil {
		return nil
	}
	return tb.err
}

// HasError returns true if a panic was captured.
// Returns false if the TryBlockWithResult itself is nil.
func (tb *TryBlockWithResult[T]) HasError() bool {
	if tb == nil {
		return false
	}
	return tb.err != nil
}

// IsHandled returns true if the error has been handled by a Catch handler.
// Returns false if the TryBlockWithResult itself is nil.
func (tb *TryBlockWithResult[T]) IsHandled() bool {
	if tb == nil {
		return false
	}
	return tb.handled
}

// String returns a friendly string representation for debugging.
func (tb *TryBlockWithResult[T]) String() string {
	if tb == nil {
		return "TryBlockWithResult{nil}"
	}
	return fmt.Sprintf("TryBlockWithResult{result: %v, err: %T(%v), handled: %v}", tb.result, tb.err, tb.err, tb.handled)
}

// TryWithResult executes the given function and captures both the return value and any panic.
func TryWithResult[T any](fn func() T) *TryBlockWithResult[T] {
	tb := &TryBlockWithResult[T]{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				tb.err = r
				debugLog("TryWithResult: captured panic of type %T: %v", r, r)
			}
		}()

		tb.result = fn()
	}()

	return tb
}

// CatchWithResult handles panics of type E in a TryBlockWithResult[T].
// If the panic value can be cast to type E, the handler is called.
func CatchWithResult[T any, E any](tb *TryBlockWithResult[T], handler func(E)) *TryBlockWithResult[T] {
	if tb == nil {
		debugLog("CatchWithResult: TryBlockWithResult is nil")
		return &TryBlockWithResult[T]{}
	}

	if handler == nil {
		debugLog("CatchWithResult: handler is nil, returning TryBlockWithResult unchanged")
		return tb
	}

	if tb.err != nil && !tb.handled {
		if err, ok := tb.err.(E); ok {
			debugLog("CatchWithResult: type %T matched, calling handler", tb.err)
			handler(err)
			tb.handled = true
		} else {
			debugLog("CatchWithResult: type %T does not match target type %T", tb.err, *new(E))
		}
	}
	return tb
}

// CatchAnyWithResult handles any unhandled panic in a TryBlockWithResult[T].
// This should typically be called last in a chain of CatchWithResult calls.
func CatchAnyWithResult[T any](tb *TryBlockWithResult[T], handler func(interface{})) *TryBlockWithResult[T] {
	if tb == nil {
		debugLog("CatchAnyWithResult: TryBlockWithResult is nil")
		return &TryBlockWithResult[T]{}
	}

	if handler == nil {
		debugLog("CatchAnyWithResult: handler is nil, returning TryBlockWithResult unchanged")
		return tb
	}

	if tb.err != nil && !tb.handled {
		debugLog("CatchAnyWithResult: handling error of type %T", tb.err)
		handler(tb.err)
		tb.handled = true
	}
	return tb
}

// Finally executes the cleanup function regardless of whether a panic occurred.
// If there was an unhandled panic, it will be re-thrown after fn executes.
// Returns the result value (or zero value if nil).
func (tb *TryBlockWithResult[T]) Finally(fn func()) T {
	if fn == nil {
		debugLog("Finally: handler is nil, returning without action")
		var zero T
		if tb != nil {
			return tb.result
		}
		return zero
	}

	if tb == nil {
		fn()
		var zero T
		return zero
	}

	defer fn()
	if tb.err != nil && !tb.handled {
		debugLog("Finally: re-throwing unhandled error of type %T: %v", tb.err, tb.err)
		panic(tb.err)
	}
	return tb.result
}

// OnSuccess executes the callback when no error occurred.
// Returns the TryBlockWithResult for method chaining.
func (tb *TryBlockWithResult[T]) OnSuccess(fn func(T)) *TryBlockWithResult[T] {
	if tb != nil && tb.err == nil && fn != nil {
		fn(tb.result)
	}
	return tb
}

// OnError executes the callback when an error occurred and marks the error as handled.
// Returns the TryBlockWithResult for method chaining.
func (tb *TryBlockWithResult[T]) OnError(fn func(interface{})) *TryBlockWithResult[T] {
	if tb != nil && tb.err != nil && !tb.handled && fn != nil {
		fn(tb.err)
		tb.handled = true
	}
	return tb
}

// OrElse returns the result if successful, or the defaultValue if an error occurred.
// Also returns defaultValue if the TryBlockWithResult is nil.
func (tb *TryBlockWithResult[T]) OrElse(defaultValue T) T {
	if tb == nil || tb.err != nil {
		return defaultValue
	}
	return tb.result
}

// OrElseGet returns the result if successful, or calls the supplier function to get a default value if an error occurred.
// Also calls supplier if the TryBlockWithResult is nil.
func (tb *TryBlockWithResult[T]) OrElseGet(supplier func() T) T {
	if tb == nil || tb.err != nil {
		return supplier()
	}
	return tb.result
}
