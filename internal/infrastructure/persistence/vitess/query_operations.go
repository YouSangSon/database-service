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

// FindWithOptions는 옵션을 사용하여 문서를 조회합니다
func (r *VitessRepository) FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *repository.FindOptions) ([]*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_with_options", collection, "success", duration)
	}()

	// 기본 쿼리
	query := `
		SELECT id, collection, data, version, created_at, updated_at
		FROM documents
		WHERE collection = ?
	`

	args := []interface{}{collection}

	// 필터 조건 추가 (JSON 필드 검색)
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

	// Sort 옵션 추가
	if opts != nil && len(opts.Sort) > 0 {
		sortClauses := []string{}
		for field, order := range opts.Sort {
			direction := "ASC"
			if order == -1 {
				direction = "DESC"
			}
			// JSON 필드 정렬을 위해 JSON_EXTRACT 사용
			if field == "created_at" || field == "updated_at" {
				sortClauses = append(sortClauses, fmt.Sprintf("%s %s", field, direction))
			} else {
				sortClauses = append(sortClauses, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') %s", field, direction))
			}
		}
		query += " ORDER BY " + strings.Join(sortClauses, ", ")
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Limit 옵션 추가
	if opts != nil && opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	// Skip (Offset) 옵션 추가
	if opts != nil && opts.Skip > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Skip)
	}

	logger.Debug(ctx, "executing find with options",
		logger.Collection(collection),
		logger.Field("query", query),
		logger.Field("args", args),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.metrics.RecordDBOperation("find_with_options", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find documents with options",
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

		// Projection 적용 (필요한 필드만 선택)
		if opts != nil && len(opts.Projection) > 0 {
			projectedData := make(map[string]interface{})
			for field, include := range opts.Projection {
				if include == 1 || include == true {
					if val, exists := data[field]; exists {
						projectedData[field] = val
					}
				}
			}
			data = projectedData
		}

		doc := entity.ReconstructDocument(id, coll, data, version, createdAt, updatedAt)
		documents = append(documents, doc)
	}

	logger.Info(ctx, "documents found with options",
		logger.Collection(collection),
		logger.Field("count", len(documents)),
		logger.Duration(time.Since(start)),
	)

	return documents, nil
}

// Upsert는 문서가 없으면 생성하고 있으면 업데이트합니다
func (r *VitessRepository) Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("upsert", collection, "success", duration)
	}()

	// 먼저 문서를 찾아봅니다
	docs, err := r.FindAll(ctx, collection, filter)
	if err != nil {
		return "", fmt.Errorf("failed to find document for upsert: %w", err)
	}

	// 문서가 존재하면 업데이트
	if len(docs) > 0 {
		doc := docs[0]

		// 업데이트 데이터 병합
		for key, value := range update {
			doc.Data()[key] = value
		}

		doc.IncrementVersion()

		if err := r.Update(ctx, doc); err != nil {
			r.metrics.RecordDBOperation("upsert", collection, "error", time.Since(start))
			return "", fmt.Errorf("failed to update document: %w", err)
		}

		logger.Info(ctx, "document updated via upsert",
			logger.Collection(collection),
			logger.DocumentID(doc.ID()),
			logger.Duration(time.Since(start)),
		)

		return doc.ID(), nil
	}

	// 문서가 없으면 생성
	newData := make(map[string]interface{})
	for key, value := range filter {
		newData[key] = value
	}
	for key, value := range update {
		newData[key] = value
	}

	doc := entity.NewDocument(collection, newData)
	if err := r.Save(ctx, doc); err != nil {
		r.metrics.RecordDBOperation("upsert", collection, "error", time.Since(start))
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	logger.Info(ctx, "document created via upsert",
		logger.Collection(collection),
		logger.DocumentID(doc.ID()),
		logger.Duration(time.Since(start)),
	)

	return doc.ID(), nil
}

// Replace는 문서를 교체합니다 (이전 문서를 반환하지 않음)
func (r *VitessRepository) Replace(ctx context.Context, collection, id string, replacement *entity.Document) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("replace", collection, "success", duration)
	}()

	// 먼저 문서가 존재하는지 확인
	_, err := r.FindByID(ctx, collection, id)
	if err != nil {
		if err == entity.ErrDocumentNotFound {
			r.metrics.RecordDBOperation("replace", collection, "not_found", time.Since(start))
			return entity.ErrDocumentNotFound
		}
		return fmt.Errorf("failed to check document existence: %w", err)
	}

	// 교체할 데이터를 JSON으로 변환
	dataJSON, err := json.Marshal(replacement.Data())
	if err != nil {
		return fmt.Errorf("failed to marshal replacement data: %w", err)
	}

	// 문서 교체 (버전 증가)
	query := `
		UPDATE documents
		SET data = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND collection = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		dataJSON,
		time.Now(),
		id,
		collection,
	)
	if err != nil {
		r.metrics.RecordDBOperation("replace", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to replace document",
			logger.Collection(collection),
			logger.DocumentID(id),
			zap.Error(err),
		)
		return fmt.Errorf("failed to replace document: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected == 0 {
		r.metrics.RecordDBOperation("replace", collection, "not_found", time.Since(start))
		return entity.ErrDocumentNotFound
	}

	logger.Info(ctx, "document replaced",
		logger.Collection(collection),
		logger.DocumentID(id),
		logger.Duration(time.Since(start)),
	)

	return nil
}

// getDB는 트랜잭션이 있으면 트랜잭션을 사용하고, 없으면 일반 DB를 사용합니다
func (r *VitessRepository) getDB(ctx context.Context) interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if tx, ok := ctx.Value("tx").(*sql.Tx); ok {
		return tx
	}
	return r.db
}
