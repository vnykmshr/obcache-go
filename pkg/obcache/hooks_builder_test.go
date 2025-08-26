package obcache

import (
	"context"
	"testing"
	"time"
)

func TestHooksBuilder(t *testing.T) {
	// Test empty hooks initially
	hooks := &Hooks{}

	// Test AddOnHit
	hitCalled := false
	hooks.AddOnHit(func(_ string, _ any) {
		hitCalled = true
	})

	if len(hooks.OnHit) != 1 {
		t.Fatal("AddOnHit should add to OnHit slice")
	}

	// Test AddOnMiss
	missCalled := false
	hooks.AddOnMiss(func(_ string) {
		missCalled = true
	})

	if len(hooks.OnMiss) != 1 {
		t.Fatal("AddOnMiss should add to OnMiss slice")
	}

	// Test AddOnEvict
	evictCalled := false
	hooks.AddOnEvict(func(_ string, _ any, _ EvictReason) {
		evictCalled = true
	})

	if len(hooks.OnEvict) != 1 {
		t.Fatal("AddOnEvict should add to OnEvict slice")
	}

	// Test AddOnInvalidate
	invalidateCalled := false
	hooks.AddOnInvalidate(func(_ string) {
		invalidateCalled = true
	})

	if len(hooks.OnInvalidate) != 1 {
		t.Fatal("AddOnInvalidate should add to OnInvalidate slice")
	}

	// Test that hooks work when used with cache
	config := NewDefaultConfig().WithHooks(hooks)
	cache, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test hit hook
	_ = cache.Set("test", "value", time.Hour)
	_, _ = cache.Get("test")

	if !hitCalled {
		t.Fatal("OnHit hook should have been called")
	}

	// Test miss hook
	_, _ = cache.Get("missing")

	if !missCalled {
		t.Fatal("OnMiss hook should have been called")
	}

	// Test invalidate hook
	_ = cache.Invalidate("test")

	if !invalidateCalled {
		t.Fatal("OnInvalidate hook should have been called")
	}

	// Test evict hook (create a small cache to force eviction)
	smallConfig := NewDefaultConfig().WithMaxEntries(1).WithHooks(hooks)
	smallCache, err := New(smallConfig)
	if err != nil {
		t.Fatalf("Failed to create small cache: %v", err)
	}

	_ = smallCache.Set("key1", "value1", time.Hour)
	_ = smallCache.Set("key2", "value2", time.Hour) // Should evict key1

	if !evictCalled {
		t.Fatal("OnEvict hook should have been called")
	}
}

func TestHooksBuilderMultiple(t *testing.T) {
	hooks := &Hooks{}

	// Add multiple hooks of same type
	hitCount := 0
	hooks.AddOnHit(func(_ string, _ any) {
		hitCount++
	})
	hooks.AddOnHit(func(_ string, _ any) {
		hitCount++
	})

	if len(hooks.OnHit) != 2 {
		t.Fatal("Should have 2 OnHit hooks")
	}

	// Test both hooks are called
	config := NewDefaultConfig().WithHooks(hooks)
	cache, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	_ = cache.Set("test", "value", time.Hour)
	_, _ = cache.Get("test")

	if hitCount != 2 {
		t.Fatalf("Expected both hit hooks to be called, got %d calls", hitCount)
	}
}

func TestEvictReasonString(t *testing.T) {
	reasons := []EvictReason{
		EvictReasonLRU,
		EvictReasonTTL,
		EvictReasonCapacity,
		EvictReason(999), // Unknown reason
	}

	for _, reason := range reasons {
		str := reason.String()
		if str == "" {
			t.Fatalf("String() should not return empty string for reason %d", reason)
		}
	}

	// Test specific string values
	if EvictReasonLRU.String() != "LRU" {
		t.Fatalf("Expected 'LRU', got '%s'", EvictReasonLRU.String())
	}

	if EvictReasonTTL.String() != "TTL" {
		t.Fatalf("Expected 'TTL', got '%s'", EvictReasonTTL.String())
	}

	if EvictReasonCapacity.String() != "Capacity" {
		t.Fatalf("Expected 'Capacity', got '%s'", EvictReasonCapacity.String())
	}
}

