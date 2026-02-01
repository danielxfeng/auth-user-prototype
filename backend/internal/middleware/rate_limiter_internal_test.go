package middleware

import (
	"testing"
	"time"
)

func TestAllowRequestCleansExpiredEntriesAtInterval(t *testing.T) {
	rl := NewRateLimiter(10*time.Millisecond, 1, 5*time.Millisecond)

	now := time.Now()
	rl.requestCounts["old"] = 2
	rl.requestExpiry["old"] = now.Add(-time.Second)
	rl.lastCleanup = now.Add(-rl.cleanupInterval - time.Second)

	_ = rl.AllowRequest("new-client")

	if _, exists := rl.requestCounts["old"]; exists {
		t.Fatalf("expected expired request count to be removed during cleanup")
	}
	if _, exists := rl.requestExpiry["old"]; exists {
		t.Fatalf("expected expired request expiry to be removed during cleanup")
	}
}

func TestUnsafeClearExpiredEntriesRemovesOnlyExpired(t *testing.T) {
	rl := NewRateLimiter(10*time.Millisecond, 1, time.Minute)

	now := time.Now()
	rl.requestCounts["expired"] = 1
	rl.requestExpiry["expired"] = now.Add(-time.Second)
	rl.requestCounts["active"] = 1
	rl.requestExpiry["active"] = now.Add(time.Second)

	unSafeClearExpiredEntries(now, rl)

	if _, exists := rl.requestCounts["expired"]; exists {
		t.Fatalf("expected expired request count to be removed")
	}
	if _, exists := rl.requestExpiry["expired"]; exists {
		t.Fatalf("expected expired request expiry to be removed")
	}
	if _, exists := rl.requestCounts["active"]; !exists {
		t.Fatalf("expected active request count to remain")
	}
	if _, exists := rl.requestExpiry["active"]; !exists {
		t.Fatalf("expected active request expiry to remain")
	}
}
