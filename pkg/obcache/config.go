package obcache

import (
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vnykmshr/obcache-go/pkg/compression"
	"github.com/vnykmshr/obcache-go/pkg/metrics"
)

// StoreType defines the type of backend store to use
type StoreType int

const (
	// StoreTypeMemory uses in-memory storage (default)
	StoreTypeMemory StoreType = iota
	// StoreTypeRedis uses Redis as backend storage
	StoreTypeRedis
)

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	// Client is a pre-configured Redis client
	// If nil, a new client will be created using Addr, Password, DB
	Client redis.Cmdable

	// Addr is the Redis server address (host:port)
	// Only used if Client is nil
	Addr string

	// Password for Redis authentication
	// Only used if Client is nil
	Password string

	// DB is the Redis database number to use
	// Only used if Client is nil
	DB int

	// KeyPrefix is prepended to all cache keys
	// Default: "obcache:"
	KeyPrefix string
}

// MetricsConfig holds metrics exporter configuration
type MetricsConfig struct {
	// Exporter is the metrics exporter to use
	Exporter metrics.Exporter

	// Enabled determines whether metrics collection is enabled
	Enabled bool

	// CacheName is the name label applied to all metrics for this cache instance
	CacheName string

	// ReportingInterval determines how often to export stats automatically
	// Set to 0 to disable automatic reporting
	ReportingInterval time.Duration

	// Labels are additional labels applied to all metrics
	Labels metrics.Labels
}

// Config defines the configuration options for a Cache instance
type Config struct {
	// StoreType determines which backend store to use
	// Default: StoreTypeMemory
	StoreType StoreType

	// MaxEntries sets the maximum number of entries in the cache (LRU)
	// Only applies to memory store
	// Default: 1000
	MaxEntries int

	// DefaultTTL sets the default time-to-live for cache entries
	// Default: 5 minutes
	DefaultTTL time.Duration

	// CleanupInterval sets how often expired entries are cleaned up
	// Only applies to memory store (Redis handles TTL automatically)
	// Default: 1 minute
	CleanupInterval time.Duration

	// KeyGenFunc defines a custom key generation function
	// If nil, DefaultKeyFunc will be used
	KeyGenFunc KeyGenFunc

	// Hooks defines event callbacks for cache operations
	Hooks *Hooks

	// Redis holds Redis-specific configuration
	// Only used when StoreType is StoreTypeRedis
	Redis *RedisConfig

	// Metrics holds metrics exporter configuration
	// If nil, no metrics will be exported
	Metrics *MetricsConfig

	// Compression holds compression configuration
	// If nil, compression will be disabled
	Compression *compression.Config
}

// KeyGenFunc defines a function that generates cache keys from function arguments
type KeyGenFunc func(args []any) string

// NewDefaultConfig returns a Config with sensible defaults for memory storage
func NewDefaultConfig() *Config {
	return &Config{
		StoreType:       StoreTypeMemory,
		MaxEntries:      1000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: time.Minute,
		KeyGenFunc:      nil, // will use DefaultKeyFunc
		Hooks:           &Hooks{},
		Redis:           nil,
	}
}

// NewRedisConfig returns a Config configured for Redis storage
func NewRedisConfig(addr string) *Config {
	return &Config{
		StoreType:       StoreTypeRedis,
		MaxEntries:      0, // Not applicable for Redis
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 0, // Redis handles TTL automatically
		KeyGenFunc:      nil, // will use DefaultKeyFunc
		Hooks:           &Hooks{},
		Redis: &RedisConfig{
			Addr:      addr,
			KeyPrefix: "obcache:",
		},
	}
}

// NewRedisConfigWithClient returns a Config configured for Redis with a pre-configured client
func NewRedisConfigWithClient(client redis.Cmdable) *Config {
	return &Config{
		StoreType:       StoreTypeRedis,
		MaxEntries:      0, // Not applicable for Redis
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 0, // Redis handles TTL automatically
		KeyGenFunc:      nil, // will use DefaultKeyFunc
		Hooks:           &Hooks{},
		Redis: &RedisConfig{
			Client:    client,
			KeyPrefix: "obcache:",
		},
	}
}

// WithMaxEntries sets the maximum number of cache entries
func (c *Config) WithMaxEntries(maxEntries int) *Config {
	c.MaxEntries = maxEntries
	return c
}

// WithDefaultTTL sets the default TTL for cache entries
func (c *Config) WithDefaultTTL(ttl time.Duration) *Config {
	c.DefaultTTL = ttl
	return c
}

// WithCleanupInterval sets the cleanup interval for expired entries
func (c *Config) WithCleanupInterval(interval time.Duration) *Config {
	c.CleanupInterval = interval
	return c
}

