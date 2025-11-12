package service

import (
	"context"
	"fmt"

	"github.com/YouSangSon/database-service/internal/database"
	"github.com/YouSangSon/database-service/internal/models"
)

// Service는 비즈니스 로직을 처리하는 서비스 레이어입니다
type Service struct {
	db database.Database
}

// NewService는 새로운 Service 인스턴스를 생성합니다
func NewService(db database.Database) *Service {
	return &Service{
		db: db,
	}
}

// Create는 새로운 문서를 생성합니다
func (s *Service) Create(ctx context.Context, req *models.CreateRequest) (*models.CreateResponse, error) {
	id, err := s.db.Create(ctx, req.Collection, req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// 생성된 문서 조회
	var doc models.Document
	if err := s.db.Read(ctx, req.Collection, id, &doc); err != nil {
		return nil, fmt.Errorf("failed to read created document: %w", err)
	}

	return &models.CreateResponse{
		ID:      id,
		Created: doc.CreatedAt,
	}, nil
}

// Read는 ID로 문서를 조회합니다
func (s *Service) Read(ctx context.Context, req *models.ReadRequest) (*models.Document, error) {
	var doc models.Document
	if err := s.db.Read(ctx, req.Collection, req.ID, &doc); err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	return &doc, nil
}

// Update는 기존 문서를 업데이트합니다
func (s *Service) Update(ctx context.Context, req *models.UpdateRequest) error {
	if err := s.db.Update(ctx, req.Collection, req.ID, req.Data); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// Delete는 문서를 삭제합니다
func (s *Service) Delete(ctx context.Context, req *models.DeleteRequest) error {
	if err := s.db.Delete(ctx, req.Collection, req.ID); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// List는 컬렉션의 문서 목록을 조회합니다
func (s *Service) List(ctx context.Context, req *models.ListRequest) (*models.ListResponse, error) {
	var docs []models.Document
	if err := s.db.List(ctx, req.Collection, req.Filter, &docs); err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return &models.ListResponse{
		Documents: docs,
		Total:     len(docs),
	}, nil
}

// HealthCheck는 데이터베이스 연결 상태를 확인합니다
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.db.Ping(ctx)
}
