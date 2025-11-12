package benchmark

import (
	"context"
	"testing"

	"github.com/YouSangSon/database-service/internal/application/usecase"
	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
)

// MockRepository는 벤치마크를 위한 모의 저장소입니다
type MockRepository struct{}

func (m *MockRepository) Save(ctx context.Context, doc *entity.Document) error {
	return nil
}

func (m *MockRepository) FindByID(ctx context.Context, collection, id string) (*entity.Document, error) {
	return entity.NewDocument(collection, map[string]interface{}{"test": "data"}), nil
}

func (m *MockRepository) FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error) {
	docs := make([]*entity.Document, 100)
	for i := range docs {
		docs[i] = entity.NewDocument(collection, map[string]interface{}{"index": i})
	}
	return docs, nil
}

func (m *MockRepository) Update(ctx context.Context, doc *entity.Document) error {
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, collection, id string) error {
	return nil
}

func (m *MockRepository) Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	return 100, nil
}

func (m *MockRepository) HealthCheck(ctx context.Context) error {
	return nil
}

// MockCache는 벤치마크를 위한 모의 캐시입니다
type MockCache struct{}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
	return nil, repository.ErrCacheNotFound
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// BenchmarkCreateDocument는 문서 생성 성능을 측정합니다
func BenchmarkCreateDocument(b *testing.B) {
	mockRepo := &MockRepository{}
	mockCache := &MockCache{}
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "benchmark_test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"index":       i,
			"description": "Benchmark test document",
		}
		uc.Create(ctx, collection, data)
	}
}

// BenchmarkGetDocument는 문서 조회 성능을 측정합니다
func BenchmarkGetDocument(b *testing.B) {
	mockRepo := &MockRepository{}
	mockCache := &MockCache{}
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "benchmark_test"
	documentID := "test123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.GetByID(ctx, collection, documentID)
	}
}

// BenchmarkUpdateDocument는 문서 업데이트 성능을 측정합니다
func BenchmarkUpdateDocument(b *testing.B) {
	mockRepo := &MockRepository{}
	mockCache := &MockCache{}
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "benchmark_test"
	documentID := "test123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"index":   i,
			"updated": true,
		}
		uc.Update(ctx, collection, documentID, data)
	}
}

// BenchmarkDeleteDocument는 문서 삭제 성능을 측정합니다
func BenchmarkDeleteDocument(b *testing.B) {
	mockRepo := &MockRepository{}
	mockCache := &MockCache{}
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "benchmark_test"
	documentID := "test123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.Delete(ctx, collection, documentID)
	}
}

// BenchmarkListDocuments는 문서 목록 조회 성능을 측정합니다
func BenchmarkListDocuments(b *testing.B) {
	mockRepo := &MockRepository{}
	mockCache := &MockCache{}
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "benchmark_test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.List(ctx, collection, 100, 0)
	}
}

// BenchmarkConcurrentOperations는 동시 작업 성능을 측정합니다
func BenchmarkConcurrentOperations(b *testing.B) {
	mockRepo := &MockRepository{}
	mockCache := &MockCache{}
	uc := usecase.NewDocumentUseCase(mockRepo, mockCache)

	ctx := context.Background()
	collection := "benchmark_test"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data := map[string]interface{}{
				"concurrent": true,
			}
			uc.Create(ctx, collection, data)
		}
	})
}
