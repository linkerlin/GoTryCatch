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
		{&struct{ X int }{1}, "struct { X int }"},
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
// PanicError direct construction edge cases
// ============================================

func TestPanicError_WithErrorValue(t *testing.T) {
	// PanicError is exported; users may construct it directly with an error value.
	inner := errors.New("inner cause")
	pe := &PanicError{Value: inner}

	// Error() must surface the inner error's message
	if got := pe.Error(); got != "inner cause" {
		t.Errorf("Expected inner message, got %q", got)
	}

	// Unwrap must return the inner error (errors.Is/As reach through)
	if !errors.Is(pe, inner) {
		t.Error("Expected errors.Is to reach inner error via Unwrap")
	}
	var target *PanicError
	if !errors.As(pe, &target) || target != pe {
		t.Error("Expected errors.As to match the PanicError itself")
	}

	// Zero-value PanicError: no value, no stack, safe to use
	var zero *PanicError
	_ = zero
	empty := &PanicError{}
	if empty.Stack() != nil {
		t.Error("Expected nil stack for empty PanicError")
	}
}

func TestPanicError_NonErrorValue(t *testing.T) {
	pe := &PanicError{Value: "plain string"}
	if got := pe.Error(); got != "panic: plain string" {
		t.Errorf("Expected 'panic: plain string', got %q", got)
	}
	if pe.Unwrap() != nil {
		t.Error("Expected nil Unwrap for non-error value")
	}
}

// ============================================
// dispatch edge cases (via Catch/CatchWithResult)
// ============================================

func TestCatchWithResult_AllNilPaths(t *testing.T) {
	// nil TryBlockWithResult + nil handler: must not panic, returns fresh block
	var nilTb *TryBlockWithResult[int]
	out := CatchWithResult[int, string](nilTb, nil)
	if out == nil {
		t.Error("Expected non-nil TryBlockWithResult for nil input")
	}

	// nil handler with existing error: unchanged
	tb := TryWithResult(func() int { panic("e") })
	if out := CatchWithResult[int, string](tb, nil); out != tb {
		t.Error("Expected same TryBlockWithResult for nil handler")
	}
}

// ============================================
// canonicalType edge cases
// ============================================

func TestCanonicalType_NilAndTypedNil(t *testing.T) {
	// typed nil pointer: CanonicalErrorType must strip to the base name
	tb := Try(func() {
		var db *trycatcherrors.DatabaseError
		panic(db)
	})
	if got := tb.CanonicalErrorType(); got != "DatabaseError" {
		t.Errorf("Expected DatabaseError for typed nil pointer, got %q", got)
	}
}

// ============================================
// TryBlockWithResult state query gaps
// ============================================

func TestTryBlockWithResult_GetErrorType(t *testing.T) {
	tb := TryWithResult(func() int { panic(trycatcherrors.NewValidationError("f", "m", 1)) })
	if got := tb.GetErrorType(); got != "errtypes.ValidationError" {
		t.Errorf("Expected 'errtypes.ValidationError', got %q", got)
	}

	tb2 := TryWithResult(func() int { return 42 })
	if got := tb2.GetErrorType(); got != "" {
		t.Errorf("Expected empty for no error, got %q", got)
	}

	var nilTb *TryBlockWithResult[int]
	if got := nilTb.GetErrorType(); got != "" {
		t.Errorf("Expected empty for nil receiver, got %q", got)
	}
}

func TestTryBlockWithResult_CanonicalErrorType(t *testing.T) {
	tb := TryWithResult(func() int { panic(trycatcherrors.NewValidationError("f", "m", 1)) })
	if got := tb.CanonicalErrorType(); got != "ValidationError" {
		t.Errorf("Expected 'ValidationError', got %q", got)
	}

	tb2 := TryWithResult(func() int { return 42 })
	if got := tb2.CanonicalErrorType(); got != "" {
		t.Errorf("Expected empty for no error, got %q", got)
	}

	var nilTb *TryBlockWithResult[int]
	if got := nilTb.CanonicalErrorType(); got != "" {
		t.Errorf("Expected empty for nil receiver, got %q", got)
	}
}

// ============================================
// Finally nil-combination gaps
// ============================================

func TestFinally_NilTryBlock_ExecutesFn(t *testing.T) {
	var ran bool
	var nilTb *TryBlock
	nilTb.Finally(func() { ran = true })
	if !ran {
		t.Error("Expected fn to run for nil TryBlock")
	}
}

func TestTryBlockWithResult_Finally_NilHandlerNilTb(t *testing.T) {
	var nilTb *TryBlockWithResult[int]
	result := nilTb.Finally(nil)
	if result != 0 {
		t.Errorf("Expected zero value, got %d", result)
	}
}

func TestOrElseGet_NilReceiver(t *testing.T) {
	var nilTb *TryBlockWithResult[string]
	if got := nilTb.OrElseGet(func() string { return "default" }); got != "default" {
		t.Errorf("Expected 'default', got %q", got)
	}
	// nil receiver + nil supplier → zero value
	if got := nilTb.OrElseGet(nil); got != "" {
		t.Errorf("Expected zero value for nil supplier, got %q", got)
	}
}

// ============================================
// CatchAsWithResult unmatched path
// ============================================

