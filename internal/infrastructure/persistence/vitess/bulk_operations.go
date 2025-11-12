package vitess

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// SaveMany는 여러 문서를 한 번에 저장합니다
func (r *VitessRepository) SaveMany(ctx context.Context, docs []*entity.Document) error {
	if len(docs) == 0 {
		return nil
	}

	start := time.Now()
	collection := docs[0].Collection()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("save_many", collection, "success", duration)
	}()

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

	// 벌크 인서트를 위한 쿼리 생성
	valueStrings := make([]string, 0, len(docs))
	valueArgs := make([]interface{}, 0, len(docs)*6)

	for _, doc := range docs {
		dataJSON, err := json.Marshal(doc.Data())
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to marshal data: %w", err)
		}

		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			doc.ID(),
			doc.Collection(),
			dataJSON,
			doc.Version(),
			doc.CreatedAt(),
			doc.UpdatedAt(),
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO documents (id, collection, data, version, created_at, updated_at)
		VALUES %s
	`, strings.Join(valueStrings, ", "))

	_, err = tx.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("save_many", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to save many documents",
			logger.Collection(collection),
			logger.Field("count", len(docs)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save many documents: %w", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info(ctx, "documents saved in bulk",
		logger.Collection(collection),
		logger.Field("count", len(docs)),
		logger.Duration(time.Since(start)),
	)

	return nil
}

// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
func (r *VitessRepository) UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("update_many", collection, "success", duration)
	}()

	// 먼저 업데이트할 문서를 찾습니다
	docs, err := r.FindAll(ctx, collection, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to find documents for update: %w", err)
	}

	if len(docs) == 0 {
		return 0, nil
	}

	// 트랜잭션 시작
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error(ctx, "failed to begin transaction", zap.Error(err))
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	var updated int64
	for _, doc := range docs {
		// 업데이트 적용
		for key, value := range update {
			doc.Data()[key] = value
		}

		doc.IncrementVersion()

		dataJSON, err := json.Marshal(doc.Data())
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("failed to marshal data: %w", err)
		}

		query := `
			UPDATE documents
			SET data = ?, version = ?, updated_at = ?
			WHERE id = ? AND collection = ?
		`

		result, err := tx.ExecContext(ctx, query,
			dataJSON,
			doc.Version(),
			time.Now(),
			doc.ID(),
			collection,
		)
		if err != nil {
			_ = tx.Rollback()
			r.metrics.RecordDBOperation("update_many", collection, "error", time.Since(start))
			logger.Error(ctx, "failed to update document",
				logger.Collection(collection),
				logger.DocumentID(doc.ID()),
				zap.Error(err),
			)
			return 0, fmt.Errorf("failed to update document: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("failed to get rows affected: %w", err)
		}

		updated += affected
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit transaction", zap.Error(err))
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info(ctx, "documents updated in bulk",
		logger.Collection(collection),
		logger.Field("count", updated),
		logger.Duration(time.Since(start)),
	)

	return updated, nil
}

// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
func (r *VitessRepository) DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("delete_many", collection, "success", duration)
	}()

	// 기본 쿼리
	query := "DELETE FROM documents WHERE collection = ?"
	args := []interface{}{collection}

	// 필터 조건 추가
	if len(filter) > 0 {
		conditions := []string{}
		for key, value := range filter {
			conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = ?", key))
			args = append(args, value)
		}
		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
		}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.metrics.RecordDBOperation("delete_many", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to delete many documents",
			logger.Collection(collection),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to delete many documents: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	logger.Info(ctx, "documents deleted in bulk",
		logger.Collection(collection),
		logger.Field("count", deleted),
		logger.Duration(time.Since(start)),
	)

	return deleted, nil
}

// BulkWrite는 여러 작업을 한 번에 실행합니다
func (r *VitessRepository) BulkWrite(ctx context.Context, operations []*repository.BulkOperation) (*repository.BulkResult, error) {
	if len(operations) == 0 {
		return &repository.BulkResult{}, nil
	}

	start := time.Now()
	collection := operations[0].Collection

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("bulk_write", collection, "success", duration)
	}()

	result := &repository.BulkResult{
		UpsertedIDs: make(map[int]interface{}),
	}

	// 트랜잭션 시작
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error(ctx, "failed to begin transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	txCtx := context.WithValue(ctx, "tx", tx)

	for i, op := range operations {
		switch op.Type {
		case "insert":
			if err := r.bulkInsert(txCtx, tx, op); err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("bulk insert failed at index %d: %w", i, err)
			}
			result.InsertedCount++

		case "update":
			matched, modified, err := r.bulkUpdate(txCtx, tx, op)
			if err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("bulk update failed at index %d: %w", i, err)
			}
			result.MatchedCount += matched
			result.ModifiedCount += modified

		case "delete":
			deleted, err := r.bulkDelete(txCtx, tx, op)
			if err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("bulk delete failed at index %d: %w", i, err)
			}
			result.DeletedCount += deleted

		case "replace":
			matched, modified, err := r.bulkReplace(txCtx, tx, op)
			if err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("bulk replace failed at index %d: %w", i, err)
			}
			result.MatchedCount += matched
			result.ModifiedCount += modified

		default:
			_ = tx.Rollback()
			return nil, fmt.Errorf("unknown bulk operation type: %s", op.Type)
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit bulk write", zap.Error(err))
		return nil, fmt.Errorf("failed to commit bulk write: %w", err)
	}

	logger.Info(ctx, "bulk write completed",
		logger.Collection(collection),
		logger.Field("operations", len(operations)),
		logger.Field("inserted", result.InsertedCount),
		logger.Field("modified", result.ModifiedCount),
		logger.Field("deleted", result.DeletedCount),
		logger.Duration(time.Since(start)),
	)

	return result, nil
}

func (r *VitessRepository) bulkInsert(ctx context.Context, tx *sql.Tx, op *repository.BulkOperation) error {
	dataJSON, err := json.Marshal(op.Document.Data())
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		INSERT INTO documents (id, collection, data, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, query,
		op.Document.ID(),
		op.Collection,
		dataJSON,
		op.Document.Version(),
		op.Document.CreatedAt(),
		op.Document.UpdatedAt(),
	)

	return err
}

func (r *VitessRepository) bulkUpdate(ctx context.Context, tx *sql.Tx, op *repository.BulkOperation) (int64, int64, error) {
	// 먼저 문서를 찾습니다
	docs, err := r.FindAll(ctx, op.Collection, op.Filter)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to find documents: %w", err)
	}

	if len(docs) == 0 {
		return 0, 0, nil
	}

	var modified int64

	// UpdateMany이면 모든 문서 업데이트, 아니면 첫 번째만
	docsToUpdate := docs
	if !op.UpdateMany {
		docsToUpdate = docs[:1]
	}

	for _, doc := range docsToUpdate {
		for key, value := range op.Update {
			doc.Data()[key] = value
		}

		doc.IncrementVersion()

		dataJSON, err := json.Marshal(doc.Data())
		if err != nil {
			return 0, 0, fmt.Errorf("failed to marshal data: %w", err)
		}

		query := `
			UPDATE documents
			SET data = ?, version = ?, updated_at = ?
			WHERE id = ? AND collection = ?
		`

		result, err := tx.ExecContext(ctx, query,
			dataJSON,
			doc.Version(),
			time.Now(),
			doc.ID(),
			op.Collection,
		)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to update document: %w", err)
		}

		affected, _ := result.RowsAffected()
		modified += affected
	}

	return int64(len(docsToUpdate)), modified, nil
}

func (r *VitessRepository) bulkDelete(ctx context.Context, tx *sql.Tx, op *repository.BulkOperation) (int64, error) {
	query := "DELETE FROM documents WHERE collection = ?"
	args := []interface{}{op.Collection}

	// 필터 조건 추가
	if len(op.Filter) > 0 {
		conditions := []string{}
		for key, value := range op.Filter {
			conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = ?", key))
			args = append(args, value)
		}
		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
		}
	}

	// DeleteMany가 아니면 LIMIT 1
	if !op.DeleteMany {
		query += " LIMIT 1"
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}

	deleted, _ := result.RowsAffected()
	return deleted, nil
}

func (r *VitessRepository) bulkReplace(ctx context.Context, tx *sql.Tx, op *repository.BulkOperation) (int64, int64, error) {
	dataJSON, err := json.Marshal(op.Document.Data())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		UPDATE documents
		SET data = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND collection = ?
	`

	result, err := tx.ExecContext(ctx, query,
		dataJSON,
		time.Now(),
		op.ReplaceOneID,
		op.Collection,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to replace document: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected, affected, nil
}
