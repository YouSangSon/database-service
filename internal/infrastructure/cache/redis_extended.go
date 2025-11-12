package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisExtended는 확장된 Redis 클라이언트입니다
type RedisExtended struct {
	client *redis.Client
}

// NewRedisExtended는 새로운 확장 Redis 클라이언트를 생성합니다
func NewRedisExtended(client *redis.Client) *RedisExtended {
	return &RedisExtended{
		client: client,
	}
}

// ===== Pub/Sub =====

// PubSubManager는 Pub/Sub 관리자입니다
type PubSubManager struct {
	client *redis.Client
	pubsub *redis.PubSub
}

// NewPubSubManager는 새로운 Pub/Sub 관리자를 생성합니다
func (r *RedisExtended) NewPubSubManager(ctx context.Context, channels ...string) *PubSubManager {
	pubsub := r.client.Subscribe(ctx, channels...)

	logger.Info(ctx, "pubsub manager created",
		logger.Field("channels", channels),
	)

	return &PubSubManager{
		client: r.client,
		pubsub: pubsub,
	}
}

// Publish는 메시지를 발행합니다
func (r *RedisExtended) Publish(ctx context.Context, channel string, message interface{}) error {
	result := r.client.Publish(ctx, channel, message)
	if result.Err() != nil {
		logger.Error(ctx, "failed to publish message",
			logger.Field("channel", channel),
			zap.Error(result.Err()),
		)
		return fmt.Errorf("failed to publish message: %w", result.Err())
	}

	subscribers := result.Val()
	logger.Debug(ctx, "message published",
		logger.Field("channel", channel),
		logger.Field("subscribers", subscribers),
	)

	return nil
}

// Subscribe는 채널을 구독합니다
func (p *PubSubManager) Subscribe(ctx context.Context, channels ...string) error {
	return p.pubsub.Subscribe(ctx, channels...)
}

// Receive는 메시지를 수신합니다
func (p *PubSubManager) Receive(ctx context.Context) (*redis.Message, error) {
	msg, err := p.pubsub.ReceiveMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to receive message: %w", err)
	}
	return msg, nil
}

// Close는 Pub/Sub 연결을 종료합니다
func (p *PubSubManager) Close() error {
	return p.pubsub.Close()
}

// ===== Rate Limiting =====

// RateLimiter는 속도 제한기입니다
type RateLimiter struct {
	client *redis.Client
	prefix string
}

// NewRateLimiter는 새로운 속도 제한기를 생성합니다
func (r *RedisExtended) NewRateLimiter(prefix string) *RateLimiter {
	return &RateLimiter{
		client: r.client,
		prefix: prefix,
	}
}

// Allow는 요청을 허용할지 확인합니다 (Token Bucket)
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	fullKey := fmt.Sprintf("%s:%s", rl.prefix, key)

	// Lua 스크립트로 원자적 처리
	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local current = tonumber(redis.call('GET', key) or "0")

		if current < limit then
			redis.call('INCR', key)
			if current == 0 then
				redis.call('EXPIRE', key, window)
			end
			return 1
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, rl.client, []string{fullKey}, limit, int64(window.Seconds())).Int()
	if err != nil {
		logger.Error(ctx, "rate limit check failed",
			logger.Field("key", key),
			zap.Error(err),
		)
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	allowed := result == 1
	if !allowed {
		logger.Debug(ctx, "rate limit exceeded",
			logger.Field("key", key),
			logger.Field("limit", limit),
		)
	}

	return allowed, nil
}

