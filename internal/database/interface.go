package database

import (
	"context"
)

// Database는 모든 데이터베이스 구현체가 따라야 할 인터페이스입니다
type Database interface {
	// Connect는 데이터베이스에 연결합니다
	Connect(ctx context.Context) error

	// Disconnect는 데이터베이스 연결을 종료합니다
	Disconnect(ctx context.Context) error

	// Ping은 데이터베이스 연결 상태를 확인합니다
	Ping(ctx context.Context) error

	// Create는 새로운 문서/레코드를 생성합니다
	Create(ctx context.Context, collection string, document interface{}) (string, error)

	// Read는 ID로 문서/레코드를 조회합니다
	Read(ctx context.Context, collection string, id string, result interface{}) error

	// Update는 기존 문서/레코드를 업데이트합니다
	Update(ctx context.Context, collection string, id string, update interface{}) error

	// Delete는 문서/레코드를 삭제합니다
	Delete(ctx context.Context, collection string, id string) error

	// List는 컬렉션/테이블의 모든 문서/레코드를 조회합니다
	List(ctx context.Context, collection string, filter interface{}, results interface{}) error

	// Query는 커스텀 쿼리를 실행합니다
	Query(ctx context.Context, collection string, query interface{}, results interface{}) error
}

// Config는 데이터베이스 연결 설정입니다
type Config struct {
	Type     string // mongodb, postgresql, mysql 등
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Options  map[string]string // 데이터베이스별 추가 옵션
}