func TestHooksBuilderChaining(t *testing.T) {
	// Test that we can add multiple hooks of different types
	hooks := &Hooks{}

	hooks.AddOnHit(func(string, any) {})
	hooks.AddOnMiss(func(string) {})
	hooks.AddOnEvict(func(string, any, EvictReason) {})
	hooks.AddOnInvalidate(func(string) {})

	if len(hooks.OnHit) != 1 {
		t.Fatal("Should have 1 OnHit hook after chaining")
	}
	if len(hooks.OnMiss) != 1 {
		t.Fatal("Should have 1 OnMiss hook after chaining")
	}
	if len(hooks.OnEvict) != 1 {
		t.Fatal("Should have 1 OnEvict hook after chaining")
	}
	if len(hooks.OnInvalidate) != 1 {
		t.Fatal("Should have 1 OnInvalidate hook after chaining")
	}
}

func TestHooksBuilderCtx(t *testing.T) {
	hooks := setupContextHooks()
	validateHookCounts(t, hooks)
	testContextHooksWithCache(t, hooks)
}

func setupContextHooks() *Hooks {
	hooks := &Hooks{}

	hooks.AddOnHitCtx(func(ctx context.Context, key string, value any, args []any) {
		// Store in global vars for validation (simplified for test)
		hitCtx = ctx
		hitKey = key
		hitValue = value
		hitArgs = args
	})

	hooks.AddOnMissCtx(func(ctx context.Context, key string, args []any) {
		missCtx = ctx
		missKey = key
		missArgs = args
	})

	hooks.AddOnInvalidateCtx(func(ctx context.Context, key string, args []any) {
		invalidateCtx = ctx
		invalidateKey = key
		invalidateArgs = args
	})

	return hooks
}

func validateHookCounts(t *testing.T, hooks *Hooks) {
	if len(hooks.OnHitCtx) != 1 {
		t.Fatal("AddOnHitCtx should add to OnHitCtx slice")
	}
	if len(hooks.OnMissCtx) != 1 {
		t.Fatal("AddOnMissCtx should add to OnMissCtx slice")
	}
	if len(hooks.OnInvalidateCtx) != 1 {
		t.Fatal("AddOnInvalidateCtx should add to OnInvalidateCtx slice")
	}
}

func testContextHooksWithCache(t *testing.T, hooks *Hooks) {
	cache := createCacheWithHooks(t, hooks)
	testHitHook(t, cache)
	testMissHook(t, cache)
	testInvalidateHook(t, cache)
}

func createCacheWithHooks(t *testing.T, hooks *Hooks) *Cache {
	config := NewDefaultConfig().WithHooks(hooks)
	cache, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	return cache
}

func testHitHook(t *testing.T, cache *Cache) {
	type testKey string
	ctx := context.WithValue(context.Background(), testKey("testKey"), "testValue")
	testArgs := []any{"arg1", "arg2"}

	_ = cache.Set("test", "value", time.Hour)
	_, _ = cache.Get("test", WithContext(ctx), WithArgs(testArgs))

	if hitCtx != ctx {
		t.Fatal("OnHitCtx hook should have received the context")
	}
	if hitKey != "test" {
		t.Fatalf("Expected key 'test', got '%s'", hitKey)
	}
	if hitValue != "value" {
		t.Fatalf("Expected value 'value', got '%v'", hitValue)
	}
	validateArgs(t, hitArgs, []any{"arg1", "arg2"})
}

func testMissHook(t *testing.T, cache *Cache) {
	type testKey string
	ctx := context.WithValue(context.Background(), testKey("missKey"), "missValue")
	testArgs := []any{"miss1", "miss2"}

	_, _ = cache.Get("missing", WithContext(ctx), WithArgs(testArgs))

	if missCtx != ctx {
		t.Fatal("OnMissCtx hook should have received the context")
	}
	if missKey != "missing" {
		t.Fatalf("Expected key 'missing', got '%s'", missKey)
	}
	validateArgs(t, missArgs, []any{"miss1", "miss2"})
}

func testInvalidateHook(t *testing.T, cache *Cache) {
	type testKey string
	ctx := context.WithValue(context.Background(), testKey("invalidateKey"), "invalidateValue")
	testArgs := []any{"inv1"}

	_ = cache.Invalidate("test", WithContext(ctx), WithArgs(testArgs))

	if invalidateCtx != ctx {
		t.Fatal("OnInvalidateCtx hook should have received the context")
	}
	if invalidateKey != "test" {
		t.Fatalf("Expected key 'test', got '%s'", invalidateKey)
	}
	if len(invalidateArgs) != 1 || invalidateArgs[0] != "inv1" {
		t.Fatalf("Expected args [inv1], got %v", invalidateArgs)
	}
}

