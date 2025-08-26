package obcache

import (
	"context"
	"sort"
	"strings"
)

// Hooks defines event callbacks for cache operations
type Hooks struct {
	// OnHit is called when a cache key is found and not expired
	OnHit []OnHitHook

	// OnMiss is called when a cache key is not found or expired
	OnMiss []OnMissHook

	// OnEvict is called when a cache entry is evicted (LRU or TTL)
	OnEvict []OnEvictHook

	// OnInvalidate is called when a cache entry is manually invalidated
	OnInvalidate []OnInvalidateHook

	// Context-aware hooks with function arguments
	// OnHitCtx is called when a cache hit occurs with context and function arguments
	OnHitCtx []OnHitHookCtx

	// OnMissCtx is called when a cache miss occurs with context and function arguments
	OnMissCtx []OnMissHookCtx

	// OnEvictCtx is called when a cache entry is evicted with context and function arguments
	OnEvictCtx []OnEvictHookCtx

	// OnInvalidateCtx is called when a cache entry is invalidated with context and function arguments
	OnInvalidateCtx []OnInvalidateHookCtx

	// Priority-based hooks - stored as prioritized lists
	OnHitPriority           []PriorityOnHitHook
	OnMissPriority          []PriorityOnMissHook
	OnEvictPriority         []PriorityOnEvictHook
	OnInvalidatePriority    []PriorityOnInvalidateHook
	OnHitCtxPriority        []PriorityOnHitHookCtx
	OnMissCtxPriority       []PriorityOnMissHookCtx
	OnEvictCtxPriority      []PriorityOnEvictHookCtx
	OnInvalidateCtxPriority []PriorityOnInvalidateHookCtx

	// Conditional hooks - only execute when conditions are met
	OnHitCtxConditional        []ConditionalOnHitHookCtx
	OnMissCtxConditional       []ConditionalOnMissHookCtx
	OnEvictCtxConditional      []ConditionalOnEvictHookCtx
	OnInvalidateCtxConditional []ConditionalOnInvalidateHookCtx
}

// Hook function type definitions
type (
	// OnHitHook is called when a cache hit occurs
	OnHitHook func(key string, value any)

	// OnMissHook is called when a cache miss occurs
	OnMissHook func(key string)

	// OnEvictHook is called when a cache entry is evicted
	OnEvictHook func(key string, value any, reason EvictReason)

	// OnInvalidateHook is called when a cache entry is invalidated
	OnInvalidateHook func(key string)

	// Context-aware hook function type definitions
	// OnHitHookCtx is called when a cache hit occurs with context and function arguments
	OnHitHookCtx func(ctx context.Context, key string, value any, args []any)

	// OnMissHookCtx is called when a cache miss occurs with context and function arguments
	OnMissHookCtx func(ctx context.Context, key string, args []any)

	// OnEvictHookCtx is called when a cache entry is evicted with context and function arguments
	OnEvictHookCtx func(ctx context.Context, key string, value any, reason EvictReason, args []any)

	// OnInvalidateHookCtx is called when a cache entry is invalidated with context and function arguments
	OnInvalidateHookCtx func(ctx context.Context, key string, args []any)
)

// HookPriority defines the execution priority for hooks
type HookPriority int

const (
	// HookPriorityHigh - Execute before other priorities (e.g., metrics collection)
	HookPriorityHigh HookPriority = 100

	// HookPriorityMedium - Standard priority (default behavior)
	HookPriorityMedium HookPriority = 50

	// HookPriorityLow - Execute after other priorities (e.g., logging, debugging)
	HookPriorityLow HookPriority = 10
)

