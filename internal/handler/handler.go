package handler

import (
	"net/http"

	"github.com/YouSangSon/database-service/internal/models"
	"github.com/YouSangSon/database-service/internal/service"
	"github.com/gin-gonic/gin"
)

// Handler는 HTTP 핸들러입니다
type Handler struct {
	service *service.Service
}

// NewHandler는 새로운 Handler 인스턴스를 생성합니다
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Create는 새로운 문서를 생성합니다
func (h *Handler) Create(c *gin.Context) {
	var req models.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	resp, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Read는 ID로 문서를 조회합니다
func (h *Handler) Read(c *gin.Context) {
	collection := c.Param("collection")
	id := c.Param("id")

	req := &models.ReadRequest{
		Collection: collection,
		ID:         id,
	}

	doc, err := h.service.Read(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// Update는 기존 문서를 업데이트합니다
func (h *Handler) Update(c *gin.Context) {
	collection := c.Param("collection")
	id := c.Param("id")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	req := &models.UpdateRequest{
		Collection: collection,
		ID:         id,
		Data:       data,
	}

	if err := h.service.Update(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document updated successfully",
	})
}

// Delete는 문서를 삭제합니다
func (h *Handler) Delete(c *gin.Context) {
	collection := c.Param("collection")
	id := c.Param("id")

	req := &models.DeleteRequest{
		Collection: collection,
		ID:         id,
	}

	if err := h.service.Delete(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document deleted successfully",
	})
}

// List는 컬렉션의 문서 목록을 조회합니다
func (h *Handler) List(c *gin.Context) {
	collection := c.Param("collection")

	req := &models.ListRequest{
		Collection: collection,
		Filter:     make(map[string]interface{}),
	}

	// 쿼리 파라미터에서 필터 추출
	if filter := c.Query("filter"); filter != "" {
		// 여기서는 간단하게 처리, 실제로는 JSON 파싱 필요
		req.Filter = make(map[string]interface{})
	}

	resp, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HealthCheck는 서비스 상태를 확인합니다
func (h *Handler) HealthCheck(c *gin.Context) {
	if err := h.service.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"healthy": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"healthy": true,
		"message": "Service is healthy",
	})
}
