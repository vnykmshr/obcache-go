package eviction

import (
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/vnykmshr/obcache-go/internal/entry"
)

// LRUStrategy implements the LRU (Least Recently Used) eviction strategy
type LRUStrategy struct {
	cache    *lru.Cache[string, *entry.Entry]
	capacity int
	mutex    sync.RWMutex
}

// NewLRUStrategy creates a new LRU eviction strategy
func NewLRUStrategy(capacity int) *LRUStrategy {
	cache, err := lru.New[string, *entry.Entry](capacity)
	if err != nil {
		// This should not happen with valid capacity, but fallback gracefully
		panic("failed to create LRU cache: " + err.Error())
	}

	return &LRUStrategy{
		cache:    cache,
		capacity: capacity,
	}
}

// Add adds an entry to the LRU tracker
func (l *LRUStrategy) Add(key string, entry *entry.Entry) (string, bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	evicted := l.cache.Add(key, entry)
	return "", evicted // LRU library handles eviction internally
}

// Get retrieves an entry and marks it as recently used
func (l *LRUStrategy) Get(key string) (*entry.Entry, bool) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.cache.Get(key)
}

// Remove removes an entry from the LRU tracker
func (l *LRUStrategy) Remove(key string) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.cache.Remove(key)
}

// Contains checks if a key exists in the LRU tracker
func (l *LRUStrategy) Contains(key string) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.cache.Contains(key)
}

// Keys returns all keys currently tracked by the LRU strategy
func (l *LRUStrategy) Keys() []string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.cache.Keys()
}

// Len returns the number of entries currently tracked
func (l *LRUStrategy) Len() int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.cache.Len()
}

// Clear removes all entries from the LRU tracker
func (l *LRUStrategy) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.cache.Purge()
}

// Capacity returns the maximum number of entries this strategy can hold
func (l *LRUStrategy) Capacity() int {
	return l.capacity
}

// Peek retrieves an entry without marking it as recently used
func (l *LRUStrategy) Peek(key string) (*entry.Entry, bool) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.cache.Peek(key)
}
