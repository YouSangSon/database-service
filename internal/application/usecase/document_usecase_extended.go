package usecase

import (
	"context"
	"fmt"

	"github.com/YouSangSon/database-service/internal/application/dto"
	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/retry"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// ReplaceDocument replaces a document completely
func (uc *DocumentUseCase) ReplaceDocument(ctx context.Context, req *dto.ReplaceDocumentRequest) (*dto.ReplaceDocumentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.ReplaceDocument")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
	)

	logger.Info(ctx, "replacing document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
	)

	// Get existing document
	existing, err := uc.docRepo.FindByID(ctx, req.Collection, req.ID)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// Create new document with same ID
	doc := &entity.Document{}
	doc.SetID(req.ID)
	doc.SetCollection(req.Collection)
	doc.SetData(req.Data)
	doc.SetVersion(existing.Version() + 1)

	// Save
	_, err = uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return uc.docRepo.Replace(ctx, doc)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to replace document", zap.Error(err))
		return nil, fmt.Errorf("failed to replace document: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)
	uc.cacheRepo.Delete(ctx, cacheKey)

	logger.Info(ctx, "document replaced successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
	)

	return &dto.ReplaceDocumentResponse{
		ID:        doc.ID(),
		Data:      doc.Data(),
		Version:   doc.Version(),
		UpdatedAt: doc.UpdatedAt(),
	}, nil
}

// SearchDocuments searches documents with filters
func (uc *DocumentUseCase) SearchDocuments(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchDocumentsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.SearchDocuments")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.Int("limit", req.Limit),
		attribute.Int("offset", req.Offset),
	)

	logger.Info(ctx, "searching documents",
		zap.String("collection", req.Collection),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset),
	)

	// Execute search
	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return uc.docRepo.Find(ctx, req.Collection, req.Filter, &repository.FindOptions{
			Sort:   req.Sort,
			Limit:  req.Limit,
			Offset: req.Offset,
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to search documents", zap.Error(err))
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	docs := result.([]*entity.Document)

	// Get total count
	count, err := uc.docRepo.Count(ctx, req.Collection, req.Filter)
	if err != nil {
		logger.Warn(ctx, "failed to count documents", zap.Error(err))
		count = int64(len(docs))
	}

	// Convert to DTO
	dtoList := make([]dto.GetDocumentResponse, len(docs))
	for i, doc := range docs {
		dtoList[i] = dto.GetDocumentResponse{
			ID:        doc.ID(),
			Data:      doc.Data(),
			Version:   doc.Version(),
			CreatedAt: doc.CreatedAt(),
			UpdatedAt: doc.UpdatedAt(),
		}
	}

	logger.Info(ctx, "documents searched successfully",
		zap.String("collection", req.Collection),
		zap.Int("count", len(docs)),
	)

	return &dto.SearchDocumentsResponse{
		Documents: dtoList,
		Total:     count,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}, nil
}

// CountDocuments counts documents matching filter
func (uc *DocumentUseCase) CountDocuments(ctx context.Context, req *dto.CountDocumentsRequest) (*dto.CountDocumentsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.CountDocuments")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "counting documents",
		zap.String("collection", req.Collection),
	)

	count, err := uc.docRepo.Count(ctx, req.Collection, req.Filter)
	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to count documents", zap.Error(err))
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	logger.Info(ctx, "documents counted successfully",
		zap.String("collection", req.Collection),
		zap.Int64("count", count),
	)

	return &dto.CountDocumentsResponse{
		Count: count,
	}, nil
}

// EstimatedCount returns estimated document count
func (uc *DocumentUseCase) EstimatedCount(ctx context.Context, req *dto.EstimatedCountRequest) (*dto.EstimatedCountResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.EstimatedCount")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "getting estimated count",
		zap.String("collection", req.Collection),
	)

	count, err := uc.docRepo.EstimatedCount(ctx, req.Collection)
	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to get estimated count", zap.Error(err))
		return nil, fmt.Errorf("failed to get estimated count: %w", err)
	}

	logger.Info(ctx, "estimated count retrieved successfully",
		zap.String("collection", req.Collection),
		zap.Int64("count", count),
	)

	return &dto.EstimatedCountResponse{
		Count: count,
	}, nil
}

// FindAndUpdate finds and updates a document atomically
func (uc *DocumentUseCase) FindAndUpdate(ctx context.Context, req *dto.FindAndUpdateRequest) (*dto.FindAndUpdateResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.FindAndUpdate")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
	)

	logger.Info(ctx, "find and update document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (*entity.Document, error) {
			return uc.docRepo.FindAndUpdate(ctx, req.Collection, req.ID, req.Update)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to find and update document", zap.Error(err))
		return nil, fmt.Errorf("failed to find and update document: %w", err)
	}

	doc := result.(*entity.Document)

	// Invalidate cache
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)
	uc.cacheRepo.Delete(ctx, cacheKey)

	logger.Info(ctx, "document found and updated successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
	)

	return &dto.FindAndUpdateResponse{
		ID:        doc.ID(),
		Data:      doc.Data(),
		Version:   doc.Version(),
		UpdatedAt: doc.UpdatedAt(),
	}, nil
}