// AllowN는 N개의 요청을 허용할지 확인합니다
func (rl *RateLimiter) AllowN(ctx context.Context, key string, count int64, limit int64, window time.Duration) (bool, error) {
	fullKey := fmt.Sprintf("%s:%s", rl.prefix, key)

	script := redis.NewScript(`
		local key = KEYS[1]
		local count = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])
		local window = tonumber(ARGV[3])
		local current = tonumber(redis.call('GET', key) or "0")

		if current + count <= limit then
			redis.call('INCRBY', key, count)
			if current == 0 then
				redis.call('EXPIRE', key, window)
			end
			return 1
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, rl.client, []string{fullKey}, count, limit, int64(window.Seconds())).Int()
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	return result == 1, nil
}

// Reset은 속도 제한을 초기화합니다
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("%s:%s", rl.prefix, key)
	return rl.client.Del(ctx, fullKey).Err()
}

// ===== Distributed Lock =====

// DistributedLock는 분산 락입니다
type DistributedLock struct {
	client   *redis.Client
	key      string
	token    string
	ttl      time.Duration
	acquired bool
}

// NewDistributedLock는 새로운 분산 락을 생성합니다
func (r *RedisExtended) NewDistributedLock(key string, ttl time.Duration) *DistributedLock {
	return &DistributedLock{
		client: r.client,
		key:    key,
		token:  fmt.Sprintf("%d", time.Now().UnixNano()),
		ttl:    ttl,
	}
}

// Acquire는 락을 획득합니다
func (dl *DistributedLock) Acquire(ctx context.Context) (bool, error) {
	// SET key token NX EX ttl
	result, err := dl.client.SetNX(ctx, dl.key, dl.token, dl.ttl).Result()
	if err != nil {
		logger.Error(ctx, "failed to acquire lock",
			logger.Field("key", dl.key),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	dl.acquired = result
	if result {
		logger.Debug(ctx, "lock acquired",
			logger.Field("key", dl.key),
			logger.Field("ttl", dl.ttl),
		)
	}

	return result, nil
}

// Release는 락을 해제합니다
func (dl *DistributedLock) Release(ctx context.Context) error {
	if !dl.acquired {
		return nil
	}

	// Lua 스크립트로 원자적 해제 (토큰 확인)
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, dl.client, []string{dl.key}, dl.token).Int()
	if err != nil {
		logger.Error(ctx, "failed to release lock",
			logger.Field("key", dl.key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result == 1 {
		dl.acquired = false
		logger.Debug(ctx, "lock released",
			logger.Field("key", dl.key),
		)
	}

	return nil
}

// Extend는 락의 TTL을 연장합니다
func (dl *DistributedLock) Extend(ctx context.Context, ttl time.Duration) (bool, error) {
	if !dl.acquired {
		return false, fmt.Errorf("lock not acquired")
	}

	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, dl.client, []string{dl.key}, dl.token, int64(ttl.Seconds())).Int()
	if err != nil {
		return false, fmt.Errorf("failed to extend lock: %w", err)
	}

	if result == 1 {
		dl.ttl = ttl
		logger.Debug(ctx, "lock extended",
			logger.Field("key", dl.key),
			logger.Field("new_ttl", ttl),
		)
	}

	return result == 1, nil
}

// ===== Distributed Counter =====

// DistributedCounter는 분산 카운터입니다
type DistributedCounter struct {
	client *redis.Client
	key    string
}

// NewDistributedCounter는 새로운 분산 카운터를 생성합니다
func (r *RedisExtended) NewDistributedCounter(key string) *DistributedCounter {
	return &DistributedCounter{
		client: r.client,
		key:    key,
	}
}

// Increment는 카운터를 증가시킵니다
func (dc *DistributedCounter) Increment(ctx context.Context, delta int64) (int64, error) {
	result, err := dc.client.IncrBy(ctx, dc.key, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}
	return result, nil
}

// Decrement는 카운터를 감소시킵니다
func (dc *DistributedCounter) Decrement(ctx context.Context, delta int64) (int64, error) {
	result, err := dc.client.DecrBy(ctx, dc.key, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement counter: %w", err)
	}
	return result, nil
}

// Get은 현재 카운터 값을 가져옵니다
func (dc *DistributedCounter) Get(ctx context.Context) (int64, error) {
	result, err := dc.client.Get(ctx, dc.key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get counter: %w", err)
	}
	return result, nil
}

// Reset은 카운터를 초기화합니다
func (dc *DistributedCounter) Reset(ctx context.Context) error {
	return dc.client.Del(ctx, dc.key).Err()
}
