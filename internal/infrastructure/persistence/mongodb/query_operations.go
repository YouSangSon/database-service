package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// FindWithOptions는 옵션을 사용하여 문서를 조회합니다 (Sort, Limit, Skip, Projection)
// 이 메서드는 복잡한 쿼리와 페이지네이션을 지원합니다
func (r *DocumentRepository) FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *repository.FindOptions) ([]*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_with_options", collection, "success", duration)
		logger.Debug(ctx, "documents found with options",
			logger.Collection(collection),
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

	// 옵션 설정
	findOpts := options.Find()

	if opts != nil {
		// Sort 옵션
		if len(opts.Sort) > 0 {
			sort := bson.D{}
			for k, v := range opts.Sort {
				sort = append(sort, bson.E{Key: k, Value: v})
			}
			findOpts.SetSort(sort)
		}

		// Limit 옵션
		if opts.Limit > 0 {
			findOpts.SetLimit(opts.Limit)
		}

		// Skip 옵션
		if opts.Skip > 0 {
			findOpts.SetSkip(opts.Skip)
		}

		// Projection 옵션
		if len(opts.Projection) > 0 {
			projection := bson.M{}
			for k, v := range opts.Projection {
				projection[k] = v
			}
			findOpts.SetProjection(projection)
		}
	}

	cursor, err := coll.Find(ctx, bsonFilter, findOpts)
	if err != nil {
		r.metrics.RecordDBOperation("find_with_options", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find documents with options",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*entity.Document
	for cursor.Next(ctx) {
		var model documentModel
		if err := cursor.Decode(&model); err != nil {
			logger.Warn(ctx, "failed to decode document", zap.Error(err))
			continue
		}

		doc := entity.ReconstructDocument(
			model.ID.Hex(),
			model.Collection,
			model.Data,
			model.Version,
			model.CreatedAt,
			model.UpdatedAt,
		)
		documents = append(documents, doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	logger.Info(ctx, "documents found with options successfully",
		logger.Collection(collection),
		logger.Count(len(documents)),
	)

	return documents, nil
}

// Upsert는 문서가 없으면 생성하고 있으면 업데이트합니다
// 원자적 연산으로 동시성 환경에서 안전합니다
func (r *DocumentRepository) Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("upsert", collection, "success", duration)
		logger.Debug(ctx, "upsert operation completed",
			logger.Collection(collection),
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

	// 업데이트 문서 생성
	updateDoc := bson.M{
		"$set": update,
		"$inc": bson.M{"version": 1},
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
		"$currentDate": bson.M{
			"updated_at": true,
		},
	}

	opts := options.Update().SetUpsert(true)

	result, err := coll.UpdateOne(ctx, bsonFilter, updateDoc, opts)
	if err != nil {
		r.metrics.RecordDBOperation("upsert", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to upsert document",
			logger.Collection(collection),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to upsert document: %w", err)
	}

	var docID string
	if result.UpsertedID != nil {
		// 새로 생성된 경우
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			docID = oid.Hex()
		}
		logger.Info(ctx, "document created via upsert",
			logger.Collection(collection),
			logger.DocumentID(docID),
		)
	} else {
		// 업데이트된 경우 - ID를 알 수 없으므로 빈 문자열 반환
		logger.Info(ctx, "document updated via upsert",
			logger.Collection(collection),
			logger.Field("matched_count", result.MatchedCount),
			logger.Field("modified_count", result.ModifiedCount),
		)
	}

	return docID, nil
}

// Replace는 문서를 교체합니다 (이전 문서를 반환하지 않음)
// FindOneAndReplace와 달리 이전 문서를 읽지 않아 더 빠릅니다
func (r *DocumentRepository) Replace(ctx context.Context, collection, id string, replacement *entity.Document) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("replace", collection, "success", duration)
		logger.Debug(ctx, "document replaced",
			logger.Collection(collection),
			logger.DocumentID(id),
			logger.Duration(duration),
		)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	// 교체할 문서 생성
	replacementDoc := &documentModel{
		Collection: replacement.Collection(),
		Data:       replacement.Data(),
		Version:    replacement.Version() + 1,
		CreatedAt:  replacement.CreatedAt(),
		UpdatedAt:  time.Now(),
	}

	result, err := coll.ReplaceOne(ctx, filter, replacementDoc)
	if err != nil {
		r.metrics.RecordDBOperation("replace", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to replace document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return fmt.Errorf("failed to replace document: %w", err)
	}

	if result.MatchedCount == 0 {
		r.metrics.RecordDBOperation("replace", collection, "not_found", time.Since(start))
		return entity.ErrDocumentNotFound
	}

	logger.Info(ctx, "document replaced successfully",
		logger.Collection(collection),
		logger.DocumentID(id),
	)

	return nil
}
