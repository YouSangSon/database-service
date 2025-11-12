package vitess

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// ExecuteRawQuery는 임의의 SQL 쿼리를 실행합니다
// query는 string (SQL 문) 또는 준비된 statement여야 합니다
//
// SELECT 쿼리의 경우 결과를 []map[string]interface{} 형태로 반환합니다
// INSERT/UPDATE/DELETE의 경우 영향받은 행 수를 반환합니다
//
// 예제:
//   // SELECT 쿼리
//   result, err := repo.ExecuteRawQuery(ctx, "SELECT * FROM documents WHERE collection = 'users'")
//
//   // INSERT 쿼리
//   result, err := repo.ExecuteRawQuery(ctx, "INSERT INTO documents (id, collection, data) VALUES ('123', 'users', '{}')")
func (r *VitessRepository) ExecuteRawQuery(ctx context.Context, query interface{}) (interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("raw_query", r.keyspace, "success", duration)
	}()

	// query를 문자열로 변환
	sqlQuery, ok := query.(string)
	if !ok {
		r.metrics.RecordDBOperation("raw_query", r.keyspace, "error", time.Since(start))
		return nil, fmt.Errorf("query must be a string, got %T", query)
	}

	logger.Debug(ctx, "executing raw SQL query",
		logger.Field("query", sqlQuery),
		logger.Field("keyspace", r.keyspace),
	)

	// 쿼리 타입 판별 (SELECT vs DML)
	isSelect := isSelectQuery(sqlQuery)

	if isSelect {
		// SELECT 쿼리 실행
		rows, err := r.db.QueryContext(ctx, sqlQuery)
		if err != nil {
			r.metrics.RecordDBOperation("raw_query", r.keyspace, "error", time.Since(start))
			logger.Error(ctx, "failed to execute SELECT query",
				logger.Field("query", sqlQuery),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to execute SELECT query: %w", err)
		}
		defer rows.Close()

		// 결과 처리
		results, err := scanRows(rows)
		if err != nil {
			r.metrics.RecordDBOperation("raw_query", r.keyspace, "error", time.Since(start))
			return nil, fmt.Errorf("failed to scan rows: %w", err)
		}

		logger.Info(ctx, "raw SELECT query executed",
			logger.Field("rows", len(results)),
			logger.Duration(time.Since(start)),
		)

		return results, nil
	}

	// DML 쿼리 (INSERT/UPDATE/DELETE) 실행
	result, err := r.db.ExecContext(ctx, sqlQuery)
	if err != nil {
		r.metrics.RecordDBOperation("raw_query", r.keyspace, "error", time.Since(start))
		logger.Error(ctx, "failed to execute DML query",
			logger.Field("query", sqlQuery),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to execute DML query: %w", err)
	}

	// 영향받은 행 수 가져오기
	affected, err := result.RowsAffected()
	if err != nil {
		logger.Warn(ctx, "failed to get rows affected", zap.Error(err))
	}

	// LastInsertId (INSERT의 경우)
	lastID, err := result.LastInsertId()
	if err != nil {
		// LastInsertId가 없는 경우 (UPDATE/DELETE) 무시
		lastID = 0
	}

	response := map[string]interface{}{
		"rows_affected": affected,
		"last_insert_id": lastID,
	}

	logger.Info(ctx, "raw DML query executed",
		logger.Field("rows_affected", affected),
		logger.Field("last_insert_id", lastID),
		logger.Duration(time.Since(start)),
	)

	return response, nil
}

// ExecuteRawQueryWithResult는 SQL 쿼리를 실행하고 결과를 지정된 변수로 스캔합니다
//
// 예제:
//   var results []struct {
//       ID         string `db:"id"`
//       Collection string `db:"collection"`
//       Data       string `db:"data"`
//   }
//   err := repo.ExecuteRawQueryWithResult(ctx,
//       "SELECT id, collection, data FROM documents WHERE collection = ?",
//       &results, "users")
func (r *VitessRepository) ExecuteRawQueryWithResult(ctx context.Context, query interface{}, result interface{}) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("raw_query_with_result", r.keyspace, "success", duration)
	}()

	// query를 문자열로 변환
	sqlQuery, ok := query.(string)
	if !ok {
		r.metrics.RecordDBOperation("raw_query_with_result", r.keyspace, "error", time.Since(start))
		return fmt.Errorf("query must be a string, got %T", query)
	}

	logger.Debug(ctx, "executing raw SQL query with result",
		logger.Field("query", sqlQuery),
	)

	// SELECT 쿼리만 지원
	if !isSelectQuery(sqlQuery) {
		r.metrics.RecordDBOperation("raw_query_with_result", r.keyspace, "error", time.Since(start))
		return fmt.Errorf("ExecuteRawQueryWithResult only supports SELECT queries")
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		r.metrics.RecordDBOperation("raw_query_with_result", r.keyspace, "error", time.Since(start))
		logger.Error(ctx, "failed to execute query",
			logger.Field("query", sqlQuery),
			zap.Error(err),
		)
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// 결과를 []map[string]interface{}로 변환하여 result에 할당
	results, err := scanRows(rows)
	if err != nil {
		r.metrics.RecordDBOperation("raw_query_with_result", r.keyspace, "error", time.Since(start))
		return fmt.Errorf("failed to scan rows: %w", err)
	}

	// result가 포인터인지 확인하고 할당
	switch v := result.(type) {
	case *[]map[string]interface{}:
		*v = results
	case *interface{}:
		*v = results
	default:
		return fmt.Errorf("result must be *[]map[string]interface{} or *interface{}, got %T", result)
	}

	logger.Info(ctx, "raw query executed with result",
		logger.Field("rows", len(results)),
		logger.Duration(time.Since(start)),
	)

	return nil
}

