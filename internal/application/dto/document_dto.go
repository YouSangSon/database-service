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
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
	Sort       string                `json:"sort"`
}

// UpdateDocumentResponse는 문서 업데이트 응답 DTO입니다
type UpdateDocumentResponse struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Version   int                    `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ReplaceDocumentRequest는 문서 교체 요청 DTO입니다
type ReplaceDocumentRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	ID         string                 `json:"id" validate:"required"`
	Data       map[string]interface{} `json:"data" validate:"required"`
}

// ReplaceDocumentResponse는 문서 교체 응답 DTO입니다
type ReplaceDocumentResponse struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Version   int                    `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SearchDocumentsRequest는 문서 검색 요청 DTO입니다
type SearchDocumentsRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Filter     map[string]interface{} `json:"filter"`
	Sort       map[string]int         `json:"sort"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
}

// SearchDocumentsResponse는 문서 검색 응답 DTO입니다
type SearchDocumentsResponse struct {
	Documents []GetDocumentResponse `json:"documents"`
	Total     int64                 `json:"total"`
	Limit     int                   `json:"limit"`
	Offset    int                   `json:"offset"`
}

// CountDocumentsRequest는 문서 개수 조회 요청 DTO입니다
type CountDocumentsRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Filter     map[string]interface{} `json:"filter"`
}

// CountDocumentsResponse는 문서 개수 조회 응답 DTO입니다
type CountDocumentsResponse struct {
	Count int64 `json:"count"`
}

// EstimatedCountRequest는 예상 문서 개수 조회 요청 DTO입니다
type EstimatedCountRequest struct {
	Collection string `json:"collection" validate:"required"`
}

// EstimatedCountResponse는 예상 문서 개수 조회 응답 DTO입니다
type EstimatedCountResponse struct {
	Count int64 `json:"count"`
}

// FindAndUpdateRequest는 문서 찾아서 업데이트 요청 DTO입니다
type FindAndUpdateRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	ID         string                 `json:"id" validate:"required"`
	Update     map[string]interface{} `json:"update" validate:"required"`
}

// FindAndUpdateResponse는 문서 찾아서 업데이트 응답 DTO입니다
type FindAndUpdateResponse struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Version   int                    `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// FindAndReplaceRequest는 문서 찾아서 교체 요청 DTO입니다
type FindAndReplaceRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	ID         string                 `json:"id" validate:"required"`
	Data       map[string]interface{} `json:"data" validate:"required"`
}

// FindAndReplaceResponse는 문서 찾아서 교체 응답 DTO입니다
type FindAndReplaceResponse struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Version   int                    `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// FindAndDeleteRequest는 문서 찾아서 삭제 요청 DTO입니다
type FindAndDeleteRequest struct {
	Collection string `json:"collection" validate:"required"`
	ID         string `json:"id" validate:"required"`
}

// FindAndDeleteResponse는 문서 찾아서 삭제 응답 DTO입니다
type FindAndDeleteResponse struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

// UpsertRequest는 문서 Upsert 요청 DTO입니다
type UpsertRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	ID         string                 `json:"id" validate:"required"`
	Data       map[string]interface{} `json:"data" validate:"required"`
}

// UpsertResponse는 문서 Upsert 응답 DTO입니다
type UpsertResponse struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Version   int                    `json:"version"`
	Upserted  bool                   `json:"upserted"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// AggregateDocumentRequest는 문서 집계 요청 DTO입니다
type AggregateDocumentRequest struct {
	Collection string                   `json:"collection" validate:"required"`
	Pipeline   []map[string]interface{} `json:"pipeline" validate:"required"`
}

// AggregateDocumentResponse는 문서 집계 응답 DTO입니다
type AggregateDocumentResponse struct {
	Results []map[string]interface{} `json:"results"`
}

// DistinctRequest는 고유값 조회 요청 DTO입니다
type DistinctRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Field      string                 `json:"field" validate:"required"`
	Filter     map[string]interface{} `json:"filter"`
}

// DistinctResponse는 고유값 조회 응답 DTO입니다
type DistinctResponse struct {
	Values []interface{} `json:"values"`
}

// BulkInsertRequest는 대량 삽입 요청 DTO입니다
type BulkInsertRequest struct {
	Collection string                   `json:"collection" validate:"required"`
	Documents  []map[string]interface{} `json:"documents" validate:"required"`
}

// BulkInsertResponse는 대량 삽입 응답 DTO입니다
type BulkInsertResponse struct {
	InsertedIDs   []string `json:"inserted_ids"`
	InsertedCount int      `json:"inserted_count"`
}

// UpdateManyRequest는 다수 문서 업데이트 요청 DTO입니다
type UpdateManyRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Filter     map[string]interface{} `json:"filter" validate:"required"`
	Update     map[string]interface{} `json:"update" validate:"required"`
}