// validateArgs is a helper function to validate argument slices
func validateArgs(t *testing.T, actual, expected []any) {
	if len(actual) != len(expected) {
		t.Fatalf("Expected args length %d, got %d", len(expected), len(actual))
	}
	for i, expectedArg := range expected {
		if actual[i] != expectedArg {
			t.Fatalf("Expected arg[%d] = %v, got %v", i, expectedArg, actual[i])
		}
	}
}

// Global variables for test validation (simplified approach)
var (
	hitCtx, missCtx, invalidateCtx    context.Context
	hitKey, missKey, invalidateKey    string
	hitValue                          any
	hitArgs, missArgs, invalidateArgs []any
)

func TestHooksPriorityOrdering(t *testing.T) {
	hooks := &Hooks{}
	var executionOrder []string

	// Add hooks with different priorities
	hooks.AddOnHitWithPriority(func(_ string, _ any) {
		executionOrder = append(executionOrder, "low")
	}, HookPriorityLow)

	hooks.AddOnHitWithPriority(func(_ string, _ any) {
		executionOrder = append(executionOrder, "high")
	}, HookPriorityHigh)

	hooks.AddOnHitWithPriority(func(_ string, _ any) {
		executionOrder = append(executionOrder, "medium")
	}, HookPriorityMedium)

	// Test that hooks execute in priority order (high -> medium -> low)
	cache := createCacheWithHooks(t, hooks)
	_ = cache.Set("test", "value", time.Hour)
	_, _ = cache.Get("test")

	if len(executionOrder) != 3 {
		t.Fatalf("Expected 3 hooks to execute, got %d", len(executionOrder))
	}

	expectedOrder := []string{"high", "medium", "low"}
	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Fatalf("Expected execution order %v, got %v", expectedOrder, executionOrder)
		}
	}
}

func TestHooksConditionalExecution(t *testing.T) {
	type debugKey string

	hooks := &Hooks{}
	var metricsExecuted, debugExecuted bool

	// Add conditional hooks
	hooks.AddOnHitCtxIf(func(_ context.Context, _ string, _ any, _ []any) {
		metricsExecuted = true
	}, KeyPrefixCondition("metrics:"))

	hooks.AddOnHitCtxIf(func(_ context.Context, _ string, _ any, _ []any) {
		debugExecuted = true
	}, ContextValueCondition(debugKey("debug"), true))

	cache := createCacheWithHooks(t, hooks)

	// Test 1: Key with metrics prefix should trigger metrics hook
	_ = cache.Set("metrics:counter", "value", time.Hour)
	_, _ = cache.Get("metrics:counter")

	if !metricsExecuted {
		t.Fatal("Metrics hook should have executed for metrics: prefixed key")
	}
	if debugExecuted {
		t.Fatal("Debug hook should not have executed without debug context")
	}

	// Reset
	metricsExecuted, debugExecuted = false, false

	// Test 2: Regular key should not trigger either hook
	_ = cache.Set("regular", "value", time.Hour)
	_, _ = cache.Get("regular")

	if metricsExecuted || debugExecuted {
		t.Fatal("No conditional hooks should execute for regular key")
	}

	// Test 3: Key with debug context should trigger debug hook
	debugCtx := context.WithValue(context.Background(), debugKey("debug"), true)
	_ = cache.Set("test", "value", time.Hour)
	_, _ = cache.Get("test", WithContext(debugCtx))

	if !debugExecuted {
		t.Fatal("Debug hook should have executed with debug context")
	}
}

func TestHookCompositionUtilities(t *testing.T) {
	var calls []string

	// Test CombineOnHitHooks
	hook1 := func(key string, value any) { calls = append(calls, "hook1") }
	hook2 := func(key string, value any) { calls = append(calls, "hook2") }
	combinedHook := CombineOnHitHooks(hook1, hook2)

	combinedHook("test", "value")

	if len(calls) != 2 || calls[0] != "hook1" || calls[1] != "hook2" {
		t.Fatalf("Expected [hook1, hook2], got %v", calls)
	}

	// Test ConditionalHook
	calls = nil
	conditionalHook := ConditionalHook(
		func(ctx context.Context, key string, value any, args []any) {
			calls = append(calls, "conditional")
		},
		KeyPrefixCondition("test:"),
	)

	// Should execute
	conditionalHook(context.Background(), "test:key", "value", nil)
	if len(calls) != 1 || calls[0] != "conditional" {
		t.Fatalf("Conditional hook should have executed for test: prefix")
	}

	// Should not execute
	calls = nil
	conditionalHook(context.Background(), "other", "value", nil)
	if len(calls) != 0 {
		t.Fatal("Conditional hook should not execute for non-matching key")
	}
}

