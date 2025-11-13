package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/application/dto"
	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence"
	"github.com/YouSangSon/database-service/internal/interfaces/http/middleware"
	"github.com/YouSangSon/database-service/internal/pkg/circuitbreaker"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"github.com/YouSangSon/database-service/internal/pkg/retry"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// DocumentUseCase는 문서 관련 유즈케이스입니다
type DocumentUseCase struct {
	docRepo        repository.DocumentRepository
	repoManager    *persistence.RepositoryManager // For multi-database support
	cacheRepo      repository.CacheRepository
	metrics        *metrics.Metrics
	circuitBreaker *circuitbreaker.CircuitBreaker
	retryConfig    retry.Config
}

// NewDocumentUseCase는 새로운 DocumentUseCase를 생성합니다
func NewDocumentUseCase(
	docRepo repository.DocumentRepository,
	cacheRepo repository.CacheRepository,
) *DocumentUseCase {
	// Circuit breaker 설정
	cb := circuitbreaker.NewCircuitBreaker("document_usecase", circuitbreaker.Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			logger.Info(context.Background(), "circuit breaker state changed",
				zap.String("name", name),
				zap.Int("from", int(from)),
				zap.Int("to", int(to)),
			)
		},
	})

	return &DocumentUseCase{
		docRepo:        docRepo,
		repoManager:    nil,
		cacheRepo:      cacheRepo,
		metrics:        metrics.GetMetrics(),
		circuitBreaker: cb,
		retryConfig:    retry.DefaultConfig(),
	}
}

// NewDocumentUseCaseWithManager creates UseCase with RepositoryManager for multi-database support
func NewDocumentUseCaseWithManager(
	repoManager *persistence.RepositoryManager,
	cacheRepo repository.CacheRepository,
) *DocumentUseCase {
	// Circuit breaker 설정
	cb := circuitbreaker.NewCircuitBreaker("document_usecase", circuitbreaker.Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			logger.Info(context.Background(), "circuit breaker state changed",
				zap.String("name", name),
				zap.Int("from", int(from)),
				zap.Int("to", int(to)),
			)
		},
	})

	return &DocumentUseCase{
		docRepo:        nil,
		repoManager:    repoManager,
		cacheRepo:      cacheRepo,
		metrics:        metrics.GetMetrics(),
		circuitBreaker: cb,
		retryConfig:    retry.DefaultConfig(),
	}
}

// getRepository gets the appropriate repository based on the database type in context
func (uc *DocumentUseCase) getRepository(ctx context.Context) (repository.DocumentRepository, error) {
	// If using single repository mode (for backwards compatibility)
	if uc.repoManager == nil {
		if uc.docRepo == nil {
			return nil, fmt.Errorf("no repository configured")
		}
		return uc.docRepo, nil
	}

	// Get database type from context
	dbType := middleware.GetDatabaseType(ctx)

	// Get repository from manager
	repo, err := uc.repoManager.GetRepository(string(dbType))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository for %s: %w", dbType, err)
	}

	return repo, nil
}

// CreateDocument는 새로운 문서를 생성합니다
func (uc *DocumentUseCase) CreateDocument(ctx context.Context, req *dto.CreateDocumentRequest) (*dto.CreateDocumentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.CreateDocument")
	defer span.End()

	// Get repository based on database type in context
	docRepo, err := uc.getRepository(ctx)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, err
	}

	dbType := middleware.GetDatabaseType(ctx)
	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("database_type", string(dbType)),
	)

	logger.Info(ctx, "creating document",
		zap.String("collection", req.Collection),
		zap.String("database_type", string(dbType)),
	)

	// 도메인 엔티티 생성
	doc, err := entity.NewDocument(req.Collection, req.Data)
	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to create domain entity", zap.Error(err))
		return nil, fmt.Errorf("invalid document: %w", err)
	}

	// Circuit breaker와 retry를 사용하여 저장
	_, err = uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return docRepo.Save(ctx, doc)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to save document", zap.Error(err))
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	// 캐시에 저장
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, doc.ID())
	if err := uc.cacheRepo.Set(ctx, cacheKey, doc, 300); err != nil {
		logger.Warn(ctx, "failed to cache document", zap.Error(err))
		// 캐시 실패는 무시
	}

	logger.Info(ctx, "document created successfully",
		zap.String("id", doc.ID()),
		zap.String("collection", req.Collection),
	)

	return &dto.CreateDocumentResponse{
		ID:        doc.ID(),
		CreatedAt: doc.CreatedAt(),
	}, nil
}

