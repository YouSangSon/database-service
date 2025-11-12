package tenant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/tenant"
)

// MemoryRepository는 인메모리 테넌트 저장소입니다 (개발/테스트용)
type MemoryRepository struct {
	tenants    map[string]*tenant.Tenant
	apiKeys    map[string]string // apiKey -> tenantID
	mu         sync.RWMutex
}

// NewMemoryRepository는 새로운 인메모리 테넌트 저장소를 생성합니다
func NewMemoryRepository() tenant.Repository {
	repo := &MemoryRepository{
		tenants: make(map[string]*tenant.Tenant),
		apiKeys: make(map[string]string),
	}

	// 기본 테넌트 초기화
	repo.initializeDefaultTenants()

	return repo
}

// initializeDefaultTenants는 기본 테넌트를 초기화합니다
func (r *MemoryRepository) initializeDefaultTenants() {
	defaultTenants := []*tenant.Tenant{
		{
			ID:       "default",
			Name:     "Default Tenant",
			Database: "database_service",
			Plan:     tenant.PlanEnterprise,
			Status:   tenant.StatusActive,
			Quotas:   tenant.GetDefaultQuotas(tenant.PlanEnterprise),
			Usage: &tenant.Usage{
				DocumentCount:   0,
				StorageUsed:     0,
				APICallsToday:   0,
				CollectionCount: 0,
				IndexCount:      0,
				UserCount:       1,
				LastUpdated:     time.Now(),
			},
			Settings:  make(map[string]string),
			Metadata:  make(map[string]string),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "demo",
			Name:     "Demo Tenant",
			Database: "tenant_demo",
			Plan:     tenant.PlanFree,
			Status:   tenant.StatusTrial,
			Quotas:   tenant.GetDefaultQuotas(tenant.PlanFree),
			Usage: &tenant.Usage{
				DocumentCount:   100,
				StorageUsed:     1024 * 1024 * 5, // 5MB
				APICallsToday:   50,
				CollectionCount: 2,
				IndexCount:      5,
				UserCount:       1,
				LastAPICall:     time.Now(),
				LastUpdated:     time.Now(),
			},
			Settings:  make(map[string]string),
			Metadata:  make(map[string]string),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, t := range defaultTenants {
		r.tenants[t.ID] = t
		// API Key 생성 (테넌트ID 기반)
		apiKey := generateAPIKey(t.ID)
		r.apiKeys[apiKey] = t.ID
	}
}

// GetByID는 ID로 테넌트를 조회합니다
func (r *MemoryRepository) GetByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.tenants[id]
	if !exists {
		return nil, tenant.ErrTenantNotFound
	}

	return t, nil
}

// GetByAPIKey는 API Key로 테넌트를 조회합니다
func (r *MemoryRepository) GetByAPIKey(ctx context.Context, apiKey string) (*tenant.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tenantID, exists := r.apiKeys[apiKey]
	if !exists {
		return nil, tenant.ErrTenantNotFound
	}

	t, exists := r.tenants[tenantID]
	if !exists {
		return nil, tenant.ErrTenantNotFound
	}

	return t, nil
}

// Create는 새로운 테넌트를 생성합니다
func (r *MemoryRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tenants[t.ID]; exists {
		return tenant.ErrInvalidTenantID
	}

	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	r.tenants[t.ID] = t

	// API Key 생성
	apiKey := generateAPIKey(t.ID)
	r.apiKeys[apiKey] = t.ID

	return nil
}

// Update는 테넌트 정보를 업데이트합니다
func (r *MemoryRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tenants[t.ID]; !exists {
		return tenant.ErrTenantNotFound
	}

	t.UpdatedAt = time.Now()
	r.tenants[t.ID] = t

	return nil
}

// Delete는 테넌트를 삭제합니다
func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tenants[id]; !exists {
		return tenant.ErrTenantNotFound
	}

	delete(r.tenants, id)

	// API Key도 삭제
	for apiKey, tenantID := range r.apiKeys {
		if tenantID == id {
			delete(r.apiKeys, apiKey)
		}
	}

	return nil
}

// UpdateUsage는 테넌트 사용량을 업데이트합니다
func (r *MemoryRepository) UpdateUsage(ctx context.Context, id string, usage *tenant.Usage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, exists := r.tenants[id]
	if !exists {
		return tenant.ErrTenantNotFound
	}

	usage.LastUpdated = time.Now()
	t.Usage = usage
	t.UpdatedAt = time.Now()

	return nil
}

// IncrementAPICallCount는 API 호출 수를 증가시킵니다
func (r *MemoryRepository) IncrementAPICallCount(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, exists := r.tenants[id]
	if !exists {
		return tenant.ErrTenantNotFound
	}

	t.Usage.APICallsToday++
	t.Usage.LastAPICall = time.Now()
	t.UpdatedAt = time.Now()

	return nil
}

// List는 테넌트 목록을 조회합니다
func (r *MemoryRepository) List(ctx context.Context, offset, limit int) ([]*tenant.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tenants []*tenant.Tenant
	i := 0
	for _, t := range r.tenants {
		if i >= offset && (limit == 0 || i < offset+limit) {
			tenants = append(tenants, t)
		}
		i++
	}

	return tenants, nil
}

// generateAPIKey는 테넌트 ID 기반으로 API Key를 생성합니다
func generateAPIKey(tenantID string) string {
	data := tenantID + time.Now().String()
	hash := sha256.Sum256([]byte(data))
	return "sk_" + hex.EncodeToString(hash[:16])
}

// GetAPIKey는 테넌트의 API Key를 반환합니다 (테스트/개발용)
func (r *MemoryRepository) GetAPIKey(tenantID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for apiKey, id := range r.apiKeys {
		if id == tenantID {
			return apiKey
		}
	}
	return ""
}
