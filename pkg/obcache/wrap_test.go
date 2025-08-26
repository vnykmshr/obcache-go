package obcache

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWrapSimpleFunction(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	callCount := int32(0)
	expensiveFunc := func(x int) int {
		atomic.AddInt32(&callCount, 1)
		return x * 2
	}

	wrapped := Wrap(cache, expensiveFunc)

	// First call should execute function
	result1 := wrapped(5)
	if result1 != 10 {
		t.Fatalf("Expected 10, got %d", result1)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once, got %d", callCount)
	}

	// Second call with same arg should use cache
	result2 := wrapped(5)
	if result2 != 10 {
		t.Fatalf("Expected 10, got %d", result2)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to still be called once, got %d", callCount)
	}

	// Different arg should call function again
	result3 := wrapped(7)
	if result3 != 14 {
		t.Fatalf("Expected 14, got %d", result3)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("Expected function to be called twice, got %d", callCount)
	}
}

func TestWrapFunctionWithError(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	callCount := int32(0)
	funcWithError := func(x int) (int, error) {
		atomic.AddInt32(&callCount, 1)
		if x < 0 {
			return 0, errors.New("negative input")
		}
		return x * 3, nil
	}

	wrapped := Wrap(cache, funcWithError)

	// Test successful call
	result1, err1 := wrapped(5)
	if err1 != nil {
		t.Fatalf("Expected no error, got %v", err1)
	}
	if result1 != 15 {
		t.Fatalf("Expected 15, got %d", result1)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once, got %d", callCount)
	}

	// Second call should use cache
	result2, err2 := wrapped(5)
	if err2 != nil {
		t.Fatalf("Expected no error, got %v", err2)
	}
	if result2 != 15 {
		t.Fatalf("Expected 15, got %d", result2)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to still be called once, got %d", callCount)
	}

	// Test error case - should not be cached
	result3, err3 := wrapped(-1)
	if err3 == nil {
		t.Fatal("Expected error for negative input")
	}
	if result3 != 0 {
		t.Fatalf("Expected 0 for error case, got %d", result3)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("Expected function to be called twice, got %d", callCount)
	}

	// Same error input should call function again (not cached)
	_, err4 := wrapped(-1)
	if err4 == nil {
		t.Fatal("Expected error for negative input")
	}
	if atomic.LoadInt32(&callCount) != 3 {
		t.Fatalf("Expected function to be called three times, got %d", callCount)
	}
}

func TestWrapWithTTL(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	callCount := int32(0)
	expensiveFunc := func(x int) int {
		atomic.AddInt32(&callCount, 1)
		return x * 2
	}

	shortTTL := 10 * time.Millisecond
	wrapped := Wrap(cache, expensiveFunc, WithTTL(shortTTL))

	// First call
	result1 := wrapped(5)
	if result1 != 10 {
		t.Fatalf("Expected 10, got %d", result1)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once, got %d", callCount)
	}

	// Wait for TTL to expire
	time.Sleep(shortTTL + 5*time.Millisecond)

	// Should call function again after TTL expires
	result2 := wrapped(5)
	if result2 != 10 {
		t.Fatalf("Expected 10, got %d", result2)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("Expected function to be called twice after TTL, got %d", callCount)
	}
}

func TestWrapWithCustomKeyFunc(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	callCount := int32(0)
	expensiveFunc := func(x, y int) int {
		atomic.AddInt32(&callCount, 1)
		return x + y
	}

	// Custom key function that ignores the second parameter
	customKeyFunc := func(args []any) string {
		return fmt.Sprintf("key-%v", args[0])
	}

	wrapped := Wrap(cache, expensiveFunc, WithKeyFunc(customKeyFunc))

	// First call
	result1 := wrapped(1, 2)
	if result1 != 3 {
		t.Fatalf("Expected 3, got %d", result1)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once, got %d", callCount)
	}

	// Second call with different y but same x should use cache (due to custom key func)
	result2 := wrapped(1, 5)
	if result2 != 3 { // Should return cached value, not 1+5=6
		t.Fatalf("Expected 3 (cached), got %d", result2)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to still be called once, got %d", callCount)
	}

	// Different x should call function again
	result3 := wrapped(2, 2)
	if result3 != 4 {
		t.Fatalf("Expected 4, got %d", result3)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("Expected function to be called twice, got %d", callCount)
	}
}

func TestWrapWithoutCache(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	callCount := int32(0)
	expensiveFunc := func(x int) int {
		atomic.AddInt32(&callCount, 1)
		return x * 2
	}

	wrapped := Wrap(cache, expensiveFunc, WithoutCache())

	// Multiple calls should always execute function
	result1 := wrapped(5)
	if result1 != 10 {
		t.Fatalf("Expected 10, got %d", result1)
	}

	result2 := wrapped(5)
	if result2 != 10 {
		t.Fatalf("Expected 10, got %d", result2)
	}

	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("Expected function to be called twice (no caching), got %d", callCount)
	}
}