// Priority-aware hook structures that wrap hooks with their priorities
type (
	// PriorityOnHitHook combines a hook with its execution priority
	PriorityOnHitHook struct {
		Priority HookPriority
		Hook     OnHitHook
	}

	// PriorityOnMissHook combines a miss hook with its execution priority
	PriorityOnMissHook struct {
		Priority HookPriority
		Hook     OnMissHook
	}

	// PriorityOnEvictHook combines an evict hook with its execution priority
	PriorityOnEvictHook struct {
		Priority HookPriority
		Hook     OnEvictHook
	}

	// PriorityOnInvalidateHook combines an invalidate hook with its execution priority
	PriorityOnInvalidateHook struct {
		Priority HookPriority
		Hook     OnInvalidateHook
	}

	// Context-aware priority hook structures
	// PriorityOnHitHookCtx combines a context-aware hit hook with its execution priority
	PriorityOnHitHookCtx struct {
		Priority HookPriority
		Hook     OnHitHookCtx
	}

	// PriorityOnMissHookCtx combines a context-aware miss hook with its execution priority
	PriorityOnMissHookCtx struct {
		Priority HookPriority
		Hook     OnMissHookCtx
	}

	// PriorityOnEvictHookCtx combines a context-aware evict hook with its execution priority
	PriorityOnEvictHookCtx struct {
		Priority HookPriority
		Hook     OnEvictHookCtx
	}

	// PriorityOnInvalidateHookCtx combines a context-aware invalidate hook with its execution priority
	PriorityOnInvalidateHookCtx struct {
		Priority HookPriority
		Hook     OnInvalidateHookCtx
	}
)

// HookCondition defines a predicate function that determines whether a hook should execute.
// It receives the same context and key information as the hook itself.
type HookCondition func(ctx context.Context, key string, args []any) bool

// Conditional hook structures that wrap context-aware hooks with execution conditions
type (
	// ConditionalOnHitHookCtx combines a context-aware hit hook with its execution condition
	ConditionalOnHitHookCtx struct {
		Condition HookCondition
		Hook      OnHitHookCtx
	}

	// ConditionalOnMissHookCtx combines a context-aware miss hook with its execution condition
	ConditionalOnMissHookCtx struct {
		Condition HookCondition
		Hook      OnMissHookCtx
	}

	// ConditionalOnEvictHookCtx combines a context-aware evict hook with its execution condition
	ConditionalOnEvictHookCtx struct {
		Condition HookCondition
		Hook      OnEvictHookCtx
	}

	// ConditionalOnInvalidateHookCtx combines a context-aware invalidate hook with its execution condition
	ConditionalOnInvalidateHookCtx struct {
		Condition HookCondition
		Hook      OnInvalidateHookCtx
	}
)

// EvictReason indicates why a cache entry was evicted
type EvictReason int

const (
	// EvictReasonLRU indicates the entry was evicted due to LRU policy
	EvictReasonLRU EvictReason = iota

	// EvictReasonTTL indicates the entry was evicted due to TTL expiration
	EvictReasonTTL

	// EvictReasonCapacity indicates the entry was evicted due to capacity limits
	EvictReasonCapacity
)

// String representations for EvictReason
const (
	evictReasonLRUString      = "LRU"
	evictReasonTTLString      = "TTL"
	evictReasonCapacityString = "Capacity"
)

func (r EvictReason) String() string {
	switch r {
	case EvictReasonLRU:
		return evictReasonLRUString
	case EvictReasonTTL:
		return evictReasonTTLString
	case EvictReasonCapacity:
		return evictReasonCapacityString
	default:
		return "Unknown"
	}
}

// AddOnHit adds an OnHit hook
func (h *Hooks) AddOnHit(hook OnHitHook) {
	h.OnHit = append(h.OnHit, hook)
}

// AddOnMiss adds an OnMiss hook
func (h *Hooks) AddOnMiss(hook OnMissHook) {
	h.OnMiss = append(h.OnMiss, hook)
}

// AddOnEvict adds an OnEvict hook
func (h *Hooks) AddOnEvict(hook OnEvictHook) {
	h.OnEvict = append(h.OnEvict, hook)
}

// AddOnInvalidate adds an OnInvalidate hook
func (h *Hooks) AddOnInvalidate(hook OnInvalidateHook) {
	h.OnInvalidate = append(h.OnInvalidate, hook)
}

