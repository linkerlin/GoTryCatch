// Run-style clause API — the idiomatic, hard-to-misuse surface of v2.
//
// Run executes fn, dispatches any panic to the declared clauses in order,
// always runs Cleanup clauses (LIFO), and returns unhandled panics as an
// error, so results flow through ordinary `if err != nil` code:
//
//	err := gotrycatch.Run(func() { query(ctx) },
//		gotrycatch.OnAs(func(e *errtypes.DatabaseError) { retry(e) }),
//		gotrycatch.On(func(e errtypes.RateLimitError) { wait(e.RetryAfter) }),
//		gotrycatch.Cleanup(func() { conn.Close() }),
//	)
//	if err != nil {
//		return fmt.Errorf("query users: %w", err)
//	}
package gotrycatch

import (
	"errors"
	"fmt"
	"runtime"
)

// ============================================
// PanicError - bridging panic values into the error world
// ============================================

// PanicError wraps a non-error panic value so it can flow through
// error-returning APIs (Run, Err). If the panic value itself implements
// error, it is returned unwrapped instead (see toError), so wrapping
// chains and errors.Is/As comparisons keep working.
type PanicError struct {
	Value interface{} // the original panic value
	pcs   []uintptr   // call stack captured at recover time
}

// Error implements the error interface.
func (e *PanicError) Error() string {
	if err, ok := e.Value.(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("panic: %v", e.Value)
}

// Unwrap exposes the wrapped error when the panic value is an error,
// enabling errors.Is/errors.As through a *PanicError.
func (e *PanicError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}

// Stack formats and returns the call stack captured when the panic was
// recovered. Because the capture happens inside the recovering defer, the
// stack starts at the panic site and includes everything up to the Run/Try
// caller. It is rebuilt on every call and safe for concurrent use.
func (e *PanicError) Stack() []string {
	var stack []string
	if e.pcs != nil {
		frames := runtime.CallersFrames(e.pcs)
		for {
			f, more := frames.Next()
			stack = append(stack, fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Function))
			if !more {
				break
			}
		}
	}
	return stack
}

// toError converts a recovered panic value into an error:
// error panics pass through unchanged; everything else becomes *PanicError.
func toError(v interface{}, pcs []uintptr) error {
	if err, ok := v.(error); ok {
		return err
	}
	return &PanicError{Value: v, pcs: pcs}
}

// asMatch matches v against error type E using errors.As semantics:
// it unwraps wrapping chains, and also accepts direct assertions of E
// so plain value/pointer panics still match.
func asMatch[E error](v interface{}) (E, bool) {
	var target E
	if err, ok := v.(error); ok {
		if errors.As(err, &target) {
			return target, true
		}
		return target, false
	}
	e, ok := v.(E)
	return e, ok
}

// ============================================
// Clause - declarative handler clauses for Run
// ============================================

// Clause is a single handler declaration for Run. Build clauses with
// On, OnAs, Any and Cleanup; do not construct Clause directly.
type Clause struct {
	match   func(interface{}) bool // nil means match anything (Any)
	run     func(interface{})
	cleanup bool // Cleanup clause: always runs, in LIFO order
}

// On matches panics whose value has exactly type E (same semantics as Catch).
// Order matters: the first matching clause handles the panic and the rest
// are skipped. A nil handler yields an inert clause.
func On[E any](handler func(E)) Clause {
	if handler == nil {
		return Clause{}
	}
	return Clause{
		match: func(v interface{}) bool { _, ok := v.(E); return ok },
		run:   func(v interface{}) { handler(v.(E)) },
	}
}

// OnAs matches panics using errors.As semantics: it penetrates
// fmt.Errorf("%w", ...) wrapping chains and accepts both E and *E
// (whichever implements error). This is the clause you want for
// library-defined error types.
func OnAs[E error](handler func(E)) Clause {
	if handler == nil {
		return Clause{}
	}
	return Clause{
		match: func(v interface{}) bool { _, ok := asMatch[E](v); return ok },
		run: func(v interface{}) {
			e, _ := asMatch[E](v)
			handler(e)
		},
	}
}

// Any matches any remaining unhandled panic. Place it last.
// A nil handler yields an inert clause.
func Any(handler func(interface{})) Clause {
	if handler == nil {
		return Clause{}
	}
	return Clause{run: func(v interface{}) { handler(v) }}
}

