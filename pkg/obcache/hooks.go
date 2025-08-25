package obcache

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

func (r EvictReason) String() string {
	switch r {
	case EvictReasonLRU:
		return "LRU"
	case EvictReasonTTL:
		return "TTL"
	case EvictReasonCapacity:
		return "Capacity"
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

// invokeOnHit calls all OnHit hooks
func (h *Hooks) invokeOnHit(key string, value any) {
	for _, hook := range h.OnHit {
		if hook != nil {
			hook(key, value)
		}
	}
}

// invokeOnMiss calls all OnMiss hooks
func (h *Hooks) invokeOnMiss(key string) {
	for _, hook := range h.OnMiss {
		if hook != nil {
			hook(key)
		}
	}
}

// invokeOnEvict calls all OnEvict hooks
func (h *Hooks) invokeOnEvict(key string, value any, reason EvictReason) {
	for _, hook := range h.OnEvict {
		if hook != nil {
			hook(key, value, reason)
		}
	}
}

// invokeOnInvalidate calls all OnInvalidate hooks
func (h *Hooks) invokeOnInvalidate(key string) {
	for _, hook := range h.OnInvalidate {
		if hook != nil {
			hook(key)
		}
	}
}