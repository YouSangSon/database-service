package vitess

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// CreateIndex는 단일 인덱스를 생성합니다
func (r *VitessRepository) CreateIndex(ctx context.Context, collection string, model repository.IndexModel) (string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("create_index", collection, "success", duration)
	}()

	// 인덱스 이름 생성
	indexName := model.Options.Name
	if indexName == "" {
		// 자동 생성: idx_collection_field1_field2
		fields := []string{}
		for field := range model.Keys {
			fields = append(fields, field)
		}
		indexName = fmt.Sprintf("idx_%s_%s", collection, strings.Join(fields, "_"))
	}

	// JSON 필드에 대한 인덱스 생성
	// MySQL 5.7+에서는 생성된 열(generated column)을 사용하거나
	// 직접 JSON_EXTRACT를 인덱스로 사용할 수 있습니다
	var indexFields []string
	for field, order := range model.Keys {
		direction := "ASC"
		if order == -1 {
			direction = "DESC"
		}

		// 표준 필드 (id, collection, created_at, updated_at)인 경우
		if field == "id" || field == "collection" || field == "created_at" || field == "updated_at" {
			indexFields = append(indexFields, fmt.Sprintf("%s %s", field, direction))
		} else {
			// JSON 필드에 대한 인덱스
			// (JSON_EXTRACT를 직접 인덱스로 사용)
			indexFields = append(indexFields, fmt.Sprintf("(CAST(JSON_EXTRACT(data, '$.%s') AS CHAR(255))) %s", field, direction))
		}
	}

	// UNIQUE 옵션
	uniqueStr := ""
	if model.Options.Unique != nil && *model.Options.Unique {
		uniqueStr = "UNIQUE"
	}

	// 인덱스 생성 쿼리
	query := fmt.Sprintf(`
		ALTER TABLE documents
		ADD %s INDEX %s (%s)
	`, uniqueStr, indexName, strings.Join(indexFields, ", "))

	logger.Debug(ctx, "creating index",
		logger.Collection(collection),
		logger.Field("index_name", indexName),
		logger.Field("query", query),
	)

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		// 이미 존재하는 인덱스인 경우 무시
		if strings.Contains(err.Error(), "Duplicate key name") {
			logger.Warn(ctx, "index already exists",
				logger.Collection(collection),
				logger.Field("index_name", indexName),
			)
			return indexName, nil
		}

		r.metrics.RecordDBOperation("create_index", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to create index",
			logger.Collection(collection),
			logger.Field("index_name", indexName),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to create index: %w", err)
	}

	logger.Info(ctx, "index created",
		logger.Collection(collection),
		logger.Field("index_name", indexName),
		logger.Duration(time.Since(start)),
	)

	return indexName, nil
}

// CreateIndexes는 여러 인덱스를 생성합니다
func (r *VitessRepository) CreateIndexes(ctx context.Context, collection string, models []repository.IndexModel) ([]string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("create_indexes", collection, "success", duration)
	}()

	indexNames := make([]string, 0, len(models))

	for _, model := range models {
		indexName, err := r.CreateIndex(ctx, collection, model)
		if err != nil {
			logger.Error(ctx, "failed to create index in batch",
				logger.Collection(collection),
				zap.Error(err),
			)
			// 에러가 발생해도 계속 진행
			continue
		}
		indexNames = append(indexNames, indexName)
	}

	logger.Info(ctx, "indexes created",
		logger.Collection(collection),
		logger.Field("count", len(indexNames)),
		logger.Duration(time.Since(start)),
	)

	return indexNames, nil
}

// DropIndex는 인덱스를 삭제합니다
func (r *VitessRepository) DropIndex(ctx context.Context, collection, indexName string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("drop_index", collection, "success", duration)
	}()

	query := fmt.Sprintf(`ALTER TABLE documents DROP INDEX %s`, indexName)

	logger.Debug(ctx, "dropping index",
		logger.Collection(collection),
		logger.Field("index_name", indexName),
	)

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		// 존재하지 않는 인덱스인 경우 무시
		if strings.Contains(err.Error(), "Can't DROP") {
			logger.Warn(ctx, "index does not exist",
				logger.Collection(collection),
				logger.Field("index_name", indexName),
			)
			return nil
		}

		r.metrics.RecordDBOperation("drop_index", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to drop index",
			logger.Collection(collection),
			logger.Field("index_name", indexName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to drop index: %w", err)
	}

	logger.Info(ctx, "index dropped",
		logger.Collection(collection),
		logger.Field("index_name", indexName),
		logger.Duration(time.Since(start)),
	)

	return nil
}

// ListIndexes는 컬렉션(테이블)의 인덱스 목록을 반환합니다
func (r *VitessRepository) ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("list_indexes", collection, "success", duration)
	}()

	// MySQL의 SHOW INDEX를 사용하여 인덱스 정보 조회
	query := `SHOW INDEX FROM documents`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.metrics.RecordDBOperation("list_indexes", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to list indexes",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer rows.Close()

	// 인덱스 정보를 저장할 맵
	indexMap := make(map[string]map[string]interface{})

	// 컬럼 이름 가져오기
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		// 동적으로 스캔할 슬라이스 생성
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			logger.Warn(ctx, "failed to scan index row", zap.Error(err))
			continue
		}

		// 결과를 맵으로 변환
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}

		// 인덱스 이름 가져오기
		indexName, ok := rowMap["Key_name"].(string)
		if !ok {
			continue
		}

		// 인덱스별로 그룹화
		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = map[string]interface{}{
				"name":    indexName,
				"unique":  rowMap["Non_unique"] == int64(0),
				"keys":    []string{},
				"type":    rowMap["Index_type"],
				"comment": rowMap["Index_comment"],
			}
		}

		// 컬럼 추가
		if columnName, ok := rowMap["Column_name"].(string); ok {
			keys := indexMap[indexName]["keys"].([]string)
			indexMap[indexName]["keys"] = append(keys, columnName)
		}
	}

	// 맵을 슬라이스로 변환
	var indexes []map[string]interface{}
	for _, indexInfo := range indexMap {
		indexes = append(indexes, indexInfo)
	}

	logger.Info(ctx, "indexes listed",
		logger.Collection(collection),
		logger.Field("count", len(indexes)),
		logger.Duration(time.Since(start)),
	)

	return indexes, nil
}

// IndexExists는 인덱스가 존재하는지 확인합니다
func (r *VitessRepository) IndexExists(ctx context.Context, collection, indexName string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE table_schema = DATABASE()
		AND table_name = 'documents'
		AND index_name = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, indexName).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		logger.Error(ctx, "failed to check index existence",
			logger.Collection(collection),
			logger.Field("index_name", indexName),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}

	return count > 0, nil
}
