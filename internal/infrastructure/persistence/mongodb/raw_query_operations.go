package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// ExecuteRawQuery는 MongoDB의 raw command를 실행합니다
// query는 bson.M, bson.D 또는 map[string]interface{} 형태여야 합니다
//
// 예제:
//   query := bson.M{
//       "find": "users",
//       "filter": bson.M{"age": bson.M{"$gt": 25}},
//   }
//   result, err := repo.ExecuteRawQuery(ctx, query)
func (r *DocumentRepository) ExecuteRawQuery(ctx context.Context, query interface{}) (interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("raw_query", "database", "success", duration)
	}()

	// query를 BSON으로 변환
	var command interface{}
	switch q := query.(type) {
	case bson.M:
		command = q
	case bson.D:
		command = q
	case map[string]interface{}:
		command = bson.M(q)
	case string:
		// JSON 문자열인 경우 파싱
		var m bson.M
		if err := bson.UnmarshalExtJSON([]byte(q), true, &m); err != nil {
			r.metrics.RecordDBOperation("raw_query", "database", "error", time.Since(start))
			logger.Error(ctx, "failed to parse query string",
				zap.String("query", q),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to parse query string: %w", err)
		}
		command = m
	default:
		r.metrics.RecordDBOperation("raw_query", "database", "error", time.Since(start))
		return nil, fmt.Errorf("unsupported query type: %T (expected bson.M, bson.D, map[string]interface{}, or JSON string)", query)
	}

	logger.Debug(ctx, "executing raw MongoDB command",
		logger.Field("command", command),
	)

	// RunCommand 실행
	var result bson.M
	err := r.database.RunCommand(ctx, command).Decode(&result)
	if err != nil {
		r.metrics.RecordDBOperation("raw_query", "database", "error", time.Since(start))
		logger.Error(ctx, "failed to execute raw command",
			logger.Field("command", command),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to execute raw command: %w", err)
	}

	logger.Info(ctx, "raw MongoDB command executed successfully",
		logger.Duration(time.Since(start)),
	)

	return result, nil
}

// ExecuteRawQueryWithResult는 MongoDB의 raw command를 실행하고 결과를 지정된 변수에 디코드합니다
//
// 예제:
//   var result struct {
//       Cursor struct {
//           FirstBatch []bson.M `bson:"firstBatch"`
//       } `bson:"cursor"`
//       OK int `bson:"ok"`
//   }
//   query := bson.M{
//       "find": "users",
//       "filter": bson.M{"age": bson.M{"$gt": 25}},
//   }
//   err := repo.ExecuteRawQueryWithResult(ctx, query, &result)
func (r *DocumentRepository) ExecuteRawQueryWithResult(ctx context.Context, query interface{}, result interface{}) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("raw_query_with_result", "database", "success", duration)
	}()

	// query를 BSON으로 변환
	var command interface{}
	switch q := query.(type) {
	case bson.M:
		command = q
	case bson.D:
		command = q
	case map[string]interface{}:
		command = bson.M(q)
	case string:
		// JSON 문자열인 경우 파싱
		var m bson.M
		if err := bson.UnmarshalExtJSON([]byte(q), true, &m); err != nil {
			r.metrics.RecordDBOperation("raw_query_with_result", "database", "error", time.Since(start))
			logger.Error(ctx, "failed to parse query string",
				zap.String("query", q),
				zap.Error(err),
			)
			return fmt.Errorf("failed to parse query string: %w", err)
		}
		command = m
	default:
		r.metrics.RecordDBOperation("raw_query_with_result", "database", "error", time.Since(start))
		return fmt.Errorf("unsupported query type: %T (expected bson.M, bson.D, map[string]interface{}, or JSON string)", query)
	}

	logger.Debug(ctx, "executing raw MongoDB command with result",
		logger.Field("command", command),
	)

	// RunCommand 실행 및 결과 디코드
	err := r.database.RunCommand(ctx, command).Decode(result)
	if err != nil {
		r.metrics.RecordDBOperation("raw_query_with_result", "database", "error", time.Since(start))
		logger.Error(ctx, "failed to execute raw command",
			logger.Field("command", command),
			zap.Error(err),
		)
		return fmt.Errorf("failed to execute raw command: %w", err)
	}

	logger.Info(ctx, "raw MongoDB command executed successfully with result",
		logger.Duration(time.Since(start)),
	)

	return nil
}

