package limiter

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
}

func NewTokenBucket(maxTokens int, refillRate int) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := time.Since(tb.lastRefill).Seconds()
	tokensToAdd := int(elapsed) * tb.refillRate

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefill = now
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

func (tb *TokenBucket) GetStatus() (int, int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens, tb.maxTokens
}
