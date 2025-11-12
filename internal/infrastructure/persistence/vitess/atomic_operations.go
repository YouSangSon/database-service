package vitess

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다 (원자적 연산)
func (r *VitessRepository) FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_and_update", collection, "success", duration)
	}()

	// 트랜잭션 시작 (원자성 보장)
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable, // 직렬화 격리 수준으로 비관적 잠금 구현
	})
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

	// FOR UPDATE를 사용하여 행 잠금 획득 (비관적 잠금)
	query := `
		SELECT id, collection, data, version, created_at, updated_at
		FROM documents
		WHERE id = ? AND collection = ?
		FOR UPDATE
	`

	var (
		docID     string
		coll      string
		dataJSON  []byte
		version   int
		createdAt time.Time
		updatedAt time.Time
	)

	err = tx.QueryRowContext(ctx, query, id, collection).Scan(
		&docID, &coll, &dataJSON, &version, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_and_update", collection, "not_found", time.Since(start))
		return nil, entity.ErrDocumentNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_and_update", collection, "error", time.Since(start))
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
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// 업데이트 적용
	for key, value := range update {
		data[key] = value
	}

	// JSON으로 다시 변환
	updatedDataJSON, err := json.Marshal(data)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to marshal updated data: %w", err)
	}

	// 문서 업데이트
	updateQuery := `
		UPDATE documents
		SET data = ?, version = ?, updated_at = ?
		WHERE id = ? AND collection = ?
	`

	newVersion := version + 1
	newUpdatedAt := time.Now()

	_, err = tx.ExecContext(ctx, updateQuery,
		updatedDataJSON,
		newVersion,
		newUpdatedAt,
		id,
		collection,
	)
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_and_update", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to update document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 업데이트된 문서 반환
	doc := entity.ReconstructDocument(docID, coll, data, newVersion, createdAt, newUpdatedAt)

	logger.Info(ctx, "document found and updated atomically",
		logger.Collection(collection),
		logger.DocumentID(id),
		logger.Duration(time.Since(start)),
	)

	return doc, nil
}

// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
func (r *VitessRepository) FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_one_and_replace", collection, "success", duration)
	}()

	// 트랜잭션 시작
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
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

	// FOR UPDATE를 사용하여 행 잠금 획득
	query := `
		SELECT id, collection, data, version, created_at, updated_at
		FROM documents
		WHERE id = ? AND collection = ?
		FOR UPDATE
	`

	var (
		docID     string
		coll      string
		dataJSON  []byte
		version   int
		createdAt time.Time
		updatedAt time.Time
	)

	err = tx.QueryRowContext(ctx, query, id, collection).Scan(
		&docID, &coll, &dataJSON, &version, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_one_and_replace", collection, "not_found", time.Since(start))
		return nil, entity.ErrDocumentNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_one_and_replace", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// 교체할 데이터를 JSON으로 변환
	replacementDataJSON, err := json.Marshal(replacement.Data())
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to marshal replacement data: %w", err)
	}

	// 문서 교체
	replaceQuery := `
		UPDATE documents
		SET data = ?, version = ?, updated_at = ?
		WHERE id = ? AND collection = ?
	`

	newVersion := version + 1
	newUpdatedAt := time.Now()

	_, err = tx.ExecContext(ctx, replaceQuery,
		replacementDataJSON,
		newVersion,
		newUpdatedAt,
		id,
		collection,
	)
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_one_and_replace", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to replace document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to replace document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 교체된 문서 반환
	doc := entity.ReconstructDocument(docID, coll, replacement.Data(), newVersion, createdAt, newUpdatedAt)

	logger.Info(ctx, "document found and replaced atomically",
		logger.Collection(collection),
		logger.DocumentID(id),
		logger.Duration(time.Since(start)),
	)

	return doc, nil
}

// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
func (r *VitessRepository) FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_one_and_delete", collection, "success", duration)
	}()

	// 트랜잭션 시작
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
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

	// FOR UPDATE를 사용하여 행 잠금 획득
	query := `
		SELECT id, collection, data, version, created_at, updated_at
		FROM documents
		WHERE id = ? AND collection = ?
		FOR UPDATE
	`

	var (
		docID     string
		coll      string
		dataJSON  []byte
		version   int
		createdAt time.Time
		updatedAt time.Time
	)

	err = tx.QueryRowContext(ctx, query, id, collection).Scan(
		&docID, &coll, &dataJSON, &version, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_one_and_delete", collection, "not_found", time.Since(start))
		return nil, entity.ErrDocumentNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_one_and_delete", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// JSON 파싱 (반환하기 위해)
	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// 문서 삭제
	deleteQuery := `DELETE FROM documents WHERE id = ? AND collection = ?`

	_, err = tx.ExecContext(ctx, deleteQuery, id, collection)
	if err != nil {
		_ = tx.Rollback()
		r.metrics.RecordDBOperation("find_one_and_delete", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to delete document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "failed to commit transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 삭제된 문서 반환
	doc := entity.ReconstructDocument(docID, coll, data, version, createdAt, updatedAt)

	logger.Info(ctx, "document found and deleted atomically",
		logger.Collection(collection),
		logger.DocumentID(id),
		logger.Duration(time.Since(start)),
	)

	return doc, nil
}
