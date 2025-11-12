// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestMongoDBIntegration은 Testcontainers를 사용한 MongoDB 통합 테스트입니다
func TestMongoDBIntegration(t *testing.T) {
	ctx := context.Background()

	// MongoDB 컨테이너 시작
	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7.0",
			ExposedPorts: []string{"27017/tcp"},
			Env: map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": "test",
				"MONGO_INITDB_ROOT_PASSWORD": "test",
			},
			WaitingFor: wait.ForLog("Waiting for connections"),
		},
		Started: true,
	})
	require.NoError(t, err)
	defer mongoContainer.Terminate(ctx)

	// 컨테이너 호스트 및 포트 가져오기
	host, err := mongoContainer.Host(ctx)
	require.NoError(t, err)

	port, err := mongoContainer.MappedPort(ctx, "27017")
	require.NoError(t, err)

	// MongoDB URI 생성
	mongoURI := "mongodb://test:test@" + host + ":" + port.Port()

	// Repository 초기화
	repo, err := mongodb.NewDocumentRepository(ctx, mongoURI, "testdb", nil)
	require.NoError(t, err)
	defer repo.Close(ctx)

	t.Run("Create and FindByID", func(t *testing.T) {
		// 문서 생성
		doc := entity.NewDocument("test_collection", map[string]interface{}{
			"name":        "Test Document",
			"description": "Integration test",
			"count":       42,
		})

		err := repo.Save(ctx, doc)
		assert.NoError(t, err)
		assert.NotEmpty(t, doc.ID())

		// 조회
		found, err := repo.FindByID(ctx, "test_collection", doc.ID())
		assert.NoError(t, err)
		assert.Equal(t, doc.ID(), found.ID())
		assert.Equal(t, "Test Document", found.Data()["name"])
		assert.Equal(t, float64(42), found.Data()["count"])
	})

	t.Run("Update with Optimistic Locking", func(t *testing.T) {
		// 문서 생성
		doc := entity.NewDocument("test_collection", map[string]interface{}{
			"name": "Original",
		})

		err := repo.Save(ctx, doc)
		require.NoError(t, err)

		// 업데이트
		doc.UpdateData(map[string]interface{}{
			"name": "Updated",
		})

		err = repo.Update(ctx, doc)
		assert.NoError(t, err)

		// 조회 및 검증
		found, err := repo.FindByID(ctx, "test_collection", doc.ID())
		assert.NoError(t, err)
		assert.Equal(t, "Updated", found.Data()["name"])
		assert.Equal(t, 2, found.Version()) // 버전이 증가했는지 확인
	})

	t.Run("Delete", func(t *testing.T) {
		// 문서 생성
		doc := entity.NewDocument("test_collection", map[string]interface{}{
			"temp": "data",
		})

		err := repo.Save(ctx, doc)
		require.NoError(t, err)

		// 삭제
		err = repo.Delete(ctx, "test_collection", doc.ID())
		assert.NoError(t, err)

		// 삭제 확인
		_, err = repo.FindByID(ctx, "test_collection", doc.ID())
		assert.Error(t, err)
		assert.Equal(t, entity.ErrDocumentNotFound, err)
	})

	t.Run("FindAll and Count", func(t *testing.T) {
		// 여러 문서 생성
		for i := 0; i < 5; i++ {
			doc := entity.NewDocument("test_collection", map[string]interface{}{
				"index": i,
				"type":  "test",
			})
			err := repo.Save(ctx, doc)
			require.NoError(t, err)
		}

		// 전체 조회
		docs, err := repo.FindAll(ctx, "test_collection", map[string]interface{}{
			"type": "test",
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(docs), 5)

		// 카운트
		count, err := repo.Count(ctx, "test_collection", map[string]interface{}{
			"type": "test",
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})

	t.Run("HealthCheck", func(t *testing.T) {
		err := repo.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}

// TestMongoDBConcurrency는 동시성 테스트입니다
func TestMongoDBConcurrency(t *testing.T) {
	ctx := context.Background()

	// MongoDB 컨테이너 시작
	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7.0",
			ExposedPorts: []string{"27017/tcp"},
			Env: map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": "test",
				"MONGO_INITDB_ROOT_PASSWORD": "test",
			},
			WaitingFor: wait.ForLog("Waiting for connections"),
		},
		Started: true,
	})
	require.NoError(t, err)
	defer mongoContainer.Terminate(ctx)

	host, err := mongoContainer.Host(ctx)
	require.NoError(t, err)

	port, err := mongoContainer.MappedPort(ctx, "27017")
	require.NoError(t, err)

	mongoURI := "mongodb://test:test@" + host + ":" + port.Port()

	repo, err := mongodb.NewDocumentRepository(ctx, mongoURI, "testdb", nil)
	require.NoError(t, err)
	defer repo.Close(ctx)

	t.Run("Concurrent Writes", func(t *testing.T) {
		const numGoroutines = 10

		done := make(chan bool, numGoroutines)

		// 동시에 문서 생성
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				doc := entity.NewDocument("concurrent_test", map[string]interface{}{
					"goroutine": index,
					"timestamp": time.Now().Unix(),
				})

				err := repo.Save(ctx, doc)
				assert.NoError(t, err)

				done <- true
			}(i)
		}

		// 모든 goroutine 완료 대기
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 결과 확인
		count, err := repo.Count(ctx, "concurrent_test", nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(numGoroutines), count)
	})
}