func TestHookConditionBuilders(t *testing.T) {
	// Test KeyPrefixCondition
	prefixCondition := KeyPrefixCondition("api:")
	if !prefixCondition(context.Background(), "api:users", nil) {
		t.Fatal("KeyPrefixCondition should match api: prefix")
	}
	if prefixCondition(context.Background(), "web:users", nil) {
		t.Fatal("KeyPrefixCondition should not match web: prefix")
	}

	// Test ContextValueCondition
	type envKey string
	ctx := context.WithValue(context.Background(), envKey("env"), "prod")
	envCondition := ContextValueCondition(envKey("env"), "prod")
	if !envCondition(ctx, "key", nil) {
		t.Fatal("ContextValueCondition should match prod environment")
	}
	if envCondition(context.Background(), "key", nil) {
		t.Fatal("ContextValueCondition should not match missing context")
	}

	// Test AndCondition
	andCondition := AndCondition(
		KeyPrefixCondition("api:"),
		ContextValueCondition(envKey("env"), "prod"),
	)
	if !andCondition(ctx, "api:users", nil) {
		t.Fatal("AndCondition should match when both conditions are true")
	}
	if andCondition(ctx, "web:users", nil) {
		t.Fatal("AndCondition should not match when first condition fails")
	}

	// Test OrCondition
	orCondition := OrCondition(
		KeyPrefixCondition("api:"),
		KeyPrefixCondition("web:"),
	)
	if !orCondition(context.Background(), "api:users", nil) {
		t.Fatal("OrCondition should match api: prefix")
	}
	if !orCondition(context.Background(), "web:users", nil) {
		t.Fatal("OrCondition should match web: prefix")
	}
	if orCondition(context.Background(), "db:users", nil) {
		t.Fatal("OrCondition should not match db: prefix")
	}
}

func TestHooksBackwardCompatibility(t *testing.T) {
	// Ensure that existing hook usage continues to work unchanged
	hooks := &Hooks{}
	var legacyCalled, ctxCalled bool

	// Add legacy hooks
	hooks.AddOnHit(func(key string, value any) {
		legacyCalled = true
	})

	// Add context-aware hooks
	hooks.AddOnHitCtx(func(ctx context.Context, key string, value any, args []any) {
		ctxCalled = true
	})

	cache := createCacheWithHooks(t, hooks)
	_ = cache.Set("test", "value", time.Hour)
	_, _ = cache.Get("test")

	if !legacyCalled {
		t.Fatal("Legacy hook should still execute")
	}
	if !ctxCalled {
		t.Fatal("Context-aware hook should still execute")
	}
}

func TestHooksExecutionOrder(t *testing.T) {
	hooks := &Hooks{}
	var executionOrder []string

	// Add hooks of different types to verify execution order
	hooks.AddOnHit(func(key string, value any) {
		executionOrder = append(executionOrder, "legacy")
	})

	hooks.AddOnHitCtx(func(ctx context.Context, key string, value any, args []any) {
		executionOrder = append(executionOrder, "context")
	})

	hooks.AddOnHitWithPriority(func(key string, value any) {
		executionOrder = append(executionOrder, "priority")
	}, HookPriorityHigh)

	hooks.AddOnHitCtxIf(func(ctx context.Context, key string, value any, args []any) {
		executionOrder = append(executionOrder, "conditional")
	}, func(ctx context.Context, key string, args []any) bool { return true })

	cache := createCacheWithHooks(t, hooks)
	_ = cache.Set("test", "value", time.Hour)
	_, _ = cache.Get("test")

	// Expected order: legacy -> context -> priority -> conditional
	expectedOrder := []string{"legacy", "context", "priority", "conditional"}
	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d hooks to execute, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Fatalf("Expected execution order %v, got %v", expectedOrder, executionOrder)
		}
	}
}
