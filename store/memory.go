package store

import (
	"sync"

	"github.com/whysooharsh/rate-limiter/limiter"
)

const (
	DefaultMaxTokens  = 10
	DefaultRefillRate = 1
)

type MemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*limiter.TokenBucket
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		buckets: make(map[string]*limiter.TokenBucket),
	}
}

func (m *MemoryStore) AddClient(clientId string, maxTokens int, refillRate int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets[clientId] = limiter.NewTokenBucket(maxTokens, refillRate)
}

func (m *MemoryStore) Allow(clientID string) bool {

	m.mu.Lock()
	bucket, exists := m.buckets[clientID]

	if !exists {
		if len(m.buckets) >= 10000 {
			m.mu.Unlock()
			return false
		}
		bucket = limiter.NewTokenBucket(DefaultMaxTokens, DefaultRefillRate)
		m.buckets[clientID] = bucket
	}
	m.mu.Unlock()

	return bucket.Allow()

}

func (m *MemoryStore) GetStatus(clientID string) (int, int) {
	m.mu.Lock()
	bucket, exists := m.buckets[clientID]
	m.mu.Unlock()

	if !exists {
		return 0, 0
	}

	return bucket.GetStatus()
}

func (m *MemoryStore) SetClient(clientID string, maxTokens int, refillRate int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.buckets[clientID]

	if !exists {
		m.buckets[clientID] = limiter.NewTokenBucket(maxTokens, refillRate)
	} else {
		m.buckets[clientID].UpdateConfig(maxTokens, refillRate)
	}
}
