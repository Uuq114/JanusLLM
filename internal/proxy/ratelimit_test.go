package proxy

import (
	"testing"
	"time"
)

func TestRequestRingUnlimited(t *testing.T) {
	ring := NewRequestRing(time.Minute, 0)
	for i := 0; i < 1000; i++ {
		if !ring.Allow() {
			t.Fatalf("expected unlimited ring to allow requests")
		}
	}
}

func TestRequestRingLimit(t *testing.T) {
	ring := NewRequestRing(time.Minute, 2)
	if !ring.Allow() {
		t.Fatalf("first request should be allowed")
	}
	if !ring.Allow() {
		t.Fatalf("second request should be allowed")
	}
	if ring.Allow() {
		t.Fatalf("third request should be blocked")
	}
}

func TestRequestRingRecoversAfterWindow(t *testing.T) {
	ring := NewRequestRing(time.Minute, 1)
	start := time.Unix(1700000000, 0)

	allowed, retryAfter := ring.AllowAt(start)
	if !allowed || retryAfter != 0 {
		t.Fatalf("first request should be allowed, got allowed=%v retry_after=%v", allowed, retryAfter)
	}

	allowed, retryAfter = ring.AllowAt(start.Add(30 * time.Second))
	if allowed {
		t.Fatalf("second request inside window should be blocked")
	}
	if retryAfter <= 0 || retryAfter > 30*time.Second {
		t.Fatalf("unexpected retry_after: %v", retryAfter)
	}

	allowed, retryAfter = ring.AllowAt(start.Add(61 * time.Second))
	if !allowed || retryAfter != 0 {
		t.Fatalf("request after window should be allowed, got allowed=%v retry_after=%v", allowed, retryAfter)
	}
}
