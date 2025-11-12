package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Watch는 컬렉션의 변경 사항을 실시간으로 감지합니다
// MongoDB Change Streams를 사용하여 실시간 데이터 변경을 모니터링합니다
// 주의: Change Streams는 MongoDB replica set 또는 sharded cluster에서만 작동합니다
//
// 사용 예시:
//
//	pipeline := []bson.M{
//		{"$match": bson.M{"operationType": bson.M{"$in": []string{"insert", "update", "delete"}}}},
//	}
//	stream, err := repo.Watch(ctx, "users", pipeline)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer stream.Close(ctx)
//
//	for stream.Next(ctx) {
//		var changeDoc bson.M
//		if err := stream.Decode(&changeDoc); err != nil {
//			log.Fatal(err)
//		}
//		fmt.Printf("Change detected: %v\n", changeDoc)
//	}
func (r *DocumentRepository) Watch(ctx context.Context, collection string, pipeline []bson.M) (*mongo.ChangeStream, error) {
	start := time.Now()

	logger.Debug(ctx, "starting change stream watch",
		logger.Collection(collection),
	)

	coll := r.database.Collection(collection)

	// Change Stream 옵션 설정
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	// pipeline이 없으면 빈 파이프라인 사용
	if pipeline == nil {
		pipeline = []bson.M{}
	}

	// Change Stream 생성
	stream, err := coll.Watch(ctx, pipeline, opts)
	if err != nil {
		r.metrics.RecordDBOperation("watch", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to create change stream",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create change stream: %w", err)
	}

	duration := time.Since(start)
	r.metrics.RecordDBOperation("watch", collection, "success", duration)
	logger.Info(ctx, "change stream created successfully",
		logger.Collection(collection),
		logger.Duration(duration),
	)

	return stream, nil
}

// WatchWithResumeToken은 resume token을 사용하여 Change Stream을 시작합니다
// 이전에 중단된 위치부터 다시 시작할 수 있습니다
func (r *DocumentRepository) WatchWithResumeToken(ctx context.Context, collection string, pipeline []bson.M, resumeToken bson.Raw) (*mongo.ChangeStream, error) {
	start := time.Now()

	logger.Debug(ctx, "starting change stream with resume token",
		logger.Collection(collection),
	)

	coll := r.database.Collection(collection)

	// Change Stream 옵션 설정
	opts := options.ChangeStream().
		SetFullDocument(options.UpdateLookup).
		SetResumeAfter(resumeToken)

	// pipeline이 없으면 빈 파이프라인 사용
	if pipeline == nil {
		pipeline = []bson.M{}
	}

	// Change Stream 생성
	stream, err := coll.Watch(ctx, pipeline, opts)
	if err != nil {
		r.metrics.RecordDBOperation("watch_resume", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to create change stream with resume token",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create change stream with resume token: %w", err)
	}

	duration := time.Since(start)
	r.metrics.RecordDBOperation("watch_resume", collection, "success", duration)
	logger.Info(ctx, "change stream created successfully with resume token",
		logger.Collection(collection),
		logger.Duration(duration),
	)

	return stream, nil
}
