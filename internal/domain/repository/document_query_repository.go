package repository

import (
	"context"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// DocumentQueryRepository는 문서 읽기 전용 저장소 인터페이스입니다 (CQRS Read Side)
// Read Replica 또는 별도의 Read Model에서 읽기 작업만 처리합니다
// 쓰기 작업은 DocumentCommandRepository를 사용해야 합니다
type DocumentQueryRepository interface {
	// ===== 기본 조회 작업 (Read Operations) =====

	// FindByID는 ID로 문서를 조회합니다
	// 가장 빠른 조회 방법 (Primary Key 사용)
	FindByID(ctx context.Context, collection, id string) (*entity.Document, error)

	// FindOne은 필터와 일치하는 첫 번째 문서를 조회합니다
	FindOne(ctx context.Context, collection string, filter map[string]interface{}) (*entity.Document, error)

	// FindAll은 필터와 일치하는 모든 문서를 조회합니다
	// 주의: 대용량 데이터의 경우 FindWithOptions 사용 권장
	FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error)

	// FindWithOptions는 옵션을 사용하여 문서를 조회합니다
	// Sort, Limit, Skip, Projection 지원
	FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *FindOptions) ([]*entity.Document, error)

	// FindByIDs는 여러 ID로 문서를 배치 조회합니다
	// N+1 쿼리 문제 해결
	FindByIDs(ctx context.Context, collection string, ids []string) ([]*entity.Document, error)

	// ===== 집계 작업 (Aggregation Operations) =====

	// Aggregate는 집계 파이프라인을 실행합니다
	// 복잡한 데이터 분석 및 변환에 사용
	Aggregate(ctx context.Context, collection string, pipeline []bson.M) ([]map[string]interface{}, error)

	// AggregateWithOptions는 옵션을 사용하여 집계를 실행합니다
	AggregateWithOptions(ctx context.Context, collection string, pipeline []bson.M, opts *AggregateOptions) ([]map[string]interface{}, error)

	// Distinct는 필드의 고유한 값을 조회합니다
	// 중복 제거된 값 목록 반환
	Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error)

	// ===== 카운트 작업 (Count Operations) =====

	// Count는 필터와 일치하는 문서 개수를 반환합니다
	// 정확한 개수가 필요한 경우 사용
	Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error)

	// EstimatedDocumentCount는 컬렉션의 추정 문서 개수를 반환합니다
	// 메타데이터 사용으로 매우 빠름 (대용량 컬렉션에 적합)
	EstimatedDocumentCount(ctx context.Context, collection string) (int64, error)

	// CountWithOptions는 옵션을 사용하여 개수를 반환합니다
	CountWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *CountOptions) (int64, error)

	// ===== 페이지네이션 (Pagination) =====

	// FindPage는 페이지 단위로 문서를 조회합니다
	// 커서 기반 페이지네이션 지원
	FindPage(ctx context.Context, collection string, filter map[string]interface{}, page *PageRequest) (*PageResponse, error)

	// FindCursorBased는 커서 기반 페이지네이션을 지원합니다
	// 대용량 데이터의 효율적인 페이징
	FindCursorBased(ctx context.Context, collection string, filter map[string]interface{}, cursor string, limit int) (*CursorPageResponse, error)

	// ===== 검색 작업 (Search Operations) =====

	// Search는 전문 검색을 수행합니다
	// Text Index가 필요함
	Search(ctx context.Context, collection, searchText string, opts *SearchOptions) ([]*entity.Document, error)

	// FindByRegex는 정규식 패턴으로 문서를 검색합니다
	FindByRegex(ctx context.Context, collection, field, pattern string) ([]*entity.Document, error)

	// ===== 컬렉션 정보 조회 =====

	// ListCollections는 데이터베이스의 컬렉션 목록을 반환합니다
	ListCollections(ctx context.Context) ([]string, error)

	// CollectionExists는 컬렉션이 존재하는지 확인합니다
	CollectionExists(ctx context.Context, name string) (bool, error)

	// GetCollectionStats는 컬렉션 통계를 반환합니다
	GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error)

	// ===== 인덱스 정보 조회 =====

	// ListIndexes는 컬렉션의 인덱스 목록을 반환합니다
	ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error)

	// GetIndexStats는 인덱스 사용 통계를 반환합니다
	GetIndexStats(ctx context.Context, collection string) ([]IndexStat, error)

	// ===== 변경 스트림 구독 (Change Streams) =====

	// Watch는 컬렉션의 변경 사항을 실시간으로 감지합니다
	// 이벤트 기반 아키텍처 구현에 사용
	Watch(ctx context.Context, collection string, pipeline []bson.M) (*mongo.ChangeStream, error)

	// WatchWithOptions는 옵션을 사용하여 변경 스트림을 생성합니다
	WatchWithOptions(ctx context.Context, collection string, pipeline []bson.M, opts *WatchOptions) (*mongo.ChangeStream, error)

	// ===== 데이터 검증 (Data Validation) =====

	// Explain은 쿼리 실행 계획을 반환합니다
	// 쿼리 최적화 및 성능 분석에 사용
	Explain(ctx context.Context, collection string, filter map[string]interface{}) (map[string]interface{}, error)

	// ValidateDocument는 문서가 스키마 검증 규칙을 만족하는지 확인합니다
	ValidateDocument(ctx context.Context, collection string, doc map[string]interface{}) (bool, error)

	// ===== Raw Query 실행 =====

	// ExecuteReadQuery는 데이터베이스별 읽기 쿼리를 실행합니다
	// MongoDB: find, aggregate 등의 명령 실행
	// Vitess: SELECT 쿼리 실행
	ExecuteReadQuery(ctx context.Context, query interface{}) (interface{}, error)

	// ExecuteReadQueryWithResult는 읽기 쿼리를 실행하고 결과를 특정 타입으로 반환합니다
	ExecuteReadQueryWithResult(ctx context.Context, query interface{}, result interface{}) error

	// ===== 캐시 통합 (Cache Integration) =====

	// FindByIDWithCache는 캐시를 사용하여 ID로 문서를 조회합니다
	// Cache-aside 패턴 구현
	FindByIDWithCache(ctx context.Context, collection, id string, ttl int) (*entity.Document, error)

	// InvalidateCache는 특정 키의 캐시를 무효화합니다
	InvalidateCache(ctx context.Context, collection, id string) error

	// WarmUpCache는 자주 사용되는 데이터를 캐시에 미리 로드합니다
	WarmUpCache(ctx context.Context, collection string, ids []string) error

	// ===== 분석 쿼리 (Analytics Queries) =====

	// GetTimeSeriesData는 시계열 데이터를 조회합니다
	// 시간 범위 기반 조회
	GetTimeSeriesData(ctx context.Context, collection string, startTime, endTime int64, interval string) ([]map[string]interface{}, error)

	// GetTopN는 상위 N개 문서를 조회합니다
	// 정렬 및 제한을 포함한 최적화된 쿼리
	GetTopN(ctx context.Context, collection string, sortField string, n int) ([]*entity.Document, error)

	// GroupBy는 필드별로 그룹화하여 집계합니다
	GroupBy(ctx context.Context, collection string, groupField string, aggregations map[string]string) ([]map[string]interface{}, error)

	// ===== 헬스체크 =====

	// HealthCheck는 읽기 저장소의 상태를 확인합니다
	// Read Replica와의 연결 상태 및 Replication Lag 확인
	HealthCheck(ctx context.Context) error

	// GetReplicationLag는 복제 지연 시간을 반환합니다 (밀리초)
	// Read Replica의 최신성 확인
	GetReplicationLag(ctx context.Context) (int64, error)

	// Close는 연결을 종료합니다
	Close(ctx context.Context) error
}