// WithKeyGenFunc sets a custom key generation function
func (c *Config) WithKeyGenFunc(fn KeyGenFunc) *Config {
	c.KeyGenFunc = fn
	return c
}

// WithHooks sets the event hooks for cache operations
func (c *Config) WithHooks(hooks *Hooks) *Config {
	c.Hooks = hooks
	return c
}

// WithRedis configures the cache to use Redis storage
func (c *Config) WithRedis(redisConfig *RedisConfig) *Config {
	c.StoreType = StoreTypeRedis
	c.Redis = redisConfig
	// Disable memory-specific settings when using Redis
	c.MaxEntries = 0
	c.CleanupInterval = 0
	return c
}

// WithRedisAddr configures the cache to use Redis with the given address
func (c *Config) WithRedisAddr(addr string) *Config {
	return c.WithRedis(&RedisConfig{
		Addr:      addr,
		KeyPrefix: "obcache:",
	})
}

// WithRedisClient configures the cache to use Redis with a pre-configured client
func (c *Config) WithRedisClient(client redis.Cmdable) *Config {
	return c.WithRedis(&RedisConfig{
		Client:    client,
		KeyPrefix: "obcache:",
	})
}

// WithRedisAuth sets Redis authentication for the current Redis config
func (c *Config) WithRedisAuth(password string) *Config {
	if c.Redis == nil {
		c.Redis = &RedisConfig{KeyPrefix: "obcache:"}
	}
	c.Redis.Password = password
	return c
}

// WithRedisDB sets the Redis database number
func (c *Config) WithRedisDB(db int) *Config {
	if c.Redis == nil {
		c.Redis = &RedisConfig{KeyPrefix: "obcache:"}
	}
	c.Redis.DB = db
	return c
}

// WithRedisKeyPrefix sets the Redis key prefix
func (c *Config) WithRedisKeyPrefix(prefix string) *Config {
	if c.Redis == nil {
		c.Redis = &RedisConfig{}
	}
	c.Redis.KeyPrefix = prefix
	return c
}

// WithMetrics configures cache metrics export
func (c *Config) WithMetrics(metricsConfig *MetricsConfig) *Config {
	c.Metrics = metricsConfig
	return c
}

// WithMetricsExporter configures metrics with the given exporter
func (c *Config) WithMetricsExporter(exporter metrics.Exporter, cacheName string) *Config {
	c.Metrics = &MetricsConfig{
		Exporter:          exporter,
		Enabled:           true,
		CacheName:         cacheName,
		ReportingInterval: 30 * time.Second,
		Labels:            make(metrics.Labels),
	}
	return c
}

// WithMetricsLabels adds labels to metrics configuration
func (c *Config) WithMetricsLabels(labels metrics.Labels) *Config {
	if c.Metrics == nil {
		c.Metrics = &MetricsConfig{
			Enabled:           false,
			Labels:            make(metrics.Labels),
			ReportingInterval: 30 * time.Second,
		}
	}
	for k, v := range labels {
		c.Metrics.Labels[k] = v
	}
	return c
}

// WithMetricsReportingInterval sets the metrics reporting interval
func (c *Config) WithMetricsReportingInterval(interval time.Duration) *Config {
	if c.Metrics == nil {
		c.Metrics = &MetricsConfig{
			Enabled:           false,
			Labels:            make(metrics.Labels),
			ReportingInterval: interval,
		}
	} else {
		c.Metrics.ReportingInterval = interval
	}
	return c
}

// WithCompression configures cache compression
func (c *Config) WithCompression(compressionConfig *compression.Config) *Config {
	c.Compression = compressionConfig
	return c
}

// WithCompressionEnabled enables compression with default settings
func (c *Config) WithCompressionEnabled(enabled bool) *Config {
	if c.Compression == nil {
		c.Compression = compression.NewDefaultConfig()
	}
	c.Compression.Enabled = enabled
	return c
}

// WithCompressionAlgorithm sets the compression algorithm
func (c *Config) WithCompressionAlgorithm(algorithm compression.CompressorType) *Config {
	if c.Compression == nil {
		c.Compression = compression.NewDefaultConfig()
	}
	c.Compression.Algorithm = algorithm
	return c
}

// WithCompressionMinSize sets the minimum size threshold for compression
func (c *Config) WithCompressionMinSize(minSize int) *Config {
	if c.Compression == nil {
		c.Compression = compression.NewDefaultConfig()
	}
	c.Compression.MinSize = minSize
	return c
}

// WithCompressionLevel sets the compression level
func (c *Config) WithCompressionLevel(level int) *Config {
	if c.Compression == nil {
		c.Compression = compression.NewDefaultConfig()
	}
	c.Compression.Level = level
	return c
}
