package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/application/dto"
	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/retry"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// CreateIndex creates an index on a collection
func (uc *DocumentUseCase) CreateIndex(ctx context.Context, req *dto.CreateIndexRequest) (*dto.CreateIndexResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.CreateIndex")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "creating index",
		zap.String("collection", req.Collection),
	)

	indexModel := repository.IndexModel{
		Keys:    req.Keys,
		Options: req.Options,
	}

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (string, error) {
			return uc.docRepo.CreateIndex(ctx, req.Collection, indexModel)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to create index", zap.Error(err))
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	indexName := result.(string)

	logger.Info(ctx, "index created successfully",
		zap.String("collection", req.Collection),
		zap.String("index_name", indexName),
	)

	return &dto.CreateIndexResponse{
		IndexName: indexName,
	}, nil
}

// CreateIndexes creates multiple indexes on a collection
func (uc *DocumentUseCase) CreateIndexes(ctx context.Context, req *dto.CreateIndexesRequest) (*dto.CreateIndexesResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.CreateIndexes")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.Int("index_count", len(req.Indexes)),
	)

	logger.Info(ctx, "creating indexes",
		zap.String("collection", req.Collection),
		zap.Int("count", len(req.Indexes)),
	)

	indexNames := make([]string, 0, len(req.Indexes))
	for _, indexDef := range req.Indexes {
		keys, ok := indexDef["keys"].(map[string]int)
		if !ok {
			// Try to convert from map[string]interface{}
			keysInterface, ok := indexDef["keys"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid keys format in index definition")
			}
			keys = make(map[string]int)
			for k, v := range keysInterface {
				if intVal, ok := v.(float64); ok {
					keys[k] = int(intVal)
				} else if intVal, ok := v.(int); ok {
					keys[k] = intVal
				}
			}
		}

		options, _ := indexDef["options"].(map[string]interface{})

		indexModel := repository.IndexModel{
			Keys:    keys,
			Options: options,
		}

		indexName, err := uc.docRepo.CreateIndex(ctx, req.Collection, indexModel)
		if err != nil {
			logger.Warn(ctx, "failed to create index", zap.Error(err))
			continue
		}

		indexNames = append(indexNames, indexName)
	}

	logger.Info(ctx, "indexes created successfully",
		zap.String("collection", req.Collection),
		zap.Int("count", len(indexNames)),
	)

	return &dto.CreateIndexesResponse{
		IndexNames: indexNames,
	}, nil
}

// DropIndex drops an index from a collection
func (uc *DocumentUseCase) DropIndex(ctx context.Context, req *dto.DropIndexRequest) (*dto.DropIndexResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.DropIndex")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("index_name", req.IndexName),
	)

	logger.Info(ctx, "dropping index",
		zap.String("collection", req.Collection),
		zap.String("index_name", req.IndexName),
	)

	_, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return uc.docRepo.DropIndex(ctx, req.Collection, req.IndexName)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to drop index", zap.Error(err))
		return nil, fmt.Errorf("failed to drop index: %w", err)
	}

	logger.Info(ctx, "index dropped successfully",
		zap.String("collection", req.Collection),
		zap.String("index_name", req.IndexName),
	)

	return &dto.DropIndexResponse{
		Success: true,
	}, nil
}

// ListIndexes lists all indexes on a collection
func (uc *DocumentUseCase) ListIndexes(ctx context.Context, req *dto.ListIndexesRequest) (*dto.ListIndexesResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.ListIndexes")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "listing indexes",
		zap.String("collection", req.Collection),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return uc.docRepo.ListIndexes(ctx, req.Collection)
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to list indexes", zap.Error(err))
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	indexes := result.([]repository.IndexModel)

	// Convert to DTO
	indexInfoList := make([]dto.IndexInfo, len(indexes))
	for i, idx := range indexes {
		unique := false
		if idx.Options != nil {
			if u, ok := idx.Options["unique"].(bool); ok {
				unique = u
			}
		}
		indexInfoList[i] = dto.IndexInfo{
			Name:   idx.Name,
			Keys:   idx.Keys,
			Unique: unique,
		}
	}

	logger.Info(ctx, "indexes listed successfully",
		zap.String("collection", req.Collection),
		zap.Int("count", len(indexes)),
	)

	return &dto.ListIndexesResponse{
		Indexes: indexInfoList,
	}, nil
}