// UpdateManyResponse는 다수 문서 업데이트 응답 DTO입니다
type UpdateManyResponse struct {
	MatchedCount  int64 `json:"matched_count"`
	ModifiedCount int64 `json:"modified_count"`
}

// DeleteManyRequest는 다수 문서 삭제 요청 DTO입니다
type DeleteManyRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Filter     map[string]interface{} `json:"filter" validate:"required"`
}

// DeleteManyResponse는 다수 문서 삭제 응답 DTO입니다
type DeleteManyResponse struct {
	DeletedCount int64 `json:"deleted_count"`
}

// BulkOperation은 대량 작업 정의 DTO입니다
type BulkOperation struct {
	Type       string                 `json:"type" validate:"required,oneof=insert update delete"`
	Collection string                 `json:"collection" validate:"required"`
	ID         string                 `json:"id"`
	Filter     map[string]interface{} `json:"filter"`
	Data       map[string]interface{} `json:"data"`
	Update     map[string]interface{} `json:"update"`
}

// BulkWriteRequest는 대량 쓰기 요청 DTO입니다
type BulkWriteRequest struct {
	Operations []BulkOperation `json:"operations" validate:"required"`
}

// BulkWriteResponse는 대량 쓰기 응답 DTO입니다
type BulkWriteResponse struct {
	InsertedCount int64    `json:"inserted_count"`
	MatchedCount  int64    `json:"matched_count"`
	ModifiedCount int64    `json:"modified_count"`
	DeletedCount  int64    `json:"deleted_count"`
	UpsertedCount int64    `json:"upserted_count"`
	UpsertedIDs   []string `json:"upserted_ids"`
}

// CreateIndexRequest는 인덱스 생성 요청 DTO입니다
type CreateIndexRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Keys       map[string]int         `json:"keys" validate:"required"`
	Options    map[string]interface{} `json:"options"`
}

// CreateIndexResponse는 인덱스 생성 응답 DTO입니다
type CreateIndexResponse struct {
	IndexName string `json:"index_name"`
}

// CreateIndexesRequest는 다수 인덱스 생성 요청 DTO입니다
type CreateIndexesRequest struct {
	Collection string                   `json:"collection" validate:"required"`
	Indexes    []map[string]interface{} `json:"indexes" validate:"required"`
}

// CreateIndexesResponse는 다수 인덱스 생성 응답 DTO입니다
type CreateIndexesResponse struct {
	IndexNames []string `json:"index_names"`
}

// DropIndexRequest는 인덱스 삭제 요청 DTO입니다
type DropIndexRequest struct {
	Collection string `json:"collection" validate:"required"`
	IndexName  string `json:"index_name" validate:"required"`
}

// DropIndexResponse는 인덱스 삭제 응답 DTO입니다
type DropIndexResponse struct {
	Success bool `json:"success"`
}

// ListIndexesRequest는 인덱스 목록 조회 요청 DTO입니다
type ListIndexesRequest struct {
	Collection string `json:"collection" validate:"required"`
}

// IndexInfo는 인덱스 정보 DTO입니다
type IndexInfo struct {
	Name   string         `json:"name"`
	Keys   map[string]int `json:"keys"`
	Unique bool           `json:"unique"`
}

// ListIndexesResponse는 인덱스 목록 조회 응답 DTO입니다
type ListIndexesResponse struct {
	Indexes []IndexInfo `json:"indexes"`
}

// CreateCollectionRequest는 컬렉션 생성 요청 DTO입니다
type CreateCollectionRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Options    map[string]interface{} `json:"options"`
}

// CreateCollectionResponse는 컬렉션 생성 응답 DTO입니다
type CreateCollectionResponse struct {
	Success bool `json:"success"`
}

// DropCollectionRequest는 컬렉션 삭제 요청 DTO입니다
type DropCollectionRequest struct {
	Collection string `json:"collection" validate:"required"`
}

// DropCollectionResponse는 컬렉션 삭제 응답 DTO입니다
type DropCollectionResponse struct {
	Success bool `json:"success"`
}

// RenameCollectionRequest는 컬렉션 이름 변경 요청 DTO입니다
type RenameCollectionRequest struct {
	OldName string `json:"old_name" validate:"required"`
	NewName string `json:"new_name" validate:"required"`
}

// RenameCollectionResponse는 컬렉션 이름 변경 응답 DTO입니다
type RenameCollectionResponse struct {
	Success bool `json:"success"`
}

// ListCollectionsRequest는 컬렉션 목록 조회 요청 DTO입니다
type ListCollectionsRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// CollectionInfo는 컬렉션 정보 DTO입니다
type CollectionInfo struct {
	Name string `json:"name"`
}

// ListCollectionsResponse는 컬렉션 목록 조회 응답 DTO입니다
type ListCollectionsResponse struct {
	Collections []CollectionInfo `json:"collections"`
}