func TestWrapSingleflight(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	callCount := int32(0)
	slowFunc := func(x int) int {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(50 * time.Millisecond) // Simulate slow operation
		return x * 2
	}

	wrapped := Wrap(cache, slowFunc)

	// Launch multiple concurrent calls with same argument
	var wg sync.WaitGroup
	results := make([]int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = wrapped(5)
		}(i)
	}

	wg.Wait()

	// All results should be the same
	for i, result := range results {
		if result != 10 {
			t.Fatalf("Result %d: expected 10, got %d", i, result)
		}
	}

	// Function should only be called once due to singleflight
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once (singleflight), got %d", callCount)
	}
}

func TestWrapMultipleReturnValues(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	callCount := int32(0)
	multiReturnFunc := func(x int) (int, string, error) {
		atomic.AddInt32(&callCount, 1)
		if x < 0 {
			return 0, "", errors.New("negative input")
		}
		return x * 2, fmt.Sprintf("result-%d", x*2), nil
	}

	wrapped := Wrap(cache, multiReturnFunc)

	// First call
	val1, str1, err1 := wrapped(5)
	if err1 != nil {
		t.Fatalf("Expected no error, got %v", err1)
	}
	if val1 != 10 {
		t.Fatalf("Expected 10, got %d", val1)
	}
	if str1 != "result-10" {
		t.Fatalf("Expected 'result-10', got %s", str1)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once, got %d", callCount)
	}

	// Second call should use cache
	val2, str2, err2 := wrapped(5)
	if err2 != nil {
		t.Fatalf("Expected no error, got %v", err2)
	}
	if val2 != 10 {
		t.Fatalf("Expected 10, got %d", val2)
	}
	if str2 != "result-10" {
		t.Fatalf("Expected 'result-10', got %s", str2)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to still be called once, got %d", callCount)
	}
}

func TestWrapValidation(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test wrapping non-function should panic
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic when wrapping non-function")
		}
	}()

	Wrap(cache, "not a function")
}

func TestValidateWrappableFunction(t *testing.T) {
	// Valid functions
	validFuncs := []any{
		func() int { return 42 },
		func(x int) int { return x * 2 },
		func(x, y int) (int, error) { return x + y, nil },
		func() (string, error) { return "test", nil },
	}

	for i, fn := range validFuncs {
		if err := ValidateWrappableFunction(fn); err != nil {
			t.Fatalf("Function %d should be valid: %v", i, err)
		}
	}

	// Invalid cases
	invalidCases := []struct {
		fn   any
		desc string
	}{
		{"not a function", "non-function"},
		{func() {}, "no return values"},
		{func(x int, y ...string) int { return x }, "variadic function"},
		{func() (int, string) { return 1, "test" }, "multiple returns without error"},
	}

	for _, tc := range invalidCases {
		if err := ValidateWrappableFunction(tc.fn); err == nil {
			t.Fatalf("Expected error for %s", tc.desc)
		}
	}
}

func TestWrapConvenienceFunctions(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test WrapSimple
	simpleFunc := func(x int) int { return x * 2 }
	wrappedSimple := WrapSimple(cache, simpleFunc)
	if result := wrappedSimple(5); result != 10 {
		t.Fatalf("WrapSimple: expected 10, got %d", result)
	}

	// Test WrapWithError with different argument to avoid cache collision
	errorFunc := func(x int) (int, error) { return x * 3, nil }
	wrappedError := WrapWithError(cache, errorFunc)
	result, err := wrappedError(7) // Use different argument
	if err != nil {
		t.Fatalf("WrapWithError: unexpected error %v", err)
	}
	if result != 21 { // 7 * 3 = 21
		t.Fatalf("WrapWithError: expected 21, got %d", result)
	}

	// Test specific arity functions
	func0 := func() int { return 42 }
	wrapped0 := WrapFunc0(cache, func0)
	if result := wrapped0(); result != 42 {
		t.Fatalf("WrapFunc0: expected 42, got %d", result)
	}

	func1 := func(x int) int { return x }
	wrapped1 := WrapFunc1(cache, func1)
	if result := wrapped1(8); result != 8 { // Use different argument
		t.Fatalf("WrapFunc1: expected 8, got %d", result)
	}

	func2 := func(x, y int) int { return x + y }
	wrapped2 := WrapFunc2(cache, func2)
	if result := wrapped2(3, 4); result != 7 {
		t.Fatalf("WrapFunc2: expected 7, got %d", result)
	}
}
