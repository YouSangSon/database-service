package vault

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// DatabaseCredentials는 데이터베이스 자격증명입니다
type DatabaseCredentials struct {
	Username      string
	Password      string
	LeaseID       string
	LeaseDuration int
	RenewAt       time.Time
	ExpiresAt     time.Time
}

// DatabaseCredentialsManager는 데이터베이스 자격증명 관리자입니다
type DatabaseCredentialsManager struct {
	client      *Client
	credentials *DatabaseCredentials
	mutex       sync.RWMutex
	stopChan    chan struct{}
	isRunning   bool
}

// NewDatabaseCredentialsManager는 새로운 데이터베이스 자격증명 관리자를 생성합니다
func NewDatabaseCredentialsManager(client *Client) *DatabaseCredentialsManager {
	return &DatabaseCredentialsManager{
		client:    client,
		stopChan:  make(chan struct{}),
		isRunning: false,
	}
}

// GetMongoDBCredentials는 MongoDB 자격증명을 가져옵니다 (캐시 또는 새로 발급)
func (m *DatabaseCredentialsManager) GetMongoDBCredentials(ctx context.Context) (*DatabaseCredentials, error) {
	m.mutex.RLock()
	if m.credentials != nil && time.Now().Before(m.credentials.ExpiresAt) {
		creds := m.credentials
		m.mutex.RUnlock()
		logger.Debug(ctx, "using cached mongodb credentials",
			logger.Field("username", creds.Username),
			logger.Field("expires_at", creds.ExpiresAt),
		)
		return creds, nil
	}
	m.mutex.RUnlock()

	// 새로운 자격증명 발급
	return m.renewMongoDBCredentials(ctx)
}

// renewMongoDBCredentials는 MongoDB 자격증명을 갱신합니다
func (m *DatabaseCredentialsManager) renewMongoDBCredentials(ctx context.Context) (*DatabaseCredentials, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 이전 자격증명 취소
	if m.credentials != nil && m.credentials.LeaseID != "" {
		if err := m.client.RevokeSecret(ctx, m.credentials.LeaseID); err != nil {
			logger.Warn(ctx, "failed to revoke old credentials",
				logger.Field("lease_id", m.credentials.LeaseID),
				zap.Error(err),
			)
		}
	}

	// 새로운 자격증명 발급
	metadata, err := m.client.GetDynamicSecret(ctx, m.client.config.MongoDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get mongodb credentials: %w", err)
	}

	username, ok := metadata.Data["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username not found in credentials")
	}

	password, ok := metadata.Data["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password not found in credentials")
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(metadata.LeaseDuration) * time.Second)
	renewAt := expiresAt.Add(-m.client.config.RenewBeforeExpiry)

	m.credentials = &DatabaseCredentials{
		Username:      username,
		Password:      password,
		LeaseID:       metadata.LeaseID,
		LeaseDuration: metadata.LeaseDuration,
		RenewAt:       renewAt,
		ExpiresAt:     expiresAt,
	}

	logger.Info(ctx, "mongodb credentials renewed",
		logger.Field("username", username),
		logger.Field("lease_id", metadata.LeaseID),
		logger.Field("lease_duration", metadata.LeaseDuration),
		logger.Field("expires_at", expiresAt),
	)

	return m.credentials, nil
}

// StartAutoRenewal은 자동 갱신을 시작합니다
func (m *DatabaseCredentialsManager) StartAutoRenewal(ctx context.Context) {
	if m.isRunning {
		logger.Warn(ctx, "auto renewal already running")
		return
	}

	m.isRunning = true
	go m.autoRenewalLoop(ctx)

	logger.Info(ctx, "mongodb credentials auto renewal started")
}

// autoRenewalLoop는 자동 갱신 루프입니다
func (m *DatabaseCredentialsManager) autoRenewalLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info(context.Background(), "stopping auto renewal due to context cancellation")
			return
		case <-m.stopChan:
			logger.Info(context.Background(), "stopping auto renewal")
			return
		case <-ticker.C:
			m.mutex.RLock()
			shouldRenew := m.credentials != nil && time.Now().After(m.credentials.RenewAt)
			m.mutex.RUnlock()

			if shouldRenew {
				if _, err := m.renewMongoDBCredentials(ctx); err != nil {
					logger.Error(ctx, "failed to auto-renew mongodb credentials",
						zap.Error(err),
					)
				}
			}
		}
	}
}

// StopAutoRenewal은 자동 갱신을 중지합니다
func (m *DatabaseCredentialsManager) StopAutoRenewal() {
	if !m.isRunning {
		return
	}

	close(m.stopChan)
	m.isRunning = false

	logger.Info(context.Background(), "mongodb credentials auto renewal stopped")
}

// GetConnectionString은 MongoDB 연결 문자열을 생성합니다
func (m *DatabaseCredentialsManager) GetConnectionString(ctx context.Context, host string, port int, database string) (string, error) {
	creds, err := m.GetMongoDBCredentials(ctx)
	if err != nil {
		return "", err
	}

	connectionString := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
		creds.Username,
		creds.Password,
		host,
		port,
		database,
	)

	return connectionString, nil
}

// RevokeCredentials는 현재 자격증명을 취소합니다
func (m *DatabaseCredentialsManager) RevokeCredentials(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.credentials == nil || m.credentials.LeaseID == "" {
		return nil
	}

	if err := m.client.RevokeSecret(ctx, m.credentials.LeaseID); err != nil {
		return fmt.Errorf("failed to revoke credentials: %w", err)
	}

	logger.Info(ctx, "mongodb credentials revoked",
		logger.Field("lease_id", m.credentials.LeaseID),
	)

	m.credentials = nil
	return nil
}

// Close는 관리자를 종료합니다
func (m *DatabaseCredentialsManager) Close(ctx context.Context) error {
	m.StopAutoRenewal()

	// 자격증명 취소
	if err := m.RevokeCredentials(ctx); err != nil {
		logger.Warn(ctx, "failed to revoke credentials on close",
			zap.Error(err),
		)
	}

	return nil
}