// PageRequest는 페이지 요청 정보입니다
type PageRequest struct {
	Page     int                    // 페이지 번호 (1부터 시작)
	PageSize int                    // 페이지 크기
	Sort     map[string]int         // 정렬 (1: 오름차순, -1: 내림차순)
	Filter   map[string]interface{} // 필터
}

// PageResponse는 페이지 응답 정보입니다
type PageResponse struct {
	Items      []*entity.Document // 문서 목록
	TotalItems int64              // 전체 항목 수
	TotalPages int                // 전체 페이지 수
	Page       int                // 현재 페이지
	PageSize   int                // 페이지 크기
	HasNext    bool               // 다음 페이지 존재 여부
	HasPrev    bool               // 이전 페이지 존재 여부
}

// CursorPageResponse는 커서 기반 페이지 응답입니다
type CursorPageResponse struct {
	Items      []*entity.Document // 문서 목록
	NextCursor string             // 다음 커서 (없으면 빈 문자열)
	HasMore    bool               // 더 많은 데이터 존재 여부
}

// SearchOptions는 검색 옵션입니다
type SearchOptions struct {
	Limit      int                    // 결과 제한
	Skip       int                    // 건너뛰기
	Sort       map[string]int         // 정렬
	Projection map[string]interface{} // 필드 선택
	Fuzzy      bool                   // 퍼지 검색 활성화
	Language   string                 // 언어 (예: "korean", "english")
}