// ExecutePreparedQuery는 준비된 statement를 사용하여 쿼리를 실행합니다
// SQL injection 방지를 위해 권장되는 방법입니다
//
// 예제:
//   result, err := repo.ExecutePreparedQuery(ctx,
//       "SELECT * FROM documents WHERE collection = ? AND version > ?",
//       "users", 5)
func (r *VitessRepository) ExecutePreparedQuery(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("prepared_query", r.keyspace, "success", duration)
	}()

	logger.Debug(ctx, "executing prepared SQL query",
		logger.Field("query", query),
		logger.Field("args", args),
	)

	// 쿼리 타입 판별
	isSelect := isSelectQuery(query)

	if isSelect {
		// SELECT 쿼리
		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			r.metrics.RecordDBOperation("prepared_query", r.keyspace, "error", time.Since(start))
			logger.Error(ctx, "failed to execute prepared SELECT query",
				logger.Field("query", query),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to execute prepared SELECT query: %w", err)
		}
		defer rows.Close()

		results, err := scanRows(rows)
		if err != nil {
			r.metrics.RecordDBOperation("prepared_query", r.keyspace, "error", time.Since(start))
			return nil, fmt.Errorf("failed to scan rows: %w", err)
		}

		logger.Info(ctx, "prepared SELECT query executed",
			logger.Field("rows", len(results)),
			logger.Duration(time.Since(start)),
		)

		return results, nil
	}

	// DML 쿼리
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.metrics.RecordDBOperation("prepared_query", r.keyspace, "error", time.Since(start))
		logger.Error(ctx, "failed to execute prepared DML query",
			logger.Field("query", query),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to execute prepared DML query: %w", err)
	}

	affected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()

	response := map[string]interface{}{
		"rows_affected": affected,
		"last_insert_id": lastID,
	}

	logger.Info(ctx, "prepared DML query executed",
		logger.Field("rows_affected", affected),
		logger.Duration(time.Since(start)),
	)

	return response, nil
}

// ExecuteBatch는 여러 SQL 문을 배치로 실행합니다
//
// 예제:
//   queries := []string{
//       "INSERT INTO documents (id, collection, data) VALUES ('1', 'users', '{}')",
//       "INSERT INTO documents (id, collection, data) VALUES ('2', 'users', '{}')",
//       "UPDATE documents SET data = '{}' WHERE id = '3'",
//   }
//   result, err := repo.ExecuteBatch(ctx, queries)
func (r *VitessRepository) ExecuteBatch(ctx context.Context, queries []string) ([]interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("batch_query", r.keyspace, "success", duration)
	}()

	// 트랜잭션 시작
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error(ctx, "failed to begin transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	results := make([]interface{}, 0, len(queries))

	for i, query := range queries {
		logger.Debug(ctx, "executing batch query",
			logger.Field("index", i),
			logger.Field("query", query),
		)

		result, err := tx.ExecContext(ctx, query)
		if err != nil {
			_ = tx.Rollback()
			r.metrics.RecordDBOperation("batch_query", r.keyspace, "error", time.Since(start))
			logger.Error(ctx, "failed to execute batch query",
				logger.Field("index", i),
				logger.Field("query", query),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to execute batch query at index %d: %w", i, err)
		}

		affected, _ := result.RowsAffected()
		lastID, _ := result.LastInsertId()

		results = append(results, map[string]interface{}{
			"rows_affected": affected,
			"last_insert_id": lastID,
		})
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit batch transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	logger.Info(ctx, "batch queries executed",
		logger.Field("count", len(queries)),
		logger.Duration(time.Since(start)),
	)

	return results, nil
}

// Helper functions

// isSelectQuery는 쿼리가 SELECT 쿼리인지 확인합니다
func isSelectQuery(query string) bool {
	// 간단한 휴리스틱: 쿼리가 SELECT, SHOW, DESCRIBE, EXPLAIN으로 시작하면 SELECT로 간주
	q := trimAndLower(query)
	return len(q) >= 6 && (q[:6] == "select" || q[:4] == "show" || q[:8] == "describe" || q[:7] == "explain")
}

// trimAndLower는 문자열을 trim하고 소문자로 변환합니다
func trimAndLower(s string) string {
	// 앞의 공백 제거
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	if i >= len(s) {
		return ""
	}
	s = s[i:]

	// 소문자로 변환 (첫 10글자만)
	if len(s) > 10 {
		s = s[:10]
	}
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

// scanRows는 sql.Rows를 []map[string]interface{}로 변환합니다
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}

	for rows.Next() {
		// 각 컬럼의 값을 저장할 슬라이스
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// map으로 변환
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// []byte를 string으로 변환
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return results, nil
}
