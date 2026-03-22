package store

import (
	"sync"
)

type ClientBucket struct {
	tokens     int
	maxTokens  int
	refillRate int
}

type MemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*ClientBucket
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		buckets: make(map[string]*ClientBucket),
	}
}

func (m *MemoryStore) AddClient(clientId string, maxTokens int, refillRate int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets[clientId] = &ClientBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}