// AggregateOptions는 집계 옵션입니다
type AggregateOptions struct {
	AllowDiskUse bool // 디스크 사용 허용 (대용량 집계)
	MaxTime      int  // 최대 실행 시간 (밀리초)
	BatchSize    int  // 배치 크기
	Collation    *Collation
}

// CountOptions는 카운트 옵션입니다
type CountOptions struct {
	Limit int64 // 카운트 제한
	Skip  int64 // 건너뛰기
	Hint  string
}

// WatchOptions는 변경 스트림 옵션입니다
type WatchOptions struct {
	FullDocument   string // "updateLookup", "whenAvailable", "required"
	ResumeAfter    interface{}
	StartAtTime    interface{}
	BatchSize      int32
	MaxAwaitTime   int64 // 밀리초
	Collation      *Collation
	StartAfter     interface{}
	ShowExpandedEvents bool
}

// Collation은 정렬 규칙입니다
type Collation struct {
	Locale          string
	CaseLevel       bool
	CaseFirst       string
	Strength        int
	NumericOrdering bool
	Alternate       string
	MaxVariable     string
	Backwards       bool
}

// CollectionStats는 컬렉션 통계입니다
type CollectionStats struct {
	Collection     string  // 컬렉션 이름
	Count          int64   // 문서 개수
	Size           int64   // 데이터 크기 (바이트)
	AvgDocSize     float64 // 평균 문서 크기
	StorageSize    int64   // 저장소 크기
	IndexCount     int     // 인덱스 개수
	TotalIndexSize int64   // 전체 인덱스 크기
}

// IndexStat는 인덱스 사용 통계입니다
type IndexStat struct {
	Name        string // 인덱스 이름
	Accesses    int64  // 접근 횟수
	Since       string // 통계 수집 시작 시간
	LastAccess  string // 마지막 접근 시간
	Size        int64  // 인덱스 크기 (바이트)
}

// QueryOptions는 쿼리 실행 옵션입니다
type QueryOptions struct {
	// ReadPreference - 읽기 선호도
	// "primary": Primary에서만 읽기
	// "primaryPreferred": Primary 우선, 없으면 Secondary
	// "secondary": Secondary에서만 읽기
	// "secondaryPreferred": Secondary 우선, 없으면 Primary
	// "nearest": 가장 가까운 노드에서 읽기
	ReadPreference string

	// MaxStaleness - 최대 복제 지연 시간 (초)
	// Secondary 읽기 시 최대 허용 지연
	MaxStaleness int

	// ReadConcern - 읽기 일관성 수준
	// "local": 로컬 데이터 읽기 (기본)
	// "available": 사용 가능한 데이터 읽기
	// "majority": 과반수 노드가 승인한 데이터 읽기
	// "linearizable": 선형화 가능한 읽기 (가장 강한 일관성)
	// "snapshot": 스냅샷 읽기 (트랜잭션용)
	ReadConcern string

	// Hint - 사용할 인덱스 지정
	// 쿼리 옵티마이저에게 힌트 제공
	Hint string

	// Comment - 쿼리 주석
	// 로그 및 프로파일링에서 쿼리 식별에 사용
	Comment string

	// MaxTime - 최대 실행 시간 (밀리초)
	// 장기 실행 쿼리 방지
	MaxTime int

	// AllowPartialResults - 부분 결과 허용
	// Sharded cluster에서 일부 샤드가 실패해도 결과 반환
	AllowPartialResults bool
}

// ReadModelConfig는 Read Model 설정입니다
type ReadModelConfig struct {
	// UseReadReplica - Read Replica 사용 여부
	UseReadReplica bool

	// ReplicaAddress - Read Replica 주소
	ReplicaAddress string

	// FallbackToPrimary - Primary로 폴백 허용
	FallbackToPrimary bool

	// MaxReplicationLag - 허용 가능한 최대 복제 지연 (밀리초)
	// 이 값을 초과하면 Primary로 폴백
	MaxReplicationLag int64

	// CacheEnabled - 캐시 활성화
	CacheEnabled bool

	// CacheTTL - 캐시 TTL (초)
	CacheTTL int

	// WarmUpCollections - 시작 시 캐시에 로드할 컬렉션 목록
	WarmUpCollections []string
}

// QueryPerformanceStats는 쿼리 성능 통계입니다
type QueryPerformanceStats struct {
	ExecutionTime    int64  // 실행 시간 (밀리초)
	DocsExamined     int64  // 검사한 문서 수
	DocsReturned     int64  // 반환한 문서 수
	IndexUsed        string // 사용된 인덱스
	CacheHit         bool   // 캐시 히트 여부
	ReplicaUsed      string // 사용된 replica (primary/secondary)
	ReplicationLag   int64  // 복제 지연 (밀리초)
}
