package tenant

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrInvalidTenantID   = errors.New("invalid tenant ID")
	ErrTenantQuotaExceeded = errors.New("tenant quota exceeded")
	ErrTenantDisabled    = errors.New("tenant is disabled")
)

// Tenant는 테넌트 정보를 나타냅니다
type Tenant struct {
	ID          string
	Name        string
	Database    string            // 테넌트별 DB 분리 (Optional)
	Plan        Plan              // 요금제
	Status      Status            // 상태
	Quotas      *Quotas           // 할당량
	Usage       *Usage            // 현재 사용량
	Settings    map[string]string // 테넌트별 설정
	Metadata    map[string]string // 메타데이터
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   *time.Time        // 만료일 (Optional)
}

// Plan은 요금제를 나타냅니다
type Plan string

const (
	PlanFree       Plan = "free"
	PlanBasic      Plan = "basic"
	PlanProfessional Plan = "professional"
	PlanEnterprise Plan = "enterprise"
)

// Status는 테넌트 상태를 나타냅니다
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDisabled  Status = "disabled"
	StatusTrial     Status = "trial"
)

// Quotas는 테넌트 할당량을 나타냅니다
type Quotas struct {
	MaxDocuments      int64   // 최대 문서 수
	MaxStorage        int64   // 최대 저장 용량 (바이트)
	MaxAPICallsPerDay int64   // 일일 API 호출 제한
	MaxCollections    int     // 최대 컬렉션 수
	MaxIndexes        int     // 최대 인덱스 수
	MaxUsers          int     // 최대 사용자 수
	RateLimitPerMin   int     // 분당 요청 제한
	EnabledFeatures   []string // 활성화된 기능 목록
}

// Usage는 테넌트 현재 사용량을 나타냅니다
type Usage struct {
	DocumentCount      int64     // 현재 문서 수
	StorageUsed        int64     // 사용 중인 저장 용량 (바이트)
	APICallsToday      int64     // 오늘 API 호출 수
	CollectionCount    int       // 현재 컬렉션 수
	IndexCount         int       // 현재 인덱스 수
	UserCount          int       // 현재 사용자 수
	LastAPICall        time.Time // 마지막 API 호출 시간
	LastUpdated        time.Time // 마지막 업데이트 시간
}

// GetDefaultQuotas는 요금제별 기본 할당량을 반환합니다
func GetDefaultQuotas(plan Plan) *Quotas {
	switch plan {
	case PlanFree:
		return &Quotas{
			MaxDocuments:      10000,
			MaxStorage:        1024 * 1024 * 100, // 100MB
			MaxAPICallsPerDay: 1000,
			MaxCollections:    5,
			MaxIndexes:        10,
			MaxUsers:          1,
			RateLimitPerMin:   60,
			EnabledFeatures:   []string{"basic_crud", "search"},
		}
	case PlanBasic:
		return &Quotas{
			MaxDocuments:      100000,
			MaxStorage:        1024 * 1024 * 1024, // 1GB
			MaxAPICallsPerDay: 10000,
			MaxCollections:    20,
			MaxIndexes:        50,
			MaxUsers:          5,
			RateLimitPerMin:   300,
			EnabledFeatures:   []string{"basic_crud", "search", "aggregation"},
		}
	case PlanProfessional:
		return &Quotas{
			MaxDocuments:      1000000,
			MaxStorage:        1024 * 1024 * 1024 * 10, // 10GB
			MaxAPICallsPerDay: 100000,
			MaxCollections:    100,
			MaxIndexes:        200,
			MaxUsers:          20,
			RateLimitPerMin:   1000,
			EnabledFeatures:   []string{"basic_crud", "search", "aggregation", "analytics", "backup"},
		}
	case PlanEnterprise:
		return &Quotas{
			MaxDocuments:      -1, // Unlimited
			MaxStorage:        -1, // Unlimited
			MaxAPICallsPerDay: -1, // Unlimited
			MaxCollections:    -1,
			MaxIndexes:        -1,
			MaxUsers:          -1,
			RateLimitPerMin:   10000,
			EnabledFeatures:   []string{"*"}, // All features
		}
	default:
		return GetDefaultQuotas(PlanFree)
	}
}

// IsActive는 테넌트가 활성 상태인지 확인합니다
func (t *Tenant) IsActive() bool {
	if t.Status != StatusActive && t.Status != StatusTrial {
		return false
	}

	// 만료일 체크
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return false
	}

	return true
}