// Context-aware hook builder methods
// AddOnHitCtx adds an OnHitCtx hook
func (h *Hooks) AddOnHitCtx(hook OnHitHookCtx) {
	h.OnHitCtx = append(h.OnHitCtx, hook)
}

// AddOnMissCtx adds an OnMissCtx hook
func (h *Hooks) AddOnMissCtx(hook OnMissHookCtx) {
	h.OnMissCtx = append(h.OnMissCtx, hook)
}

// AddOnEvictCtx adds an OnEvictCtx hook
func (h *Hooks) AddOnEvictCtx(hook OnEvictHookCtx) {
	h.OnEvictCtx = append(h.OnEvictCtx, hook)
}

// AddOnInvalidateCtx adds an OnInvalidateCtx hook
func (h *Hooks) AddOnInvalidateCtx(hook OnInvalidateHookCtx) {
	h.OnInvalidateCtx = append(h.OnInvalidateCtx, hook)
}

// Priority-based hook registration methods
// AddOnHitWithPriority adds an OnHit hook with specified priority
func (h *Hooks) AddOnHitWithPriority(hook OnHitHook, priority HookPriority) {
	h.OnHitPriority = append(h.OnHitPriority, PriorityOnHitHook{
		Priority: priority,
		Hook:     hook,
	})
}

// AddOnMissWithPriority adds an OnMiss hook with specified priority
func (h *Hooks) AddOnMissWithPriority(hook OnMissHook, priority HookPriority) {
	h.OnMissPriority = append(h.OnMissPriority, PriorityOnMissHook{
		Priority: priority,
		Hook:     hook,
	})
}

// AddOnEvictWithPriority adds an OnEvict hook with specified priority
func (h *Hooks) AddOnEvictWithPriority(hook OnEvictHook, priority HookPriority) {
	h.OnEvictPriority = append(h.OnEvictPriority, PriorityOnEvictHook{
		Priority: priority,
		Hook:     hook,
	})
}

// AddOnInvalidateWithPriority adds an OnInvalidate hook with specified priority
func (h *Hooks) AddOnInvalidateWithPriority(hook OnInvalidateHook, priority HookPriority) {
	h.OnInvalidatePriority = append(h.OnInvalidatePriority, PriorityOnInvalidateHook{
		Priority: priority,
		Hook:     hook,
	})
}

// Context-aware priority hook registration methods
// AddOnHitCtxWithPriority adds an OnHitCtx hook with specified priority
func (h *Hooks) AddOnHitCtxWithPriority(hook OnHitHookCtx, priority HookPriority) {
	h.OnHitCtxPriority = append(h.OnHitCtxPriority, PriorityOnHitHookCtx{
		Priority: priority,
		Hook:     hook,
	})
}

// AddOnMissCtxWithPriority adds an OnMissCtx hook with specified priority
func (h *Hooks) AddOnMissCtxWithPriority(hook OnMissHookCtx, priority HookPriority) {
	h.OnMissCtxPriority = append(h.OnMissCtxPriority, PriorityOnMissHookCtx{
		Priority: priority,
		Hook:     hook,
	})
}

// AddOnEvictCtxWithPriority adds an OnEvictCtx hook with specified priority
func (h *Hooks) AddOnEvictCtxWithPriority(hook OnEvictHookCtx, priority HookPriority) {
	h.OnEvictCtxPriority = append(h.OnEvictCtxPriority, PriorityOnEvictHookCtx{
		Priority: priority,
		Hook:     hook,
	})
}

// AddOnInvalidateCtxWithPriority adds an OnInvalidateCtx hook with specified priority
func (h *Hooks) AddOnInvalidateCtxWithPriority(hook OnInvalidateHookCtx, priority HookPriority) {
	h.OnInvalidateCtxPriority = append(h.OnInvalidateCtxPriority, PriorityOnInvalidateHookCtx{
		Priority: priority,
		Hook:     hook,
	})
}

