package limiter

type Store interface {
	Allow(clientId string) bool
	GetStatus(clientId string) (int, int)
}

type Config struct {
	MaxTokens  int
	RefillRate int
}
