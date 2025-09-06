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

// TryBlock represents a try block that can catch and handle panics
type TryBlock struct {
	err     interface{}
	handled bool
}

// Try executes the given function and captures any panic that occurs.
// It returns a TryBlock that can be used with Catch and Finally methods.
func Try(fn func()) *TryBlock {
	tb := &TryBlock{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				tb.err = r
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

// CatchWithReturn handles panics of the specified type T and allows the handler to return a value.
// If the panic value can be cast to type T, the handler function is called and its return value
// is returned along with the TryBlock.
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

// CatchAny handles any unhandled panic, regardless of type.
// This method should typically be called last in a chain of Catch calls.
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

// Finally executes the given function regardless of whether a panic occurred.
// If there was an unhandled panic, it will be re-thrown after the finally block executes.
func (tb *TryBlock) Finally(fn func()) {
	if tb == nil {
		fn()
		return
	}

	defer fn()
	if tb.err != nil && !tb.handled {
		panic(tb.err) // Re-throw unhandled exception
	}
}

// Throw creates a panic with the given value.
// This is a convenience function to make code more readable.
func Throw(err interface{}) {
	panic(err)
}