// AddOnHitCtxIf adds a conditional context-aware hit hook that only executes when the condition is met
func (h *Hooks) AddOnHitCtxIf(hook OnHitHookCtx, condition HookCondition) {
	h.OnHitCtxConditional = append(h.OnHitCtxConditional, ConditionalOnHitHookCtx{
		Condition: condition,
		Hook:      hook,
	})
}

// AddOnMissCtxIf adds a conditional context-aware miss hook that only executes when the condition is met
func (h *Hooks) AddOnMissCtxIf(hook OnMissHookCtx, condition HookCondition) {
	h.OnMissCtxConditional = append(h.OnMissCtxConditional, ConditionalOnMissHookCtx{
		Condition: condition,
		Hook:      hook,
	})
}

// AddOnEvictCtxIf adds a conditional context-aware evict hook that only executes when the condition is met
func (h *Hooks) AddOnEvictCtxIf(hook OnEvictHookCtx, condition HookCondition) {
	h.OnEvictCtxConditional = append(h.OnEvictCtxConditional, ConditionalOnEvictHookCtx{
		Condition: condition,
		Hook:      hook,
	})
}

// AddOnInvalidateCtxIf adds a conditional context-aware invalidate hook that only executes when the condition is met
func (h *Hooks) AddOnInvalidateCtxIf(hook OnInvalidateHookCtx, condition HookCondition) {
	h.OnInvalidateCtxConditional = append(h.OnInvalidateCtxConditional, ConditionalOnInvalidateHookCtx{
		Condition: condition,
		Hook:      hook,
	})
}

// Hook composition utility functions

// CombineOnHitHooks combines multiple hit hooks into a single hook that executes all of them
func CombineOnHitHooks(hooks ...OnHitHook) OnHitHook {
	return func(key string, value any) {
		for _, hook := range hooks {
			if hook != nil {
				hook(key, value)
			}
		}
	}
}

// CombineOnHitHooksCtx combines multiple context-aware hit hooks into a single hook
func CombineOnHitHooksCtx(hooks ...OnHitHookCtx) OnHitHookCtx {
	return func(ctx context.Context, key string, value any, args []any) {
		for _, hook := range hooks {
			if hook != nil {
				hook(ctx, key, value, args)
			}
		}
	}
}

// CombineOnMissHooks combines multiple miss hooks into a single hook that executes all of them
func CombineOnMissHooks(hooks ...OnMissHook) OnMissHook {
	return func(key string) {
		for _, hook := range hooks {
			if hook != nil {
				hook(key)
			}
		}
	}
}

// CombineOnMissHooksCtx combines multiple context-aware miss hooks into a single hook
func CombineOnMissHooksCtx(hooks ...OnMissHookCtx) OnMissHookCtx {
	return func(ctx context.Context, key string, args []any) {
		for _, hook := range hooks {
			if hook != nil {
				hook(ctx, key, args)
			}
		}
	}
}

// ConditionalHook wraps a context-aware hook with a condition, creating a conditional hook
func ConditionalHook(hook OnHitHookCtx, condition HookCondition) OnHitHookCtx {
	return func(ctx context.Context, key string, value any, args []any) {
		if condition != nil && condition(ctx, key, args) {
			hook(ctx, key, value, args)
		}
	}
}

// ConditionalMissHook wraps a context-aware miss hook with a condition
func ConditionalMissHook(hook OnMissHookCtx, condition HookCondition) OnMissHookCtx {
	return func(ctx context.Context, key string, args []any) {
		if condition != nil && condition(ctx, key, args) {
			hook(ctx, key, args)
		}
	}
}

// Common condition builders

// KeyPrefixCondition creates a condition that checks if the key starts with a specific prefix
func KeyPrefixCondition(prefix string) HookCondition {
	return func(ctx context.Context, key string, args []any) bool {
		return strings.HasPrefix(key, prefix)
	}
}