// CreateCollection creates a new collection
func (uc *DocumentUseCase) CreateCollection(ctx context.Context, req *dto.CreateCollectionRequest) (*dto.CreateCollectionResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.CreateCollection")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "creating collection",
		zap.String("collection", req.Collection),
	)

	_, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return uc.docRepo.CreateCollection(ctx, req.Collection, req.Options)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to create collection", zap.Error(err))
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	logger.Info(ctx, "collection created successfully",
		zap.String("collection", req.Collection),
	)

	return &dto.CreateCollectionResponse{
		Success: true,
	}, nil
}

// DropCollection drops a collection
func (uc *DocumentUseCase) DropCollection(ctx context.Context, req *dto.DropCollectionRequest) (*dto.DropCollectionResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.DropCollection")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "dropping collection",
		zap.String("collection", req.Collection),
	)

	_, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return uc.docRepo.DropCollection(ctx, req.Collection)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to drop collection", zap.Error(err))
		return nil, fmt.Errorf("failed to drop collection: %w", err)
	}

	logger.Info(ctx, "collection dropped successfully",
		zap.String("collection", req.Collection),
	)

	return &dto.DropCollectionResponse{
		Success: true,
	}, nil
}

// RenameCollection renames a collection
func (uc *DocumentUseCase) RenameCollection(ctx context.Context, req *dto.RenameCollectionRequest) (*dto.RenameCollectionResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.RenameCollection")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("old_name", req.OldName),
		attribute.String("new_name", req.NewName),
	)

	logger.Info(ctx, "renaming collection",
		zap.String("old_name", req.OldName),
		zap.String("new_name", req.NewName),
	)

	_, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return uc.docRepo.RenameCollection(ctx, req.OldName, req.NewName)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to rename collection", zap.Error(err))
		return nil, fmt.Errorf("failed to rename collection: %w", err)
	}

	logger.Info(ctx, "collection renamed successfully",
		zap.String("old_name", req.OldName),
		zap.String("new_name", req.NewName),
	)

	return &dto.RenameCollectionResponse{
		Success: true,
	}, nil
}

// ListCollections lists all collections
func (uc *DocumentUseCase) ListCollections(ctx context.Context, req *dto.ListCollectionsRequest) (*dto.ListCollectionsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.ListCollections")
	defer span.End()

	logger.Info(ctx, "listing collections")

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return uc.docRepo.ListCollections(ctx, req.Filter)
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to list collections", zap.Error(err))
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	collections := result.([]string)

	// Convert to DTO
	collectionInfoList := make([]dto.CollectionInfo, len(collections))
	for i, name := range collections {
		collectionInfoList[i] = dto.CollectionInfo{
			Name: name,
		}
	}

	logger.Info(ctx, "collections listed successfully",
		zap.Int("count", len(collections)),
	)

	return &dto.ListCollectionsResponse{
		Collections: collectionInfoList,
	}, nil
}

// CollectionExists checks if a collection exists
func (uc *DocumentUseCase) CollectionExists(ctx context.Context, req *dto.CollectionExistsRequest) (*dto.CollectionExistsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.CollectionExists")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "checking if collection exists",
		zap.String("collection", req.Collection),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return uc.docRepo.CollectionExists(ctx, req.Collection)
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to check collection existence", zap.Error(err))
		return nil, fmt.Errorf("failed to check collection existence: %w", err)
	}

	exists := result.(bool)

	logger.Info(ctx, "collection existence checked",
		zap.String("collection", req.Collection),
		zap.Bool("exists", exists),
	)

	return &dto.CollectionExistsResponse{
		Exists: exists,
	}, nil
}

