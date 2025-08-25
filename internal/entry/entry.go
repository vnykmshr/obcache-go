package entry

import (
	"time"
)

// Entry represents a cache entry with value and TTL information
type Entry struct {
	// Value is the cached value
	Value any

	// ExpiresAt indicates when this entry expires (nil means no expiration)
	ExpiresAt *time.Time

	// CreatedAt is when this entry was created
	CreatedAt time.Time

	// AccessedAt is when this entry was last accessed (for LRU)
	AccessedAt time.Time
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
	return time.Since(e.AccessedAt)
}

// Touch updates the last accessed time to now
func (e *Entry) Touch() {
	e.AccessedAt = time.Now()
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
	if e.ExpiresAt == nil {
		return "Entry{no-expiry}"
	}
	return "Entry{expires: " + e.ExpiresAt.Format(time.RFC3339) + "}"
}