// ContextValueCondition creates a condition that checks for a specific context value
func ContextValueCondition(ctxKey, expectedValue any) HookCondition {
	return func(ctx context.Context, key string, args []any) bool {
		return ctx.Value(ctxKey) == expectedValue
	}
}

// AndCondition combines multiple conditions with AND logic
func AndCondition(conditions ...HookCondition) HookCondition {
	return func(ctx context.Context, key string, args []any) bool {
		for _, condition := range conditions {
			if condition != nil && !condition(ctx, key, args) {
				return false
			}
		}
		return true
	}
}

// OrCondition combines multiple conditions with OR logic
func OrCondition(conditions ...HookCondition) HookCondition {
	return func(ctx context.Context, key string, args []any) bool {
		for _, condition := range conditions {
			if condition != nil && condition(ctx, key, args) {
				return true
			}
		}
		return false
	}
}

// invokeOnHitWithCtx calls all OnHit and OnHitCtx hooks with context and args
func (h *Hooks) invokeOnHitWithCtx(ctx context.Context, key string, value any, args []any) {
	// Call legacy hooks
	for _, hook := range h.OnHit {
		if hook != nil {
			hook(key, value)
		}
	}
	// Call context-aware hooks
	for _, hook := range h.OnHitCtx {
		if hook != nil {
			hook(ctx, key, value, args)
		}
	}

	// Call priority-based legacy hooks (sorted by priority, highest first)
	priorityHooks := make([]PriorityOnHitHook, len(h.OnHitPriority))
	copy(priorityHooks, h.OnHitPriority)
	sort.Slice(priorityHooks, func(i, j int) bool {
		return priorityHooks[i].Priority > priorityHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(key, value)
		}
	}

	// Call priority-based context-aware hooks (sorted by priority, highest first)
	priorityCtxHooks := make([]PriorityOnHitHookCtx, len(h.OnHitCtxPriority))
	copy(priorityCtxHooks, h.OnHitCtxPriority)
	sort.Slice(priorityCtxHooks, func(i, j int) bool {
		return priorityCtxHooks[i].Priority > priorityCtxHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityCtxHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(ctx, key, value, args)
		}
	}

	// Call conditional context-aware hooks (only if conditions are met)
	for _, conditionalHook := range h.OnHitCtxConditional {
		if conditionalHook.Hook != nil && conditionalHook.Condition != nil {
			if conditionalHook.Condition(ctx, key, args) {
				conditionalHook.Hook(ctx, key, value, args)
			}
		}
	}
}

// invokeOnMissWithCtx calls all OnMiss and OnMissCtx hooks with context and args
func (h *Hooks) invokeOnMissWithCtx(ctx context.Context, key string, args []any) {
	// Call legacy hooks
	for _, hook := range h.OnMiss {
		if hook != nil {
			hook(key)
		}
	}
	// Call context-aware hooks
	for _, hook := range h.OnMissCtx {
		if hook != nil {
			hook(ctx, key, args)
		}
	}

	// Call priority-based legacy hooks (sorted by priority, highest first)
	priorityHooks := make([]PriorityOnMissHook, len(h.OnMissPriority))
	copy(priorityHooks, h.OnMissPriority)
	sort.Slice(priorityHooks, func(i, j int) bool {
		return priorityHooks[i].Priority > priorityHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(key)
		}
	}

	// Call priority-based context-aware hooks (sorted by priority, highest first)
	priorityCtxHooks := make([]PriorityOnMissHookCtx, len(h.OnMissCtxPriority))
	copy(priorityCtxHooks, h.OnMissCtxPriority)
	sort.Slice(priorityCtxHooks, func(i, j int) bool {
		return priorityCtxHooks[i].Priority > priorityCtxHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityCtxHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(ctx, key, args)
		}
	}

	// Call conditional context-aware hooks (only if conditions are met)
	for _, conditionalHook := range h.OnMissCtxConditional {
		if conditionalHook.Hook != nil && conditionalHook.Condition != nil {
			if conditionalHook.Condition(ctx, key, args) {
				conditionalHook.Hook(ctx, key, args)
			}
		}
	}
}