// CollectionExistsRequest는 컬렉션 존재 확인 요청 DTO입니다
type CollectionExistsRequest struct {
	Collection string `json:"collection" validate:"required"`
}

// CollectionExistsResponse는 컬렉션 존재 확인 응답 DTO입니다
type CollectionExistsResponse struct {
	Exists bool `json:"exists"`
}

// TransactionOperation은 트랜잭션 작업 정의 DTO입니다
type TransactionOperation struct {
	Type       string                 `json:"type" validate:"required,oneof=insert update delete"`
	Collection string                 `json:"collection" validate:"required"`
	ID         string                 `json:"id"`
	Data       map[string]interface{} `json:"data"`
	Update     map[string]interface{} `json:"update"`
	Filter     map[string]interface{} `json:"filter"`
}

// ExecuteTransactionRequest는 트랜잭션 실행 요청 DTO입니다
type ExecuteTransactionRequest struct {
	Operations []TransactionOperation `json:"operations" validate:"required"`
}

// ExecuteTransactionResponse는 트랜잭션 실행 응답 DTO입니다
type ExecuteTransactionResponse struct {
	Success       bool     `json:"success"`
	InsertedIDs   []string `json:"inserted_ids,omitempty"`
	ModifiedCount int64    `json:"modified_count,omitempty"`
	DeletedCount  int64    `json:"deleted_count,omitempty"`
}

// RawQueryRequest는 원시 쿼리 실행 요청 DTO입니다
type RawQueryRequest struct {
	Query      string                   `json:"query" validate:"required"`
	Parameters []interface{}            `json:"parameters"`
	Options    map[string]interface{}   `json:"options"`
}

// RawQueryResponse는 원시 쿼리 실행 응답 DTO입니다
type RawQueryResponse struct {
	Results interface{} `json:"results"`
}

// RawQueryTypedRequest는 타입이 있는 원시 쿼리 실행 요청 DTO입니다
type RawQueryTypedRequest struct {
	Query      string                 `json:"query" validate:"required"`
	Parameters []interface{}          `json:"parameters"`
	ResultType string                 `json:"result_type" validate:"required"`
}

// RawQueryTypedResponse는 타입이 있는 원시 쿼리 실행 응답 DTO입니다
type RawQueryTypedResponse struct {
	Results interface{} `json:"results"`
}

// HealthCheckResponse는 헬스 체크 응답 DTO입니다
type HealthCheckResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// DatabaseHealthRequest는 데이터베이스 헬스 체크 요청 DTO입니다
type DatabaseHealthRequest struct {
	DatabaseType string `json:"database_type" validate:"required"`
}

// DatabaseHealthResponse는 데이터베이스 헬스 체크 응답 DTO입니다
type DatabaseHealthResponse struct {
	Status       string    `json:"status"`
	DatabaseType string    `json:"database_type"`
	ResponseTime int64     `json:"response_time_ms"`
	Timestamp    time.Time `json:"timestamp"`
}

// MetricsResponse는 메트릭스 응답 DTO입니다
type MetricsResponse struct {
	Requests      int64              `json:"requests"`
	Errors        int64              `json:"errors"`
	Latency       float64            `json:"latency_ms"`
	Uptime        int64              `json:"uptime_seconds"`
	DatabaseStats map[string]DBStats `json:"database_stats"`
}

// DBStats는 데이터베이스 통계 DTO입니다
type DBStats struct {
	Connections int64   `json:"connections"`
	Operations  int64   `json:"operations"`
	AvgLatency  float64 `json:"avg_latency_ms"`
}

// DatabaseStatsRequest는 데이터베이스 통계 요청 DTO입니다
type DatabaseStatsRequest struct {
	DatabaseType string `json:"database_type" validate:"required"`
}

// DatabaseStatsResponse는 데이터베이스 통계 응답 DTO입니다
type DatabaseStatsResponse struct {
	DatabaseType    string  `json:"database_type"`
	Collections     int     `json:"collections"`
	TotalDocuments  int64   `json:"total_documents"`
	TotalSize       int64   `json:"total_size_bytes"`
	AvgDocumentSize float64 `json:"avg_document_size_bytes"`
}

// CollectionStatsRequest는 컬렉션 통계 요청 DTO입니다
type CollectionStatsRequest struct {
	Collection string `json:"collection" validate:"required"`
}

// CollectionStatsResponse는 컬렉션 통계 응답 DTO입니다
type CollectionStatsResponse struct {
	Collection      string  `json:"collection"`
	DocumentCount   int64   `json:"document_count"`
	Size            int64   `json:"size_bytes"`
	AvgDocumentSize float64 `json:"avg_document_size_bytes"`
	IndexCount      int     `json:"index_count"`
}

// APIResponse는 공통 API 응답 래퍼입니다
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// APIError는 API 오류 응답입니다
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