// GetDocument는 문서를 조회합니다
func (uc *DocumentUseCase) GetDocument(ctx context.Context, req *dto.GetDocumentRequest) (*dto.GetDocumentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.GetDocument")
	defer span.End()

	// Get repository based on database type in context
	docRepo, err := uc.getRepository(ctx)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, err
	}

	dbType := middleware.GetDatabaseType(ctx)
	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
		attribute.String("database_type", string(dbType)),
	)

	logger.Info(ctx, "getting document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
		zap.String("database_type", string(dbType)),
	)

	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)

	// 캐시에서 조회 시도
	cachedData, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil {
		uc.metrics.RecordCacheHit("document")
		logger.Debug(ctx, "cache hit", zap.String("key", cacheKey))

		// 캐시 데이터를 DTO로 변환
		data, _ := json.Marshal(cachedData)
		var doc entity.Document
		json.Unmarshal(data, &doc)

		return &dto.GetDocumentResponse{
			ID:        req.ID,
			Data:      doc.Data(),
			Version:   doc.Version(),
			CreatedAt: doc.CreatedAt(),
			UpdatedAt: doc.UpdatedAt(),
		}, nil
	}

	uc.metrics.RecordCacheMiss("document")
	logger.Debug(ctx, "cache miss", zap.String("key", cacheKey))

	// DB에서 조회
	var doc *entity.Document
	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return retry.DoWithValue(ctx, uc.retryConfig, func(ctx context.Context) (*entity.Document, error) {
			return docRepo.FindByID(ctx, req.Collection, req.ID)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to get document", zap.Error(err))
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	doc = result.(*entity.Document)

	// 캐시에 저장
	if err := uc.cacheRepo.Set(ctx, cacheKey, doc, 300); err != nil {
		logger.Warn(ctx, "failed to cache document", zap.Error(err))
	}

	logger.Info(ctx, "document retrieved successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
	)

	return &dto.GetDocumentResponse{
		ID:        doc.ID(),
		Data:      doc.Data(),
		Version:   doc.Version(),
		CreatedAt: doc.CreatedAt(),
		UpdatedAt: doc.UpdatedAt(),
	}, nil
}

// UpdateDocument는 문서를 업데이트합니다
func (uc *DocumentUseCase) UpdateDocument(ctx context.Context, req *dto.UpdateDocumentRequest) error {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.UpdateDocument")
	defer span.End()

	// Get repository based on database type in context
	docRepo, err := uc.getRepository(ctx)
	if err != nil {
		tracing.RecordError(ctx, err)
		return err
	}

	dbType := middleware.GetDatabaseType(ctx)
	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
		attribute.Int("version", req.Version),
		attribute.String("database_type", string(dbType)),
	)

	logger.Info(ctx, "updating document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
		zap.String("database_type", string(dbType)),
	)

	// 기존 문서 조회
	doc, err := docRepo.FindByID(ctx, req.Collection, req.ID)
	if err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("failed to find document: %w", err)
	}

	// 버전 확인
	if doc.Version() != req.Version {
		return entity.ErrVersionConflict
	}

	// 업데이트
	if err := doc.Update(req.Data); err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("failed to update document: %w", err)
	}

	// Circuit breaker와 retry를 사용하여 저장
	_, err = uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return docRepo.Update(ctx, doc)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to update document", zap.Error(err))
		return fmt.Errorf("failed to update document: %w", err)
	}

	// 캐시 무효화
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)
	if err := uc.cacheRepo.Delete(ctx, cacheKey); err != nil {
		logger.Warn(ctx, "failed to invalidate cache", zap.Error(err))
	}

	logger.Info(ctx, "document updated successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
	)

	return nil
}

// DeleteDocument는 문서를 삭제합니다
func (uc *DocumentUseCase) DeleteDocument(ctx context.Context, req *dto.DeleteDocumentRequest) error {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.DeleteDocument")
	defer span.End()

	// Get repository based on database type in context
	docRepo, err := uc.getRepository(ctx)
	if err != nil {
		tracing.RecordError(ctx, err)
		return err
	}

	dbType := middleware.GetDatabaseType(ctx)
	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.String("id", req.ID),
		attribute.String("database_type", string(dbType)),
	)

	logger.Info(ctx, "deleting document",
		zap.String("collection", req.Collection),
		zap.String("id", req.ID),
		zap.String("database_type", string(dbType)),
	)

	// Circuit breaker와 retry를 사용하여 삭제
	_, err = uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, retry.Do(ctx, uc.retryConfig, func(ctx context.Context) error {
			return docRepo.Delete(ctx, req.Collection, req.ID)
		})
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to delete document", zap.Error(err))
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// 캐시 무효화
	cacheKey := fmt.Sprintf("document:%s:%s", req.Collection, req.ID)
	if err := uc.cacheRepo.Delete(ctx, cacheKey); err != nil {
		logger.Warn(ctx, "failed to invalidate cache", zap.Error(err))
	}

	logger.Info(ctx, "document deleted successfully",
		zap.String("id", req.ID),
		zap.String("collection", req.Collection),
	)

	return nil
}

// ListDocuments는 문서 목록을 조회합니다
func (uc *DocumentUseCase) ListDocuments(ctx context.Context, req *dto.ListDocumentsRequest) (*dto.ListDocumentsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "DocumentUseCase.ListDocuments")
	defer span.End()

	// Get repository based on database type in context
	docRepo, err := uc.getRepository(ctx)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, err
	}

	dbType := middleware.GetDatabaseType(ctx)
	tracing.SetAttributes(ctx,
		attribute.String("collection", req.Collection),
		attribute.Int("page", req.Page),
		attribute.Int("page_size", req.PageSize),
		attribute.String("database_type", string(dbType)),
	)

	logger.Info(ctx, "listing documents",
		zap.String("collection", req.Collection),
		zap.Int("page", req.Page),
		zap.Int("page_size", req.PageSize),
		zap.String("database_type", string(dbType)),
	)

	// Circuit breaker를 사용하여 조회
	result, err := uc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return docRepo.FindAll(ctx, req.Collection, req.Filter)
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		logger.Error(ctx, "failed to list documents", zap.Error(err))
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	docs := result.([]*entity.Document)

	// 총 개수 조회
	count, err := docRepo.Count(ctx, req.Collection, req.Filter)
	if err != nil {
		logger.Warn(ctx, "failed to count documents", zap.Error(err))
		count = int64(len(docs))
	}

	// DTO로 변환
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

	logger.Info(ctx, "documents listed successfully",
		zap.String("collection", req.Collection),
		zap.Int("count", len(docs)),
	)

	return &dto.ListDocumentsResponse{
		Documents:  dtoList,
		TotalCount: count,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}
