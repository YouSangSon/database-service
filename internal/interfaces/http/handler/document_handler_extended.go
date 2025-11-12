package handler

import (
	"net/http"

	"github.com/YouSangSon/database-service/internal/application/dto"
	"github.com/YouSangSon/database-service/internal/application/usecase"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DocumentHandlerExtended는 확장된 문서 관련 HTTP 핸들러입니다
type DocumentHandlerExtended struct {
	documentUC *usecase.DocumentUseCase
}

// NewDocumentHandlerExtended는 새로운 DocumentHandlerExtended를 생성합니다
func NewDocumentHandlerExtended(documentUC *usecase.DocumentUseCase) *DocumentHandlerExtended {
	return &DocumentHandlerExtended{
		documentUC: documentUC,
	}
}

// Replace replaces a document
func (h *DocumentHandlerExtended) Replace(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	id := c.Param("id")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req := &dto.ReplaceDocumentRequest{
		Collection: collection,
		ID:         id,
		Data:       data,
	}

	resp, err := h.documentUC.ReplaceDocument(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to replace document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "REPLACE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Document replaced successfully",
	})
}

// Search searches for documents
func (h *DocumentHandlerExtended) Search(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.SearchDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.SearchDocuments(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to search documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "SEARCH_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// Count counts documents
func (h *DocumentHandlerExtended) Count(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.CountDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.CountDocuments(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to count documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "COUNT_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// EstimatedCount returns estimated document count
func (h *DocumentHandlerExtended) EstimatedCount(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	req := &dto.EstimatedCountRequest{
		Collection: collection,
	}

	resp, err := h.documentUC.EstimatedCount(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to get estimated count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "ESTIMATED_COUNT_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// FindAndUpdate finds and updates a document
func (h *DocumentHandlerExtended) FindAndUpdate(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	id := c.Param("id")

	var update map[string]interface{}
	if err := c.ShouldBindJSON(&update); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req := &dto.FindAndUpdateRequest{
		Collection: collection,
		ID:         id,
		Update:     update,
	}

	resp, err := h.documentUC.FindAndUpdate(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to find and update document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "FIND_AND_UPDATE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Document found and updated successfully",
	})
}

// FindAndReplace finds and replaces a document
func (h *DocumentHandlerExtended) FindAndReplace(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	id := c.Param("id")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req := &dto.FindAndReplaceRequest{
		Collection: collection,
		ID:         id,
		Data:       data,
	}

	resp, err := h.documentUC.FindAndReplace(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to find and replace document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "FIND_AND_REPLACE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Document found and replaced successfully",
	})
}

// FindAndDelete finds and deletes a document
func (h *DocumentHandlerExtended) FindAndDelete(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	id := c.Param("id")

	req := &dto.FindAndDeleteRequest{
		Collection: collection,
		ID:         id,
	}

	resp, err := h.documentUC.FindAndDelete(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to find and delete document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "FIND_AND_DELETE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Document found and deleted successfully",
	})
}

// Upsert upserts a document
func (h *DocumentHandlerExtended) Upsert(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.Upsert(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to upsert document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "UPSERT_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Document upserted successfully",
	})
}

// Distinct retrieves distinct values
func (h *DocumentHandlerExtended) Distinct(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.DistinctRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.Distinct(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to get distinct values", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "DISTINCT_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// BulkInsert inserts multiple documents
func (h *DocumentHandlerExtended) BulkInsert(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.BulkInsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	resp, err := h.documentUC.BulkInsert(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to bulk insert documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "BULK_INSERT_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Documents inserted successfully",
	})
}

// UpdateMany updates multiple documents
func (h *DocumentHandlerExtended) UpdateMany(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.UpdateManyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.UpdateMany(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to update many documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "UPDATE_MANY_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Documents updated successfully",
	})
}

// DeleteMany deletes multiple documents
func (h *DocumentHandlerExtended) DeleteMany(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.DeleteManyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.DeleteMany(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to delete many documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "DELETE_MANY_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Documents deleted successfully",
	})
}

// BulkWrite executes multiple write operations
func (h *DocumentHandlerExtended) BulkWrite(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.BulkWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	resp, err := h.documentUC.BulkWrite(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to execute bulk write", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "BULK_WRITE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Bulk write executed successfully",
	})
}

// CreateIndex creates an index
func (h *DocumentHandlerExtended) CreateIndex(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.CreateIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.CreateIndex(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to create index", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "CREATE_INDEX_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Index created successfully",
	})
}

// CreateIndexes creates multiple indexes
func (h *DocumentHandlerExtended) CreateIndexes(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.CreateIndexesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.CreateIndexes(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to create indexes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "CREATE_INDEXES_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Indexes created successfully",
	})
}

