package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// BulkWrite는 여러 작업을 한 번에 실행합니다
// 다양한 작업(insert, update, delete, replace)을 하나의 요청으로 처리하여 성능을 향상시킵니다
func (r *DocumentRepository) BulkWrite(ctx context.Context, operations []*repository.BulkOperation) (*repository.BulkResult, error) {
	if len(operations) == 0 {
		return &repository.BulkResult{}, nil
	}

	start := time.Now()

	// 컬렉션별로 작업을 그룹화
	collectionOps := make(map[string][]mongo.WriteModel)
	for _, op := range operations {
		model, err := r.convertToBulkWriteModel(op)
		if err != nil {
			logger.Error(ctx, "failed to convert bulk operation",
				logger.Collection(op.Collection),
				zap.Error(err),
			)
			continue
		}
		collectionOps[op.Collection] = append(collectionOps[op.Collection], model)
	}

	// 각 컬렉션별로 벌크 작업 실행
	result := &repository.BulkResult{
		UpsertedIDs: make(map[int]interface{}),
	}

	for collName, models := range collectionOps {
		coll := r.database.Collection(collName)

		// 순서대로 실행하지 않음 (ordered=false) - 더 나은 성능
		opts := mongo.NewBulkWriteOptions().SetOrdered(false)

		bulkResult, err := coll.BulkWrite(ctx, models, opts)
		if err != nil {
			r.metrics.RecordDBOperation("bulk_write", collName, "error", time.Since(start))
			logger.Error(ctx, "bulk write operation failed",
				logger.Collection(collName),
				zap.Error(err),
			)
			// 부분적으로 성공한 경우도 있으므로 계속 진행
			if bulkResult != nil {
				result.InsertedCount += bulkResult.InsertedCount
				result.MatchedCount += bulkResult.MatchedCount
				result.ModifiedCount += bulkResult.ModifiedCount
				result.DeletedCount += bulkResult.DeletedCount
				result.UpsertedCount += bulkResult.UpsertedCount
			}
			continue
		}

		// 결과 집계
		result.InsertedCount += bulkResult.InsertedCount
		result.MatchedCount += bulkResult.MatchedCount
		result.ModifiedCount += bulkResult.ModifiedCount
		result.DeletedCount += bulkResult.DeletedCount
		result.UpsertedCount += bulkResult.UpsertedCount

		// Upserted IDs 추가
		for idx, id := range bulkResult.UpsertedIDs {
			result.UpsertedIDs[int(idx)] = id
		}

		logger.Info(ctx, "bulk write completed for collection",
			logger.Collection(collName),
			logger.Field("inserted", bulkResult.InsertedCount),
			logger.Field("matched", bulkResult.MatchedCount),
			logger.Field("modified", bulkResult.ModifiedCount),
			logger.Field("deleted", bulkResult.DeletedCount),
			logger.Field("upserted", bulkResult.UpsertedCount),
		)
	}

	duration := time.Since(start)
	r.metrics.RecordDBOperation("bulk_write", "multiple", "success", duration)
	logger.Info(ctx, "bulk write operation completed",
		logger.Duration(duration),
		logger.Field("total_operations", len(operations)),
		logger.Field("total_inserted", result.InsertedCount),
		logger.Field("total_modified", result.ModifiedCount),
		logger.Field("total_deleted", result.DeletedCount),
	)

	return result, nil
}

// convertToBulkWriteModel은 BulkOperation을 mongo.WriteModel로 변환합니다
func (r *DocumentRepository) convertToBulkWriteModel(op *repository.BulkOperation) (mongo.WriteModel, error) {
	switch op.Type {
	case "insert":
		if op.Document == nil {
			return nil, fmt.Errorf("document is required for insert operation")
		}
		model := &documentModel{
			Collection: op.Document.Collection(),
			Data:       op.Document.Data(),
			Version:    op.Document.Version(),
			CreatedAt:  op.Document.CreatedAt(),
			UpdatedAt:  op.Document.UpdatedAt(),
		}
		return mongo.NewInsertOneModel().SetDocument(model), nil

	case "update":
		if len(op.Filter) == 0 {
			return nil, fmt.Errorf("filter is required for update operation")
		}
		if len(op.Update) == 0 {
			return nil, fmt.Errorf("update is required for update operation")
		}

		filter := bson.M(op.Filter)
		update := bson.M{
			"$set": op.Update,
			"$inc": bson.M{"version": 1},
			"$currentDate": bson.M{
				"updated_at": true,
			},
		}

		if op.UpdateMany {
			model := mongo.NewUpdateManyModel().
				SetFilter(filter).
				SetUpdate(update).
				SetUpsert(op.Upsert)
			return model, nil
		}

		model := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(op.Upsert)
		return model, nil

	case "delete":
		if len(op.Filter) == 0 {
			return nil, fmt.Errorf("filter is required for delete operation")
		}

		filter := bson.M(op.Filter)

		if op.DeleteMany {
			return mongo.NewDeleteManyModel().SetFilter(filter), nil
		}
		return mongo.NewDeleteOneModel().SetFilter(filter), nil

	case "replace":
		if op.ReplaceOneID == "" {
			return nil, fmt.Errorf("id is required for replace operation")
		}
		if op.Document == nil {
			return nil, fmt.Errorf("document is required for replace operation")
		}

		objectID, err := primitive.ObjectIDFromHex(op.ReplaceOneID)
		if err != nil {
			return nil, fmt.Errorf("invalid id format: %w", err)
		}

		filter := bson.M{"_id": objectID}
		replacement := &documentModel{
			Collection: op.Document.Collection(),
			Data:       op.Document.Data(),
			Version:    op.Document.Version() + 1,
			CreatedAt:  op.Document.CreatedAt(),
			UpdatedAt:  time.Now(),
		}

		model := mongo.NewReplaceOneModel().
			SetFilter(filter).
			SetReplacement(replacement).
			SetUpsert(op.Upsert)
		return model, nil

	default:
		return nil, fmt.Errorf("unknown operation type: %s", op.Type)
	}
}
