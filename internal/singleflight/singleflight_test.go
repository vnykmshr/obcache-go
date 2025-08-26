package singleflight

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSingleflightBasic(t *testing.T) {
	g := &Group[string, int]{}

	callCount := int32(0)
	fn := func() (int, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(10 * time.Millisecond)
		return 42, nil
	}

	// Single call
	v, err, shared := g.Do("key1", fn)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if v != 42 {
		t.Fatalf("Expected 42, got %d", v)
	}
	if shared {
		t.Fatal("Expected shared to be false for single call")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once, got %d", callCount)
	}
}

func TestSingleflightDeduplication(t *testing.T) {
	g := &Group[string, int]{}

	callCount := int32(0)
	fn := func() (int, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(50 * time.Millisecond) // Slow operation
		return 123, nil
	}

	// Launch multiple concurrent calls
	var wg sync.WaitGroup
	results := make([]int, 10)
	errors := make([]error, 10)
	shared := make([]bool, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errors[idx], shared[idx] = g.Do("same-key", fn)
		}(i)
	}

	wg.Wait()

	// All results should be the same
	for i := 0; i < 10; i++ {
		if errors[i] != nil {
			t.Fatalf("Result %d: unexpected error %v", i, errors[i])
		}
		if results[i] != 123 {
			t.Fatalf("Result %d: expected 123, got %d", i, results[i])
		}
	}

	// Function should only be called once
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("Expected function to be called once, got %d", callCount)
	}

	// At least some calls should be marked as shared
	sharedCount := 0
	for _, isShared := range shared {
		if isShared {
			sharedCount++
		}
	}
	if sharedCount == 0 {
		t.Fatal("Expected some calls to be marked as shared")
	}
}

func TestSingleflightError(t *testing.T) {
	g := &Group[string, int]{}

	testError := errors.New("test error")
	fn := func() (int, error) {
		return 0, testError
	}

	v, err, shared := g.Do("error-key", fn)
	if err != testError {
		t.Fatalf("Expected test error, got %v", err)
	}
	if v != 0 {
		t.Fatalf("Expected 0 for error case, got %d", v)
	}
	if shared {
		t.Fatal("Expected shared to be false for single call")
	}
}

func TestSingleflightDifferentKeys(t *testing.T) {
	g := &Group[string, int]{}

	callCount := int32(0)
	fn1 := func() (int, error) {
		atomic.AddInt32(&callCount, 1)
		return 10, nil
	}
	fn2 := func() (int, error) {
		atomic.AddInt32(&callCount, 1)
		return 20, nil
	}

	// Test sequential calls with different keys
	// Singleflight only deduplicates concurrent calls, not sequential ones
	result1, _, _ := g.Do("key1", fn1)
	if result1 != 10 {
		t.Fatalf("Expected first call to return 10, got %d", result1)
	}

	result2, _, _ := g.Do("key2", fn2)
	if result2 != 20 {
		t.Fatalf("Expected second call to return 20, got %d", result2)
	}

	// Each call to a different key should execute its function
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("Expected function to be called twice, got %d", callCount)
	}
}

func TestSingleflightForget(t *testing.T) {
	g := &Group[string, int]{}

	callCount := int32(0)
	fn := func() (int, error) {
		atomic.AddInt32(&callCount, 1)
		return 42, nil
	}

	// First call
	v1, err1, _ := g.Do("forget-key", fn)
	if err1 != nil || v1 != 42 {
		t.Fatalf("First call failed: %v, %d", err1, v1)
	}

	// Forget the key
	g.Forget("forget-key")

	// Second call should execute function again
	v2, err2, _ := g.Do("forget-key", fn)
	if err2 != nil || v2 != 42 {
		t.Fatalf("Second call failed: %v, %d", err2, v2)
	}

	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("Expected function to be called twice, got %d", callCount)
	}
}

func TestSingleflightDoChan(t *testing.T) {
	g := &Group[string, int]{}

	fn := func() (int, error) {
		time.Sleep(20 * time.Millisecond)
		return 99, nil
	}

	// Test DoChan
	ch := g.DoChan("chan-key", fn)

	select {
	case result := <-ch:
		if result.Err != nil {
			t.Fatalf("Expected no error, got %v", result.Err)
		}
		if result.Val != 99 {
			t.Fatalf("Expected 99, got %d", result.Val)
		}
		if result.Shared {
			t.Fatal("Expected shared to be false for single call")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DoChan timed out")
	}
}

func TestSingleflightDoContext(t *testing.T) {
	g := &Group[string, int]{}

	fn := func() (int, error) {
		time.Sleep(100 * time.Millisecond) // Long operation
		return 42, nil
	}

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	v, err, shared := g.DoContext(ctx, "context-key", fn)
	if err != context.DeadlineExceeded {
		t.Fatalf("Expected context deadline exceeded, got %v", err)
	}
	if v != 0 {
		t.Fatalf("Expected 0 for cancelled context, got %d", v)
	}
	if shared {
		t.Fatal("Expected shared to be false for cancelled call")
	}
}

func TestSingleflightDoContextSuccess(t *testing.T) {
	g := &Group[string, int]{}

	fn := func() (int, error) {
		return 42, nil
	}

	// Test successful context call
	ctx := context.Background()
	v, err, shared := g.DoContext(ctx, "context-success-key", fn)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if v != 42 {
		t.Fatalf("Expected 42, got %d", v)
	}
	if shared {
		t.Fatal("Expected shared to be false for single call")
	}
}

func TestSingleflightInFlight(t *testing.T) {
	g := &Group[string, int]{}

	// Check initial in-flight count
	if count := g.InFlight(); count != 0 {
		t.Fatalf("Expected 0 in-flight calls initially, got %d", count)
	}

	fn := func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 42, nil
	}

	// Start a slow operation
	done := make(chan bool)
	go func() {
		g.Do("inflight-key", fn)
		done <- true
	}()

	// Check in-flight count during operation
	time.Sleep(10 * time.Millisecond) // Let the operation start
	if count := g.InFlight(); count != 1 {
		t.Fatalf("Expected 1 in-flight call, got %d", count)
	}

	// Wait for completion
	<-done

	// Check in-flight count after completion
	if count := g.InFlight(); count != 0 {
		t.Fatalf("Expected 0 in-flight calls after completion, got %d", count)
	}
}

func TestSingleflightConcurrentDifferentTypes(t *testing.T) {
	// Test with different key and value types
	gInt := &Group[int, string]{}
	gString := &Group[string, int]{}

	// Test int keys, string values
	v1, err1, _ := gInt.Do(123, func() (string, error) {
		return "test", nil
	})
	if err1 != nil || v1 != "test" {
		t.Fatalf("Int key group failed: %v, %s", err1, v1)
	}

	// Test string keys, int values
	v2, err2, _ := gString.Do("test", func() (int, error) {
		return 456, nil
	})
	if err2 != nil || v2 != 456 {
		t.Fatalf("String key group failed: %v, %d", err2, v2)
	}
}
