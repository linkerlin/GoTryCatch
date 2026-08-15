package gotrycatch

import (
	"errors"
	"testing"
)

var benchErr = errors.New("bench error")

// 原生 recover 基线：快乐路径
func BenchmarkRawRecover_NoPanic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		func() {
			defer func() { _ = recover() }()
			benchWork()
		}()
	}
}

// 原生 recover 基线：panic 路径
func BenchmarkRawRecover_WithPanic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		func() {
			defer func() { _ = recover() }()
			panic(benchErr)
		}()
	}
}

// error 返回值基线（Go 惯用对照）
func BenchmarkErrorReturn(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := benchWorkErr(); err != nil {
			b.Fatal(err)
		}
	}
}

// 本库：Try 快乐路径
func BenchmarkTry_NoPanic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tb := Try(benchWork)
		if tb.HasError() {
			b.Fatal("unexpected error")
		}
	}
}

// 本库：Try + panic + Catch 命中
func BenchmarkTry_WithPanicAndCatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tb := Try(func() { panic("bench error") })
		tb = Catch[string](tb, func(err string) {})
		_ = tb
	}
}

// 本库：TryWithResult 快乐路径 + OrElse
func BenchmarkTryWithResult(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tb := TryWithResult(func() int { return 42 })
		_ = tb.OrElse(0)
	}
}

// v2：Run 快乐路径（子句构造在循环外，不计入）
func BenchmarkRun_NoPanic(b *testing.B) {
	cleanup := Cleanup(func() {})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := Run(func() {}, cleanup); err != nil {
			b.Fatal(err)
		}
	}
}

// v2：Run + panic + On 命中
func BenchmarkRun_WithPanicAndOn(b *testing.B) {
	clause := On(func(e string) {})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := Run(func() { panic("bench error") }, clause); err != nil {
			b.Fatal(err)
		}
	}
}

// v2：Run 未处理 panic → error（含 PanicError 堆栈捕获）
func BenchmarkRun_UnhandledPanic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := Run(func() { panic("bench error") }); err == nil {
			b.Fatal("expected error")
		}
	}
}

// v2：Run1 快乐路径
func BenchmarkRun1_NoPanic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Run1(func() int { return 42 }); err != nil {
			b.Fatal(err)
		}
	}
}

func benchWork() {}

func benchWorkErr() (int, error) {
	return 42, nil
}
