package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/cassandra"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/elasticsearch"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/mongodb"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/mysql"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/postgresql"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/vitess"
	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/gocql/gocql"
	"go.mongodb.org/mongo-driver/mongo"
)

// RepositoryManager manages all database repositories
type RepositoryManager struct {
	mongoRepo         repository.DocumentRepository
	postgresRepo      repository.DocumentRepository
	mysqlRepo         repository.DocumentRepository
	cassandraRepo     repository.DocumentRepository
	elasticsearchRepo repository.DocumentRepository
	vitessRepo        repository.DocumentRepository

	mu sync.RWMutex
}

// NewRepositoryManager creates a new RepositoryManager
func NewRepositoryManager() *RepositoryManager {
	return &RepositoryManager{}
}

// InitializeMongoDB initializes MongoDB repository
func (rm *RepositoryManager) InitializeMongoDB(ctx context.Context, client *mongo.Client, database string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	db := client.Database(database)
	rm.mongoRepo = mongodb.NewMongoDocumentRepository(db)
	return nil
}

// InitializePostgreSQL initializes PostgreSQL repository
func (rm *RepositoryManager) InitializePostgreSQL(ctx context.Context, db *sql.DB) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.postgresRepo = postgresql.NewPostgreSQLRepository(db)
	return nil
}

// InitializeMySQL initializes MySQL repository
func (rm *RepositoryManager) InitializeMySQL(ctx context.Context, db *sql.DB) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.mysqlRepo = mysql.NewMySQLRepository(db)
	return nil
}

// InitializeCassandra initializes Cassandra repository
func (rm *RepositoryManager) InitializeCassandra(ctx context.Context, session *gocql.Session, keyspace string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.cassandraRepo = cassandra.NewCassandraRepository(session, keyspace)
	return nil
}

// InitializeElasticsearch initializes Elasticsearch repository
func (rm *RepositoryManager) InitializeElasticsearch(ctx context.Context, client *es.Client) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.elasticsearchRepo = elasticsearch.NewElasticsearchRepository(client)
	return nil
}

// InitializeVitess initializes Vitess repository
func (rm *RepositoryManager) InitializeVitess(ctx context.Context, db *sql.DB) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.vitessRepo = vitess.NewVitessRepository(db)
	return nil
}

// GetRepository returns the appropriate repository based on database type
func (rm *RepositoryManager) GetRepository(dbType string) (repository.DocumentRepository, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	switch dbType {
	case "mongodb":
		if rm.mongoRepo == nil {
			return nil, fmt.Errorf("MongoDB repository not initialized")
		}
		return rm.mongoRepo, nil
	case "postgresql":
		if rm.postgresRepo == nil {
			return nil, fmt.Errorf("PostgreSQL repository not initialized")
		}
		return rm.postgresRepo, nil
	case "mysql":
		if rm.mysqlRepo == nil {
			return nil, fmt.Errorf("MySQL repository not initialized")
		}
		return rm.mysqlRepo, nil
	case "cassandra":
		if rm.cassandraRepo == nil {
			return nil, fmt.Errorf("Cassandra repository not initialized")
		}
		return rm.cassandraRepo, nil
	case "elasticsearch":
		if rm.elasticsearchRepo == nil {
			return nil, fmt.Errorf("Elasticsearch repository not initialized")
		}
		return rm.elasticsearchRepo, nil
	case "vitess":
		if rm.vitessRepo == nil {
			return nil, fmt.Errorf("Vitess repository not initialized")
		}
		return rm.vitessRepo, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// Close closes all database connections
func (rm *RepositoryManager) Close() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Close connections if needed
	// Most repositories handle their own cleanup
	return nil
}
