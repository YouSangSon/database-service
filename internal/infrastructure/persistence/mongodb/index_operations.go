package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// CreateIndex는 단일 인덱스를 생성합니다
func (r *DocumentRepository) CreateIndex(ctx context.Context, collection string, model repository.IndexModel) (string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("create_index", collection, "success", duration)
		logger.Debug(ctx, "index created",
			logger.Collection(collection),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	// 인덱스 키 변환
	keys := bson.D{}
	for k, v := range model.Keys {
		keys = append(keys, bson.E{Key: k, Value: v})
	}

	// 인덱스 옵션 설정
	indexModel := mongo.IndexModel{
		Keys: keys,
	}

	if model.Options != nil {
		opts := options.Index()

		if model.Options.Unique != nil {
			opts.SetUnique(*model.Options.Unique)
		}
		if model.Options.Name != "" {
			opts.SetName(model.Options.Name)
		}
		if model.Options.Background != nil {
			opts.SetBackground(*model.Options.Background)
		}
		if model.Options.Sparse != nil {
			opts.SetSparse(*model.Options.Sparse)
		}
		if model.Options.ExpireAfter != nil {
			opts.SetExpireAfterSeconds(*model.Options.ExpireAfter)
		}
		if model.Options.PartialFilter != nil {
			opts.SetPartialFilterExpression(model.Options.PartialFilter)
		}

		indexModel.Options = opts
	}

	indexName, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		r.metrics.RecordDBOperation("create_index", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to create index",
			logger.Collection(collection),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to create index: %w", err)
	}

	logger.Info(ctx, "index created successfully",
		logger.Collection(collection),
		logger.Field("index_name", indexName),
	)

	return indexName, nil
}

// CreateIndexes는 여러 인덱스를 생성합니다
func (r *DocumentRepository) CreateIndexes(ctx context.Context, collection string, models []repository.IndexModel) ([]string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("create_indexes", collection, "success", duration)
		logger.Debug(ctx, "multiple indexes created",
			logger.Collection(collection),
			logger.Count(len(models)),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	// 인덱스 모델 변환
	indexModels := make([]mongo.IndexModel, len(models))
	for i, model := range models {
		keys := bson.D{}
		for k, v := range model.Keys {
			keys = append(keys, bson.E{Key: k, Value: v})
		}

		indexModel := mongo.IndexModel{
			Keys: keys,
		}

		if model.Options != nil {
			opts := options.Index()

			if model.Options.Unique != nil {
				opts.SetUnique(*model.Options.Unique)
			}
			if model.Options.Name != "" {
				opts.SetName(model.Options.Name)
			}
			if model.Options.Background != nil {
				opts.SetBackground(*model.Options.Background)
			}
			if model.Options.Sparse != nil {
				opts.SetSparse(*model.Options.Sparse)
			}
			if model.Options.ExpireAfter != nil {
				opts.SetExpireAfterSeconds(*model.Options.ExpireAfter)
			}
			if model.Options.PartialFilter != nil {
				opts.SetPartialFilterExpression(model.Options.PartialFilter)
			}

			indexModel.Options = opts
		}

		indexModels[i] = indexModel
	}

	indexNames, err := coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		r.metrics.RecordDBOperation("create_indexes", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to create indexes",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	logger.Info(ctx, "indexes created successfully",
		logger.Collection(collection),
		logger.Count(len(indexNames)),
	)

	return indexNames, nil
}

// DropIndex는 인덱스를 삭제합니다
func (r *DocumentRepository) DropIndex(ctx context.Context, collection, indexName string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("drop_index", collection, "success", duration)
		logger.Debug(ctx, "index dropped",
			logger.Collection(collection),
			logger.Field("index_name", indexName),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	_, err := coll.Indexes().DropOne(ctx, indexName)
	if err != nil {
		r.metrics.RecordDBOperation("drop_index", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to drop index",
			logger.Collection(collection),
			logger.Field("index_name", indexName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to drop index: %w", err)
	}

	logger.Info(ctx, "index dropped successfully",
		logger.Collection(collection),
		logger.Field("index_name", indexName),
	)

	return nil
}

// ListIndexes는 컬렉션의 인덱스 목록을 반환합니다
func (r *DocumentRepository) ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("list_indexes", collection, "success", duration)
		logger.Debug(ctx, "indexes listed",
			logger.Collection(collection),
			logger.Duration(duration),
		)
	}()

	coll := r.database.Collection(collection)

	cursor, err := coll.Indexes().List(ctx)
	if err != nil {
		r.metrics.RecordDBOperation("list_indexes", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to list indexes",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var indexes []map[string]interface{}
	if err := cursor.All(ctx, &indexes); err != nil {
		logger.Error(ctx, "failed to decode indexes",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to decode indexes: %w", err)
	}

	logger.Info(ctx, "indexes listed successfully",
		logger.Collection(collection),
		logger.Count(len(indexes)),
	)

	return indexes, nil
}
