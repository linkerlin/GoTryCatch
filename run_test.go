package gotrycatch

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	trycatcherrors "github.com/linkerlin/gotrycatch/errors"
)

// ============================================
// Run basics
// ============================================

func TestRun_NoPanic(t *testing.T) {
	var ran bool
	err := Run(func() { ran = true })
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !ran {
		t.Error("Expected fn to run")
	}
}

func TestRun_NilFn(t *testing.T) {
	if err := Run(nil); err != nil {
		t.Errorf("Expected nil error for nil fn, got %v", err)
	}
}

func TestRun_OnExactMatch(t *testing.T) {
	var caught string
	err := Run(func() { panic("boom") },
		On(func(e string) { caught = e }),
	)
	if err != nil {
		t.Errorf("Expected handled (nil error), got %v", err)
	}
	if caught != "boom" {
		t.Errorf("Expected 'boom', got %q", caught)
	}
}

func TestRun_FirstMatchWins(t *testing.T) {
	var order []string
	err := Run(func() { panic("x") },
		On(func(e int) { order = append(order, "int") }),
		On(func(e string) { order = append(order, "string1") }),
		On(func(e string) { order = append(order, "string2") }),
		Any(func(v interface{}) { order = append(order, "any") }),
	)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(order) != 1 || order[0] != "string1" {
		t.Errorf("Expected only first string clause to run, got %v", order)
	}
}

func TestRun_AnyFallback(t *testing.T) {
	var anyCalled bool
	err := Run(func() { panic(42) },
		On(func(e string) { t.Error("string clause must not run") }),
		Any(func(v interface{}) { anyCalled = true }),
	)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !anyCalled {
		t.Error("Expected Any fallback to run")
	}
}

// ============================================
// OnAs: errors.As penetration
// ============================================

func TestRun_OnAs_PenetratesWrapping(t *testing.T) {
	var caught trycatcherrors.DatabaseError
	err := Run(func() {
		panic(fmt.Errorf("query failed: %w", trycatcherrors.NewDatabaseError("SELECT", "users", errors.New("conn refused"))))
	},
		OnAs(func(e trycatcherrors.DatabaseError) { caught = e }),
	)
	if err != nil {
		t.Errorf("Expected handled, got %v", err)
	}
	if caught.Table != "users" {
		t.Errorf("Expected DatabaseError to be extracted, got %+v", caught)
	}
}

func TestRun_OnAs_PointerForm(t *testing.T) {
	// pointer panics match pointer clauses — E and *E are distinct As targets
	dbErr := &trycatcherrors.DatabaseError{Operation: "DELETE", Table: "rows"}
	var caught *trycatcherrors.DatabaseError
	err := Run(func() { panic(fmt.Errorf("wrap: %w", dbErr)) },
		OnAs(func(e *trycatcherrors.DatabaseError) { caught = e }),
	)
	if err != nil {
		t.Errorf("Expected handled, got %v", err)
	}
	if caught != dbErr {
		t.Errorf("Expected same pointer, got %p", caught)
	}
}

func TestRun_OnAs_ValuePanicDirect(t *testing.T) {
	// errtypes constructors return value types; OnAs must accept them without wrapping
	var caught trycatcherrors.ValidationError
	err := Run(func() {
		panic(trycatcherrors.NewValidationError("email", "bad", 1))
	},
		OnAs(func(e trycatcherrors.ValidationError) { caught = e }),
	)
	if err != nil {
		t.Errorf("Expected handled, got %v", err)
	}
	if caught.Field != "email" {
		t.Errorf("Expected field 'email', got %q", caught.Field)
	}
}

func TestRun_OnAs_DoesNotMatchUnrelated(t *testing.T) {
	var anyCalled bool
	err := Run(func() { panic("plain string") },
		OnAs(func(e *trycatcherrors.DatabaseError) { t.Error("must not match") }),
		Any(func(v interface{}) { anyCalled = true }),
	)
	if err != nil {
		t.Errorf("Expected Any to handle, got %v", err)
	}
	if !anyCalled {
		t.Error("Expected Any fallback")
	}
}

// ============================================
// Unhandled panics → error
// ============================================

func TestRun_Unhandled_ReturnsError(t *testing.T) {
	err := Run(func() { panic("nobody catches me") })
	if err == nil {
		t.Fatal("Expected non-nil error")
	}
	var pe *PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("Expected *PanicError, got %T", err)
	}
	if pe.Value != "nobody catches me" {
		t.Errorf("Expected Value preserved, got %v", pe.Value)
	}
	if got := err.Error(); got != "panic: nobody catches me" {
		t.Errorf("Unexpected Error(): %q", got)
	}
}

