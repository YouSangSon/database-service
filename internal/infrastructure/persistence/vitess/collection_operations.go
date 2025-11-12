package vitess

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// CreateCollection은 컬렉션을 생성합니다
// Vitess에서는 실제로 새 테이블을 생성하지 않고, documents 테이블의 collection 필드로 구분합니다
// 따라서 이 메서드는 논리적 검증만 수행합니다
func (r *VitessRepository) CreateCollection(ctx context.Context, name string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("create_collection", name, "success", duration)
	}()

	// 컬렉션이 이미 존재하는지 확인
	exists, err := r.CollectionExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		logger.Warn(ctx, "collection already exists",
			logger.Collection(name),
		)
		return fmt.Errorf("collection %s already exists", name)
	}

	// Vitess에서는 documents 테이블을 공유하므로
	// 실제 테이블 생성은 하지 않고 로그만 남깁니다
	logger.Info(ctx, "collection created (logical)",
		logger.Collection(name),
		logger.Duration(time.Since(start)),
	)

	return nil
}

// DropCollection은 컬렉션을 삭제합니다
// 해당 컬렉션의 모든 문서를 삭제합니다
func (r *VitessRepository) DropCollection(ctx context.Context, name string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("drop_collection", name, "success", duration)
	}()

	// 컬렉션이 존재하는지 확인
	exists, err := r.CollectionExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !exists {
		logger.Warn(ctx, "collection does not exist",
			logger.Collection(name),
		)
		return fmt.Errorf("collection %s does not exist", name)
	}

	// 해당 컬렉션의 모든 문서 삭제
	query := `DELETE FROM documents WHERE collection = ?`

	result, err := r.db.ExecContext(ctx, query, name)
	if err != nil {
		r.metrics.RecordDBOperation("drop_collection", name, "error", time.Since(start))
		logger.Error(ctx, "failed to drop collection",
			logger.Collection(name),
			zap.Error(err),
		)
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	deleted, _ := result.RowsAffected()

	logger.Info(ctx, "collection dropped",
		logger.Collection(name),
		logger.Field("documents_deleted", deleted),
		logger.Duration(time.Since(start)),
	)

	return nil
}

// RenameCollection은 컬렉션 이름을 변경합니다
// 모든 문서의 collection 필드를 업데이트합니다
func (r *VitessRepository) RenameCollection(ctx context.Context, oldName, newName string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("rename_collection", oldName, "success", duration)
	}()

	// 이전 컬렉션이 존재하는지 확인
	exists, err := r.CollectionExists(ctx, oldName)
	if err != nil {
		return fmt.Errorf("failed to check old collection existence: %w", err)
	}

	if !exists {
		logger.Warn(ctx, "old collection does not exist",
			logger.Collection(oldName),
		)
		return fmt.Errorf("collection %s does not exist", oldName)
	}

	// 새 컬렉션 이름이 이미 존재하는지 확인
	newExists, err := r.CollectionExists(ctx, newName)
	if err != nil {
		return fmt.Errorf("failed to check new collection existence: %w", err)
	}

	if newExists {
		logger.Warn(ctx, "new collection name already exists",
			logger.Collection(newName),
		)
		return fmt.Errorf("collection %s already exists", newName)
	}

	// 트랜잭션 시작
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

	// collection 필드 업데이트
	query := `UPDATE documents SET collection = ? WHERE collection = ?`

	result, err := tx.ExecContext(ctx, query, newName, oldName)
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("rename_collection", oldName, "error", time.Since(start))
		logger.Error(ctx, "failed to rename collection",
			logger.Collection(oldName),
			logger.Field("new_name", newName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to rename collection: %w", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	updated, _ := result.RowsAffected()

	logger.Info(ctx, "collection renamed",
		logger.Collection(oldName),
		logger.Field("new_name", newName),
		logger.Field("documents_updated", updated),
		logger.Duration(time.Since(start)),
	)

	return nil
}

// ListCollections는 데이터베이스의 컬렉션 목록을 반환합니다
func (r *VitessRepository) ListCollections(ctx context.Context) ([]string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("list_collections", "all", "success", duration)
	}()

	// DISTINCT를 사용하여 고유한 컬렉션 이름 조회
	query := `SELECT DISTINCT collection FROM documents ORDER BY collection`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.metrics.RecordDBOperation("list_collections", "all", "error", time.Since(start))
		logger.Error(ctx, "failed to list collections", zap.Error(err))
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer rows.Close()

	var collections []string
	for rows.Next() {
		var collection string
		if err := rows.Scan(&collection); err != nil {
			logger.Warn(ctx, "failed to scan collection name", zap.Error(err))
			continue
		}
		collections = append(collections, collection)
	}

	logger.Info(ctx, "collections listed",
		logger.Field("count", len(collections)),
		logger.Duration(time.Since(start)),
	)

	return collections, nil
}

