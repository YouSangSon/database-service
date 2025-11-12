package repository

import (
	"context"

	"github.com/YouSangSon/database-service/internal/domain/entity"
)

// DocumentRepository는 문서 저장소 인터페이스입니다
type DocumentRepository interface {
	// Save는 문서를 저장합니다
	Save(ctx context.Context, doc *entity.Document) error

	// FindByID는 ID로 문서를 조회합니다
	FindByID(ctx context.Context, collection, id string) (*entity.Document, error)

	// Update는 문서를 업데이트합니다 (낙관적 잠금 포함)
	Update(ctx context.Context, doc *entity.Document) error

	// Delete는 문서를 삭제합니다
	Delete(ctx context.Context, collection, id string) error

	// FindAll은 컬렉션의 모든 문서를 조회합니다
	FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error)

	// Count는 문서 개수를 반환합니다
	Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error)

	// HealthCheck는 저장소의 상태를 확인합니다
	HealthCheck(ctx context.Context) error
}

// CacheRepository는 캐시 저장소 인터페이스입니다
type CacheRepository interface {
	// Get은 캐시에서 값을 가져옵니다
	Get(ctx context.Context, key string) (interface{}, error)

	// Set은 캐시에 값을 저장합니다
	Set(ctx context.Context, key string, value interface{}, ttl int) error

	// Delete는 캐시에서 값을 삭제합니다
	Delete(ctx context.Context, key string) error

	// Exists는 키가 존재하는지 확인합니다
	Exists(ctx context.Context, key string) (bool, error)
}
