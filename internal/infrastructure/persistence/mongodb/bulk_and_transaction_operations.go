package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// ===== 벌크 작업 (Bulk Operations) =====

// SaveMany는 여러 문서를 한 번에 저장합니다 (Bulk Insert)
func (r *DocumentRepository) SaveMany(ctx context.Context, docs []*entity.Document) error {
	if len(docs) == 0 {
		return nil
	}

	start := time.Now()
	collection := docs[0].Collection()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("save_many", collection, "success", duration)
		logger.Debug(ctx, "documents saved in bulk",
			logger.Collection(collection),
			logger.Count(len(docs)),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	// 문서 모델로 변환
	models := make([]interface{}, len(docs))
	for i, doc := range docs {
		models[i] = &documentModel{
			Collection: doc.Collection(),
			Data:       doc.Data(),
			Version:    doc.Version(),
			CreatedAt:  doc.CreatedAt(),
			UpdatedAt:  doc.UpdatedAt(),
		}
	}

	// Bulk Insert with ordered=false for better performance
	opts := options.InsertMany().SetOrdered(false)
	result, err := coll.InsertMany(ctx, models, opts)
	if err != nil {
		r.metrics.RecordDBOperation("save_many", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to save documents in bulk",
			logger.Collection(collection),
			logger.Count(len(docs)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save documents: %w", err)
	}

	// ID 설정
	for i, id := range result.InsertedIDs {
		if oid, ok := id.(primitive.ObjectID); ok {
			docs[i].SetID(oid.Hex())
		}
	}

	return nil
}

// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
func (r *DocumentRepository) UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("update_many", collection, "success", duration)
		logger.Debug(ctx, "multiple documents updated",
			logger.Collection(collection),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	// 필터 변환
	bsonFilter := bson.M(filter)
	if len(bsonFilter) == 0 {
		return 0, fmt.Errorf("filter cannot be empty for update_many operation")
	}

	// 업데이트 문서 생성
	updateDoc := bson.M{
		"$set": update,
		"$inc": bson.M{"version": 1},
		"$currentDate": bson.M{
			"updated_at": true,
		},
	}

	result, err := coll.UpdateMany(ctx, bsonFilter, updateDoc)
	if err != nil {
		r.metrics.RecordDBOperation("update_many", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to update multiple documents",
			logger.Collection(collection),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to update documents: %w", err)
	}

	logger.Info(ctx, "multiple documents updated successfully",
		logger.Collection(collection),
		logger.Count(int(result.ModifiedCount)),
	)

	return result.ModifiedCount, nil
}

// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
func (r *DocumentRepository) DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("delete_many", collection, "success", duration)
		logger.Debug(ctx, "multiple documents deleted",
			logger.Collection(collection),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	// 필터 변환
	bsonFilter := bson.M(filter)
	if len(bsonFilter) == 0 {
		return 0, fmt.Errorf("filter cannot be empty for delete_many operation")
	}

	result, err := coll.DeleteMany(ctx, bsonFilter)
	if err != nil {
		r.metrics.RecordDBOperation("delete_many", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to delete multiple documents",
			logger.Collection(collection),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}

	logger.Info(ctx, "multiple documents deleted successfully",
		logger.Collection(collection),
		logger.Count(int(result.DeletedCount)),
	)

	return result.DeletedCount, nil
}

// ===== 트랜잭션 (Transaction) =====

// WithTransaction은 트랜잭션 내에서 함수를 실행합니다
// MongoDB 트랜잭션은 replica set 또는 sharded cluster에서만 작동합니다
func (r *DocumentRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	start := time.Now()

	session, err := r.client.StartSession()
	if err != nil {
		logger.Error(ctx, "failed to start session",
			zap.Error(err),
		)
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	logger.Debug(ctx, "starting transaction")

	// 트랜잭션 실행
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	if err != nil {
		logger.Error(ctx, "transaction failed",
			logger.Duration(time.Since(start)),
			zap.Error(err),
		)
		return fmt.Errorf("transaction failed: %w", err)
	}

	logger.Info(ctx, "transaction completed successfully",
		logger.Duration(time.Since(start)),
	)

	return nil
}