// ExecuteTransaction executes multiple operations in a transaction
func (uc *DocumentUseCase) ExecuteTransaction(ctx context.Context, req *dto.ExecuteTransactionRequest) (*dto.ExecuteTransactionResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.ExecuteTransaction")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.Int("operation_count", len(req.Operations)),
	)

	logger.Info(ctx, "executing transaction",
		zap.Int("operation_count", len(req.Operations)),
	)

	insertedIDs := make([]string, 0)
	var modifiedCount, deletedCount int64

	// Execute transaction
	err := uc.docRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, op := range req.Operations {
			switch op.Type {
			case "insert":
				doc, err := entity.NewDocument(op.Collection, op.Data)
				if err != nil {
					return fmt.Errorf("failed to create document: %w", err)
				}
				if op.ID != "" {
					doc.SetID(op.ID)
				}
				if err := uc.docRepo.Save(txCtx, doc); err != nil {
					return fmt.Errorf("failed to insert document: %w", err)
				}
				insertedIDs = append(insertedIDs, doc.ID())

			case "update":
				var err error
				if op.ID != "" {
					_, err = uc.docRepo.FindAndUpdate(txCtx, op.Collection, op.ID, op.Update)
					if err == nil {
						modifiedCount++
					}
				} else {
					result, err := uc.docRepo.UpdateMany(txCtx, op.Collection, op.Filter, op.Update)
					if err == nil {
						modifiedCount += result.ModifiedCount
					}
				}
				if err != nil {
					return fmt.Errorf("failed to update document: %w", err)
				}

			case "delete":
				if op.ID != "" {
					if err := uc.docRepo.Delete(txCtx, op.Collection, op.ID); err != nil {
						return fmt.Errorf("failed to delete document: %w", err)
					}
					deletedCount++
				} else {
					deleted, err := uc.docRepo.DeleteMany(txCtx, op.Collection, op.Filter)
					if err != nil {
						return fmt.Errorf("failed to delete documents: %w", err)
					}
					deletedCount += deleted
				}

			default:
				return fmt.Errorf("unsupported operation type: %s", op.Type)
			}
		}
		return nil
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "transaction failed", zap.Error(err))
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	logger.Info(ctx, "transaction executed successfully",
		zap.Int("operation_count", len(req.Operations)),
		zap.Int("inserted", len(insertedIDs)),
		zap.Int64("modified", modifiedCount),
		zap.Int64("deleted", deletedCount),
	)

	return &dto.ExecuteTransactionResponse{
		Success:       true,
		InsertedIDs:   insertedIDs,
		ModifiedCount: modifiedCount,
		DeletedCount:  deletedCount,
	}, nil
}

// ExecuteRawQuery executes a raw database query
func (uc *DocumentUseCase) ExecuteRawQuery(ctx context.Context, req *dto.RawQueryRequest) (*dto.RawQueryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.ExecuteRawQuery")
	defer span.End()

	logger.Info(ctx, "executing raw query")

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		var results interface{}
		err := uc.docRepo.ExecuteRawQuery(ctx, req.Query, req.Parameters, &results)
		return results, err
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to execute raw query", zap.Error(err))
		return nil, fmt.Errorf("failed to execute raw query: %w", err)
	}

	logger.Info(ctx, "raw query executed successfully")

	return &dto.RawQueryResponse{
		Results: result,
	}, nil
}

// ExecuteRawQueryTyped executes a raw database query with typed result
func (uc *DocumentUseCase) ExecuteRawQueryTyped(ctx context.Context, req *dto.RawQueryTypedRequest) (*dto.RawQueryTypedResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.ExecuteRawQueryTyped")
	defer span.End()

	logger.Info(ctx, "executing raw query with typed result",
		zap.String("result_type", req.ResultType),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		var results interface{}
		err := uc.docRepo.ExecuteRawQueryWithResult(ctx, req.Query, req.Parameters, &results)
		return results, err
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to execute raw query", zap.Error(err))
		return nil, fmt.Errorf("failed to execute raw query: %w", err)
	}

	logger.Info(ctx, "raw query with typed result executed successfully")

	return &dto.RawQueryTypedResponse{
		Results: result,
	}, nil
}

// HealthCheck performs a health check
func (uc *DocumentUseCase) HealthCheck(ctx context.Context) (*dto.HealthCheckResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.HealthCheck")
	defer span.End()

	logger.Info(ctx, "performing health check")

	services := make(map[string]string)

	// Check database
	if err := uc.docRepo.Ping(ctx); err != nil {
		services["database"] = "unhealthy"
		logger.Error(ctx, "database health check failed", zap.Error(err))
	} else {
		services["database"] = "healthy"
	}

	// Check cache
	if err := uc.cacheRepo.Ping(ctx); err != nil {
		services["cache"] = "unhealthy"
		logger.Warn(ctx, "cache health check failed", zap.Error(err))
	} else {
		services["cache"] = "healthy"
	}

	status := "healthy"
	if services["database"] != "healthy" {
		status = "unhealthy"
	}

	logger.Info(ctx, "health check completed",
		zap.String("status", status),
	)

	return &dto.HealthCheckResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
	}, nil
}