// Cleanup registers a function that always runs after dispatch, regardless
// of success, handled or unhandled panic — even if a handler panics.
// Multiple cleanups run in reverse registration order (like defer).
func Cleanup(fn func()) Clause {
	return Clause{
		run:     func(interface{}) { fn() },
		cleanup: true,
	}
}

// ============================================
// Run / Run1
// ============================================

// Run executes fn and dispatches any panic to the clauses in declaration
// order. It returns nil when fn succeeded or the panic was handled, and a
// non-nil error for unhandled panics (see PanicError for value wrapping).
//
// Guarantees:
//   - Cleanup clauses always run, LIFO, even when a handler panics;
//   - a panic inside a handler propagates (after cleanups) — same as v1;
//   - the returned error unwraps to the original error on error panics,
//     so errors.Is/errors.As work end-to-end.
//
// A nil fn is a no-op returning nil.
func Run(fn func(), clauses ...Clause) (unhandled error) {
	var cleanups []func(interface{})
	for _, c := range clauses {
		if c.cleanup && c.run != nil {
			cleanups = append(cleanups, c.run)
		}
	}
	if len(cleanups) > 0 {
		defer func() {
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i](nil)
			}
		}()
	}

	defer func() {
		if r := recover(); r != nil {
			unhandled = dispatchClauses(r, clauses)
		}
	}()

	if fn != nil {
		fn()
	}
	return nil
}

// Run1 is Run for functions with a return value. On success it returns the
// result and nil; on an unhandled panic it returns the zero value of T and
// the error. CatchAs clauses receive the typed handler:
//
//	v, err := gotrycatch.Run1(func() int { return compute() },
//		gotrycatch.OnAs(func(e *errtypes.ValidationError) { /* ... */ }),
//	)
func Run1[T any](fn func() T, clauses ...Clause) (T, error) {
	var result T
	err := Run(func() { result = fn() }, clauses...)
	if err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}

// dispatchClauses runs the first matching non-cleanup clause against r.
// Returns nil when handled, or the unhandled panic bridged via toError
// (error panics pass through; others become *PanicError).
func dispatchClauses(r interface{}, clauses []Clause) error {
	for _, c := range clauses {
		if c.cleanup || c.run == nil {
			continue
		}
		if c.match == nil || c.match(r) {
			c.run(r)
			return nil
		}
	}
	return toError(r, callers())
}

// ============================================
// CatchAs - errors.As semantics for the Try/TryWithResult chains
// ============================================

// CatchAs handles panics of error type E using errors.As semantics,
// penetrating fmt.Errorf("%w", ...) wrappers and matching E or *E.
// Use it exactly like Catch; the v1 exact-assertion behavior of Catch
// is unchanged.
func CatchAs[E error](tb *TryBlock, handler func(E)) *TryBlock {
	if tb == nil {
		debugLog("CatchAs: TryBlock is nil, returning empty TryBlock")
		return &TryBlock{}
	}
	if handler == nil {
		debugLog("CatchAs: handler is nil, returning TryBlock unchanged")
		return tb
	}
	if tb.err != nil && !tb.handled {
		if e, ok := asMatch[E](tb.err); ok {
			debugLog("CatchAs: matched %T via errors.As", e)
			handler(e)
			tb.handled = true
		} else {
			var zero E
			debugLog("CatchAs: no %T in error chain of %T", zero, tb.err)
		}
	}
	return tb
}

// CatchAsWithResult is CatchAs for TryBlockWithResult chains.
func CatchAsWithResult[T any, E error](tb *TryBlockWithResult[T], handler func(E)) *TryBlockWithResult[T] {
	if tb == nil {
		debugLog("CatchAsWithResult: TryBlockWithResult is nil")
		return &TryBlockWithResult[T]{}
	}
	if handler == nil {
		debugLog("CatchAsWithResult: handler is nil, returning TryBlockWithResult unchanged")
		return tb
	}
	if tb.err != nil && !tb.handled {
		if e, ok := asMatch[E](tb.err); ok {
			debugLog("CatchAsWithResult: matched %T via errors.As", e)
			handler(e)
			tb.handled = true
		}
	}
	return tb
}
