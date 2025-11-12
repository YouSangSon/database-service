package repository

import (
	"context"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// FindOptions는 조회 옵션입니다
type FindOptions struct {
	Sort       map[string]int         // 정렬 (1: 오름차순, -1: 내림차순)
	Limit      int64                  // 제한
	Skip       int64                  // 건너뛰기
	Projection map[string]interface{} // 필드 선택
}

// IndexModel은 인덱스 모델입니다
type IndexModel struct {
	Keys    map[string]interface{} // 인덱스 키
	Options *IndexOptions          // 인덱스 옵션
}

// IndexOptions는 인덱스 생성 옵션입니다
type IndexOptions struct {
	Unique         *bool  // 고유 인덱스 여부
	Name           string // 인덱스 이름
	Background     *bool  // 백그라운드 생성
	Sparse         *bool  // Sparse 인덱스
	ExpireAfter    *int32 // TTL (초)
	PartialFilter  bson.M // 부분 인덱스 필터
	TextIndexField string // 텍스트 인덱스 필드
}

// BulkOperation은 벌크 작업입니다
type BulkOperation struct {
	Type         string                 // insert, update, delete, replace
	Collection   string                 // 컬렉션 이름
	Filter       map[string]interface{} // 필터 (update, delete, replace에 사용)
	Document     *entity.Document       // 문서 (insert, replace에 사용)
	Update       map[string]interface{} // 업데이트 내용 (update에 사용)
	Upsert       bool                   // Upsert 여부 (update, replace에 사용)
	DeleteMany   bool                   // 여러 문서 삭제 여부 (delete에 사용)
	UpdateMany   bool                   // 여러 문서 업데이트 여부 (update에 사용)
	ReplaceOneID string                 // Replace할 문서 ID (replace에 사용)
}

// BulkResult는 벌크 작업 결과입니다
type BulkResult struct {
	InsertedCount int64
	MatchedCount  int64
	ModifiedCount int64
	DeletedCount  int64
	UpsertedCount int64
	UpsertedIDs   map[int]interface{}
}

// DocumentRepository는 문서 저장소 인터페이스입니다
type DocumentRepository interface {
	// ===== 기본 CRUD =====

	// Save는 문서를 저장합니다
	Save(ctx context.Context, doc *entity.Document) error

	// SaveMany는 여러 문서를 한 번에 저장합니다
	SaveMany(ctx context.Context, docs []*entity.Document) error

	// FindByID는 ID로 문서를 조회합니다
	FindByID(ctx context.Context, collection, id string) (*entity.Document, error)

	// FindAll은 컬렉션의 모든 문서를 조회합니다
	FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error)

	// FindWithOptions는 옵션을 사용하여 문서를 조회합니다 (Sort, Limit, Skip, Projection)
	FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *FindOptions) ([]*entity.Document, error)

	// Update는 문서를 업데이트합니다 (낙관적 잠금 포함)
	Update(ctx context.Context, doc *entity.Document) error

	// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
	UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error)

	// Replace는 문서를 교체합니다 (이전 문서를 반환하지 않음)
	Replace(ctx context.Context, collection, id string, replacement *entity.Document) error

	// Delete는 문서를 삭제합니다
	Delete(ctx context.Context, collection, id string) error

	// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
	DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error)

	// ===== 원자적 연산 (Atomic Operations) =====

	// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다 (비관적 잠금)
	FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error)

	// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
	FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error)

	// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
	FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error)

	// Upsert는 문서가 없으면 생성하고 있으면 업데이트합니다
	Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error)

	// ===== 집계 (Aggregation) =====

	// Aggregate는 집계 파이프라인을 실행합니다
	Aggregate(ctx context.Context, collection string, pipeline []bson.M) ([]map[string]interface{}, error)

	// Distinct는 고유한 값을 조회합니다
	Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error)

	// Count는 문서 개수를 반환합니다 (정확한 개수)
	Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error)

	// EstimatedDocumentCount는 컬렉션의 추정 문서 개수를 반환합니다 (빠름)
	EstimatedDocumentCount(ctx context.Context, collection string) (int64, error)

	// ===== 벌크 작업 (Bulk Operations) =====

	// BulkWrite는 여러 작업을 한 번에 실행합니다
	BulkWrite(ctx context.Context, operations []*BulkOperation) (*BulkResult, error)

	// ===== 인덱스 관리 (Index Management) =====

	// CreateIndex는 단일 인덱스를 생성합니다
	CreateIndex(ctx context.Context, collection string, model IndexModel) (string, error)

	// CreateIndexes는 여러 인덱스를 생성합니다
	CreateIndexes(ctx context.Context, collection string, models []IndexModel) ([]string, error)

	// DropIndex는 인덱스를 삭제합니다
	DropIndex(ctx context.Context, collection, indexName string) error

	// ListIndexes는 컬렉션의 인덱스 목록을 반환합니다
	ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error)

	// ===== 컬렉션 관리 (Collection Management) =====

	// CreateCollection은 컬렉션을 생성합니다
	CreateCollection(ctx context.Context, name string) error

	// DropCollection은 컬렉션을 삭제합니다
	DropCollection(ctx context.Context, name string) error

	// RenameCollection은 컬렉션 이름을 변경합니다
	RenameCollection(ctx context.Context, oldName, newName string) error

	// ListCollections는 데이터베이스의 컬렉션 목록을 반환합니다
	ListCollections(ctx context.Context) ([]string, error)

	// CollectionExists는 컬렉션이 존재하는지 확인합니다
	CollectionExists(ctx context.Context, name string) (bool, error)

	// ===== Change Streams =====

	// Watch는 컬렉션의 변경 사항을 실시간으로 감지합니다
	Watch(ctx context.Context, collection string, pipeline []bson.M) (*mongo.ChangeStream, error)

	// ===== 트랜잭션 (Transaction) =====

	// WithTransaction은 트랜잭션 내에서 함수를 실행합니다
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// ===== 헬스체크 =====

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
