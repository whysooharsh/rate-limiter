package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStore(url string) *RedisStore {
	opts, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	return &RedisStore{
		client: redis.NewClient(opts),
		ctx:    context.Background(),
	}
}

const luaScript = `
local key = KEYS[1]
local default_max_tokens = tonumber(ARGV[1])
local default_refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill', 'max_tokens', 'refill_rate')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])
local max_tokens = tonumber(bucket[3]) or default_max_tokens
local refill_rate = tonumber(bucket[4]) or default_refill_rate

if tokens == nil then
    tokens = max_tokens
    last_refill = now
end

local elapsed = math.max(0, now - last_refill)
local refill = 0
if refill_rate > 0 then
    local time_per_token = 1000 / refill_rate
    if time_per_token > 0 then
        refill = math.floor(elapsed / time_per_token)
        if refill > 0 then
            tokens = math.min(max_tokens, tokens + refill)
            last_refill = last_refill + math.floor(refill * time_per_token)
        end
    else
        tokens = max_tokens
        last_refill = now
    end
end

local allowed = 0
if tokens > 0 then
    tokens = tokens - 1
    allowed = 1
end

redis.call('HSET', key, 'tokens', tokens, 'last_refill', last_refill, 'max_tokens', max_tokens, 'refill_rate', refill_rate)
redis.call('EXPIRE', key, 3600)
return {allowed, tokens, max_tokens}
`

const luaStatusScript = `
local key = KEYS[1]
local default_max_tokens = tonumber(ARGV[1])
local default_refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill', 'max_tokens', 'refill_rate')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])
local max_tokens = tonumber(bucket[3]) or default_max_tokens
local refill_rate = tonumber(bucket[4]) or default_refill_rate

if tokens == nil then
    return {max_tokens, max_tokens}
end

local elapsed = math.max(0, now - last_refill)
local refill = 0
if refill_rate > 0 then
    local time_per_token = 1000 / refill_rate
    if time_per_token > 0 then
        refill = math.floor(elapsed / time_per_token)
    else
        refill = max_tokens
    end
end
tokens = math.min(max_tokens, tokens + refill)

return {tokens, max_tokens}
`

// this removes the race where a second
// request could mutate the bucket between Allow and a separate GetStatus.
func (r *RedisStore) Allow(clientID string) (bool, int, int) {
	key := "rate:" + clientID
	now := time.Now().UnixMilli()

	result, err := r.client.Eval(r.ctx, luaScript, []string{key},
		DefaultMaxTokens, DefaultRefillRate, now).Slice()

	if err != nil {
		return true, 0, 0 // fail open — if Redis is down, allow the request
		// tradeoff -> availability > consistency
	}

	allowed := result[0].(int64) == 1
	tokens := int(result[1].(int64))
	maxTokens := int(result[2].(int64))

	return allowed, tokens, maxTokens
}

func (r *RedisStore) GetStatus(clientID string) (int, int) {
	key := "rate:" + clientID
	now := time.Now().UnixMilli()

	result, err := r.client.Eval(r.ctx, luaStatusScript, []string{key},
		DefaultMaxTokens, DefaultRefillRate, now).Result()

	if err != nil {
		return 0, 0
	}

	vals := result.([]interface{})
	tokens := int(vals[0].(int64))
	maxTokens := int(vals[1].(int64))

	return tokens, maxTokens
}

func (r *RedisStore) SetClient(clientID string, maxTokens int, refillRate int) {
	key := "rate:" + clientID
	r.client.HSet(r.ctx, key,
		"tokens", maxTokens,
		"max_tokens", maxTokens,
		"refill_rate", refillRate,
		"last_refill", time.Now().UnixMilli(),
	)
	r.client.Expire(r.ctx, key, 3600*time.Second)
}
