package vitess

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

// VitessRepository는 Vitess 기반 문서 저장소입니다
type VitessRepository struct {
	db       *sql.DB
	keyspace string
	metrics  *metrics.Metrics
}

// Config는 Vitess 설정입니다
type Config struct {
	Host            string
	Port            int
	Keyspace        string
	Username        string
	Password        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewVitessRepository는 새로운 Vitess 저장소를 생성합니다
func NewVitessRepository(cfg *Config) (repository.DocumentRepository, error) {
	// Vitess는 MySQL 프로토콜 사용
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Keyspace,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open vitess connection: %w", err)
	}

	// 연결 풀 설정
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	// 연결 확인
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping vitess: %w", err)
	}

	repo := &VitessRepository{
		db:       db,
		keyspace: cfg.Keyspace,
		metrics:  metrics.GetMetrics(),
	}

	// 테이블 초기화
	if err := repo.initTables(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}

	logger.Info(context.Background(), "vitess repository initialized",
		logger.Field("keyspace", cfg.Keyspace),
		logger.Field("host", cfg.Host),
	)

	return repo, nil
}

// initTables는 필요한 테이블을 생성합니다
func (r *VitessRepository) initTables(ctx context.Context) error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS documents (
			id VARCHAR(255) PRIMARY KEY,
			collection VARCHAR(255) NOT NULL,
			data JSON NOT NULL,
			version INT NOT NULL DEFAULT 0,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			INDEX idx_collection (collection),
			INDEX idx_created_at (created_at),
			INDEX idx_updated_at (updated_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	_, err := r.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create documents table: %w", err)
	}

	return nil
}