// FindAndReplace finds and replaces a document atomically
func (uc *DocumentUseCase) FindAndReplace(ctx context.Context, req *dto.FindAndReplaceRequest) (*dto.FindAndReplaceResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.FindAndReplace")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
	)

	logger.Info(ctx, "find and replace document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (*entity.Document, error) {
			return uc.docRepo.FindAndReplace(ctx, req.Collection, req.ID, req.Data)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to find and replace document", zap.Error(err))
		return nil, fmt.Errorf("failed to find and replace document: %w", err)
	}

	doc := result.(*entity.Document)

	// Invalidate cache
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)
	uc.cacheRepo.Delete(ctx, cacheKey)

	logger.Info(ctx, "document found and replaced successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
	)

	return &dto.FindAndReplaceResponse{
		ID:        doc.ID(),
		Data:      doc.Data(),
		Version:   doc.Version(),
		UpdatedAt: doc.UpdatedAt(),
	}, nil
}

// FindAndDelete finds and deletes a document atomically
func (uc *DocumentUseCase) FindAndDelete(ctx context.Context, req *dto.FindAndDeleteRequest) (*dto.FindAndDeleteResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.FindAndDelete")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
	)

	logger.Info(ctx, "find and delete document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (*entity.Document, error) {
			return uc.docRepo.FindAndDelete(ctx, req.Collection, req.ID)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to find and delete document", zap.Error(err))
		return nil, fmt.Errorf("failed to find and delete document: %w", err)
	}

	doc := result.(*entity.Document)

	// Invalidate cache
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)
	uc.cacheRepo.Delete(ctx, cacheKey)

	logger.Info(ctx, "document found and deleted successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
	)

	return &dto.FindAndDeleteResponse{
		ID:   doc.ID(),
		Data: doc.Data(),
	}, nil
}

// Upsert inserts or updates a document
func (uc *DocumentUseCase) Upsert(ctx context.Context, req *dto.UpsertRequest) (*dto.UpsertResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.Upsert")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
	)

	logger.Info(ctx, "upserting document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
	)

	// Try to find existing document
	existing, err := uc.docRepo.FindByID(ctx, req.Collection, req.ID)
	upserted := err != nil // If not found, it's an insert

	// Create or update document
	doc := &entity.Document{}
	doc.SetID(req.ID)
	doc.SetCollection(req.Collection)
	doc.SetData(req.Data)
	if !upserted {
		doc.SetVersion(existing.Version() + 1)
	} else {
		doc.SetVersion(1)
	}

	// Execute upsert
	_, err = uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return uc.docRepo.Upsert(ctx, doc)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to upsert document", zap.Error(err))
		return nil, fmt.Errorf("failed to upsert document: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)
	uc.cacheRepo.Delete(ctx, cacheKey)

	logger.Info(ctx, "document upserted successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
		zap.Bool("upserted", upserted),
	)

	return &dto.UpsertResponse{
		ID:        doc.ID(),
		Data:      doc.Data(),
		Version:   doc.Version(),
		Upserted:  upserted,
		UpdatedAt: doc.UpdatedAt(),
	}, nil
}

// AggregateDocuments runs an aggregation pipeline
func (uc *DocumentUseCase) AggregateDocuments(ctx context.Context, req *dto.AggregateDocumentRequest) (*dto.AggregateDocumentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.AggregateDocuments")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "aggregating documents",
		zap.String("collection", req.Collection),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return uc.docRepo.Aggregate(ctx, req.Collection, req.Pipeline)
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to aggregate documents", zap.Error(err))
		return nil, fmt.Errorf("failed to aggregate documents: %w", err)
	}

	results := result.([]map[string]interface{})

	logger.Info(ctx, "documents aggregated successfully",
		zap.String("collection", req.Collection),
		zap.Int("result_count", len(results)),
	)

	return &dto.AggregateDocumentResponse{
		Results: results,
	}, nil
}

// Distinct retrieves distinct values for a field
func (uc *DocumentUseCase) Distinct(ctx context.Context, req *dto.DistinctRequest) (*dto.DistinctResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.Distinct")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("field", req.Field),
	)

	logger.Info(ctx, "getting distinct values",
		zap.String("collection", req.Collection),
		zap.String("field", req.Field),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return uc.docRepo.Distinct(ctx, req.Collection, req.Field, req.Filter)
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to get distinct values", zap.Error(err))
		return nil, fmt.Errorf("failed to get distinct values: %w", err)
	}

	values := result.([]interface{})

	logger.Info(ctx, "distinct values retrieved successfully",
		zap.String("collection", req.Collection),
		zap.String("field", req.Field),
		zap.Int("value_count", len(values)),
	)

	return &dto.DistinctResponse{
		Values: values,
	}, nil
}

