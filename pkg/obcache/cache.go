package obcache

import (
	"context"
	"sync"
	"time"

	"github.com/vnykmshr/obcache-go/internal/entry"
	"github.com/vnykmshr/obcache-go/internal/singleflight"
	"github.com/vnykmshr/obcache-go/internal/store"
	"github.com/vnykmshr/obcache-go/internal/store/memory"
)

// Context keys for cache-specific metadata
type contextKey string

const (
	// CacheTagsKey can be used to store cache tags in context
	CacheTagsKey contextKey = "cache_tags"
)

// CacheTagsFromContext extracts cache tags from context
func CacheTagsFromContext(ctx context.Context) []string {
	if tags, ok := ctx.Value(CacheTagsKey).([]string); ok {
		return tags
	}
	return nil
}

// WithCacheTags adds cache tags to context
func WithCacheTags(ctx context.Context, tags []string) context.Context {
	return context.WithValue(ctx, CacheTagsKey, tags)
}

// CacheCallContext contains context and function arguments for cache operations
type CacheCallContext struct {
	ctx  context.Context
	args []any
}

// CacheOption configures cache operation behavior
type CacheOption func(*CacheCallContext)

// WithContext sets the context for a cache operation
func WithContext(ctx context.Context) CacheOption {
	return func(cctx *CacheCallContext) {
		cctx.ctx = ctx
	}
}

// WithArgs sets the function arguments for a cache operation
func WithArgs(args []any) CacheOption {
	return func(cctx *CacheCallContext) {
		cctx.args = args
	}
}

// newCacheCallContext creates a new CacheCallContext with defaults
func newCacheCallContext(options ...CacheOption) *CacheCallContext {
	cctx := &CacheCallContext{
		ctx:  context.Background(),
		args: nil,
	}
	for _, opt := range options {
		opt(cctx)
	}
	return cctx
}

// Cache is the main cache implementation with LRU and TTL support
type Cache struct {
	config *Config
	store  store.Store
	stats  *Stats
	hooks  *Hooks
	sf     *singleflight.Group[string, any]
	mu     sync.RWMutex
}

// New creates a new Cache instance with the given configuration
func New(config *Config) (*Cache, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	// Create memory store with cleanup if cleanup interval is set
	var memStore store.Store
	var err error

	if config.CleanupInterval > 0 {
		memStore, err = memory.NewWithCleanup(config.MaxEntries, config.CleanupInterval)
	} else {
		memStore, err = memory.New(config.MaxEntries)
	}

	if err != nil {
		return nil, err
	}

	cache := &Cache{
		config: config,
		store:  memStore,
		stats:  &Stats{},
		hooks:  config.Hooks,
		sf:     &singleflight.Group[string, any]{},
	}

	// Set up store callbacks for statistics and hooks
	if lruStore, ok := memStore.(store.LRUStore); ok {
		lruStore.SetEvictCallback(func(key string, value any) {
			cache.stats.incEvictions()
			if cache.hooks != nil {
				cache.hooks.invokeOnEvict(key, value, EvictReasonLRU)
			}
		})
	}

	if ttlStore, ok := memStore.(store.TTLStore); ok {
		ttlStore.SetCleanupCallback(func(key string, value any) {
			cache.stats.incEvictions()
			if cache.hooks != nil {
				cache.hooks.invokeOnEvict(key, value, EvictReasonTTL)
			}
		})
	}

	return cache, nil
}

// Get retrieves a value from the cache by key
func (c *Cache) Get(key string, options ...CacheOption) (any, bool) {
	cctx := newCacheCallContext(options...)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.store.Get(key)
	if !found {
		c.stats.incMisses()
		if c.hooks != nil {
			c.hooks.invokeOnMissWithCtx(cctx.ctx, key, cctx.args)
		}
		return nil, false
	}

	c.stats.incHits()
	if c.hooks != nil {
		c.hooks.invokeOnHitWithCtx(cctx.ctx, key, entry.Value, cctx.args)
	}

	return entry.Value, true
}