// CheckQuota는 할당량을 초과했는지 확인합니다
func (t *Tenant) CheckQuota(quotaType QuotaType) error {
	if !t.IsActive() {
		return ErrTenantDisabled
	}

	switch quotaType {
	case QuotaTypeDocuments:
		if t.Quotas.MaxDocuments != -1 && t.Usage.DocumentCount >= t.Quotas.MaxDocuments {
			return ErrTenantQuotaExceeded
		}
	case QuotaTypeStorage:
		if t.Quotas.MaxStorage != -1 && t.Usage.StorageUsed >= t.Quotas.MaxStorage {
			return ErrTenantQuotaExceeded
		}
	case QuotaTypeAPICalls:
		if t.Quotas.MaxAPICallsPerDay != -1 && t.Usage.APICallsToday >= t.Quotas.MaxAPICallsPerDay {
			return ErrTenantQuotaExceeded
		}
	case QuotaTypeCollections:
		if t.Quotas.MaxCollections != -1 && t.Usage.CollectionCount >= t.Quotas.MaxCollections {
			return ErrTenantQuotaExceeded
		}
	case QuotaTypeIndexes:
		if t.Quotas.MaxIndexes != -1 && t.Usage.IndexCount >= t.Quotas.MaxIndexes {
			return ErrTenantQuotaExceeded
		}
	}

	return nil
}

// HasFeature는 특정 기능이 활성화되어 있는지 확인합니다
func (t *Tenant) HasFeature(feature string) bool {
	// Enterprise 플랜은 모든 기능 사용 가능
	for _, f := range t.Quotas.EnabledFeatures {
		if f == "*" || f == feature {
			return true
		}
	}
	return false
}

// GetDatabase는 테넌트의 데이터베이스 이름을 반환합니다
func (t *Tenant) GetDatabase() string {
	if t.Database != "" {
		return t.Database
	}
	// 기본 데이터베이스에 테넌트 prefix 사용
	return "tenant_" + t.ID
}

// QuotaType은 할당량 유형입니다
type QuotaType string

const (
	QuotaTypeDocuments   QuotaType = "documents"
	QuotaTypeStorage     QuotaType = "storage"
	QuotaTypeAPICalls    QuotaType = "api_calls"
	QuotaTypeCollections QuotaType = "collections"
	QuotaTypeIndexes     QuotaType = "indexes"
)

// Context key for tenant
type contextKey string

const TenantContextKey contextKey = "tenant"

// FromContext는 context에서 테넌트 정보를 추출합니다
func FromContext(ctx context.Context) (*Tenant, bool) {
	tenant, ok := ctx.Value(TenantContextKey).(*Tenant)
	return tenant, ok
}

// NewContext는 테넌트 정보를 포함한 새로운 context를 생성합니다
func NewContext(ctx context.Context, tenant *Tenant) context.Context {
	return context.WithValue(ctx, TenantContextKey, tenant)
}

// Repository는 테넌트 저장소 인터페이스입니다
type Repository interface {
	// GetByID는 ID로 테넌트를 조회합니다
	GetByID(ctx context.Context, id string) (*Tenant, error)

	// GetByAPIKey는 API Key로 테넌트를 조회합니다
	GetByAPIKey(ctx context.Context, apiKey string) (*Tenant, error)

	// Create는 새로운 테넌트를 생성합니다
	Create(ctx context.Context, tenant *Tenant) error

	// Update는 테넌트 정보를 업데이트합니다
	Update(ctx context.Context, tenant *Tenant) error

	// Delete는 테넌트를 삭제합니다
	Delete(ctx context.Context, id string) error

	// UpdateUsage는 테넌트 사용량을 업데이트합니다
	UpdateUsage(ctx context.Context, id string, usage *Usage) error

	// IncrementAPICallCount는 API 호출 수를 증가시킵니다
	IncrementAPICallCount(ctx context.Context, id string) error

	// List는 테넌트 목록을 조회합니다
	List(ctx context.Context, offset, limit int) ([]*Tenant, error)
}

// Service는 테넌트 서비스 인터페이스입니다
type Service interface {
	// ValidateTenant는 테넌트의 유효성을 검증합니다
	ValidateTenant(ctx context.Context, tenantID string) (*Tenant, error)

	// CheckQuota는 테넌트의 할당량을 확인합니다
	CheckQuota(ctx context.Context, tenantID string, quotaType QuotaType) error

	// RecordAPICall은 API 호출을 기록합니다
	RecordAPICall(ctx context.Context, tenantID string) error

	// GetTenantStats는 테넌트 통계를 조회합니다
	GetTenantStats(ctx context.Context, tenantID string) (*TenantStats, error)
}

// TenantStats는 테넌트 통계입니다
type TenantStats struct {
	TenantID            string
	DocumentCount       int64
	StorageUsed         int64
	APICallsToday       int64
	APICallsThisMonth   int64
	TopCollections      []CollectionStat
	AverageResponseTime float64 // 밀리초
	ErrorRate           float64 // 백분율
	LastActive          time.Time
}

// CollectionStat은 컬렉션 통계입니다
type CollectionStat struct {
	Name          string
	DocumentCount int64
	Size          int64
	IndexCount    int
}