func TestRun_Unhandled_ErrorPanic_Unwraps(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := Run(func() { panic(fmt.Errorf("wrap1: %w", fmt.Errorf("wrap2: %w", sentinel))) })
	if !errors.Is(err, sentinel) {
		t.Errorf("Expected errors.Is to reach sentinel, got %v", err)
	}
	// error panics pass through unwrapped
	if _, isPanicErr := err.(*PanicError); isPanicErr {
		t.Error("error panics should pass through unwrapped")
	}
}

func TestPanicError_Stack(t *testing.T) {
	err := Run(func() { panic("with stack") })
	var pe *PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("Expected *PanicError, got %T", err)
	}
	stack := pe.Stack()
	if len(stack) == 0 {
		t.Error("Expected non-empty stack")
	}
	if !strings.Contains(stack[0], "run_test.go") {
		t.Errorf("Expected top frame in this test file, got %q", stack[0])
	}
	// rebuilt per call, no cached mutation
	if again := pe.Stack(); len(again) != len(stack) {
		t.Error("Stack() should be deterministic per call")
	}
}

// ============================================
// Cleanup semantics
// ============================================

func TestRun_Cleanup_AlwaysRuns(t *testing.T) {
	steps := map[string]int{}
	record := func(s string) Clause { return Cleanup(func() { steps[s]++ }) }

	// success
	Run(func() {}, record("success"))
	// handled
	Run(func() { panic("x") }, On(func(e string) {}), record("handled"))
	// unhandled
	Run(func() { panic("x") }, record("unhandled"))

	for k, v := range steps {
		if v != 1 {
			t.Errorf("Cleanup %q ran %d times, want 1", k, v)
		}
	}
}

func TestRun_Cleanup_ReverseOrder(t *testing.T) {
	var order []string
	Run(func() { panic("x") },
		Cleanup(func() { order = append(order, "first") }),
		Cleanup(func() { order = append(order, "second") }),
	)
	if len(order) != 2 || order[0] != "second" || order[1] != "first" {
		t.Errorf("Expected LIFO order [second first], got %v", order)
	}
}

func TestRun_HandlerPanic_StillCleansUp(t *testing.T) {
	var cleaned bool
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected handler panic to propagate")
		} else if r != "handler panic" {
			t.Errorf("Expected 'handler panic', got %v", r)
		}
		if !cleaned {
			t.Error("Expected Cleanup to run despite handler panic")
		}
	}()

	Run(func() { panic("original") },
		On(func(e string) { panic("handler panic") }),
		Cleanup(func() { cleaned = true }),
	)
}

func TestRun_NilHandlers_AreInert(t *testing.T) {
	// On(nil) and Any(nil) must not match (inert clauses), panic stays unhandled
	err := Run(func() { panic("x") },
		On(func(e string) {}),
	)
	_ = err // just must not hang or blow up
	err = Run(func() { panic("x") }, Any(nil))
	if err == nil {
		t.Error("Any(nil) should be inert, leaving panic unhandled")
	}
}

// ============================================
// Run1
// ============================================

func TestRun1_Success(t *testing.T) {
	v, err := Run1(func() int { return 42 })
	if err != nil || v != 42 {
		t.Errorf("Expected (42, nil), got (%d, %v)", v, err)
	}
}

func TestRun1_Unhandled(t *testing.T) {
	v, err := Run1(func() int { panic("boom") })
	if err == nil {
		t.Error("Expected error")
	}
	if v != 0 {
		t.Errorf("Expected zero value, got %d", v)
	}
}

func TestRun1_Handled(t *testing.T) {
	var caught string
	v, err := Run1(func() int { panic("boom") },
		On(func(e string) { caught = e }),
	)
	if err != nil {
		t.Errorf("Expected nil error when handled, got %v", err)
	}
	if caught != "boom" {
		t.Errorf("Expected handler to run, got %q", caught)
	}
	if v != 0 {
		t.Errorf("Expected zero value on panic, got %d", v)
	}
}

// ============================================
// Err() bridge on TryBlock / TryBlockWithResult
// ============================================

func TestTryBlock_Err(t *testing.T) {
	// no error
	if err := Try(func() {}).Err(); err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	// nil receiver
	var nilTb *TryBlock
	if err := nilTb.Err(); err != nil {
		t.Errorf("Expected nil for nil receiver, got %v", err)
	}

	// error panic passes through as-is
	orig := errors.New("original")
	err := Try(func() { panic(orig) }).Err()
	if err != orig {
		t.Errorf("Expected original error, got %v", err)
	}

	// non-error panic wrapped
	err = Try(func() { panic("str") }).Err()
	var pe *PanicError
	if !errors.As(err, &pe) || pe.Value != "str" {
		t.Errorf("Expected *PanicError{str}, got %v", err)
	}
}

