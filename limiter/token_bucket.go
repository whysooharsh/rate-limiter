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
	if tb.refillRate <= 0 {
		return
	}
	durationPerToken := time.Second / time.Duration(tb.refillRate)
	if durationPerToken == 0 {
		durationPerToken = 1
	}
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

func (tb *TokenBucket) Allow() (bool, int, int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true, tb.tokens, tb.maxTokens
	}
	return false, tb.tokens, tb.maxTokens
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

	if tb.tokens > tb.maxTokens {
		tb.tokens = maxTokens
	}
}
