package cassandra

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

// Config는 Cassandra 연결 설정입니다
type Config struct {
	Hosts    []string // Cassandra cluster hosts
	Port     int
	Keyspace string
	Username string
	Password string

	// Connection Settings
	Consistency      string        // ONE, QUORUM, ALL, etc.
	Timeout          time.Duration
	ConnectTimeout   time.Duration
	NumConns         int // 호스트당 연결 수
	ReconnectInterval time.Duration

	// Retry Policy
	MaxRetries int
}

// NewClient는 Cassandra 클라이언트를 생성합니다
func NewClient(ctx context.Context, config *Config) (*gocql.Session, error) {
	cluster := gocql.NewCluster(config.Hosts...)
	cluster.Port = config.Port
	cluster.Keyspace = config.Keyspace

	// Authentication
	if config.Username != "" && config.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: config.Username,
			Password: config.Password,
		}
	}

	// Consistency
	consistency := getConsistency(config.Consistency)
	cluster.Consistency = consistency

	// Timeouts
	if config.Timeout > 0 {
		cluster.Timeout = config.Timeout
	} else {
		cluster.Timeout = 10 * time.Second // 기본값
	}

	if config.ConnectTimeout > 0 {
		cluster.ConnectTimeout = config.ConnectTimeout
	} else {
		cluster.ConnectTimeout = 10 * time.Second // 기본값
	}

	// Connection Pool
	if config.NumConns > 0 {
		cluster.NumConns = config.NumConns
	} else {
		cluster.NumConns = 2 // 기본값
	}

	// Retry Policy
	if config.MaxRetries > 0 {
		cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: config.MaxRetries}
	} else {
		cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3} // 기본값
	}

	// Reconnection Policy
	if config.ReconnectInterval > 0 {
		cluster.ReconnectInterval = config.ReconnectInterval
	} else {
		cluster.ReconnectInterval = 10 * time.Second // 기본값
	}

	// Protocol Version
	cluster.ProtoVersion = 4

	// Create Session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// Close는 Cassandra 세션을 닫습니다
func Close(session *gocql.Session) {
	if session != nil {
		session.Close()
	}
}

// getConsistency는 문자열에서 Consistency 레벨을 반환합니다
func getConsistency(consistency string) gocql.Consistency {
	switch consistency {
	case "ONE":
		return gocql.One
	case "TWO":
		return gocql.Two
	case "THREE":
		return gocql.Three
	case "QUORUM":
		return gocql.Quorum
	case "ALL":
		return gocql.All
	case "LOCAL_QUORUM":
		return gocql.LocalQuorum
	case "EACH_QUORUM":
		return gocql.EachQuorum
	case "LOCAL_ONE":
		return gocql.LocalOne
	case "ANY":
		return gocql.Any
	default:
		return gocql.Quorum // 기본값
	}
}

// CreateKeyspace는 Keyspace를 생성합니다
func CreateKeyspace(session *gocql.Session, keyspace string, replicationFactor int) error {
	query := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH REPLICATION = {
			'class': 'SimpleStrategy',
			'replication_factor': %d
		}
	`, keyspace, replicationFactor)

	return session.Query(query).Exec()
}

// DropKeyspace는 Keyspace를 삭제합니다
func DropKeyspace(session *gocql.Session, keyspace string) error {
	query := fmt.Sprintf(`DROP KEYSPACE IF EXISTS %s`, keyspace)
	return session.Query(query).Exec()
}
