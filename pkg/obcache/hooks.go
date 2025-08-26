package obcache

import "context"

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
}
