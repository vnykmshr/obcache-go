package obcache

import (
	"testing"
	"time"
)

func TestHooksBuilder(t *testing.T) {
	// Test empty hooks initially
	hooks := &Hooks{}

	// Test AddOnHit
	hitCalled := false
	hooks.AddOnHit(func(key string, value any) {
		hitCalled = true
	})

	if len(hooks.OnHit) != 1 {
		t.Fatal("AddOnHit should add to OnHit slice")
	}

	// Test AddOnMiss
	missCalled := false
	hooks.AddOnMiss(func(key string) {
		missCalled = true
	})

	if len(hooks.OnMiss) != 1 {
		t.Fatal("AddOnMiss should add to OnMiss slice")
	}

	// Test AddOnEvict
	evictCalled := false
	hooks.AddOnEvict(func(key string, value any, reason EvictReason) {
		evictCalled = true
	})

	if len(hooks.OnEvict) != 1 {
		t.Fatal("AddOnEvict should add to OnEvict slice")
	}

	// Test AddOnInvalidate
	invalidateCalled := false
	hooks.AddOnInvalidate(func(key string) {
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
	hooks.AddOnHit(func(key string, value any) {
		hitCount++
	})
	hooks.AddOnHit(func(key string, value any) {
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
