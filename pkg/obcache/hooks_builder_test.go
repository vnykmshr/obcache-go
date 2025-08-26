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