// DropIndex drops an index
func (h *DocumentHandlerExtended) DropIndex(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	indexName := c.Param("index_name")

	req := &dto.DropIndexRequest{
		Collection: collection,
		IndexName:  indexName,
	}

	resp, err := h.documentUC.DropIndex(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to drop index", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "DROP_INDEX_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Index dropped successfully",
	})
}

// ListIndexes lists indexes
func (h *DocumentHandlerExtended) ListIndexes(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	req := &dto.ListIndexesRequest{
		Collection: collection,
	}

	resp, err := h.documentUC.ListIndexes(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to list indexes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "LIST_INDEXES_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// CreateCollection creates a collection
func (h *DocumentHandlerExtended) CreateCollection(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	resp, err := h.documentUC.CreateCollection(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to create collection", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "CREATE_COLLECTION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Collection created successfully",
	})
}

// DropCollection drops a collection
func (h *DocumentHandlerExtended) DropCollection(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	req := &dto.DropCollectionRequest{
		Collection: collection,
	}

	resp, err := h.documentUC.DropCollection(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to drop collection", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "DROP_COLLECTION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Collection dropped successfully",
	})
}

// RenameCollection renames a collection
func (h *DocumentHandlerExtended) RenameCollection(c *gin.Context) {
	ctx := c.Request.Context()

	oldName := c.Param("old_name")

	var data struct {
		NewName string `json:"new_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	req := &dto.RenameCollectionRequest{
		OldName: oldName,
		NewName: data.NewName,
	}

	resp, err := h.documentUC.RenameCollection(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to rename collection", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "RENAME_COLLECTION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Collection renamed successfully",
	})
}

// ListCollections lists all collections
func (h *DocumentHandlerExtended) ListCollections(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.ListCollectionsRequest
	// Optional filter from query or body
	c.ShouldBindJSON(&req)

	resp, err := h.documentUC.ListCollections(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to list collections", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "LIST_COLLECTIONS_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// CollectionExists checks if a collection exists
func (h *DocumentHandlerExtended) CollectionExists(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	req := &dto.CollectionExistsRequest{
		Collection: collection,
	}

	resp, err := h.documentUC.CollectionExists(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to check collection existence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "COLLECTION_EXISTS_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// ExecuteTransaction executes a transaction
func (h *DocumentHandlerExtended) ExecuteTransaction(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.ExecuteTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	resp, err := h.documentUC.ExecuteTransaction(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to execute transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "TRANSACTION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Transaction executed successfully",
	})
}

// ExecuteRaw executes a raw query
func (h *DocumentHandlerExtended) ExecuteRaw(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.RawQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	resp, err := h.documentUC.ExecuteRawQuery(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to execute raw query", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "RAW_QUERY_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// ExecuteRawTyped executes a raw query with typed result
func (h *DocumentHandlerExtended) ExecuteRawTyped(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.RawQueryTypedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	resp, err := h.documentUC.ExecuteRawQueryTyped(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to execute raw query", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "RAW_QUERY_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// Health performs a health check
func (h *DocumentHandlerExtended) Health(c *gin.Context) {
	ctx := c.Request.Context()

	resp, err := h.documentUC.HealthCheck(ctx)
	if err != nil {
		logger.Error(ctx, "health check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "HEALTH_CHECK_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	statusCode := http.StatusOK
	if resp.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// DatabaseHealth checks database health
func (h *DocumentHandlerExtended) DatabaseHealth(c *gin.Context) {
	ctx := c.Request.Context()

	dbType := c.Param("db_type")

	req := &dto.DatabaseHealthRequest{
		DatabaseType: dbType,
	}

	resp, err := h.documentUC.DatabaseHealth(ctx, req)
	if err != nil {
		logger.Error(ctx, "database health check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "DATABASE_HEALTH_CHECK_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	statusCode := http.StatusOK
	if resp.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// GetMetrics retrieves system metrics
func (h *DocumentHandlerExtended) GetMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	resp, err := h.documentUC.GetMetrics(ctx)
	if err != nil {
		logger.Error(ctx, "failed to get metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "GET_METRICS_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// GetDatabaseStats retrieves database statistics
func (h *DocumentHandlerExtended) GetDatabaseStats(c *gin.Context) {
	ctx := c.Request.Context()

	dbType := c.Param("db_type")

	req := &dto.DatabaseStatsRequest{
		DatabaseType: dbType,
	}

	resp, err := h.documentUC.GetDatabaseStats(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to get database stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "GET_DATABASE_STATS_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// GetCollectionStats retrieves collection statistics
func (h *DocumentHandlerExtended) GetCollectionStats(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	req := &dto.CollectionStatsRequest{
		Collection: collection,
	}

	resp, err := h.documentUC.GetCollectionStats(ctx, req)
	if err != nil {
		logger.Error(ctx, "failed to get collection stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error: &dto.APIError{
				Code:    "GET_COLLECTION_STATS_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    resp,
	})
}
