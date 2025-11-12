package handler

import (
	"fmt"
	"net/http"

	"github.com/YouSangSon/database-service/internal/application/dto"
	"github.com/YouSangSon/database-service/internal/application/usecase"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DocumentHandler는 문서 관련 HTTP 핸들러입니다
type DocumentHandler struct {
	documentUC *usecase.DocumentUseCase
}

// NewDocumentHandler는 새로운 DocumentHandler를 생성합니다
func NewDocumentHandler(documentUC *usecase.DocumentUseCase) *DocumentHandler {
	return &DocumentHandler{
		documentUC: documentUC,
	}
}

// Create godoc
// @Summary      Create a new document
// @Description  Create a new document in the specified collection
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateDocumentRequest  true  "Document creation request"
// @Success      201      {object}  dto.CreateDocumentResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /api/v1/documents [post]
func (h *DocumentHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	resp, err := h.documentUC.CreateDocument(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to create document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create document",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetByID godoc
// @Summary      Get document by ID
// @Description  Retrieve a document by collection and ID
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        collection  path      string  true  "Collection name"
// @Param        id          path      string  true  "Document ID"
// @Success      200         {object}  dto.GetDocumentResponse
// @Failure      404         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Router       /api/v1/documents/{collection}/{id} [get]
func (h *DocumentHandler) GetByID(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	id := c.Param("id")

	req := &dto.GetDocumentRequest{
		Collection: collection,
		ID:         id,
	}

	resp, err := h.documentUC.GetDocument(ctx, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "document not found" {
			statusCode = http.StatusNotFound
		}
		logger.Error(ctx, "failed to get document", zap.Error(err))
		c.JSON(statusCode, ErrorResponse{
			Error:   "Failed to get document",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Update godoc
// @Summary      Update a document
// @Description  Update a document by collection and ID
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        collection  path      string                     true  "Collection name"
// @Param        id          path      string                     true  "Document ID"
// @Param        request     body      dto.UpdateDocumentRequest  true  "Document update request"
// @Success      200         {object}  dto.UpdateDocumentResponse
// @Failure      400         {object}  ErrorResponse
// @Failure      404         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Router       /api/v1/documents/{collection}/{id} [put]
func (h *DocumentHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	id := c.Param("id")

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	req := &dto.UpdateDocumentRequest{
		Collection: collection,
		ID:         id,
		Data:       updateData,
	}

	resp, err := h.documentUC.UpdateDocument(ctx, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "document not found" {
			statusCode = http.StatusNotFound
		}
		logger.Error(ctx, "failed to update document", zap.Error(err))
		c.JSON(statusCode, ErrorResponse{
			Error:   "Failed to update document",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Delete godoc
// @Summary      Delete a document
// @Description  Delete a document by collection and ID
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        collection  path      string  true  "Collection name"
// @Param        id          path      string  true  "Document ID"
// @Success      204         "No Content"
// @Failure      404         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Router       /api/v1/documents/{collection}/{id} [delete]
func (h *DocumentHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")
	id := c.Param("id")

	req := &dto.DeleteDocumentRequest{
		Collection: collection,
		ID:         id,
	}

	if err := h.documentUC.DeleteDocument(ctx, req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "document not found" {
			statusCode = http.StatusNotFound
		}
		logger.Error(ctx, "failed to delete document", zap.Error(err))
		c.JSON(statusCode, ErrorResponse{
			Error:   "Failed to delete document",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// List godoc
// @Summary      List documents
// @Description  List documents in a collection with pagination
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        collection  path      string  true   "Collection name"
// @Param        limit       query     int     false  "Limit (default 10)"
// @Param        offset      query     int     false  "Offset (default 0)"
// @Param        sort        query     string  false  "Sort field (e.g., created_at:-1)"
// @Success      200         {object}  dto.ListDocumentsResponse
// @Failure      400         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Router       /api/v1/documents/{collection} [get]
func (h *DocumentHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.ListDocumentsRequest
	req.Collection = collection
	req.Limit = 10 // default
	req.Offset = 0  // default

	if limit, ok := c.GetQuery("limit"); ok {
		if l, err := parseInt(limit); err == nil {
			req.Limit = l
		}
	}

	if offset, ok := c.GetQuery("offset"); ok {
		if o, err := parseInt(offset); err == nil {
			req.Offset = o
		}
	}

	if sort, ok := c.GetQuery("sort"); ok {
		req.Sort = sort
	}

	resp, err := h.documentUC.ListDocuments(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to list documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list documents",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Aggregate godoc
// @Summary      Aggregate documents
// @Description  Run aggregation pipeline on a collection
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        collection  path      string                      true  "Collection name"
// @Param        request     body      dto.AggregateDocumentRequest  true  "Aggregation request"
// @Success      200         {object}  dto.AggregateDocumentResponse
// @Failure      400         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Router       /api/v1/documents/{collection}/aggregate [post]
func (h *DocumentHandler) Aggregate(c *gin.Context) {
	ctx := c.Request.Context()

	collection := c.Param("collection")

	var req dto.AggregateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	req.Collection = collection

	resp, err := h.documentUC.AggregateDocuments(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to aggregate documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to aggregate documents",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ExecuteRawQuery godoc
// @Summary      Execute raw query
// @Description  Execute a raw database query (MongoDB RunCommand or SQL)
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RawQueryRequest  true  "Raw query request"
// @Success      200      {object}  dto.RawQueryResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /api/v1/documents/raw-query [post]
func (h *DocumentHandler) ExecuteRawQuery(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.RawQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	resp, err := h.documentUC.ExecuteRawQuery(ctx, &req)
	if err != nil {
		logger.Error(ctx, "failed to execute raw query", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to execute raw query",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Helper functions

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// ErrorResponse는 에러 응답 구조체입니다
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
