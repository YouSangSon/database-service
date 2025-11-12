package repository

import (
	"context"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
)

// DocumentCommandRepository는 문서 쓰기 전용 저장소 인터페이스입니다 (CQRS Write Side)
// 주 데이터베이스(Primary)에 대한 쓰기 작업만 처리합니다
type DocumentCommandRepository interface {
	// ===== 기본 쓰기 작업 (Create, Update, Delete) =====

	// Save는 문서를 저장합니다
	Save(ctx context.Context, doc *entity.Document) error

	// SaveMany는 여러 문서를 한 번에 저장합니다
	// Bulk insert 최적화를 통해 성능 향상
	SaveMany(ctx context.Context, docs []*entity.Document) error

	// Update는 문서를 업데이트합니다 (낙관적 잠금 포함)
	// Version 필드를 사용한 동시성 제어
	Update(ctx context.Context, doc *entity.Document) error

	// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
	// 반환값: 업데이트된 문서 개수
	UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error)

	// Replace는 문서를 완전히 교체합니다
	// Update와 다르게 전체 문서를 교체
	Replace(ctx context.Context, collection, id string, replacement *entity.Document) error

	// Delete는 문서를 삭제합니다
	Delete(ctx context.Context, collection, id string) error

	// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
	// 반환값: 삭제된 문서 개수
	DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error)

	// ===== 원자적 쓰기 연산 (Atomic Write Operations) =====

	// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다
	// 비관적 잠금을 통한 원자적 업데이트
	FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error)

	// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
	FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error)

	// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
	// 삭제 전 문서 내용이 필요한 경우 사용
	FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error)

	// Upsert는 문서가 없으면 생성하고 있으면 업데이트합니다
	// 반환값: 생성 또는 업데이트된 문서 ID
	Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error)

	// ===== 벌크 쓰기 작업 (Bulk Write Operations) =====

	// BulkWrite는 여러 쓰기 작업을 한 번에 실행합니다
	// 네트워크 왕복을 줄여 성능 향상
	BulkWrite(ctx context.Context, operations []*BulkOperation) (*BulkResult, error)

	// ===== 인덱스 관리 (Index Management) =====

	// CreateIndex는 단일 인덱스를 생성합니다
	// 반환값: 생성된 인덱스 이름
	CreateIndex(ctx context.Context, collection string, model IndexModel) (string, error)

	// CreateIndexes는 여러 인덱스를 생성합니다
	// 반환값: 생성된 인덱스 이름 목록
	CreateIndexes(ctx context.Context, collection string, models []IndexModel) ([]string, error)

	// DropIndex는 인덱스를 삭제합니다
	DropIndex(ctx context.Context, collection, indexName string) error

	// ===== 컬렉션 관리 (Collection Management) =====

	// CreateCollection은 컬렉션을 생성합니다
	CreateCollection(ctx context.Context, name string) error

	// DropCollection은 컬렉션을 삭제합니다
	// 주의: 데이터가 영구 삭제됨
	DropCollection(ctx context.Context, name string) error

	// RenameCollection은 컬렉션 이름을 변경합니다
	RenameCollection(ctx context.Context, oldName, newName string) error

	// ===== 트랜잭션 (Transaction) =====

	// WithTransaction은 트랜잭션 내에서 함수를 실행합니다
	// ACID 보장이 필요한 복잡한 쓰기 작업에 사용
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// ===== 변경 스트림 설정 (Change Stream Setup) =====

	// EnableChangeDataCapture는 CDC를 활성화하여 모든 변경사항을 Kafka로 발행합니다
	// 이벤트 소싱 및 CQRS의 Read Model 동기화에 사용
	EnableChangeDataCapture(ctx context.Context, collections []string) error

	// DisableChangeDataCapture는 CDC를 비활성화합니다
	DisableChangeDataCapture(ctx context.Context) error

	// ===== Raw Command 실행 =====

	// ExecuteWriteCommand는 데이터베이스별 쓰기 명령을 실행합니다
	// MongoDB: RunCommand를 사용하여 임의의 쓰기 명령 실행
	// Vitess: INSERT, UPDATE, DELETE SQL 실행
	ExecuteWriteCommand(ctx context.Context, command interface{}) (interface{}, error)

	// ===== 헬스체크 =====

	// HealthCheck는 쓰기 저장소의 상태를 확인합니다
	// Primary 노드와의 연결 상태 확인
	HealthCheck(ctx context.Context) error

	// Close는 연결을 종료합니다
	Close(ctx context.Context) error
}

// CommandResult는 쓰기 작업 결과를 나타냅니다
type CommandResult struct {
	Success       bool
	AffectedRows  int64
	InsertedID    string
	ModifiedCount int64
	DeletedCount  int64
	ErrorMessage  string
}

// WriteOptions는 쓰기 작업 옵션입니다
type WriteOptions struct {
	// WriteConcern - 쓰기 승인 수준
	// "majority" (기본): 과반수 노드 승인
	// "1": Primary 노드만 승인
	WriteConcern string

	// Timeout - 작업 타임아웃 (밀리초)
	Timeout int

	// RetryWrites - 재시도 가능한 쓰기 활성화
	RetryWrites bool

	// Ordered - 벌크 작업 시 순서 보장 여부
	// true: 에러 발생 시 중단
	// false: 에러 발생해도 계속 실행
	Ordered bool

	// PublishCDC - CDC 이벤트 발행 여부
	PublishCDC bool
}

// WritePreference는 쓰기 선호도를 나타냅니다
type WritePreference struct {
	// Mode - 쓰기 모드 (primary, primaryPreferred)
	Mode string

	// MaxStaleness - 최대 지연 시간 (초)
	MaxStaleness int

	// Tags - 노드 태그 기반 라우팅
	Tags map[string]string
}

// TransactionOptions는 트랜잭션 옵션입니다
type TransactionOptions struct {
	// ReadConcern - 읽기 일관성 수준
	ReadConcern string

	// WriteConcern - 쓰기 승인 수준
	WriteConcern string

	// ReadPreference - 읽기 선호도
	ReadPreference string

	// MaxCommitTime - 최대 커밋 시간 (밀리초)
	MaxCommitTime int
}

// IndexStrategy는 인덱스 생성 전략입니다
type IndexStrategy struct {
	// CreateOnWrite - 쓰기 시 자동 인덱스 생성
	CreateOnWrite bool

	// Background - 백그라운드로 인덱스 생성
	Background bool

	// BuildIndexes - 벌크 삽입 후 인덱스 재구성
	BuildIndexes []string
}

// CDCConfig는 Change Data Capture 설정입니다
type CDCConfig struct {
	// Enabled - CDC 활성화 여부
	Enabled bool

	// KafkaTopic - Kafka 토픽 이름 패턴
	// 예: "db.events.{collection}.{operation}"
	KafkaTopicPattern string

	// IncludeFullDocument - 전체 문서 포함 여부
	IncludeFullDocument bool

	// Operations - 감지할 작업 (insert, update, delete, replace)
	Operations []string

	// BatchSize - 배치 크기
	BatchSize int
}