// Set stores a value in the cache with the specified key and TTL
// If ttl is 0 or negative, the default TTL from config is used
// If both are 0 or negative, the entry never expires
func (c *Cache) Set(key string, value any, ttl time.Duration, _ ...CacheOption) error {
	if ttl <= 0 {
		ttl = c.config.DefaultTTL
	}

	var cacheEntry *entry.Entry
	if ttl > 0 {
		cacheEntry = entry.New(value, ttl)
	} else {
		cacheEntry = entry.NewWithoutTTL(value)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.store.Set(key, cacheEntry)
	if err == nil {
		c.updateKeyCount()
	}

	return err
}

// Warmup preloads a value into the cache (alias for Set with default TTL)
func (c *Cache) Warmup(key string, value any) error {
	return c.Set(key, value, c.config.DefaultTTL)
}

// WarmupWithTTL preloads a value into the cache with specific TTL
func (c *Cache) WarmupWithTTL(key string, value any, ttl time.Duration) error {
	return c.Set(key, value, ttl)
}

// Invalidate removes a specific key from the cache
func (c *Cache) Invalidate(key string, options ...CacheOption) error {
	cctx := newCacheCallContext(options...)

	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.store.Delete(key)
	if err == nil {
		c.stats.incInvalidations()
		c.updateKeyCount()

		if c.hooks != nil {
			c.hooks.invokeOnInvalidateWithCtx(cctx.ctx, key, cctx.args)
		}
	}

	return err
}

// InvalidateAll removes all entries from the cache
func (c *Cache) InvalidateAll(options ...CacheOption) error {
	cctx := newCacheCallContext(options...)

	c.mu.Lock()
	defer c.mu.Unlock()

	keys := c.store.Keys()
	err := c.store.Clear()
	if err == nil {
		for _, key := range keys {
			c.stats.incInvalidations()
			if c.hooks != nil {
				c.hooks.invokeOnInvalidateWithCtx(cctx.ctx, key, cctx.args)
			}
		}
		c.updateKeyCount()
	}

	return err
}

// Stats returns the current cache statistics
func (c *Cache) Stats() *Stats {
	c.updateKeyCount()
	return c.stats
}

// Keys returns all current cache keys
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.store.Keys()
}

// Len returns the current number of entries in the cache
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.store.Len()
}

// Has checks if a key exists in the cache (without updating access time)
func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.store.Get(key)
	return found && !entry.IsExpired()
}

// GetWithTTL retrieves a value and its remaining TTL
func (c *Cache) GetWithTTL(key string, options ...CacheOption) (value any, ttl time.Duration, found bool) {
	cctx := newCacheCallContext(options...)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.store.Get(key)
	if !found {
		c.stats.incMisses()
		if c.hooks != nil {
			c.hooks.invokeOnMissWithCtx(cctx.ctx, key, cctx.args)
		}
		return nil, 0, false
	}

	c.stats.incHits()
	if c.hooks != nil {
		c.hooks.invokeOnHitWithCtx(cctx.ctx, key, entry.Value, cctx.args)
	}

	return entry.Value, entry.TTL(), true
}

// Close closes the cache and cleans up resources
func (c *Cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.store.Close()
}

// Cleanup manually triggers cleanup of expired entries
// Returns the number of entries removed
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttlStore, ok := c.store.(store.TTLStore); ok {
		removed := ttlStore.Cleanup()
		c.updateKeyCount()
		return removed
	}

	return 0
}

// updateKeyCount updates the key count statistic
func (c *Cache) updateKeyCount() {
	count := int64(c.store.Len())
	c.stats.setKeyCount(count)
}

// getKeyGenFunc returns the key generation function to use
func (c *Cache) getKeyGenFunc() KeyGenFunc {
	if c.config.KeyGenFunc != nil {
		return c.config.KeyGenFunc
	}
	return DefaultKeyFunc
}
