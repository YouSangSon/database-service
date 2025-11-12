package vitess

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// Aggregate는 집계 파이프라인을 실행합니다
// MongoDB의 aggregation pipeline을 SQL로 변환합니다
func (r *VitessRepository) Aggregate(ctx context.Context, collection string, pipeline []bson.M) ([]map[string]interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("aggregate", collection, "success", duration)
	}()

	// 기본 쿼리
	query := "SELECT data FROM documents WHERE collection = ?"
	args := []interface{}{collection}

	// 파이프라인 분석 및 SQL로 변환
	var whereClauses []string
	var groupBy string
	var having string
	var orderBy string
	var limit string
	var skip string

	for _, stage := range pipeline {
		for key, value := range stage {
			switch key {
			case "$match":
				// $match를 WHERE 절로 변환
				if matchConditions, ok := value.(bson.M); ok {
					for field, condition := range matchConditions {
						whereClauses = append(whereClauses, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = ?", field))
						args = append(args, condition)
					}
				}

			case "$sort":
				// $sort를 ORDER BY로 변환
				if sortFields, ok := value.(bson.M); ok {
					var sortClauses []string
					for field, order := range sortFields {
						direction := "ASC"
						if order == -1 {
							direction = "DESC"
						}
						sortClauses = append(sortClauses, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') %s", field, direction))
					}
					orderBy = " ORDER BY " + strings.Join(sortClauses, ", ")
				}

			case "$limit":
				// $limit를 LIMIT로 변환
				if limitVal, ok := value.(int); ok {
					limit = fmt.Sprintf(" LIMIT %d", limitVal)
				} else if limitVal, ok := value.(int64); ok {
					limit = fmt.Sprintf(" LIMIT %d", limitVal)
				}

			case "$skip":
				// $skip를 OFFSET으로 변환
				if skipVal, ok := value.(int); ok {
					skip = fmt.Sprintf(" OFFSET %d", skipVal)
				} else if skipVal, ok := value.(int64); ok {
					skip = fmt.Sprintf(" OFFSET %d", skipVal)
				}

			case "$group":
				// $group을 GROUP BY로 변환 (간단한 케이스만 지원)
				groupBy = " GROUP BY id"

			case "$project":
				// $project는 애플리케이션 레벨에서 처리
				// (SQL에서는 결과 후처리로 구현)
			}
		}
	}

	// WHERE 절 추가
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}

	// GROUP BY, ORDER BY, LIMIT, OFFSET 추가
	query += groupBy + orderBy + limit + skip

	logger.Debug(ctx, "executing aggregate",
		logger.Collection(collection),
		logger.Field("query", query),
		logger.Field("args", args),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.metrics.RecordDBOperation("aggregate", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to execute aggregate",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to execute aggregate: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var dataJSON []byte
		if err := rows.Scan(&dataJSON); err != nil {
			logger.Warn(ctx, "failed to scan row", zap.Error(err))
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			logger.Warn(ctx, "failed to unmarshal data", zap.Error(err))
			continue
		}

		results = append(results, data)
	}

	logger.Info(ctx, "aggregate executed",
		logger.Collection(collection),
		logger.Field("results", len(results)),
		logger.Duration(time.Since(start)),
	)

	return results, nil
}

// Distinct는 고유한 값을 조회합니다
func (r *VitessRepository) Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("distinct", collection, "success", duration)
	}()

	// JSON_EXTRACT를 사용하여 특정 필드의 고유 값 조회
	query := fmt.Sprintf(`
		SELECT DISTINCT JSON_EXTRACT(data, '$.%s') as value
		FROM documents
		WHERE collection = ?
	`, field)

	args := []interface{}{collection}

	// 필터 조건 추가
	if len(filter) > 0 {
		conditions := []string{}
		for key, value := range filter {
			conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = ?", key))
			args = append(args, value)
		}
		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
		}
	}

	logger.Debug(ctx, "executing distinct",
		logger.Collection(collection),
		logger.Field("field", field),
		logger.Field("query", query),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.metrics.RecordDBOperation("distinct", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to execute distinct",
			logger.Collection(collection),
			logger.Field("field", field),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to execute distinct: %w", err)
	}
	defer rows.Close()

	var values []interface{}
	for rows.Next() {
		var value interface{}
		if err := rows.Scan(&value); err != nil {
			logger.Warn(ctx, "failed to scan row", zap.Error(err))
			continue
		}

		// NULL이 아닌 값만 추가
		if value != nil {
			// JSON 문자열로 반환되는 경우 파싱
			if strValue, ok := value.([]byte); ok {
				var parsed interface{}
				if err := json.Unmarshal(strValue, &parsed); err == nil {
					values = append(values, parsed)
				} else {
					values = append(values, string(strValue))
				}
			} else if strValue, ok := value.(string); ok {
				var parsed interface{}
				if err := json.Unmarshal([]byte(strValue), &parsed); err == nil {
					values = append(values, parsed)
				} else {
					values = append(values, strValue)
				}
			} else {
				values = append(values, value)
			}
		}
	}

	logger.Info(ctx, "distinct executed",
		logger.Collection(collection),
		logger.Field("field", field),
		logger.Field("count", len(values)),
		logger.Duration(time.Since(start)),
	)

	return values, nil
}

// EstimatedDocumentCount는 컬렉션의 추정 문서 개수를 반환합니다
func (r *VitessRepository) EstimatedDocumentCount(ctx context.Context, collection string) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("estimated_count", collection, "success", duration)
	}()

	// MySQL의 EXPLAIN을 사용하여 추정 행 수를 가져옵니다
	// 또는 information_schema를 사용할 수 있습니다
	// 여기서는 빠른 COUNT(*)를 사용합니다 (Vitess는 최적화되어 있음)
	query := `
		SELECT COUNT(*)
		FROM documents
		WHERE collection = ?
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, collection).Scan(&count)
	if err != nil {
		r.metrics.RecordDBOperation("estimated_count", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to get estimated document count",
			logger.Collection(collection),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to get estimated document count: %w", err)
	}

	logger.Info(ctx, "estimated document count retrieved",
		logger.Collection(collection),
		logger.Field("count", count),
		logger.Duration(time.Since(start)),
	)

	return count, nil
}

// CountWithFilter는 필터와 일치하는 문서 개수를 반환합니다 (정확한 개수)
// 이미 Count 메서드가 있지만, 명확성을 위해 별도 구현
func (r *VitessRepository) CountWithFilter(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("count_with_filter", collection, "success", duration)
	}()

	query := "SELECT COUNT(*) FROM documents WHERE collection = ?"
	args := []interface{}{collection}

	// 필터 조건 추가
	if len(filter) > 0 {
		conditions := []string{}
		for key, value := range filter {
			conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = ?", key))
			args = append(args, value)
		}
		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
		}
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.metrics.RecordDBOperation("count_with_filter", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to count documents",
			logger.Collection(collection),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	logger.Info(ctx, "documents counted with filter",
		logger.Collection(collection),
		logger.Field("count", count),
		logger.Duration(time.Since(start)),
	)

	return count, nil
}