// BulkInsert inserts multiple documents
func (uc *DocumentUseCase) BulkInsert(ctx context.Context, req *dto.BulkInsertRequest) (*dto.BulkInsertResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.BulkInsert")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.Int("document_count", len(req.Documents)),
	)

	logger.Info(ctx, "bulk inserting documents",
		zap.String("collection", req.Collection),
		zap.Int("count", len(req.Documents)),
	)

	// Convert to domain entities
	docs := make([]*entity.Document, len(req.Documents))
	for i, data := range req.Documents {
		doc, err := entity.NewDocument(req.Collection, data)
		if err != nil {
			tracing.RecordError(ctx, err)
			return nil, fmt.Errorf("invalid document at index %d: %w", i, err)
		}
		docs[i] = doc
	}

	// Execute bulk insert
	_, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return uc.docRepo.BulkInsert(ctx, docs)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to bulk insert documents", zap.Error(err))
		return nil, fmt.Errorf("failed to bulk insert documents: %w", err)
	}

	// Collect IDs
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID()
	}

	logger.Info(ctx, "documents bulk inserted successfully",
		zap.String("collection", req.Collection),
		zap.Int("count", len(docs)),
	)

	return &dto.BulkInsertResponse{
		InsertedIDs:   ids,
		InsertedCount: len(ids),
	}, nil
}

// UpdateMany updates multiple documents
func (uc *DocumentUseCase) UpdateMany(ctx context.Context, req *dto.UpdateManyRequest) (*dto.UpdateManyResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.UpdateMany")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "updating many documents",
		zap.String("collection", req.Collection),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (*repository.UpdateResult, error) {
			return uc.docRepo.UpdateMany(ctx, req.Collection, req.Filter, req.Update)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to update many documents", zap.Error(err))
		return nil, fmt.Errorf("failed to update many documents: %w", err)
	}

	updateResult := result.(*repository.UpdateResult)

	logger.Info(ctx, "documents updated successfully",
		zap.String("collection", req.Collection),
		zap.Int64("matched", updateResult.MatchedCount),
		zap.Int64("modified", updateResult.ModifiedCount),
	)

	return &dto.UpdateManyResponse{
		MatchedCount:  updateResult.MatchedCount,
		ModifiedCount: updateResult.ModifiedCount,
	}, nil
}

// DeleteMany deletes multiple documents
func (uc *DocumentUseCase) DeleteMany(ctx context.Context, req *dto.DeleteManyRequest) (*dto.DeleteManyResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.DeleteMany")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
	)

	logger.Info(ctx, "deleting many documents",
		zap.String("collection", req.Collection),
	)

	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (int64, error) {
			return uc.docRepo.DeleteMany(ctx, req.Collection, req.Filter)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to delete many documents", zap.Error(err))
		return nil, fmt.Errorf("failed to delete many documents: %w", err)
	}

	deletedCount := result.(int64)

	logger.Info(ctx, "documents deleted successfully",
		zap.String("collection", req.Collection),
		zap.Int64("deleted", deletedCount),
	)

	return &dto.DeleteManyResponse{
		DeletedCount: deletedCount,
	}, nil
}

// BulkWrite executes multiple write operations
func (uc *DocumentUseCase) BulkWrite(ctx context.Context, req *dto.BulkWriteRequest) (*dto.BulkWriteResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.BulkWrite")
	defer span.End()

	tracing.SetAttributes(ctx,
		attribute.Int("operation_count", len(req.Operations)),
	)

	logger.Info(ctx, "executing bulk write",
		zap.Int("operation_count", len(req.Operations)),
	)

	// Convert to repository bulk operations
	operations := make([]*repository.BulkOperation, len(req.Operations))
	for i, op := range req.Operations {
		bulkOp := &repository.BulkOperation{
			Type:       op.Type,
			Collection: op.Collection,
			Filter:     op.Filter,
			Data:       op.Data,
			Update:     op.Update,
		}
		if op.ID != "" {
			bulkOp.Filter = map[string]interface{}{"id": op.ID}
		}
		operations[i] = bulkOp
	}

	// Execute bulk write
	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (*repository.BulkResult, error) {
			return uc.docRepo.BulkWrite(ctx, operations)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to execute bulk write", zap.Error(err))
		return nil, fmt.Errorf("failed to execute bulk write: %w", err)
	}

	bulkResult := result.(*repository.BulkResult)

	logger.Info(ctx, "bulk write executed successfully",
		zap.Int64("inserted", bulkResult.InsertedCount),
		zap.Int64("modified", bulkResult.ModifiedCount),
		zap.Int64("deleted", bulkResult.DeletedCount),
	)

	return &dto.BulkWriteResponse{
		InsertedCount: bulkResult.InsertedCount,
		MatchedCount:  bulkResult.MatchedCount,
		ModifiedCount: bulkResult.ModifiedCount,
		DeletedCount:  bulkResult.DeletedCount,
		UpsertedCount: bulkResult.UpsertedCount,
		UpsertedIDs:   bulkResult.UpsertedIDs,
	}, nil
}
