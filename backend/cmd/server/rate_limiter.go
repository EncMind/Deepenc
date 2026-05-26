package main

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

func NewRateLimiter(requestsPerMinute int, burst int, ttl time.Duration) *RateLimiter {
	if requestsPerMinute <= 0 {
		return nil
	}

	limit := rate.Every(time.Minute / time.Duration(requestsPerMinute))
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     limit,
		burst:    burst,
		ttl:      ttl,
	}

	go rl.cleanupVisitors()

	return rl
}

func (rl *RateLimiter) getVisitor(id string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[id]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[id] = &visitor{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for id, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.ttl {
				delete(rl.visitors, id)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(id string) bool {
	if rl == nil {
		return true
	}

	return rl.getVisitor(id).Allow()
}