// RunAggregateCommand는 aggregation pipeline을 raw command로 실행합니다
//
// 예제:
//   pipeline := []bson.M{
//       {"$match": bson.M{"status": "active"}},
//       {"$group": bson.M{"_id": "$category", "count": bson.M{"$sum": 1}}},
//   }
//   result, err := repo.RunAggregateCommand(ctx, "products", pipeline)
func (r *DocumentRepository) RunAggregateCommand(ctx context.Context, collection string, pipeline []bson.M) ([]bson.M, error) {
	start := time.Now()

	command := bson.M{
		"aggregate": collection,
		"pipeline":  pipeline,
		"cursor":    bson.M{},
	}

	var result struct {
		Cursor struct {
			FirstBatch []bson.M `bson:"firstBatch"`
			ID         int64    `bson:"id"`
			NS         string   `bson:"ns"`
		} `bson:"cursor"`
		OK int `bson:"ok"`
	}

	err := r.ExecuteRawQueryWithResult(ctx, command, &result)
	if err != nil {
		r.metrics.RecordDBOperation("aggregate_command", collection, "error", time.Since(start))
		return nil, err
	}

	logger.Info(ctx, "aggregate command executed",
		logger.Collection(collection),
		logger.Field("results", len(result.Cursor.FirstBatch)),
		logger.Duration(time.Since(start)),
	)

	return result.Cursor.FirstBatch, nil
}

// RunMapReduceCommand는 MapReduce를 실행합니다 (legacy, 일반적으로 aggregation 사용 권장)
//
// 예제:
//   mapFunction := "function() { emit(this.category, 1); }"
//   reduceFunction := "function(key, values) { return Array.sum(values); }"
//   result, err := repo.RunMapReduceCommand(ctx, "products", mapFunction, reduceFunction)
func (r *DocumentRepository) RunMapReduceCommand(ctx context.Context, collection, mapFunction, reduceFunction string, options bson.M) (interface{}, error) {
	start := time.Now()

	command := bson.M{
		"mapReduce": collection,
		"map":       mapFunction,
		"reduce":    reduceFunction,
		"out":       bson.M{"inline": 1},
	}

	// 추가 옵션 병합
	for k, v := range options {
		command[k] = v
	}

	result, err := r.ExecuteRawQuery(ctx, command)
	if err != nil {
		r.metrics.RecordDBOperation("mapreduce_command", collection, "error", time.Since(start))
		return nil, err
	}

	logger.Info(ctx, "mapreduce command executed",
		logger.Collection(collection),
		logger.Duration(time.Since(start)),
	)

	return result, nil
}

// GetCollectionStats는 컬렉션 통계를 반환합니다
//
// 예제:
//   stats, err := repo.GetCollectionStats(ctx, "users")
//   fmt.Printf("Document count: %v\n", stats["count"])
//   fmt.Printf("Storage size: %v\n", stats["storageSize"])
func (r *DocumentRepository) GetCollectionStats(ctx context.Context, collection string) (bson.M, error) {
	start := time.Now()

	command := bson.M{
		"collStats": collection,
	}

	result, err := r.ExecuteRawQuery(ctx, command)
	if err != nil {
		r.metrics.RecordDBOperation("collstats_command", collection, "error", time.Since(start))
		return nil, err
	}

	stats, ok := result.(bson.M)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	logger.Info(ctx, "collection stats retrieved",
		logger.Collection(collection),
		logger.Duration(time.Since(start)),
	)

	return stats, nil
}

// GetDatabaseStats는 데이터베이스 통계를 반환합니다
//
// 예제:
//   stats, err := repo.GetDatabaseStats(ctx)
//   fmt.Printf("Total collections: %v\n", stats["collections"])
//   fmt.Printf("Total data size: %v\n", stats["dataSize"])
func (r *DocumentRepository) GetDatabaseStats(ctx context.Context) (bson.M, error) {
	start := time.Now()

	command := bson.M{
		"dbStats": 1,
		"scale":   1024, // KB 단위
	}

	result, err := r.ExecuteRawQuery(ctx, command)
	if err != nil {
		r.metrics.RecordDBOperation("dbstats_command", "database", "error", time.Since(start))
		return nil, err
	}

	stats, ok := result.(bson.M)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	logger.Info(ctx, "database stats retrieved",
		logger.Duration(time.Since(start)),
	)

	return stats, nil
}
