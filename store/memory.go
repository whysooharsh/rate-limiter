package store

import (
	"sync"
)

type MemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*limiter.tokenBucket
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		buckets: make(map[string]*limiter.tokenBucket),
	}
}

func (m *MemoryStore) AddClient(clientId string, maxTokens int, refillRate int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets[clientId] = limiter.newTokenBucket(maxTokens, refillRate)
}

func (m *MemoryStore) Allow(clientID string) bool {

	m.mu.Lock()
	bucket, exists := m.buckets[clientID]
	m.mu.Unlock()

	if !exists {
		return false
	}
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
