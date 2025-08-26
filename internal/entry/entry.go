package entry

import (
	"sync"
	"time"
)

// Entry represents a cache entry with value and TTL information
type Entry struct {
	// Value is the cached value (may be compressed bytes if IsCompressed is true)
	Value any

	// ExpiresAt indicates when this entry expires (nil means no expiration)
	ExpiresAt *time.Time

	// CreatedAt is when this entry was created
	CreatedAt time.Time

	// AccessedAt is when this entry was last accessed (for LRU)
	// Protected by mu for concurrent access
	AccessedAt time.Time
	mu         sync.RWMutex

	// Compression metadata
	IsCompressed     bool   // Whether the value is compressed
	CompressorName   string // Name of the compressor used (for debugging/metrics)
	OriginalSize     int    // Original size before compression (0 if not compressed)
	CompressedSize   int    // Size after compression (0 if not compressed)
}

// New creates a new cache entry with the given value and TTL
func New(value any, ttl time.Duration) *Entry {
	now := time.Now()
	entry := &Entry{
		Value:      value,
		CreatedAt:  now,
		AccessedAt: now,
	}

	if ttl > 0 {
		expiry := now.Add(ttl)
		entry.ExpiresAt = &expiry
	}

	return entry
}

// NewWithoutTTL creates a new cache entry without expiration
func NewWithoutTTL(value any) *Entry {
	now := time.Now()
	return &Entry{
		Value:      value,
		ExpiresAt:  nil,
		CreatedAt:  now,
		AccessedAt: now,
	}
}

// IsExpired returns true if the entry has expired
func (e *Entry) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}

// TTL returns the time remaining until expiration
// Returns 0 if the entry has no expiration or has already expired
func (e *Entry) TTL() time.Duration {
	if e.ExpiresAt == nil {
		return 0 // No expiration
	}

	remaining := time.Until(*e.ExpiresAt)
	if remaining < 0 {
		return 0 // Already expired
	}

	return remaining
}

// Age returns how long ago this entry was created
func (e *Entry) Age() time.Duration {
	return time.Since(e.CreatedAt)
}

// TimeSinceLastAccess returns how long ago this entry was last accessed
func (e *Entry) TimeSinceLastAccess() time.Duration {
	e.mu.RLock()
	accessedAt := e.AccessedAt
	e.mu.RUnlock()
	return time.Since(accessedAt)
}

// Touch updates the last accessed time to now
func (e *Entry) Touch() {
	e.mu.Lock()
	e.AccessedAt = time.Now()
	e.mu.Unlock()
}

// UpdateExpiry updates the expiration time with a new TTL from now
func (e *Entry) UpdateExpiry(ttl time.Duration) {
	if ttl > 0 {
		expiry := time.Now().Add(ttl)
		e.ExpiresAt = &expiry
	} else {
		e.ExpiresAt = nil
	}
}

// HasExpiry returns true if the entry has an expiration time set
func (e *Entry) HasExpiry() bool {
	return e.ExpiresAt != nil
}

// String returns a string representation of the entry (for debugging)
func (e *Entry) String() string {
	status := "Entry{"
	if e.IsCompressed {
		status += "compressed, "
	}
	if e.ExpiresAt == nil {
		status += "no-expiry}"
	} else {
		status += "expires: " + e.ExpiresAt.Format(time.RFC3339) + "}"
	}
	return status
}

// CompressionRatio returns the compression ratio (original/compressed)
// Returns 1.0 if not compressed
func (e *Entry) CompressionRatio() float64 {
	if !e.IsCompressed || e.CompressedSize == 0 {
		return 1.0
	}
	return float64(e.OriginalSize) / float64(e.CompressedSize)
}

// SpaceSaved returns the bytes saved through compression
// Returns 0 if not compressed
func (e *Entry) SpaceSaved() int {
	if !e.IsCompressed {
		return 0
	}
	return e.OriginalSize - e.CompressedSize
}

// SetCompressionInfo sets compression metadata for the entry
func (e *Entry) SetCompressionInfo(compressorName string, originalSize, compressedSize int) {
	e.IsCompressed = true
	e.CompressorName = compressorName
	e.OriginalSize = originalSize
	e.CompressedSize = compressedSize
}
