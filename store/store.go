package store

type Store interface {
	Allow(clientID string) (bool, int, int)
	GetStatus(clientID string) (int, int, bool)
	SetClient(clientID string, maxTokens int, refillRate int)
}
