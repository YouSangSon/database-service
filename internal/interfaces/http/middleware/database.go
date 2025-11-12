package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	DatabaseTypeMongoDB       DatabaseType = "mongodb"
	DatabaseTypePostgreSQL    DatabaseType = "postgresql"
	DatabaseTypeMySQL         DatabaseType = "mysql"
	DatabaseTypeCassandra     DatabaseType = "cassandra"
	DatabaseTypeElasticsearch DatabaseType = "elasticsearch"
	DatabaseTypeVitess        DatabaseType = "vitess"
)

// Context keys for database type
type contextKey string

const (
	DatabaseTypeContextKey contextKey = "database_type"
)

// DatabaseSelector는 데이터베이스 선택 미들웨어입니다
func DatabaseSelector() gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Database-Type 헤더에서 데이터베이스 타입 읽기
		dbType := c.GetHeader("X-Database-Type")

		// 기본값은 MongoDB
		if dbType == "" {
			dbType = string(DatabaseTypeMongoDB)
		}

		// 유효한 데이터베이스 타입인지 확인
		if !isValidDatabaseType(dbType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_DATABASE_TYPE",
					"message": "Invalid database type. Supported types: mongodb, postgresql, mysql, cassandra, elasticsearch, vitess",
				},
			})
			c.Abort()
			return
		}

		// 컨텍스트에 데이터베이스 타입 저장
		ctx := context.WithValue(c.Request.Context(), DatabaseTypeContextKey, DatabaseType(dbType))
		c.Request = c.Request.WithContext(ctx)

		// 다음 핸들러로 전달
		c.Next()
	}
}

// isValidDatabaseType은 데이터베이스 타입이 유효한지 확인합니다
func isValidDatabaseType(dbType string) bool {
	validTypes := []string{
		string(DatabaseTypeMongoDB),
		string(DatabaseTypePostgreSQL),
		string(DatabaseTypeMySQL),
		string(DatabaseTypeCassandra),
		string(DatabaseTypeElasticsearch),
		string(DatabaseTypeVitess),
	}

	for _, validType := range validTypes {
		if dbType == validType {
			return true
		}
	}

	return false
}

// GetDatabaseType은 컨텍스트에서 데이터베이스 타입을 가져옵니다
func GetDatabaseType(ctx context.Context) DatabaseType {
	dbType, ok := ctx.Value(DatabaseTypeContextKey).(DatabaseType)
	if !ok {
		return DatabaseTypeMongoDB // 기본값
	}
	return dbType
}
