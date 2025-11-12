package middleware

import (
	"net/http"
	"strings"

	"github.com/YouSangSon/database-service/internal/domain/tenant"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// TenantIDHeader는 테넌트 ID를 전달하는 HTTP 헤더입니다
	TenantIDHeader = "X-Tenant-ID"

	// APIKeyHeader는 API Key를 전달하는 HTTP 헤더입니다
	APIKeyHeader = "X-API-Key"

	// TenantContextKey는 Gin context에 테넌트 정보를 저장하는 키입니다
	TenantContextKey = "tenant"
)

// TenantMiddleware는 Multi-tenancy를 지원하는 미들웨어입니다
// 요청 헤더에서 테넌트 정보를 추출하고 검증합니다
func TenantMiddleware(tenantRepo tenant.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// 1. 테넌트 ID 추출 (헤더 또는 API Key)
		tenantID := c.GetHeader(TenantIDHeader)
		apiKey := c.GetHeader(APIKeyHeader)

		var t *tenant.Tenant
		var err error

		// API Key가 있으면 우선 사용
		if apiKey != "" {
			t, err = tenantRepo.GetByAPIKey(ctx, apiKey)
			if err != nil {
				logger.Warn(ctx, "invalid API key",
					zap.String("api_key", maskAPIKey(apiKey)),
					zap.Error(err),
				)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_api_key",
					"message": "The provided API key is invalid or expired",
				})
				c.Abort()
				return
			}
		} else if tenantID != "" {
			// Tenant ID로 조회
			t, err = tenantRepo.GetByID(ctx, tenantID)
			if err != nil {
				logger.Warn(ctx, "tenant not found",
					zap.String("tenant_id", tenantID),
					zap.Error(err),
				)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "tenant_not_found",
					"message": "The specified tenant does not exist",
				})
				c.Abort()
				return
			}
		} else {
			// 테넌트 정보가 없으면 기본 테넌트 사용 (개발 모드)
			logger.Warn(ctx, "no tenant information provided, using default tenant")
			t = getDefaultTenant()
		}

		// 2. 테넌트 상태 확인
		if !t.IsActive() {
			logger.Warn(ctx, "inactive tenant attempted access",
				zap.String("tenant_id", t.ID),
				zap.String("status", string(t.Status)),
			)
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "tenant_inactive",
				"message": "Your account is currently inactive. Please contact support.",
				"status":  t.Status,
			})
			c.Abort()
			return
		}

		// 3. API 호출 수 체크
		if err := t.CheckQuota(tenant.QuotaTypeAPICalls); err != nil {
			logger.Warn(ctx, "tenant quota exceeded",
				zap.String("tenant_id", t.ID),
				zap.String("quota_type", string(tenant.QuotaTypeAPICalls)),
			)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "quota_exceeded",
				"message": "Daily API call limit exceeded. Please upgrade your plan or wait until tomorrow.",
				"quota": gin.H{
					"limit":   t.Quotas.MaxAPICallsPerDay,
					"current": t.Usage.APICallsToday,
				},
			})
			c.Abort()
			return
		}

		// 4. Context에 테넌트 정보 추가
		ctx = tenant.NewContext(ctx, t)
		c.Request = c.Request.WithContext(ctx)
		c.Set(TenantContextKey, t)

		// 5. 로깅용 필드 추가
		logger.WithFields(ctx,
			zap.String("tenant_id", t.ID),
			zap.String("tenant_name", t.Name),
			zap.String("plan", string(t.Plan)),
		)

		// 6. API 호출 카운트 증가 (비동기)
		go func() {
			if err := tenantRepo.IncrementAPICallCount(c.Request.Context(), t.ID); err != nil {
				logger.Error(ctx, "failed to increment API call count",
					zap.String("tenant_id", t.ID),
					zap.Error(err),
				)
			}
		}()

		c.Next()
	}
}