// CollectionExists는 컬렉션이 존재하는지 확인합니다
func (r *VitessRepository) CollectionExists(ctx context.Context, name string) (bool, error) {
	query := `SELECT COUNT(*) FROM documents WHERE collection = ? LIMIT 1`

	var count int
	err := r.db.QueryRowContext(ctx, query, name).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		logger.Error(ctx, "failed to check collection existence",
			logger.Collection(name),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}

	exists := count > 0

	logger.Debug(ctx, "collection existence checked",
		logger.Collection(name),
		logger.Field("exists", exists),
	)

	return exists, nil
}

// Watch는 컬렉션의 변경 사항을 실시간으로 감지합니다
// Vitess/MySQL에서는 Change Streams를 직접 지원하지 않으므로
// 이 메서드는 에러를 반환합니다
// 실시간 변경 감지가 필요한 경우 Kafka CDC를 사용해야 합니다
func (r *VitessRepository) Watch(ctx context.Context, collection string, pipeline []interface{}) (*mongo.ChangeStream, error) {
	logger.Warn(ctx, "watch is not supported in Vitess, use Kafka CDC instead",
		logger.Collection(collection),
	)
	return nil, fmt.Errorf("watch is not supported in Vitess/MySQL, please use Kafka CDC for real-time change detection")
}

// GetCollectionStats는 컬렉션의 통계 정보를 반환합니다
func (r *VitessRepository) GetCollectionStats(ctx context.Context, name string) (map[string]interface{}, error) {
	start := time.Now()

	// 문서 개수
	countQuery := `SELECT COUNT(*) FROM documents WHERE collection = ?`
	var count int64
	if err := r.db.QueryRowContext(ctx, countQuery, name).Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	// 평균 문서 크기
	sizeQuery := `SELECT AVG(LENGTH(data)) FROM documents WHERE collection = ?`
	var avgSize sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, sizeQuery, name).Scan(&avgSize); err != nil {
		return nil, fmt.Errorf("failed to get average document size: %w", err)
	}

	// 총 크기
	totalSizeQuery := `SELECT SUM(LENGTH(data)) FROM documents WHERE collection = ?`
	var totalSize sql.NullInt64
	if err := r.db.QueryRowContext(ctx, totalSizeQuery, name).Scan(&totalSize); err != nil {
		return nil, fmt.Errorf("failed to get total size: %w", err)
	}

	// 최신/최고 버전
	versionQuery := `SELECT MAX(version) FROM documents WHERE collection = ?`
	var maxVersion sql.NullInt64
	if err := r.db.QueryRowContext(ctx, versionQuery, name).Scan(&maxVersion); err != nil {
		return nil, fmt.Errorf("failed to get max version: %w", err)
	}

	stats := map[string]interface{}{
		"collection":         name,
		"document_count":     count,
		"average_size_bytes": 0.0,
		"total_size_bytes":   int64(0),
		"max_version":        int64(0),
		"query_time_ms":      time.Since(start).Milliseconds(),
	}

	if avgSize.Valid {
		stats["average_size_bytes"] = avgSize.Float64
	}
	if totalSize.Valid {
		stats["total_size_bytes"] = totalSize.Int64
	}
	if maxVersion.Valid {
		stats["max_version"] = maxVersion.Int64
	}

	logger.Info(ctx, "collection stats retrieved",
		logger.Collection(name),
		logger.Field("document_count", count),
		logger.Duration(time.Since(start)),
	)

	return stats, nil
}