// Save는 문서를 저장합니다
func (r *VitessRepository) Save(ctx context.Context, doc *entity.Document) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("save", doc.Collection(), "success", duration)
		logger.Debug(ctx, "document saved",
			logger.Collection(doc.Collection()),
			logger.Duration(duration),
		)
	}()

	// JSON으로 변환
	dataJSON, err := json.Marshal(doc.Data())
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		INSERT INTO documents (id, collection, data, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		doc.ID(),
		doc.Collection(),
		dataJSON,
		doc.Version(),
		doc.CreatedAt(),
		doc.UpdatedAt(),
	)
	if err != nil {
		r.metrics.RecordDBOperation("save", doc.Collection(), "error", time.Since(start))
		logger.Error(ctx, "failed to save document",
			logger.Collection(doc.Collection()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save document: %w", err)
	}

	// ID 설정 (생성된 경우)
	if doc.ID() == "" {
		id, err := result.LastInsertId()
		if err == nil {
			doc.SetID(fmt.Sprintf("%d", id))
		}
	}

	return nil
}

// FindByID는 ID로 문서를 조회합니다
func (r *VitessRepository) FindByID(ctx context.Context, collection, id string) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find", collection, "success", duration)
	}()

	query := `
		SELECT id, collection, data, version, created_at, updated_at
		FROM documents
		WHERE id = ? AND collection = ?
	`

	var (
		docID      string
		coll       string
		dataJSON   []byte
		version    int
		createdAt  time.Time
		updatedAt  time.Time
	)

	err := r.db.QueryRowContext(ctx, query, id, collection).Scan(
		&docID, &coll, &dataJSON, &version, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		r.metrics.RecordDBOperation("find", collection, "not_found", time.Since(start))
		return nil, entity.ErrDocumentNotFound
	}
	if err != nil {
		r.metrics.RecordDBOperation("find", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// JSON 파싱
	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	doc := entity.ReconstructDocument(docID, coll, data, version, createdAt, updatedAt)
	return doc, nil
}

// Update는 문서를 업데이트합니다 (낙관적 잠금)
func (r *VitessRepository) Update(ctx context.Context, doc *entity.Document) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("update", doc.Collection(), "success", duration)
	}()

	dataJSON, err := json.Marshal(doc.Data())
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		UPDATE documents
		SET data = ?, version = ?, updated_at = ?
		WHERE id = ? AND collection = ? AND version = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		dataJSON,
		doc.Version(),
		doc.UpdatedAt(),
		doc.ID(),
		doc.Collection(),
		doc.Version()-1, // 낙관적 잠금
	)
	if err != nil {
		r.metrics.RecordDBOperation("update", doc.Collection(), "error", time.Since(start))
		logger.Error(ctx, "failed to update document",
			logger.Collection(doc.Collection()),
			logger.DocumentID(doc.ID()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update document: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected == 0 {
		r.metrics.RecordDBOperation("update", doc.Collection(), "conflict", time.Since(start))
		return entity.ErrVersionConflict
	}

	return nil
}

// Delete는 문서를 삭제합니다
func (r *VitessRepository) Delete(ctx context.Context, collection, id string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("delete", collection, "success", duration)
	}()

	query := `DELETE FROM documents WHERE id = ? AND collection = ?`

	result, err := r.db.ExecContext(ctx, query, id, collection)
	if err != nil {
		r.metrics.RecordDBOperation("delete", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to delete document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete document: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected == 0 {
		r.metrics.RecordDBOperation("delete", collection, "not_found", time.Since(start))
		return entity.ErrDocumentNotFound
	}

	return nil
}

// FindAll은 컬렉션의 모든 문서를 조회합니다
func (r *VitessRepository) FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_all", collection, "success", duration)
	}()

	query := `
		SELECT id, collection, data, version, created_at, updated_at
		FROM documents
		WHERE collection = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, collection)
	if err != nil {
		r.metrics.RecordDBOperation("find_all", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find documents",
			logger.Collection(collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}
	defer rows.Close()

	var documents []*entity.Document
	for rows.Next() {
		var (
			id        string
			coll      string
			dataJSON  []byte
			version   int
			createdAt time.Time
			updatedAt time.Time
		)

		if err := rows.Scan(&id, &coll, &dataJSON, &version, &createdAt, &updatedAt); err != nil {
			logger.Warn(ctx, "failed to scan row", zap.Error(err))
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			logger.Warn(ctx, "failed to unmarshal data", zap.Error(err))
			continue
		}

		doc := entity.ReconstructDocument(id, coll, data, version, createdAt, updatedAt)
		documents = append(documents, doc)
	}

	return documents, nil
}

// Count는 문서 개수를 반환합니다
func (r *VitessRepository) Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	query := `SELECT COUNT(*) FROM documents WHERE collection = ?`

	var count int64
	err := r.db.QueryRowContext(ctx, query, collection).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// HealthCheck는 저장소의 상태를 확인합니다
func (r *VitessRepository) HealthCheck(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// Close는 데이터베이스 연결을 종료합니다
func (r *VitessRepository) Close() error {
	return r.db.Close()
}

// WithTransaction은 트랜잭션 내에서 함수를 실행합니다
func (r *VitessRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	start := time.Now()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error(ctx, "failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// 트랜잭션 컨텍스트 생성
	txCtx := context.WithValue(ctx, "tx", tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			logger.Error(ctx, "failed to rollback transaction", zap.Error(rbErr))
		}
		logger.Error(ctx, "transaction failed", zap.Error(err))
		return fmt.Errorf("transaction failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info(ctx, "transaction completed successfully",
		logger.Duration(time.Since(start)),
	)

	return nil
}

// 나머지 메서드는 기본 구현 제공 (필요시 확장)
func (r *VitessRepository) SaveMany(ctx context.Context, docs []*entity.Document) error {
	return fmt.Errorf("SaveMany not yet implemented for Vitess")
}

func (r *VitessRepository) UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	return 0, fmt.Errorf("UpdateMany not yet implemented for Vitess")
}

func (r *VitessRepository) FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error) {
	return nil, fmt.Errorf("FindAndUpdate not yet implemented for Vitess")
}

func (r *VitessRepository) FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error) {
	return nil, fmt.Errorf("FindOneAndReplace not yet implemented for Vitess")
}

func (r *VitessRepository) FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error) {
	return nil, fmt.Errorf("FindOneAndDelete not yet implemented for Vitess")
}

func (r *VitessRepository) DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	return 0, fmt.Errorf("DeleteMany not yet implemented for Vitess")
}

func (r *VitessRepository) Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error) {
	return "", fmt.Errorf("Upsert not yet implemented for Vitess")
}

func (r *VitessRepository) FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *repository.FindOptions) ([]*entity.Document, error) {
	return nil, fmt.Errorf("FindWithOptions not yet implemented for Vitess")
}

func (r *VitessRepository) Replace(ctx context.Context, collection, id string, replacement *entity.Document) error {
	return fmt.Errorf("Replace not yet implemented for Vitess")
}

func (r *VitessRepository) Aggregate(ctx context.Context, collection string, pipeline []interface{}) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("Aggregate not yet implemented for Vitess")
}

func (r *VitessRepository) Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("Distinct not yet implemented for Vitess")
}

func (r *VitessRepository) EstimatedDocumentCount(ctx context.Context, collection string) (int64, error) {
	return 0, fmt.Errorf("EstimatedDocumentCount not yet implemented for Vitess")
}

func (r *VitessRepository) BulkWrite(ctx context.Context, operations []*repository.BulkOperation) (*repository.BulkResult, error) {
	return nil, fmt.Errorf("BulkWrite not yet implemented for Vitess")
}

func (r *VitessRepository) CreateIndex(ctx context.Context, collection string, model repository.IndexModel) (string, error) {
	return "", fmt.Errorf("CreateIndex not yet implemented for Vitess")
}

func (r *VitessRepository) CreateIndexes(ctx context.Context, collection string, models []repository.IndexModel) ([]string, error) {
	return nil, fmt.Errorf("CreateIndexes not yet implemented for Vitess")
}

func (r *VitessRepository) DropIndex(ctx context.Context, collection, indexName string) error {
	return fmt.Errorf("DropIndex not yet implemented for Vitess")
}

func (r *VitessRepository) ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("ListIndexes not yet implemented for Vitess")
}

func (r *VitessRepository) CreateCollection(ctx context.Context, name string) error {
	return fmt.Errorf("CreateCollection not yet implemented for Vitess")
}

func (r *VitessRepository) DropCollection(ctx context.Context, name string) error {
	return fmt.Errorf("DropCollection not yet implemented for Vitess")
}

func (r *VitessRepository) RenameCollection(ctx context.Context, oldName, newName string) error {
	return fmt.Errorf("RenameCollection not yet implemented for Vitess")
}

func (r *VitessRepository) ListCollections(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("ListCollections not yet implemented for Vitess")
}

func (r *VitessRepository) CollectionExists(ctx context.Context, name string) (bool, error) {
	return false, fmt.Errorf("CollectionExists not yet implemented for Vitess")
}

func (r *VitessRepository) Watch(ctx context.Context, collection string, pipeline []interface{}) (interface{}, error) {
	return nil, fmt.Errorf("Watch not yet implemented for Vitess")
}
