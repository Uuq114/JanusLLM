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
