package ratelimit

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	cleanupInterval = time.Minute
	idleTTL         = 3 * time.Minute
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Limiter struct {
	mu      sync.Mutex
	clients map[string]*client
	rps     rate.Limit
	burst   int
}

func New(rps float64, burst int) (*Limiter, error) {
	if rps <= 0 || math.IsNaN(rps) || math.IsInf(rps, 0) || burst <= 0 {
		return nil, fmt.Errorf("rate limiter requires finite positive rps and burst")
	}

	return &Limiter{
		clients: make(map[string]*client),
		rps:     rate.Limit(rps),
		burst:   burst,
	}, nil
}

func (l *Limiter) Allow(key string) bool {
	if key == "" {
		return false
	}

	l.mu.Lock()
	c, ok := l.clients[key]
	if !ok {
		c = &client{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.clients[key] = c
	}
	c.lastSeen = time.Now()
	l.mu.Unlock()

	return c.limiter.Allow()
}

func (l *Limiter) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			l.removeIdle(now)
		}
	}
}

func (l *Limiter) removeIdle(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for key, c := range l.clients {
		if now.Sub(c.lastSeen) > idleTTL {
			delete(l.clients, key)
		}
	}
}