func TestTryBlockWithResult_Err(t *testing.T) {
	if err := TryWithResult(func() int { return 1 }).Err(); err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	orig := errors.New("orig")
	if err := TryWithResult(func() int { panic(orig) }).Err(); err != orig {
		t.Errorf("Expected original error, got %v", err)
	}
	var nilTb *TryBlockWithResult[int]
	if err := nilTb.Err(); err != nil {
		t.Errorf("Expected nil for nil receiver, got %v", err)
	}
}

// ============================================
// CatchAs on the classic chains
// ============================================

func TestCatchAs_WrappedPanic(t *testing.T) {
	inner := trycatcherrors.NewDatabaseError("INSERT", "logs", errors.New("disk full"))
	tb := Try(func() { panic(fmt.Errorf("job failed: %w", inner)) })

	var caught trycatcherrors.DatabaseError
	tb = CatchAs(tb, func(e trycatcherrors.DatabaseError) { caught = e })

	if !tb.IsHandled() {
		t.Error("Expected handled")
	}
	if caught.Table != "logs" {
		t.Errorf("Expected DatabaseError extracted, got %+v", caught)
	}
}

func TestCatchAs_ExactSemanticsUnchangedForCatch(t *testing.T) {
	inner := trycatcherrors.NewValidationError("f", "m", 7)
	tb := Try(func() { panic(fmt.Errorf("wrapped: %w", inner)) })

	// plain Catch must NOT match through the wrapper (v1 semantics preserved)
	tb = Catch[trycatcherrors.ValidationError](tb, func(e trycatcherrors.ValidationError) {
		t.Error("plain Catch must not penetrate wrappers")
	})
	if tb.IsHandled() {
		t.Error("Expected unhandled")
	}
}

func TestCatchAs_NilSafety(t *testing.T) {
	var nilTb *TryBlock
	out := CatchAs(nilTb, func(e *trycatcherrors.DatabaseError) {})
	if out == nil {
		t.Error("Expected non-nil TryBlock for nil input")
	}

	tb := Try(func() { panic("x") })
	if out := CatchAs[*trycatcherrors.DatabaseError](tb, nil); out != tb {
		t.Error("Expected same TryBlock for nil handler")
	}
}

func TestCatchAsWithResult_WrappedPanic(t *testing.T) {
	inner := trycatcherrors.NewRateLimitError("api", 1, 2, 3)
	tb := TryWithResult(func() string { panic(fmt.Errorf("call failed: %w", inner)) })

	var retryAfter int
	tb = CatchAsWithResult(tb, func(e trycatcherrors.RateLimitError) { retryAfter = e.RetryAfter })

	if !tb.IsHandled() {
		t.Error("Expected handled")
	}
	if retryAfter != 3 {
		t.Errorf("Expected retryAfter 3, got %d", retryAfter)
	}
}

// ============================================
// CanonicalErrorType
// ============================================

func TestCanonicalErrorType(t *testing.T) {
	cases := []struct {
		panicv interface{}
		want   string
	}{
		{trycatcherrors.NewValidationError("f", "m", 1), "ValidationError"},
		{&struct{ X int }{1}, ""},
		{"plain", "string"},
		{42, "int"},
	}
	for i, c := range cases {
		tb := Try(func() { panic(c.panicv) })
		if got := tb.CanonicalErrorType(); got != c.want {
			t.Errorf("case %d: got %q, want %q", i, got, c.want)
		}
	}
	if got := Try(func() {}).CanonicalErrorType(); got != "" {
		t.Errorf("Expected empty for no error, got %q", got)
	}
	var nilTb *TryBlock
	if got := nilTb.CanonicalErrorType(); got != "" {
		t.Errorf("Expected empty for nil receiver, got %q", got)
	}
}

// ============================================
// Nesting & concurrency
// ============================================

func TestRun_Nested(t *testing.T) {
	var innerRan, outerCaught bool
	err := Run(func() {
		if e := Run(func() { panic("inner") }); e != nil {
			innerRan = true
			panic(e)
		}
	},
		On(func(e *PanicError) { outerCaught = true }),
	)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if !innerRan || !outerCaught {
		t.Errorf("innerRan=%v outerCaught=%v", innerRan, outerCaught)
	}
}

func TestRun_Concurrent(t *testing.T) {
	const goroutines = 10
	const iterations = 100
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch i % 3 {
				case 0:
					_ = Run(func() {})
				case 1:
					_ = Run(func() { panic("x") }, On(func(e string) {}))
				default:
					_ = Run(func() { panic(i) }, Cleanup(func() {}))
				}
			}
		}(g)
	}
	wg.Wait()
}
