package store

import (
	"sync"
	"time"

	"github.com/whysooharsh/rate-limiter/limiter"
)

const (
	DefaultMaxTokens  = 10
	DefaultRefillRate = 1
	maxClients        = 10000
	idleTTL           = 10 * time.Minute
	evictInterval     = 1 * time.Minute
)

type bucketEntry struct {
	bucket   *limiter.TokenBucket
	lastSeen time.Time
}

type MemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*bucketEntry
}

func NewMemoryStore() *MemoryStore {

	m := &MemoryStore{
		buckets: make(map[string]*bucketEntry),
	}
	go m.evictLoop()
	return m
}

func (m *MemoryStore) evictLoop() {

	ticker := time.NewTicker(evictInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		m.mu.Lock()
		for clientID, entry := range m.buckets {
			if now.Sub(entry.lastSeen) >= idleTTL {
				delete(m.buckets, clientID)
			}
		}
		m.mu.Unlock()
	}

}

func (m *MemoryStore) AddClient(clientId string, maxTokens int, refillRate int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets[clientId] = &bucketEntry{
		bucket:   limiter.NewTokenBucket(maxTokens, refillRate),
		lastSeen: time.Now(),
	}
}

func (m *MemoryStore) Allow(clientID string) (bool, int, int) {

	m.mu.Lock()
	bucket, exists := m.buckets[clientID]

	if !exists {
		if len(m.buckets) >= maxClients {
			m.mu.Unlock()
			return false, 0, 0
		}
		bucket = &bucketEntry{
			bucket:   limiter.NewTokenBucket(DefaultMaxTokens, DefaultRefillRate),
			lastSeen: time.Now(),
		}
		m.buckets[clientID] = bucket
	} else {
		bucket.lastSeen = time.Now()
	}
	m.mu.Unlock()

	return bucket.bucket.Allow()

}

func (m *MemoryStore) GetStatus(clientID string) (int, int) {
	m.mu.Lock()
	bucket, exists := m.buckets[clientID]
	if exists {
		bucket.lastSeen = time.Now()
	}
	m.mu.Unlock()

	if !exists {
		return 0, 0
	}

	return bucket.bucket.GetStatus()
}

func (m *MemoryStore) SetClient(clientID string, maxTokens int, refillRate int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bucket, exists := m.buckets[clientID]

	if !exists {
		m.buckets[clientID] = &bucketEntry{
			bucket:   limiter.NewTokenBucket(maxTokens, refillRate),
			lastSeen: time.Now(),
		}
	} else {
		bucket.bucket.UpdateConfig(maxTokens, refillRate)
		bucket.lastSeen = time.Now()
	}
}
