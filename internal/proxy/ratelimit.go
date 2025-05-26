package proxy

import (
	"sync"
	"time"
)

type RequestRing struct {
	window      time.Duration
	maxRequests int
	bufferSize  int
	ring        []time.Time
	writePos    int
	mu          sync.RWMutex
}

func NewRequestRing(window time.Duration, maxRequests int) *RequestRing {
	return &RequestRing{
		window:      window,
		maxRequests: maxRequests,
		bufferSize:  maxRequests + 10, // 多预留一点防止边界问题
		ring:        make([]time.Time, maxRequests+10),
	}
}

func (r *RequestRing) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	count := 0
	for i := 0; i < r.bufferSize; i++ {
		if r.ring[i].After(cutoff) {
			count++
		}
	}

	if count >= r.maxRequests {
		return false
	}

	r.ring[r.writePos] = now
	r.writePos = (r.writePos + 1) % r.bufferSize
	return true
}