// DatabaseHealth checks database health
func (uc *DocumentUseCase) DatabaseHealth(ctx context.Context, req *dto.DatabaseHealthRequest) (*dto.DatabaseHealthResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.DatabaseHealth")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("database_type", req.DatabaseType),
	)

	logger.Info(ctx, "checking database health",
		zap.String("database_type", req.DatabaseType),
	)

	start := time.Now()
	err := uc.docRepo.Ping(ctx)
	responseTime := time.Since(start).Milliseconds()

	status := "healthy"
	if err != nil {
		status = "unhealthy"
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "database health check failed", zap.Error(err))
	}

	logger.Info(ctx, "database health checked",
		zap.String("database_type", req.DatabaseType),
		zap.String("status", status),
		zap.Int64("response_time_ms", responseTime),
	)

	return &dto.DatabaseHealthResponse{
		Status:       status,
		DatabaseType: req.DatabaseType,
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
	}, nil
}

// GetMetrics retrieves system metrics
func (uc *DocumentUseCase) GetMetrics(ctx context.Context) (*dto.MetricsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.GetMetrics")
	defer span.End()

	logger.Info(ctx, "retrieving metrics")

	// This is a placeholder implementation
	// In a real system, you would collect actual metrics from Prometheus
	metrics := &dto.MetricsResponse{
		Requests:      0, // Would be collected from Prometheus metrics
		Errors:        0, // Would be collected from Prometheus metrics
		Latency:       0, // Would be calculated from histogram metrics
		Uptime:        0, // Would be calculated from start time
		DatabaseStats: make(map[string]dto.DBStats),
	}

	logger.Info(ctx, "metrics retrieved successfully")

	return metrics, nil
}

// GetDatabaseStats retrieves database statistics
func (uc *DocumentUseCase) GetDatabaseStats(ctx context.Context, req *dto.DatabaseStatsRequest) (*dto.DatabaseStatsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.GetDatabaseStats")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("database_type", req.DatabaseType),
	)

	logger.Info(ctx, "retrieving database stats",
		zap.String("database_type", req.DatabaseType),
	)

	// Get collections
	collections, err := uc.docRepo.ListCollections(ctx, nil)
	if err != nil {
		logger.Warn(ctx, "failed to list collections", zap.Error(err))
		collections = []string{}
	}

	// This is a placeholder implementation
	// In a real system, you would collect actual statistics
	stats := &dto.DatabaseStatsResponse{
		DatabaseType:    req.DatabaseType,
		Collections:     len(collections),
		TotalDocuments:  0,
		TotalSize:       0,
		AvgDocumentSize: 0,
	}

	logger.Info(ctx, "database stats retrieved successfully",
		zap.String("database_type", req.DatabaseType),
	)

	return stats, nil
}

// GetCollectionStats retrieves collection statistics
func (uc *DocumentUseCase) GetCollectionStats(ctx context.Context, req *dto.CollectionStatsRequest) (*dto.CollectionStatsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.GetCollectionStats")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "retrieving collection stats",
		zap.String("collection", req.Collection),
	)

	// Get document count
	count, err := uc.docRepo.Count(ctx, req.Collection, nil)
	if err != nil {
		logger.Warn(ctx, "failed to count documents", zap.Error(err))
		count = 0
	}

	// Get indexes
	indexes, err := uc.docRepo.ListIndexes(ctx, req.Collection)
	if err != nil {
		logger.Warn(ctx, "failed to list indexes", zap.Error(err))
		indexes = []repository.IndexModel{}
	}

	// This is a placeholder implementation
	stats := &dto.CollectionStatsResponse{
		Collection:      req.Collection,
		DocumentCount:   count,
		Size:            0,
		AvgDocumentSize: 0,
		IndexCount:      len(indexes),
	}

	logger.Info(ctx, "collection stats retrieved successfully",
		zap.String("collection", req.Collection),
	)

	return stats, nil
}
