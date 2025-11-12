package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// Aggregate는 집계 파이프라인을 실행합니다
// MongoDB의 강력한 집계 프레임워크를 사용하여 복잡한 데이터 처리를 수행합니다
func (r *DocumentRepository) Aggregate(ctx context.Context, collection string, pipeline []bson.M) ([]map[string]interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("aggregate", collection, "success", duration)
		logger.Debug(ctx, "aggregation pipeline executed",
			logger.Collection(collection),
			logger.Duration(duration),
			logger.Count(len(pipeline)),
		)
	}()

	coll := r.database.Collection(collection)

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		r.metrics.RecordDBOperation("aggregate", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to execute aggregation pipeline",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to execute aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	if err := cursor.All(ctx, &results); err != nil {
		logger.Error(ctx, "failed to decode aggregation results",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to decode aggregation results: %w", err)
	}

	logger.Info(ctx, "aggregation completed successfully",
		logger.Collection(collection),
		logger.Count(len(results)),
	)

	return results, nil
}

// Distinct는 고유한 값을 조회합니다
// 지정된 필드의 고유한 값 목록을 반환합니다
func (r *DocumentRepository) Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("distinct", collection, "success", duration)
		logger.Debug(ctx, "distinct values retrieved",
			logger.Collection(collection),
			logger.Field("field", field),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	var bsonFilter bson.M
	if filter != nil {
		bsonFilter = bson.M(filter)
	} else {
		bsonFilter = bson.M{}
	}

	values, err := coll.Distinct(ctx, field, bsonFilter)
	if err != nil {
		r.metrics.RecordDBOperation("distinct", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to get distinct values",
			logger.Collection(collection),
			logger.Field("field", field),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get distinct values: %w", err)
	}

	logger.Info(ctx, "distinct values retrieved successfully",
		logger.Collection(collection),
		logger.Field("field", field),
		logger.Count(len(values)),
	)

	return values, nil
}

// EstimatedDocumentCount는 컬렉션의 추정 문서 개수를 반환합니다
// 메타데이터를 사용하므로 매우 빠르지만 정확하지 않을 수 있습니다
func (r *DocumentRepository) EstimatedDocumentCount(ctx context.Context, collection string) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("estimated_count", collection, "success", duration)
		logger.Debug(ctx, "estimated document count retrieved",
			logger.Collection(collection),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		r.metrics.RecordDBOperation("estimated_count", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to get estimated document count",
			logger.Collection(collection),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to get estimated count: %w", err)
	}

	logger.Info(ctx, "estimated document count retrieved successfully",
		logger.Collection(collection),
		logger.Field("count", count),
	)

	return count, nil
}
