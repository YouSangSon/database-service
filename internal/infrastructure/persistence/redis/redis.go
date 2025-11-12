package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CacheRepository는 Redis 기반 캐시 저장소입니다
type CacheRepository struct {
	client *redis.Client
}

// NewCacheRepository는 새로운 Redis 캐시 저장소를 생성합니다
func NewCacheRepository(addr, password string, db int) (repository.CacheRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     100,              // 연결 풀 크기
		MinIdleConns: 10,                // 최소 유휴 연결 수
		MaxRetries:   3,                 // 최대 재시도 횟수
		DialTimeout:  5 * time.Second,   // 연결 타임아웃
		ReadTimeout:  3 * time.Second,   // 읽기 타임아웃
		WriteTimeout: 3 * time.Second,   // 쓰기 타임아웃
		PoolTimeout:  4 * time.Second,   // 풀 타임아웃
	})

	// 연결 확인
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheRepository{
		client: client,
	}, nil
}

// Get은 캐시에서 값을 가져옵니다
func (r *CacheRepository) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	defer func() {
		logger.Debug(ctx, "cache get operation",
			zap.String("key", key),
			zap.Duration("duration", time.Since(start)),
		)
	}()

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found: %s", key)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	// JSON 역직렬화
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return result, nil
}

// Set은 캐시에 값을 저장합니다
func (r *CacheRepository) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	start := time.Now()
	defer func() {
		logger.Debug(ctx, "cache set operation",
			zap.String("key", key),
			zap.Int("ttl", ttl),
			zap.Duration("duration", time.Since(start)),
		)
	}()

	// JSON 직렬화
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	expiration := time.Duration(ttl) * time.Second
	if err := r.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	return nil
}

// Delete는 캐시에서 값을 삭제합니다
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		logger.Debug(ctx, "cache delete operation",
			zap.String("key", key),
			zap.Duration("duration", time.Since(start)),
		)
	}()

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	return nil
}

// Exists는 키가 존재하는지 확인합니다
func (r *CacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// Close는 Redis 연결을 종료합니다
func (r *CacheRepository) Close() error {
	return r.client.Close()
}

// Ping은 Redis 연결 상태를 확인합니다
func (r *CacheRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetClient는 Redis 클라이언트를 반환합니다 (테스트용)
func (r *CacheRepository) GetClient() *redis.Client {
	return r.client
}
