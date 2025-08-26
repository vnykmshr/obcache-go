package eviction

import (
	"github.com/vnykmshr/obcache-go/internal/entry"
)

// Strategy defines the interface for eviction strategies
type Strategy interface {
	// Add adds an entry to the eviction strategy tracker
	// Returns the key of an entry to evict if capacity is exceeded, empty string otherwise
	Add(key string, entry *entry.Entry) (evictKey string, evicted bool)

	// Get retrieves an entry and updates its position in the eviction order
	Get(key string) (*entry.Entry, bool)

	// Remove removes an entry from the eviction strategy tracker
	Remove(key string) bool

	// Contains checks if a key exists in the eviction strategy tracker
	Contains(key string) bool

	// Keys returns all keys currently tracked by the strategy
	Keys() []string

	// Len returns the number of entries currently tracked
	Len() int

	// Clear removes all entries from the strategy
	Clear()

	// Capacity returns the maximum number of entries this strategy can hold
	Capacity() int

	// Peek retrieves an entry without updating its position in the eviction order
	Peek(key string) (*entry.Entry, bool)
}

// EvictionType represents the type of eviction strategy
type EvictionType string

const (
	// LRU - Least Recently Used eviction
	LRU EvictionType = "lru"

	// LFU - Least Frequently Used eviction
	LFU EvictionType = "lfu"

	// FIFO - First In, First Out eviction
	FIFO EvictionType = "fifo"
)

// Config holds configuration for eviction strategies
type Config struct {
	Type     EvictionType
	Capacity int
}

// NewStrategy creates a new eviction strategy based on the given config
func NewStrategy(config Config) Strategy {
	switch config.Type {
	case LRU:
		return NewLRUStrategy(config.Capacity)
	case LFU:
		return NewLFUStrategy(config.Capacity)
	case FIFO:
		return NewFIFOStrategy(config.Capacity)
	default:
		// Default to LRU
		return NewLRUStrategy(config.Capacity)
	}
}
