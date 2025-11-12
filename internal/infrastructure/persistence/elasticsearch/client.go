package elasticsearch

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

// Config는 Elasticsearch 연결 설정입니다
type Config struct {
	Addresses []string // Elasticsearch cluster URLs
	Username  string
	Password  string
	APIKey    string // API Key authentication

	// TLS Settings
	InsecureSkipVerify bool
	CertificatePath    string

	// Connection Settings
	MaxRetries    int
	RetryOnStatus []int
	Timeout       time.Duration

	// Compression
	EnableCompression bool
}

// NewClient는 Elasticsearch 클라이언트를 생성합니다
func NewClient(ctx context.Context, config *Config) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: config.Addresses,
	}

	// Authentication
	if config.APIKey != "" {
		cfg.APIKey = config.APIKey
	} else if config.Username != "" && config.Password != "" {
		cfg.Username = config.Username
		cfg.Password = config.Password
	}

	// TLS Configuration
	if config.InsecureSkipVerify {
		cfg.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Retry Configuration
	if config.MaxRetries > 0 {
		cfg.MaxRetries = config.MaxRetries
	} else {
		cfg.MaxRetries = 3 // 기본값
	}

	if len(config.RetryOnStatus) > 0 {
		cfg.RetryOnStatus = config.RetryOnStatus
	}

	// Timeout
	if config.Timeout > 0 {
		if cfg.Transport == nil {
			cfg.Transport = &http.Transport{}
		}
		if transport, ok := cfg.Transport.(*http.Transport); ok {
			transport.ResponseHeaderTimeout = config.Timeout
		}
	}

	// Compression
	if config.EnableCompression {
		cfg.EnableMetrics = true
		cfg.EnableDebugLogger = false
	}

	// Create Client
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Test Connection
	res, err := client.Ping(client.Ping.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch ping failed: %s", res.String())
	}

	return client, nil
}

// Close는 Elasticsearch 클라이언트를 닫습니다
// Note: go-elasticsearch 클라이언트는 Close 메서드가 없습니다
// HTTP 연결은 자동으로 관리됩니다
func Close(client *elasticsearch.Client) {
	// No-op for Elasticsearch client
	// Connection pooling is handled automatically
}
