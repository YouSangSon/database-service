package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config는 PostgreSQL 연결 설정입니다
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string // disable, require, verify-ca, verify-full

	// Connection Pool Settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewClient는 PostgreSQL 클라이언트를 생성합니다
func NewClient(ctx context.Context, config *Config) (*sql.DB, error) {
	// Connection string 생성
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection Pool 설정
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25) // 기본값
	}

	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(5) // 기본값
	}

	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute) // 기본값
	}

	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	} else {
		db.SetConnMaxIdleTime(1 * time.Minute) // 기본값
	}

	// 연결 테스트
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close는 데이터베이스 연결을 닫습니다
func Close(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}
