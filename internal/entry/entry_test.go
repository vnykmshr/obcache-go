package entry

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	value := "test-value"
	ttl := 10 * time.Second

	entry := New(value, ttl)

	if entry.Value != value {
		t.Fatalf("Expected value %v, got %v", value, entry.Value)
	}

	if entry.ExpiresAt == nil {
		t.Fatal("Expected ExpiresAt to be set")
	}

	expectedExpiry := time.Now().Add(ttl)
	if entry.ExpiresAt.Before(expectedExpiry.Add(-time.Second)) ||
		entry.ExpiresAt.After(expectedExpiry.Add(time.Second)) {
		t.Fatal("ExpiresAt not set correctly")
	}

	if entry.CreatedAt.IsZero() {
		t.Fatal("Expected CreatedAt to be set")
	}

	if entry.AccessedAt.IsZero() {
		t.Fatal("Expected AccessedAt to be set")
	}
}

func TestNewWithoutTTL(t *testing.T) {
	value := 42

	entry := NewWithoutTTL(value)

	if entry.Value != value {
		t.Fatalf("Expected value %v, got %v", value, entry.Value)
	}

	if entry.ExpiresAt != nil {
		t.Fatal("Expected ExpiresAt to be nil for no TTL")
	}

	if entry.CreatedAt.IsZero() {
		t.Fatal("Expected CreatedAt to be set")
	}

	if entry.AccessedAt.IsZero() {
		t.Fatal("Expected AccessedAt to be set")
	}
}

func TestIsExpired(t *testing.T) {
	// Test non-expired entry
	entry := New("value", time.Hour)
	if entry.IsExpired() {
		t.Fatal("Entry should not be expired")
	}

	// Test expired entry - create one that's already expired
	expiredEntry := &Entry{
		Value:      "value",
		ExpiresAt:  func() *time.Time { t := time.Now().Add(-time.Hour); return &t }(),
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}
	if !expiredEntry.IsExpired() {
		t.Fatal("Entry should be expired")
	}

	// Test entry without TTL
	noTTLEntry := NewWithoutTTL("value")
	if noTTLEntry.IsExpired() {
		t.Fatal("Entry without TTL should never expire")
	}
}

func TestTTL(t *testing.T) {
	// Test entry with TTL
	entry := New("value", time.Hour)
	ttl := entry.TTL()

	if ttl <= 0 || ttl > time.Hour {
		t.Fatalf("Expected TTL close to 1 hour, got %v", ttl)
	}

	// Test entry without TTL
	noTTLEntry := NewWithoutTTL("value")
	if noTTLEntry.TTL() != 0 {
		t.Fatal("Entry without TTL should return 0 TTL")
	}
}

func TestAge(t *testing.T) {
	entry := New("value", time.Hour)

	// Sleep briefly to ensure age is measurable
	time.Sleep(time.Millisecond)

	age := entry.Age()
	if age <= 0 {
		t.Fatal("Entry age should be positive")
	}

	if age > time.Second {
		t.Fatal("Entry age should be very small")
	}
}

func TestTimeSinceLastAccess(t *testing.T) {
	entry := New("value", time.Hour)

	// Sleep briefly to ensure time difference is measurable
	time.Sleep(time.Millisecond)

	timeSince := entry.TimeSinceLastAccess()
	if timeSince <= 0 {
		t.Fatal("Time since last access should be positive")
	}

	if timeSince > time.Second {
		t.Fatal("Time since last access should be very small")
	}
}

func TestTouch(t *testing.T) {
	entry := New("value", time.Hour)

	// Get initial access time
	initialAccess := entry.AccessedAt

	// Sleep and touch
	time.Sleep(2 * time.Millisecond)
	entry.Touch()

	// Access time should be updated
	if !entry.AccessedAt.After(initialAccess) {
		t.Fatal("Touch should update AccessedAt")
	}
}

func TestUpdateExpiry(t *testing.T) {
	entry := New("value", time.Hour)

	// Get initial expiry
	initialExpiry := *entry.ExpiresAt

	// Sleep and update expiry
	time.Sleep(time.Millisecond)
	newTTL := 2 * time.Hour
	entry.UpdateExpiry(newTTL)

	// Expiry should be updated
	if !entry.ExpiresAt.After(initialExpiry) {
		t.Fatal("UpdateExpiry should update ExpiresAt")
	}

	// Test with zero TTL (should remove expiry)
	entry.UpdateExpiry(0)
	if entry.ExpiresAt != nil {
		t.Fatal("UpdateExpiry with 0 TTL should remove expiry")
	}
}

func TestHasExpiry(t *testing.T) {
	// Test entry with TTL
	entry := New("value", time.Hour)
	if !entry.HasExpiry() {
		t.Fatal("Entry with TTL should have expiry")
	}

	// Test entry without TTL
	noTTLEntry := NewWithoutTTL("value")
	if noTTLEntry.HasExpiry() {
		t.Fatal("Entry without TTL should not have expiry")
	}
}

func TestString(t *testing.T) {
	// Test entry with TTL
	entry := New("test-value", time.Hour)
	str := entry.String()
	if str == "" {
		t.Fatal("String should return non-empty representation")
	}
	if !contains(str, "expires") {
		t.Fatal("String should contain expiry information")
	}

	// Test entry without TTL
	noTTLEntry := NewWithoutTTL("test-value")
	noTTLStr := noTTLEntry.String()
	if !contains(noTTLStr, "no-expiry") {
		t.Fatal("String should contain no-expiry information")
	}
}

func TestConcurrentTouch(_ *testing.T) {
	entry := New("value", time.Hour)

	// Test concurrent access to Touch method
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			entry.Touch()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race conditions
}

func TestConcurrentTimeSinceLastAccess(_ *testing.T) {
	entry := New("value", time.Hour)

	// Test concurrent access to TimeSinceLastAccess method
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_ = entry.TimeSinceLastAccess()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race conditions
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexContains(s, substr))))
}

func indexContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
