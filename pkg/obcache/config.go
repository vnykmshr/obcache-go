package obcache

import (
	"time"
)

// Config defines the configuration options for a Cache instance
type Config struct {
	// MaxEntries sets the maximum number of entries in the cache (LRU)
	// Default: 1000
	MaxEntries int

	// DefaultTTL sets the default time-to-live for cache entries
	// Default: 5 minutes
	DefaultTTL time.Duration

	// CleanupInterval sets how often expired entries are cleaned up
	// Default: 1 minute
	CleanupInterval time.Duration

	// KeyGenFunc defines a custom key generation function
	// If nil, DefaultKeyFunc will be used
	KeyGenFunc KeyGenFunc

	// Hooks defines event callbacks for cache operations
	Hooks *Hooks
}

// KeyGenFunc defines a function that generates cache keys from function arguments
type KeyGenFunc func(args []any) string

// NewDefaultConfig returns a Config with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		MaxEntries:      1000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: time.Minute,
		KeyGenFunc:      nil, // will use DefaultKeyFunc
		Hooks:          &Hooks{},
	}
}

// WithMaxEntries sets the maximum number of cache entries
func (c *Config) WithMaxEntries(max int) *Config {
	c.MaxEntries = max
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