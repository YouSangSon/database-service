package models

import (
	"time"
)

// Document는 범용 문서 모델입니다
type Document struct {
	ID        string                 `json:"id" bson:"_id,omitempty"`
	Data      map[string]interface{} `json:"data" bson:"data"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" bson:"updated_at"`
}

// CreateRequest는 문서 생성 요청입니다
type CreateRequest struct {
	Collection string                 `json:"collection" binding:"required"`
	Data       map[string]interface{} `json:"data" binding:"required"`
}

// CreateResponse는 문서 생성 응답입니다
type CreateResponse struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
}

// ReadRequest는 문서 조회 요청입니다
type ReadRequest struct {
	Collection string `json:"collection" binding:"required"`
	ID         string `json:"id" binding:"required"`
}

// UpdateRequest는 문서 업데이트 요청입니다
type UpdateRequest struct {
	Collection string                 `json:"collection" binding:"required"`
	ID         string                 `json:"id" binding:"required"`
	Data       map[string]interface{} `json:"data" binding:"required"`
}

// DeleteRequest는 문서 삭제 요청입니다
type DeleteRequest struct {
	Collection string `json:"collection" binding:"required"`
	ID         string `json:"id" binding:"required"`
}

// ListRequest는 문서 목록 조회 요청입니다
type ListRequest struct {
	Collection string                 `json:"collection" binding:"required"`
	Filter     map[string]interface{} `json:"filter"`
	Limit      int                    `json:"limit"`
	Skip       int                    `json:"skip"`
}

// ListResponse는 문서 목록 조회 응답입니다
type ListResponse struct {
	Documents []Document `json:"documents"`
	Total     int        `json:"total"`
}

// ErrorResponse는 에러 응답입니다
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
