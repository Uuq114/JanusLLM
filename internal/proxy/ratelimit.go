package proxy

import (
	"sync"
	"time"
)

type RequestRing struct {
	window      time.Duration
	maxRequests int
	requests    []time.Time
	unlimited   bool
	mu          sync.Mutex
}

func NewRequestRing(window time.Duration, maxRequests int) *RequestRing {
	if maxRequests <= 0 {
		return &RequestRing{
			window:      window,
			maxRequests: 0,
			unlimited:   true,
		}
	}

	return &RequestRing{
		window:      window,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0, maxRequests),
	}
}

func (r *RequestRing) Allow() bool {
	allowed, _ := r.AllowAt(time.Now())
	return allowed
}

func (r *RequestRing) AllowAt(now time.Time) (bool, time.Duration) {
	if r.unlimited {
		return true, 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.pruneLocked(now)
	if len(r.requests) >= r.maxRequests {
		retryAfter := r.requests[0].Add(r.window).Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	r.requests = append(r.requests, now)
	return true, 0
}

func (r *RequestRing) pruneLocked(now time.Time) {
	cutoff := now.Add(-r.window)
	firstValid := 0
	for firstValid < len(r.requests) {
		if r.requests[firstValid].After(cutoff) {
			break
		}
		firstValid++
	}
	if firstValid == 0 {
		return
	}
	r.requests = append(r.requests[:0], r.requests[firstValid:]...)
}
