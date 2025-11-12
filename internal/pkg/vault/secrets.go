package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// GetSecret는 정적 시크릿을 가져옵니다 (KV v2)
func (c *Client) GetSecret(ctx context.Context, path string) (*SecretMetadata, error) {
	// 캐시 확인
	if c.config.CacheEnabled {
		if cached := c.getCachedSecret(path); cached != nil {
			logger.Debug(ctx, "secret retrieved from cache",
				logger.Field("path", path),
			)
			return cached, nil
		}
	}

	// Vault에서 시크릿 가져오기
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		logger.Error(ctx, "failed to read secret",
			logger.Field("path", path),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}

	if secret == nil {
		return nil, fmt.Errorf("secret not found at path: %s", path)
	}

	// KV v2 데이터 추출
	var data map[string]interface{}
	if secret.Data["data"] != nil {
		data = secret.Data["data"].(map[string]interface{})
	} else {
		data = secret.Data
	}

	metadata := &SecretMetadata{
		LeaseID:       secret.LeaseID,
		LeaseDuration: secret.LeaseDuration,
		Renewable:     secret.Renewable,
		Data:          data,
		CreatedAt:     time.Now(),
	}

	// 캐시에 저장
	if c.config.CacheEnabled {
		c.cacheSecret(path, metadata)
	}

	logger.Info(ctx, "secret retrieved successfully",
		logger.Field("path", path),
		logger.Field("renewable", secret.Renewable),
	)

	return metadata, nil
}

// GetDynamicSecret는 동적 시크릿을 가져옵니다 (데이터베이스 자격증명)
func (c *Client) GetDynamicSecret(ctx context.Context, path string) (*SecretMetadata, error) {
	// 캐시 확인
	if c.config.CacheEnabled {
		if cached := c.getCachedSecret(path); cached != nil && !cached.IsExpired() {
			logger.Debug(ctx, "dynamic secret retrieved from cache",
				logger.Field("path", path),
			)
			return cached, nil
		}
	}

	// Vault에서 동적 시크릿 가져오기
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		logger.Error(ctx, "failed to read dynamic secret",
			logger.Field("path", path),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to read dynamic secret: %w", err)
	}

	if secret == nil {
		return nil, fmt.Errorf("dynamic secret not found at path: %s", path)
	}

	metadata := &SecretMetadata{
		LeaseID:       secret.LeaseID,
		LeaseDuration: secret.LeaseDuration,
		Renewable:     secret.Renewable,
		Data:          secret.Data,
		CreatedAt:     time.Now(),
	}

	// 캐시에 저장
	if c.config.CacheEnabled {
		c.cacheSecret(path, metadata)
	}

	logger.Info(ctx, "dynamic secret retrieved successfully",
		logger.Field("path", path),
		logger.Field("lease_id", secret.LeaseID),
		logger.Field("lease_duration", secret.LeaseDuration),
		logger.Field("renewable", secret.Renewable),
	)

	return metadata, nil
}

// GetMongoDBCredentials는 MongoDB 동적 자격증명을 가져옵니다
func (c *Client) GetMongoDBCredentials(ctx context.Context) (username, password string, err error) {
	metadata, err := c.GetDynamicSecret(ctx, c.config.MongoDBPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get mongodb credentials: %w", err)
	}

	username, ok := metadata.Data["username"].(string)
	if !ok {
		return "", "", fmt.Errorf("username not found in mongodb credentials")
	}

	password, ok = metadata.Data["password"].(string)
	if !ok {
		return "", "", fmt.Errorf("password not found in mongodb credentials")
	}

	logger.Info(ctx, "mongodb credentials retrieved",
		logger.Field("username", username),
		logger.Field("lease_duration", metadata.LeaseDuration),
	)

	return username, password, nil
}

// GetRedisCredentials는 Redis 자격증명을 가져옵니다
func (c *Client) GetRedisCredentials(ctx context.Context) (password string, err error) {
	metadata, err := c.GetSecret(ctx, c.config.RedisPath)
	if err != nil {
		return "", fmt.Errorf("failed to get redis credentials: %w", err)
	}

	password, ok := metadata.Data["password"].(string)
	if !ok {
		return "", fmt.Errorf("password not found in redis credentials")
	}

	logger.Info(ctx, "redis credentials retrieved")

	return password, nil
}

// PutSecret는 시크릿을 저장합니다 (KV v2)
func (c *Client) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	// KV v2 형식으로 데이터 래핑
	wrappedData := map[string]interface{}{
		"data": data,
	}

	_, err := c.client.Logical().Write(path, wrappedData)
	if err != nil {
		logger.Error(ctx, "failed to write secret",
			logger.Field("path", path),
			zap.Error(err),
		)
		return fmt.Errorf("failed to write secret: %w", err)
	}

	// 캐시 무효화
	c.invalidateCache(path)

	logger.Info(ctx, "secret written successfully",
		logger.Field("path", path),
	)

	return nil
}

// DeleteSecret는 시크릿을 삭제합니다
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	_, err := c.client.Logical().Delete(path)
	if err != nil {
		logger.Error(ctx, "failed to delete secret",
			logger.Field("path", path),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// 캐시 무효화
	c.invalidateCache(path)

	logger.Info(ctx, "secret deleted successfully",
		logger.Field("path", path),
	)

	return nil
}

// RevokeSecret는 동적 시크릿의 리스를 취소합니다
func (c *Client) RevokeSecret(ctx context.Context, leaseID string) error {
	if leaseID == "" {
		return fmt.Errorf("lease_id is required")
	}

	err := c.client.Sys().Revoke(leaseID)
	if err != nil {
		logger.Error(ctx, "failed to revoke secret",
			logger.Field("lease_id", leaseID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to revoke secret: %w", err)
	}

	logger.Info(ctx, "secret revoked successfully",
		logger.Field("lease_id", leaseID),
	)

	return nil
}

// ===== 캐시 관리 =====

// getCachedSecret는 캐시에서 시크릿을 가져옵니다
func (c *Client) getCachedSecret(path string) *SecretMetadata {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	metadata, exists := c.cache[path]
	if !exists {
		return nil
	}

	// 만료 확인
	if metadata.IsExpired() {
		return nil
	}

	return metadata
}

// cacheSecret는 시크릿을 캐시에 저장합니다
func (c *Client) cacheSecret(path string, metadata *SecretMetadata) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache[path] = metadata
}

// invalidateCache는 캐시를 무효화합니다
func (c *Client) invalidateCache(path string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	delete(c.cache, path)
}

// ClearCache는 모든 캐시를 삭제합니다
func (c *Client) ClearCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache = make(map[string]*SecretMetadata)

	logger.Info(context.Background(), "vault cache cleared")
}