func TestCatchAsWithResult_NonMatching(t *testing.T) {
	var handlerCalled bool
	tb := TryWithResult(func() int { panic("plain string") })
	tb = CatchAsWithResult(tb, func(e trycatcherrors.DatabaseError) { handlerCalled = true })
	if handlerCalled {
		t.Error("Handler must not be called for non-matching panic")
	}
	if tb.IsHandled() {
		t.Error("Expected unhandled")
	}
}

// ============================================
// Debug-mode branch coverage
// ============================================

func TestDispatch_DebugMismatchBranch(t *testing.T) {
	original := IsDebug()
	defer SetDebug(original)
	SetDebug(true)

	// 开启调试 + 类型不匹配 → debugLog 的 mismatch 分支（含 *new(T) 求值）
	tb := Try(func() { panic(42) })
	tb = Catch[string](tb, func(err string) {})
	if tb.IsHandled() {
		t.Error("Expected unhandled for type mismatch")
	}
}

func TestCatchAs_DebugAndMismatchBranches(t *testing.T) {
	original := IsDebug()
	defer SetDebug(original)
	SetDebug(true)

	// 未匹配 → else 分支（debug 输出 "no %T in error chain"）
	tb := Try(func() { panic("plain") })
	tb = CatchAs(tb, func(e trycatcherrors.DatabaseError) {})
	if tb.IsHandled() {
		t.Error("Expected unhandled")
	}

	// 匹配 → debug 命中分支
	tb2 := Try(func() { panic(trycatcherrors.NewValidationError("f", "m", 1)) })
	tb2 = CatchAs(tb2, func(e trycatcherrors.ValidationError) {})
	if !tb2.IsHandled() {
		t.Error("Expected handled")
	}

	// CatchAsWithResult：nil handler 分支
	var nilTb *TryBlockWithResult[int]
	if out := CatchAsWithResult[int, error](nilTb, nil); out == nil {
		t.Error("Expected non-nil TryBlockWithResult for nil input + nil handler")
	}

	// CatchAsWithResult：匹配时 debug 命中分支
	tb3 := TryWithResult(func() int { panic(trycatcherrors.NewConfigError("k", "v", "r")) })
	tb3 = CatchAsWithResult(tb3, func(e trycatcherrors.ConfigError) {})
	if !tb3.IsHandled() {
		t.Error("Expected handled")
	}
}

// ============================================
// Clause nil-handler branches + asMatch tail
// ============================================

func TestClauses_NilHandlers_Inert(t *testing.T) {
	// On(nil) / OnAs(nil)：返回空子句（run 为 nil），panic 保持未处理
	err := Run(func() { panic("x") },
		On(func(e string) {}),
		OnAs(func(e error) {}),
		On(func(e int) { t.Error("must not run") }),
	)
	if err != nil {
		t.Errorf("Expected handled by non-nil On, got %v", err)
	}
}

func TestClauses_OnAsNilAndAsTail(t *testing.T) {
	// OnAs 不匹配：asMatch 走 v.(E) 兜底并失败（非 error 值 + 目标类型不匹配）
	err := Run(func() { panic(42) },
		OnAs(func(e *trycatcherrors.DatabaseError) { t.Error("must not match") }),
	)
	var pe *PanicError
	if !errors.As(err, &pe) {
		t.Errorf("Expected unhandled PanicError, got %v", err)
	}
	if pe.Value != 42 {
		t.Errorf("Expected panic value 42 preserved, got %v", pe.Value)
	}
}

// ============================================
// canonicalType nil-type tail
// ============================================

func TestCanonicalType_UnknownTypeName(t *testing.T) {
	// 匿名结构体没有 Name()，canonicalType 回退 t.String()（非空）
	tb := Try(func() { panic(struct{ X int }{1}) })
	if got := tb.CanonicalErrorType(); got != "struct { X int }" {
		t.Errorf("Expected 'struct { X int }', got %q", got)
	}

	// 切片类型：Name() 同样为空，走 String() 回退
	tb2 := Try(func() { panic([]int{1, 2}) })
	if got := tb2.CanonicalErrorType(); got != "[]int" {
		t.Errorf("Expected '[]int', got %q", got)
	}

	// nil interface 的防御分支（reflect.TypeOf(nil) == nil → %T 兜底）
	if got := canonicalType(nil); got != "<nil>" {
		t.Errorf("Expected '<nil>', got %q", got)
	}
}

func TestCatchAsWithResult_NilHandlerNonNilTb(t *testing.T) {
	// tb 非 nil + handler nil → 原样返回（debug 分支）
	tb := TryWithResult(func() int { panic("e") })
	if out := CatchAsWithResult[int, error](tb, nil); out != tb {
		t.Error("Expected same TryBlockWithResult for nil handler")
	}
}

func TestClauses_OnAsErrorAsFail(t *testing.T) {
	// panic 值是 error 但链上不含目标类型 → asMatch 返回 (target, false)
	sentinel := errors.New("plain sentinel")
	var handlerCalled bool
	err := Run(func() { panic(sentinel) },
		OnAs(func(e *trycatcherrors.DatabaseError) { handlerCalled = true }),
	)
	if handlerCalled {
		t.Error("Handler must not be called")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Expected sentinel to flow through, got %v", err)
	}
}

func TestClauses_OnNil_OnAsNil(t *testing.T) {
	// On(nil) / OnAs(nil) 返回空子句：不得崩溃，panic 保持未处理
	err := Run(func() { panic("x") },
		On[string](nil),
		OnAs[error](nil),
	)
	var pe *PanicError
	if !errors.As(err, &pe) {
		t.Errorf("Expected unhandled PanicError, got %v", err)
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
