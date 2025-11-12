package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// CreateCollection은 컬렉션을 생성합니다
func (r *DocumentRepository) CreateCollection(ctx context.Context, name string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("create_collection", name, "success", duration)
		logger.Debug(ctx, "collection created",
			logger.Collection(name),
			logger.Duration(duration),
		)
	}()

	err := r.database.CreateCollection(ctx, name)
	if err != nil {
		// 이미 존재하는 경우는 에러로 처리하지 않음
		if mongo.IsDuplicateKeyError(err) {
			logger.Info(ctx, "collection already exists",
				logger.Collection(name),
			)
			return nil
		}

		r.metrics.RecordDBOperation("create_collection", name, "error", time.Since(start))
		logger.Error(ctx, "failed to create collection",
			logger.Collection(name),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create collection: %w", err)
	}

	logger.Info(ctx, "collection created successfully",
		logger.Collection(name),
	)

	return nil
}

// DropCollection은 컬렉션을 삭제합니다
func (r *DocumentRepository) DropCollection(ctx context.Context, name string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("drop_collection", name, "success", duration)
		logger.Debug(ctx, "collection dropped",
			logger.Collection(name),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(name)
	err := coll.Drop(ctx)
	if err != nil {
		r.metrics.RecordDBOperation("drop_collection", name, "error", time.Since(start))
		logger.Error(ctx, "failed to drop collection",
			logger.Collection(name),
			zap.Error(err),
		)
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	logger.Info(ctx, "collection dropped successfully",
		logger.Collection(name),
	)

	return nil
}

// RenameCollection은 컬렉션 이름을 변경합니다
func (r *DocumentRepository) RenameCollection(ctx context.Context, oldName, newName string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("rename_collection", oldName, "success", duration)
		logger.Debug(ctx, "collection renamed",
			logger.Field("old_name", oldName),
			logger.Field("new_name", newName),
			logger.Duration(duration),
		)
	}()

	// MongoDB의 renameCollection 명령 실행
	command := bson.D{
		{Key: "renameCollection", Value: r.database.Name() + "." + oldName},
		{Key: "to", Value: r.database.Name() + "." + newName},
	}

	err := r.database.RunCommand(ctx, command).Err()
	if err != nil {
		r.metrics.RecordDBOperation("rename_collection", oldName, "error", time.Since(start))
		logger.Error(ctx, "failed to rename collection",
			logger.Field("old_name", oldName),
			logger.Field("new_name", newName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to rename collection: %w", err)
	}

	logger.Info(ctx, "collection renamed successfully",
		logger.Field("old_name", oldName),
		logger.Field("new_name", newName),
	)

	return nil
}

// ListCollections는 데이터베이스의 컬렉션 목록을 반환합니다
func (r *DocumentRepository) ListCollections(ctx context.Context) ([]string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("list_collections", "database", "success", duration)
		logger.Debug(ctx, "collections listed",
			logger.Duration(duration),
		)
	}()

	collections, err := r.database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		r.metrics.RecordDBOperation("list_collections", "database", "error", time.Since(start))
		logger.Error(ctx, "failed to list collections",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	logger.Info(ctx, "collections listed successfully",
		logger.Count(len(collections)),
	)

	return collections, nil
}

// CollectionExists는 컬렉션이 존재하는지 확인합니다
func (r *DocumentRepository) CollectionExists(ctx context.Context, name string) (bool, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		logger.Debug(ctx, "collection existence checked",
			logger.Collection(name),
			logger.Duration(duration),
		)
	}()

	collections, err := r.database.ListCollectionNames(ctx, bson.M{"name": name})
	if err != nil {
		logger.Error(ctx, "failed to check collection existence",
			logger.Collection(name),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}

	exists := len(collections) > 0
	logger.Info(ctx, "collection existence checked",
		logger.Collection(name),
		logger.Field("exists", exists),
	)

	return exists, nil
}
