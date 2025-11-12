package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	vault "github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// Client는 Vault 클라이언트 래퍼입니다
type Client struct {
	client      *vault.Client
	config      *Config
	cache       map[string]*SecretMetadata
	cacheMutex  sync.RWMutex
	renewers    map[string]*vault.LifetimeWatcher
	renewMutex  sync.RWMutex
	stopChan    chan struct{}
	isRunning   bool
}

// NewClient는 새로운 Vault 클라이언트를 생성합니다
func NewClient(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid vault config: %w", err)
	}

	// Vault 클라이언트 설정
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = cfg.Address

	// TLS 설정
	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}

		// CA 인증서 설정
		if cfg.CACert != "" {
			caCert, err := ioutil.ReadFile(cfg.CACert)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA cert: %w", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}

		// 클라이언트 인증서 설정
		if cfg.ClientCert != "" && cfg.ClientKey != "" {
			cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load client cert: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		vaultConfig.HttpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	// Vault 클라이언트 생성
	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// 네임스페이스 설정
	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	vaultClient := &Client{
		client:    client,
		config:    cfg,
		cache:     make(map[string]*SecretMetadata),
		renewers:  make(map[string]*vault.LifetimeWatcher),
		stopChan:  make(chan struct{}),
		isRunning: false,
	}

	// 인증
	if err := vaultClient.authenticate(); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	logger.Info(context.Background(), "vault client initialized successfully",
		logger.Field("address", cfg.Address),
		logger.Field("auth_method", cfg.AuthMethod),
	)

	return vaultClient, nil
}

// authenticate는 Vault에 인증합니다
func (c *Client) authenticate() error {
	switch c.config.AuthMethod {
	case "token":
		c.client.SetToken(c.config.Token)
		// 토큰 유효성 확인
		_, err := c.client.Auth().Token().LookupSelf()
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}
		logger.Info(context.Background(), "authenticated with token")

	case "approle":
		data := map[string]interface{}{
			"role_id":   c.config.RoleID,
			"secret_id": c.config.SecretID,
		}
		secret, err := c.client.Logical().Write("auth/approle/login", data)
		if err != nil {
			return fmt.Errorf("approle login failed: %w", err)
		}
		if secret == nil || secret.Auth == nil {
			return fmt.Errorf("approle login returned no auth info")
		}
		c.client.SetToken(secret.Auth.ClientToken)
		logger.Info(context.Background(), "authenticated with approle",
			logger.Field("role_id", c.config.RoleID),
		)

	case "kubernetes":
		// Kubernetes 서비스 어카운트 토큰 읽기
		jwt, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			return fmt.Errorf("failed to read k8s service account token: %w", err)
		}

		data := map[string]interface{}{
			"role": c.config.K8sRole,
			"jwt":  string(jwt),
		}
		secret, err := c.client.Logical().Write("auth/kubernetes/login", data)
		if err != nil {
			return fmt.Errorf("kubernetes login failed: %w", err)
		}
		if secret == nil || secret.Auth == nil {
			return fmt.Errorf("kubernetes login returned no auth info")
		}
		c.client.SetToken(secret.Auth.ClientToken)
		logger.Info(context.Background(), "authenticated with kubernetes",
			logger.Field("role", c.config.K8sRole),
		)

	default:
		return fmt.Errorf("unsupported auth method: %s", c.config.AuthMethod)
	}

	return nil
}

// StartRenewal은 자동 갱신을 시작합니다
func (c *Client) StartRenewal(ctx context.Context) {
	if c.isRunning {
		logger.Warn(ctx, "renewal already running")
		return
	}

	c.isRunning = true
	go c.renewalLoop(ctx)

	logger.Info(ctx, "vault renewal started",
		logger.Field("interval", c.config.RenewInterval),
	)
}

// renewalLoop는 자동 갱신 루프입니다
func (c *Client) renewalLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.RenewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info(context.Background(), "stopping vault renewal due to context cancellation")
			return
		case <-c.stopChan:
			logger.Info(context.Background(), "stopping vault renewal")
			return
		case <-ticker.C:
			c.renewSecrets(ctx)
		}
	}
}

// renewSecrets는 만료 예정인 시크릿을 갱신합니다
func (c *Client) renewSecrets(ctx context.Context) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	for path, metadata := range c.cache {
		if metadata.ShouldRenew(c.config.RenewBeforeExpiry) {
			go func(p string, m *SecretMetadata) {
				if err := c.renewSecret(ctx, p, m); err != nil {
					logger.Error(ctx, "failed to renew secret",
						logger.Field("path", p),
						zap.Error(err),
					)
				}
			}(path, metadata)
		}
	}
}

// renewSecret는 개별 시크릿을 갱신합니다
func (c *Client) renewSecret(ctx context.Context, path string, metadata *SecretMetadata) error {
	if !metadata.Renewable || metadata.LeaseID == "" {
		logger.Debug(ctx, "secret is not renewable",
			logger.Field("path", path),
		)
		return nil
	}

	// 리스 갱신
	secret, err := c.client.Sys().Renew(metadata.LeaseID, 0)
	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}

	// 캐시 업데이트
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	metadata.LeaseDuration = secret.LeaseDuration
	metadata.CreatedAt = time.Now()

	logger.Info(ctx, "secret renewed successfully",
		logger.Field("path", path),
		logger.Field("lease_duration", secret.LeaseDuration),
	)

	return nil
}

// StopRenewal은 자동 갱신을 중지합니다
func (c *Client) StopRenewal() {
	if !c.isRunning {
		return
	}

	close(c.stopChan)
	c.isRunning = false

	// 모든 lifetime watcher 중지
	c.renewMutex.Lock()
	defer c.renewMutex.Unlock()

	for _, watcher := range c.renewers {
		watcher.Stop()
	}

	logger.Info(context.Background(), "vault renewal stopped")
}

// HealthCheck는 Vault 연결 상태를 확인합니다
func (c *Client) HealthCheck(ctx context.Context) error {
	health, err := c.client.Sys().Health()
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	logger.Debug(ctx, "vault health check passed",
		logger.Field("version", health.Version),
		logger.Field("cluster_name", health.ClusterName),
	)

	return nil
}

// Close는 클라이언트를 종료합니다
func (c *Client) Close() error {
	c.StopRenewal()

	// 캐시 정리
	c.cacheMutex.Lock()
	c.cache = make(map[string]*SecretMetadata)
	c.cacheMutex.Unlock()

	logger.Info(context.Background(), "vault client closed")
	return nil
}

// GetClient는 내부 Vault 클라이언트를 반환합니다
func (c *Client) GetClient() *vault.Client {
	return c.client
}
