package vault

import (
	"fmt"
	"time"
)

// Config는 Vault 클라이언트 설정입니다
type Config struct {
	// Vault 서버 주소
	Address string

	// 인증 토큰
	Token string

	// 인증 방법 (token, approle, kubernetes)
	AuthMethod string

	// AppRole 설정
	RoleID   string
	SecretID string

	// Kubernetes 설정
	K8sRole           string
	K8sServiceAccount string

	// 네임스페이스
	Namespace string

	// TLS 설정
	TLSEnabled     bool
	TLSSkipVerify  bool
	CACert         string
	ClientCert     string
	ClientKey      string

	// 시크릿 경로 설정
	MongoDBPath string // MongoDB 동적 자격증명 경로
	RedisPath   string // Redis 자격증명 경로
	SecretsPath string // 정적 시크릿 경로
	TransitPath string // Transit 암호화 경로

	// 리뉴얼 설정
	RenewInterval      time.Duration // 자동 갱신 간격
	RenewBeforeExpiry  time.Duration // 만료 전 갱신 시간
	MaxRetries         int           // 최대 재시도 횟수
	RetryInterval      time.Duration // 재시도 간격

	// 캐시 설정
	CacheEnabled bool
	CacheTTL     time.Duration
}

// DefaultConfig는 기본 Vault 설정을 반환합니다
func DefaultConfig() *Config {
	return &Config{
		Address:            "http://localhost:8200",
		AuthMethod:         "token",
		Namespace:          "",
		TLSEnabled:         false,
		TLSSkipVerify:      false,
		MongoDBPath:        "database/creds/mongodb-role",
		RedisPath:          "secret/data/redis",
		SecretsPath:        "secret/data/app",
		TransitPath:        "transit",
		RenewInterval:      15 * time.Minute,
		RenewBeforeExpiry:  5 * time.Minute,
		MaxRetries:         3,
		RetryInterval:      5 * time.Second,
		CacheEnabled:       true,
		CacheTTL:           5 * time.Minute,
	}
}

// Validate는 설정을 검증합니다
func (c *Config) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("vault address is required")
	}

	switch c.AuthMethod {
	case "token":
		if c.Token == "" {
			return fmt.Errorf("vault token is required for token auth")
		}
	case "approle":
		if c.RoleID == "" || c.SecretID == "" {
			return fmt.Errorf("role_id and secret_id are required for approle auth")
		}
	case "kubernetes":
		if c.K8sRole == "" {
			return fmt.Errorf("kubernetes role is required for kubernetes auth")
		}
	default:
		return fmt.Errorf("unsupported auth method: %s", c.AuthMethod)
	}

	if c.RenewInterval <= 0 {
		c.RenewInterval = 15 * time.Minute
	}

	if c.RenewBeforeExpiry <= 0 {
		c.RenewBeforeExpiry = 5 * time.Minute
	}

	return nil
}

// SecretMetadata는 시크릿 메타데이터입니다
type SecretMetadata struct {
	LeaseID       string
	LeaseDuration int
	Renewable     bool
	Data          map[string]interface{}
	CreatedAt     time.Time
}

// IsExpired는 시크릿이 만료되었는지 확인합니다
func (s *SecretMetadata) IsExpired() bool {
	if s.LeaseDuration == 0 {
		return false
	}
	expiryTime := s.CreatedAt.Add(time.Duration(s.LeaseDuration) * time.Second)
	return time.Now().After(expiryTime)
}

// ShouldRenew는 시크릿을 갱신해야 하는지 확인합니다
func (s *SecretMetadata) ShouldRenew(renewBeforeExpiry time.Duration) bool {
	if !s.Renewable || s.LeaseDuration == 0 {
		return false
	}
	expiryTime := s.CreatedAt.Add(time.Duration(s.LeaseDuration) * time.Second)
	renewTime := expiryTime.Add(-renewBeforeExpiry)
	return time.Now().After(renewTime)
}