// TenantRateLimitMiddleware는 테넌트별 rate limiting을 적용합니다
func TenantRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Context에서 테넌트 정보 가져오기
		t, exists := c.Get(TenantContextKey)
		if !exists {
			c.Next()
			return
		}

		tenant := t.(*tenant.Tenant)

		// 테넌트별 rate limit은 이미 rate limiting 미들웨어에서 처리
		// 여기서는 추가 검증만 수행
		if tenant.Quotas.RateLimitPerMin > 0 {
			// Rate limit 정보를 헤더에 추가
			c.Header("X-RateLimit-Limit", string(rune(tenant.Quotas.RateLimitPerMin)))
			c.Header("X-RateLimit-Remaining", "N/A") // Redis에서 가져와야 함
		}

		c.Next()
	}
}

// RequireFeature는 특정 기능이 활성화된 테넌트만 접근할 수 있도록 제한합니다
func RequireFeature(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Context에서 테넌트 정보 가져오기
		t, exists := c.Get(TenantContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Tenant information not found",
			})
			c.Abort()
			return
		}

		tenant := t.(*tenant.Tenant)

		// 기능 활성화 여부 확인
		if !tenant.HasFeature(feature) {
			logger.Warn(c.Request.Context(), "feature not enabled for tenant",
				zap.String("tenant_id", tenant.ID),
				zap.String("feature", feature),
				zap.String("plan", string(tenant.Plan)),
			)
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "feature_not_available",
				"message": "This feature is not available in your current plan",
				"feature": feature,
				"plan":    tenant.Plan,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePlan은 특정 요금제 이상의 테넌트만 접근할 수 있도록 제한합니다
func RequirePlan(minPlan tenant.Plan) gin.HandlerFunc {
	planLevels := map[tenant.Plan]int{
		tenant.PlanFree:         1,
		tenant.PlanBasic:        2,
		tenant.PlanProfessional: 3,
		tenant.PlanEnterprise:   4,
	}

	return func(c *gin.Context) {
		// Context에서 테넌트 정보 가져오기
		t, exists := c.Get(TenantContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Tenant information not found",
			})
			c.Abort()
			return
		}

		tenant := t.(*tenant.Tenant)

		// 요금제 레벨 확인
		currentLevel := planLevels[tenant.Plan]
		requiredLevel := planLevels[minPlan]

		if currentLevel < requiredLevel {
			logger.Warn(c.Request.Context(), "insufficient plan level",
				zap.String("tenant_id", tenant.ID),
				zap.String("current_plan", string(tenant.Plan)),
				zap.String("required_plan", string(minPlan)),
			)
			c.JSON(http.StatusForbidden, gin.H{
				"error":        "plan_upgrade_required",
				"message":      "This endpoint requires a higher plan level",
				"current_plan": tenant.Plan,
				"required_plan": minPlan,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TenantDatabaseIsolation은 테넌트별 데이터베이스 분리를 위한 미들웨어입니다
// 각 테넌트는 별도의 데이터베이스 또는 컬렉션 prefix를 사용합니다
func TenantDatabaseIsolation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Context에서 테넌트 정보 가져오기
		t, exists := c.Get(TenantContextKey)
		if !exists {
			c.Next()
			return
		}

		tenant := t.(*tenant.Tenant)

		// 데이터베이스 이름을 헤더에 추가 (Repository layer에서 사용)
		c.Set("tenant_database", tenant.GetDatabase())

		c.Next()
	}
}

// getDefaultTenant는 개발/테스트용 기본 테넌트를 반환합니다
func getDefaultTenant() *tenant.Tenant {
	return &tenant.Tenant{
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
		},
		Settings: make(map[string]string),
		Metadata: make(map[string]string),
	}
}

// maskAPIKey는 API Key를 마스킹합니다 (로깅용)
func maskAPIKey(apiKey string) string {
	if len(apiKey) < 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// GetTenantFromContext는 Gin context에서 테넌트 정보를 가져옵니다
func GetTenantFromContext(c *gin.Context) (*tenant.Tenant, bool) {
	t, exists := c.Get(TenantContextKey)
	if !exists {
		return nil, false
	}
	tenant, ok := t.(*tenant.Tenant)
	return tenant, ok
}
