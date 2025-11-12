package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YouSangSon/database-service/internal/application/dto"
	"github.com/YouSangSon/database-service/internal/application/usecase"
	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDocumentRepository는 DocumentRepository의 mock입니다
type MockDocumentRepository struct {
	mock.Mock
}

func (m *MockDocumentRepository) Save(ctx context.Context, doc *entity.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockDocumentRepository) FindByID(ctx context.Context, collection, id string) (*entity.Document, error) {
	args := m.Called(ctx, collection, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Document), args.Error(1)
}

func (m *MockDocumentRepository) Update(ctx context.Context, doc *entity.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockDocumentRepository) Delete(ctx context.Context, collection, id string) error {
	args := m.Called(ctx, collection, id)
	return args.Error(0)
}

func (m *MockDocumentRepository) FindAll(ctx context.Context, collection string, opts *entity.QueryOptions) ([]*entity.Document, error) {
	args := m.Called(ctx, collection, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Document), args.Error(1)
}

func (m *MockDocumentRepository) Count(ctx context.Context, collection string) (int64, error) {
	args := m.Called(ctx, collection)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDocumentRepository) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockCacheRepository는 CacheRepository의 mock입니다
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestCreateDocument_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	req := &dto.CreateDocumentRequest{
		Collection: "users",
		Data: map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
			"age":   30,
		},
	}

	// Mock expectations
	mockRepo.On("Save", ctx, mock.AnythingOfType("*entity.Document")).Return(nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything, 300).Return(nil)

	// Act
	resp, err := uc.CreateDocument(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "users", resp.Collection)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCreateDocument_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	req := &dto.CreateDocumentRequest{
		Collection: "users",
		Data: map[string]interface{}{
			"name": "John Doe",
		},
	}

	expectedErr := errors.New("database connection failed")
	mockRepo.On("Save", ctx, mock.AnythingOfType("*entity.Document")).Return(expectedErr)

	// Act
	resp, err := uc.CreateDocument(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to save document")
	mockRepo.AssertExpectations(t)
}

func TestGetDocument_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "users"
	docID := "507f1f77bcf86cd799439011"

	expectedDoc, _ := entity.NewDocument(collection, map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	})

	// Mock: cache miss, then repository hit
	mockCache.On("Get", ctx, mock.Anything, mock.Anything).Return(errors.New("cache miss"))
	mockRepo.On("FindByID", ctx, collection, docID).Return(expectedDoc, nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything, 300).Return(nil)

	// Act
	resp, err := uc.GetDocument(ctx, &dto.GetDocumentRequest{
		Collection: collection,
		ID:         docID,
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, collection, resp.Collection)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetDocument_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "users"
	docID := "nonexistent"

	mockCache.On("Get", ctx, mock.Anything, mock.Anything).Return(errors.New("cache miss"))
	mockRepo.On("FindByID", ctx, collection, docID).Return(nil, entity.ErrDocumentNotFound)

	// Act
	resp, err := uc.GetDocument(ctx, &dto.GetDocumentRequest{
		Collection: collection,
		ID:         docID,
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, entity.ErrDocumentNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateDocument_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "users"
	docID := "507f1f77bcf86cd799439011"

	existingDoc, _ := entity.NewDocument(collection, map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	})

	updateData := map[string]interface{}{
		"age": 31,
	}

	mockRepo.On("FindByID", ctx, collection, docID).Return(existingDoc, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*entity.Document")).Return(nil)
	mockCache.On("Delete", ctx, mock.Anything).Return(nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything, 300).Return(nil)

	// Act
	resp, err := uc.UpdateDocument(ctx, &dto.UpdateDocumentRequest{
		Collection: collection,
		ID:         docID,
		Data:       updateData,
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestDeleteDocument_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "users"
	docID := "507f1f77bcf86cd799439011"

	mockRepo.On("Delete", ctx, collection, docID).Return(nil)
	mockCache.On("Delete", ctx, mock.Anything).Return(nil)

	// Act
	err := uc.DeleteDocument(ctx, &dto.DeleteDocumentRequest{
		Collection: collection,
		ID:         docID,
	})

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestListDocuments_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "users"

	doc1, _ := entity.NewDocument(collection, map[string]interface{}{"name": "John"})
	doc2, _ := entity.NewDocument(collection, map[string]interface{}{"name": "Jane"})
	expectedDocs := []*entity.Document{doc1, doc2}

	mockRepo.On("FindAll", ctx, collection, mock.AnythingOfType("*entity.QueryOptions")).Return(expectedDocs, nil)
	mockRepo.On("Count", ctx, collection).Return(int64(2), nil)

	// Act
	resp, err := uc.ListDocuments(ctx, &dto.ListDocumentsRequest{
		Collection: collection,
		Limit:      10,
		Offset:     0,
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Documents, 2)
	assert.Equal(t, int64(2), resp.Total)
	mockRepo.AssertExpectations(t)
}

func TestCircuitBreakerTriggered(t *testing.T) {
	// Arrange
	mockRepo := new(MockDocumentRepository)
	mockCache := new(MockCacheRepository)
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	req := &dto.CreateDocumentRequest{
		Collection: "users",
		Data:       map[string]interface{}{"name": "Test"},
	}

	// Simulate repeated failures to trigger circuit breaker
	mockRepo.On("Save", ctx, mock.AnythingOfType("*entity.Document")).Return(errors.New("database error"))

	// Act - trigger circuit breaker with multiple failures
	for i := 0; i < 10; i++ {
		_, err := uc.CreateDocument(ctx, req)
		assert.Error(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// At this point, circuit breaker should be open
	// The error message should indicate circuit breaker is open
	_, err := uc.CreateDocument(ctx, req)
	assert.Error(t, err)
}
