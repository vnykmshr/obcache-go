package obcache

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vnykmshr/obcache-go/internal/entry"
	"github.com/vnykmshr/obcache-go/internal/singleflight"
	"github.com/vnykmshr/obcache-go/internal/store"
	"github.com/vnykmshr/obcache-go/internal/store/memory"
	redisstore "github.com/vnykmshr/obcache-go/internal/store/redis"
	"github.com/vnykmshr/obcache-go/pkg/compression"
	"github.com/vnykmshr/obcache-go/pkg/metrics"
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

	// Compression
	compressor compression.Compressor

	// Metrics
	metricsExporter metrics.Exporter
	metricsLabels   metrics.Labels
	metricsStop     chan struct{}
	metricsWg       sync.WaitGroup
}

// New creates a new Cache instance with the given configuration
func New(config *Config) (*Cache, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	// Create the appropriate store based on configuration
	var cacheStore store.Store
	var err error

	switch config.StoreType {
	case StoreTypeMemory:
		cacheStore, err = createMemoryStore(config)
	case StoreTypeRedis:
		cacheStore, err = createRedisStore(config)
	default:
		return nil, fmt.Errorf("unsupported store type: %v", config.StoreType)
	}

	if err != nil {
		return nil, err
	}

	cache := &Cache{
		config: config,
		store:  cacheStore,
		stats:  &Stats{},
		hooks:  config.Hooks,
		sf:     &singleflight.Group[string, any]{},
	}

	// Initialize compression if configured
	if err := cache.initializeCompression(); err != nil {
		return nil, fmt.Errorf("failed to initialize compression: %w", err)
	}

	// Initialize metrics if configured
	if err := cache.initializeMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Set up store callbacks for statistics and hooks
	if lruStore, ok := cacheStore.(store.LRUStore); ok {
		lruStore.SetEvictCallback(func(key string, value any) {
			cache.stats.incEvictions()
			if cache.hooks != nil {
				cache.hooks.invokeOnEvict(key, value, EvictReasonLRU)
			}
		})
	}

	if ttlStore, ok := cacheStore.(store.TTLStore); ok {
		ttlStore.SetCleanupCallback(func(key string, value any) {
			cache.stats.incEvictions()
			if cache.hooks != nil {
				cache.hooks.invokeOnEvict(key, value, EvictReasonTTL)
			}
		})
	}

	return cache, nil
}

// createMemoryStore creates a memory-based store
func createMemoryStore(config *Config) (store.Store, error) {
	if config.CleanupInterval > 0 {
		return memory.NewWithCleanup(config.MaxEntries, config.CleanupInterval)
	}
	return memory.New(config.MaxEntries)
}

// createRedisStore creates a Redis-based store
func createRedisStore(config *Config) (store.Store, error) {
	if config.Redis == nil {
		return nil, fmt.Errorf("Redis configuration is required when using StoreTypeRedis")
	}

	redisConfig := &redisstore.Config{
		DefaultTTL: config.DefaultTTL,
		KeyPrefix:  config.Redis.KeyPrefix,
		Context:    context.Background(),
	}

	// Use provided client or create a new one
	if config.Redis.Client != nil {
		redisConfig.Client = config.Redis.Client
	} else {
		// Create Redis client from connection parameters
		client := redis.NewClient(&redis.Options{
			Addr:     config.Redis.Addr,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
		})

		// Test the connection
		ctx := context.Background()
		if err := client.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}

		redisConfig.Client = client
	}

	return redisstore.New(redisConfig)
}

// Get retrieves a value from the cache by key
func (c *Cache) Get(key string, options ...CacheOption) (any, bool) {
	start := time.Now()
	defer func() {
		c.recordCacheOperation(metrics.OperationGet, time.Since(start))
	}()

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

	// Decompress value if needed
	value, err := c.decompressValue(entry)
	if err != nil {
		// If decompression fails, treat as cache miss
		c.stats.incMisses()
		return nil, false
	}

	c.stats.incHits()
	if c.hooks != nil {
		c.hooks.invokeOnHitWithCtx(cctx.ctx, key, value, cctx.args)
	}

	return value, true
}

