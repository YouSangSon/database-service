package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다
// 이 메서드는 비관적 잠금(Pessimistic Lock)처럼 동작합니다
func (r *DocumentRepository) FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_and_update", collection, "success", duration)
		logger.Debug(ctx, "document found and updated",
			logger.Collection(collection),
			logger.DocumentID(id),
			logger.Duration(duration),
		)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	// 업데이트 문서 생성
	updateDoc := bson.M{
		"$set": update,
		"$inc": bson.M{"version": 1},
		"$currentDate": bson.M{
			"updated_at": true,
		},
	}

	// 업데이트 후의 문서를 반환
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var model documentModel
	err = coll.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			r.metrics.RecordDBOperation("find_and_update", collection, "not_found", time.Since(start))
			return nil, entity.ErrDocumentNotFound
		}
		r.metrics.RecordDBOperation("find_and_update", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find and update document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find and update document: %w", err)
	}

	doc := entity.ReconstructDocument(
		model.ID.Hex(),
		model.Collection,
		model.Data,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)

	logger.Info(ctx, "document found and updated successfully",
		logger.Collection(collection),
		logger.DocumentID(id),
		logger.Version(doc.Version()),
	)

	return doc, nil
}

// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
func (r *DocumentRepository) FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_and_replace", collection, "success", duration)
		logger.Debug(ctx, "document found and replaced",
			logger.Collection(collection),
			logger.DocumentID(id),
			logger.Duration(duration),
		)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
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

	// 교체 후의 문서를 반환
	opts := options.FindOneAndReplace().SetReturnDocument(options.After)

	var model documentModel
	err = coll.FindOneAndReplace(ctx, filter, replacementDoc, opts).Decode(&model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			r.metrics.RecordDBOperation("find_and_replace", collection, "not_found", time.Since(start))
			return nil, entity.ErrDocumentNotFound
		}
		r.metrics.RecordDBOperation("find_and_replace", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find and replace document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find and replace document: %w", err)
	}

	doc := entity.ReconstructDocument(
		model.ID.Hex(),
		model.Collection,
		model.Data,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)

	logger.Info(ctx, "document found and replaced successfully",
		logger.Collection(collection),
		logger.DocumentID(id),
		logger.Version(doc.Version()),
	)

	return doc, nil
}

// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
// 원자적 연산으로 동시성 환경에서 안전합니다
func (r *DocumentRepository) FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_and_delete", collection, "success", duration)
		logger.Debug(ctx, "document found and deleted",
			logger.Collection(collection),
			logger.DocumentID(id),
			logger.Duration(duration),
		)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	var model documentModel
	err = coll.FindOneAndDelete(ctx, filter).Decode(&model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			r.metrics.RecordDBOperation("find_and_delete", collection, "not_found", time.Since(start))
			return nil, entity.ErrDocumentNotFound
		}
		r.metrics.RecordDBOperation("find_and_delete", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find and delete document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find and delete document: %w", err)
	}

	doc := entity.ReconstructDocument(
		model.ID.Hex(),
		model.Collection,
		model.Data,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)

	logger.Info(ctx, "document found and deleted successfully",
		logger.Collection(collection),
		logger.DocumentID(id),
	)

	return doc, nil
}
