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
	durationPerToken := time.Second / time.Duration(tb.refillRate)
	elapsed := time.Since(tb.lastRefill)
	tokensToAdd := int(elapsed / durationPerToken)

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefill = tb.lastRefill.Add(time.Duration(tokensToAdd) * durationPerToken)
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
	tb.refill()
	return tb.tokens, tb.maxTokens
}

func (tb *TokenBucket) UpdateConfig(maxTokens int, refillRate int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.maxTokens = maxTokens
	tb.refillRate = refillRate
}
