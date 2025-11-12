package dto

import "time"

// CreateDocumentRequest는 문서 생성 요청 DTO입니다
type CreateDocumentRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Data       map[string]interface{} `json:"data" validate:"required"`
}

// CreateDocumentResponse는 문서 생성 응답 DTO입니다
type CreateDocumentResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// GetDocumentRequest는 문서 조회 요청 DTO입니다
type GetDocumentRequest struct {
	Collection string `json:"collection" validate:"required"`
	ID         string `json:"id" validate:"required"`
}

// GetDocumentResponse는 문서 조회 응답 DTO입니다
type GetDocumentResponse struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Version   int                    `json:"version"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// UpdateDocumentRequest는 문서 업데이트 요청 DTO입니다
type UpdateDocumentRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	ID         string                 `json:"id" validate:"required"`
	Data       map[string]interface{} `json:"data" validate:"required"`
	Version    int                    `json:"version" validate:"required"`
}

// DeleteDocumentRequest는 문서 삭제 요청 DTO입니다
type DeleteDocumentRequest struct {
	Collection string `json:"collection" validate:"required"`
	ID         string `json:"id" validate:"required"`
}

// ListDocumentsRequest는 문서 목록 조회 요청 DTO입니다
type ListDocumentsRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Filter     map[string]interface{} `json:"filter"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
}

// ListDocumentsResponse는 문서 목록 조회 응답 DTO입니다
type ListDocumentsResponse struct {
	Documents  []GetDocumentResponse `json:"documents"`
	TotalCount int64                 `json:"total_count"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
}