// invokeOnEvict calls all OnEvict and OnEvictCtx hooks
func (h *Hooks) invokeOnEvict(key string, value any, reason EvictReason) {
	h.invokeOnEvictWithCtx(context.Background(), key, value, reason, nil)
}

// invokeOnEvictWithCtx calls all OnEvict and OnEvictCtx hooks with context and args
func (h *Hooks) invokeOnEvictWithCtx(ctx context.Context, key string, value any, reason EvictReason, args []any) {
	// Call legacy hooks
	for _, hook := range h.OnEvict {
		if hook != nil {
			hook(key, value, reason)
		}
	}
	// Call context-aware hooks
	for _, hook := range h.OnEvictCtx {
		if hook != nil {
			hook(ctx, key, value, reason, args)
		}
	}

	// Call priority-based legacy hooks (sorted by priority, highest first)
	priorityHooks := make([]PriorityOnEvictHook, len(h.OnEvictPriority))
	copy(priorityHooks, h.OnEvictPriority)
	sort.Slice(priorityHooks, func(i, j int) bool {
		return priorityHooks[i].Priority > priorityHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(key, value, reason)
		}
	}

	// Call priority-based context-aware hooks (sorted by priority, highest first)
	priorityCtxHooks := make([]PriorityOnEvictHookCtx, len(h.OnEvictCtxPriority))
	copy(priorityCtxHooks, h.OnEvictCtxPriority)
	sort.Slice(priorityCtxHooks, func(i, j int) bool {
		return priorityCtxHooks[i].Priority > priorityCtxHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityCtxHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(ctx, key, value, reason, args)
		}
	}

	// Call conditional context-aware hooks (only if conditions are met)
	for _, conditionalHook := range h.OnEvictCtxConditional {
		if conditionalHook.Hook != nil && conditionalHook.Condition != nil {
			if conditionalHook.Condition(ctx, key, args) {
				conditionalHook.Hook(ctx, key, value, reason, args)
			}
		}
	}
}

// invokeOnInvalidateWithCtx calls all OnInvalidate and OnInvalidateCtx hooks with context and args
func (h *Hooks) invokeOnInvalidateWithCtx(ctx context.Context, key string, args []any) {
	// Call legacy hooks
	for _, hook := range h.OnInvalidate {
		if hook != nil {
			hook(key)
		}
	}
	// Call context-aware hooks
	for _, hook := range h.OnInvalidateCtx {
		if hook != nil {
			hook(ctx, key, args)
		}
	}

	// Call priority-based legacy hooks (sorted by priority, highest first)
	priorityHooks := make([]PriorityOnInvalidateHook, len(h.OnInvalidatePriority))
	copy(priorityHooks, h.OnInvalidatePriority)
	sort.Slice(priorityHooks, func(i, j int) bool {
		return priorityHooks[i].Priority > priorityHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(key)
		}
	}

	// Call priority-based context-aware hooks (sorted by priority, highest first)
	priorityCtxHooks := make([]PriorityOnInvalidateHookCtx, len(h.OnInvalidateCtxPriority))
	copy(priorityCtxHooks, h.OnInvalidateCtxPriority)
	sort.Slice(priorityCtxHooks, func(i, j int) bool {
		return priorityCtxHooks[i].Priority > priorityCtxHooks[j].Priority // Higher priority first
	})
	for _, priorityHook := range priorityCtxHooks {
		if priorityHook.Hook != nil {
			priorityHook.Hook(ctx, key, args)
		}
	}

	// Call conditional context-aware hooks (only if conditions are met)
	for _, conditionalHook := range h.OnInvalidateCtxConditional {
		if conditionalHook.Hook != nil && conditionalHook.Condition != nil {
			if conditionalHook.Condition(ctx, key, args) {
				conditionalHook.Hook(ctx, key, args)
			}
		}
	}
}
