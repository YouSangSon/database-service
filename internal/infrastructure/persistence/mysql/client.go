package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Config는 MySQL 연결 설정입니다
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string

	// Connection Parameters
	Charset            string // utf8mb4
	ParseTime          bool   // true: time.Time 자동 파싱
	Loc                string // UTC, Local, etc.
	AllowNativePasswords bool

	// Connection Pool Settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewClient는 MySQL 클라이언트를 생성합니다
func NewClient(ctx context.Context, config *Config) (*sql.DB, error) {
	// Connection string 생성 (DSN format)
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		getCharset(config.Charset),
		getParseTime(config.ParseTime),
		getLoc(config.Loc),
	)

	if config.AllowNativePasswords {
		dsn += "&allowNativePasswords=true"
	}

	db, err := sql.Open("mysql", dsn)
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

func getCharset(charset string) string {
	if charset == "" {
		return "utf8mb4"
	}
	return charset
}

func getParseTime(parseTime bool) bool {
	return parseTime // 기본값은 false이지만, true 권장
}

func getLoc(loc string) string {
	if loc == "" {
		return "UTC"
	}
	return loc
}