// Set stores a value in the cache with the specified key and TTL
// If ttl is 0 or negative, the default TTL from config is used
// If both are 0 or negative, the entry never expires
func (c *Cache) Set(key string, value any, ttl time.Duration, _ ...CacheOption) error {
	start := time.Now()
	defer func() {
		c.recordCacheOperation(metrics.OperationSet, time.Since(start))
	}()

	if ttl <= 0 {
		ttl = c.config.DefaultTTL
	}

	// Prepare entry with compression if enabled
	cacheEntry, err := c.createCompressedEntry(value, ttl)
	if err != nil {
		return fmt.Errorf("failed to create compressed entry: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	err = c.store.Set(key, cacheEntry)
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

	// Stop metrics reporting if running
	if c.metricsStop != nil {
		close(c.metricsStop)
		c.metricsWg.Wait()
	}

	// Close metrics exporter
	if c.metricsExporter != nil {
		c.metricsExporter.Close()
	}

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

// createCompressedEntry creates a cache entry with compression if applicable
func (c *Cache) createCompressedEntry(value any, ttl time.Duration) (*entry.Entry, error) {
	var cacheEntry *entry.Entry
	if ttl > 0 {
		cacheEntry = entry.New(nil, ttl) // We'll set the value after compression
	} else {
		cacheEntry = entry.NewWithoutTTL(nil)
	}

	// Only try compression if it's enabled
	if c.config.Compression != nil && c.config.Compression.Enabled {
		// Serialize and compress the value
		compressed, isCompressed, err := compression.SerializeAndCompress(
			value,
			c.compressor,
			c.config.Compression.MinSize,
		)
		if err != nil {
			return nil, err
		}

		if isCompressed {
			// Store compressed data and metadata
			cacheEntry.Value = compressed

			// Calculate original size by serializing without compression
			serialized, _, serErr := compression.SerializeAndCompress(value, compression.NewNoOpCompressor(), 0)
			originalSize := len(serialized)
			if serErr != nil {
				// Fallback to approximate size if serialization fails
				originalSize = c.approximateSize(value)
			}

			cacheEntry.SetCompressionInfo(c.compressor.Name(), originalSize, len(compressed))
		} else {
			// Store uncompressed data
			cacheEntry.Value = compressed // This is actually the uncompressed serialized data
		}
	} else {
		// No compression, store value directly
		cacheEntry.Value = value
	}

	return cacheEntry, nil
}

// decompressValue decompresses a cached value if needed
func (c *Cache) decompressValue(entry *entry.Entry) (any, error) {
	// Check if compression was used during storage
	if c.config.Compression != nil && c.config.Compression.Enabled {
		// Value was stored with compression logic (might be compressed or serialized)
		data, ok := entry.Value.([]byte)
		if !ok {
			return nil, fmt.Errorf("serialized value is not []byte")
		}

		var result any
		err := compression.DecompressAndDeserialize(data, entry.IsCompressed, c.compressor, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize value: %w", err)
		}

		return result, nil
	} else {
		// No compression was configured, return value directly
		return entry.Value, nil
	}
}

// approximateSize estimates the memory size of a value
func (c *Cache) approximateSize(value any) int {
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case int, int8, int16, int32, int64:
		return 8
	case uint, uint8, uint16, uint32, uint64:
		return 8
	case float32:
		return 4
	case float64:
		return 8
	case bool:
		return 1
	default:
		// For complex types, use reflection to estimate
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			return rv.Len() * 8 // Rough estimate
		case reflect.Map:
			return rv.Len() * 16 // Rough estimate for key-value pairs
		case reflect.Struct:
			return rv.NumField() * 8 // Rough estimate
		default:
			return 64 // Default fallback
		}
	}
}

// initializeCompression sets up compression if enabled
func (c *Cache) initializeCompression() error {
	if c.config.Compression == nil {
		c.config.Compression = compression.NewDefaultConfig()
	}

	compressor, err := compression.NewCompressor(c.config.Compression)
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	c.compressor = compressor
	return nil
}

// initializeMetrics sets up metrics collection if enabled
func (c *Cache) initializeMetrics() error {
	if c.config.Metrics == nil || !c.config.Metrics.Enabled || c.config.Metrics.Exporter == nil {
		c.metricsExporter = metrics.NewNoOpExporter()
		return nil
	}

	c.metricsExporter = c.config.Metrics.Exporter

	// Prepare metrics labels with cache name
	c.metricsLabels = make(metrics.Labels)
	if c.config.Metrics.CacheName != "" {
		c.metricsLabels["cache_name"] = c.config.Metrics.CacheName
	} else {
		c.metricsLabels["cache_name"] = "default"
	}

	// Add any additional labels from config
	for k, v := range c.config.Metrics.Labels {
		c.metricsLabels[k] = v
	}

	// Start automatic stats reporting if interval is configured
	if c.config.Metrics.ReportingInterval > 0 {
		c.metricsStop = make(chan struct{})
		c.metricsWg.Add(1)
		go c.metricsReporter()
	}

	return nil
}

// metricsReporter periodically exports cache statistics
func (c *Cache) metricsReporter() {
	defer c.metricsWg.Done()

	ticker := time.NewTicker(c.config.Metrics.ReportingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.exportCurrentStats()
		case <-c.metricsStop:
			// Final stats export before shutting down
			c.exportCurrentStats()
			return
		}
	}
}

// exportCurrentStats exports the current statistics to metrics
func (c *Cache) exportCurrentStats() {
	if c.metricsExporter != nil {
		c.metricsExporter.ExportStats(c.stats, c.metricsLabels)
	}
}

// recordCacheOperation records a cache operation with timing for metrics
func (c *Cache) recordCacheOperation(operation metrics.Operation, duration time.Duration) {
	if c.metricsExporter != nil {
		c.metricsExporter.RecordCacheOperation(operation, duration, c.metricsLabels)
	}
}
