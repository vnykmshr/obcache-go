package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vnykmshr/obcache-go/pkg/obcache"
)

// Example demonstrating advanced hook features:
// 1. Priority-based hook ordering (metrics before logging)
// 2. Conditional hook execution (environment-specific, key-based filtering)
// 3. Hook composition utilities

func main() {
	fmt.Println("=== Advanced Hook Features Demo ===")

	// Create hooks instance
	hooks := &obcache.Hooks{}

	// Example 1: Priority-based Hook Ordering
	fmt.Println("1. Priority-based Hook Ordering:")
	setupPriorityHooks(hooks)

	// Example 2: Conditional Hook Execution
	fmt.Println("\n2. Conditional Hook Execution:")
	setupConditionalHooks(hooks)

	// Example 3: Hook Composition Utilities
	fmt.Println("\n3. Hook Composition Utilities:")
	demonstrateHookComposition()

	// Create cache with all hooks
	config := obcache.NewDefaultConfig().WithHooks(hooks)
	cache, err := obcache.New(config)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}

	fmt.Println("\n=== Testing Hook Execution ===")

	// Test different scenarios
	testScenario1(cache) // Metrics key with production context
	testScenario2(cache) // Debug key with debug context
	testScenario3(cache) // Regular key
}

func setupPriorityHooks(hooks *obcache.Hooks) {
	// High priority: Metrics collection (should run first)
	hooks.AddOnHitWithPriority(func(key string, value any) {
		fmt.Printf("  📊 [HIGH] Metrics: Recording hit for key '%s'\n", key)
	}, obcache.HookPriorityHigh)

	// Medium priority: Business logic
	hooks.AddOnHitWithPriority(func(key string, value any) {
		fmt.Printf("  💼 [MEDIUM] Business: Processing hit for key '%s'\n", key)
	}, obcache.HookPriorityMedium)

	// Low priority: Logging (should run last)
	hooks.AddOnHitWithPriority(func(key string, value any) {
		fmt.Printf("  📝 [LOW] Logging: Cache hit recorded for key '%s'\n", key)
	}, obcache.HookPriorityLow)
}

func setupConditionalHooks(hooks *obcache.Hooks) {
	// Conditional hook: Only execute for metrics keys
	hooks.AddOnHitCtxIf(func(ctx context.Context, key string, value any, args []any) {
		fmt.Printf("  📈 Prometheus: Incrementing counter for metrics key '%s'\n", key)
	}, obcache.KeyPrefixCondition("metrics:"))

	// Conditional hook: Only execute in debug mode
	hooks.AddOnHitCtxIf(func(ctx context.Context, key string, value any, args []any) {
		fmt.Printf("  🐛 Debug: Detailed logging for key '%s', value: %v\n", key, value)
	}, obcache.ContextValueCondition("debug", true))

	// Complex conditional hook: Combine conditions with AND
	hooks.AddOnMissCtxIf(func(ctx context.Context, key string, args []any) {
		fmt.Printf("  🚨 Alert: Production metrics miss for key '%s'\n", key)
	}, obcache.AndCondition(
		obcache.KeyPrefixCondition("metrics:"),
		obcache.ContextValueCondition("env", "production"),
	))

	// Environment-specific hook using OR condition
	hooks.AddOnHitCtxIf(func(ctx context.Context, key string, value any, args []any) {
		fmt.Printf("  🏷️  Environment: Special handling for key '%s'\n", key)
	}, obcache.OrCondition(
		obcache.ContextValueCondition("env", "staging"),
		obcache.ContextValueCondition("env", "production"),
	))
}

func demonstrateHookComposition() {
	// Create individual hooks
	metricsHook := func(key string, value any) {
		fmt.Printf("  📊 Combined: Metrics recorded for '%s'\n", key)
	}

	auditHook := func(key string, value any) {
		fmt.Printf("  🔍 Combined: Audit trail for '%s'\n", key)
	}

	// Combine hooks into a single hook
	combinedHook := obcache.CombineOnHitHooks(metricsHook, auditHook)

	fmt.Println("  Executing combined hook:")
	combinedHook("user:123", "John Doe")

	// Create a conditional wrapper
	conditionalHook := obcache.ConditionalHook(
		func(ctx context.Context, key string, value any, args []any) {
			fmt.Printf("  ✅ Conditional: Hook executed for key '%s'\n", key)
		},
		func(ctx context.Context, key string, args []any) bool {
			return strings.HasPrefix(key, "api:")
		},
	)

	fmt.Println("  Testing conditional wrapper:")
	conditionalHook(context.Background(), "api:endpoint", "data", nil) // Should execute
	conditionalHook(context.Background(), "web:page", "data", nil)     // Should not execute
}

func testScenario1(cache *obcache.Cache) {
	fmt.Println("\n--- Scenario 1: Metrics key with production context ---")

	key := "metrics:api_requests"

	// Set and get to trigger hooks
	_ = cache.Set(key, 150, time.Hour)
	_, _ = cache.Get(key)

	// Test miss scenario
	_, _ = cache.Get("metrics:missing_key")
}

func testScenario2(cache *obcache.Cache) {
	fmt.Println("\n--- Scenario 2: Debug key with debug context ---")

	key := "debug:session_data"

	// Set and get to trigger hooks
	_ = cache.Set(key, map[string]string{"user": "alice", "session": "abc123"}, time.Hour)
	_, _ = cache.Get(key)
}

func testScenario3(cache *obcache.Cache) {
	fmt.Println("\n--- Scenario 3: Regular key (minimal hooks) ---")

	key := "user:profile:123"

	// Set and get to trigger hooks
	_ = cache.Set(key, "profile data", time.Hour)
	_, _ = cache.Get(key)
}
