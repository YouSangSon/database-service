package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MySQLRepository는 MySQL 기반 문서 저장소입니다
type MySQLRepository struct {
	db *sql.DB
}

// NewMySQLRepository는 MySQL 저장소를 생성합니다
func NewMySQLRepository(db *sql.DB) repository.DocumentRepository {
	return &MySQLRepository{db: db}
}

// ensureTableExists는 컬렉션(테이블)이 존재하는지 확인하고 없으면 생성합니다
func (r *MySQLRepository) ensureTableExists(ctx context.Context, collection string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			data JSON NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			version INTEGER NOT NULL DEFAULT 1,
			metadata JSON DEFAULT ('{}')
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, quoteIdentifier(collection))

	_, err := r.db.ExecContext(ctx, query)
	return err
}

// quoteIdentifier는 MySQL 식별자를 인용합니다
func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// ===== 기본 CRUD =====

// Save는 문서를 저장합니다
func (r *MySQLRepository) Save(ctx context.Context, doc *entity.Document) error {
	if err := r.ensureTableExists(ctx, doc.Collection); err != nil {
		return fmt.Errorf("failed to ensure table exists: %w", err)
	}

	dataJSON, err := json.Marshal(doc.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, data, created_at, updated_at, version, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, quoteIdentifier(doc.Collection))

	_, err = r.db.ExecContext(ctx, query,
		doc.ID,
		dataJSON,
		doc.CreatedAt,
		doc.UpdatedAt,
		doc.Version,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

// SaveMany는 여러 문서를 한 번에 저장합니다
func (r *MySQLRepository) SaveMany(ctx context.Context, docs []*entity.Document) error {
	if len(docs) == 0 {
		return nil
	}

	collection := docs[0].Collection
	if err := r.ensureTableExists(ctx, collection); err != nil {
		return fmt.Errorf("failed to ensure table exists: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO %s (id, data, created_at, updated_at, version, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, quoteIdentifier(collection))

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, doc := range docs {
		dataJSON, err := json.Marshal(doc.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}

		metadataJSON, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		_, err = stmt.ExecContext(ctx, doc.ID, dataJSON, doc.CreatedAt, doc.UpdatedAt, doc.Version, metadataJSON)
		if err != nil {
			return fmt.Errorf("failed to insert document: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByID는 ID로 문서를 조회합니다
func (r *MySQLRepository) FindByID(ctx context.Context, collection, id string) (*entity.Document, error) {
	query := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s
		WHERE id = ?
	`, quoteIdentifier(collection))

	var doc entity.Document
	var dataJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&dataJSON,
		&doc.CreatedAt,
		&doc.UpdatedAt,
		&doc.Version,
		&metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	doc.Collection = collection

	if err := json.Unmarshal(dataJSON, &doc.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &doc, nil
}

// FindAll은 컬렉션의 모든 문서를 조회합니다
func (r *MySQLRepository) FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s
		%s
	`, quoteIdentifier(collection), whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	return r.scanDocuments(rows, collection)
}

// FindWithOptions는 옵션을 사용하여 문서를 조회합니다
func (r *MySQLRepository) FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *repository.FindOptions) ([]*entity.Document, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s
		%s
		%s
		%s
		%s
	`, quoteIdentifier(collection),
		whereClause,
		r.buildOrderBy(opts.Sort),
		r.buildLimit(opts.Limit),
		r.buildOffset(opts.Skip),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	return r.scanDocuments(rows, collection)
}

// Update는 문서를 업데이트합니다
func (r *MySQLRepository) Update(ctx context.Context, doc *entity.Document) error {
	dataJSON, err := json.Marshal(doc.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET data = ?, updated_at = ?, version = version + 1, metadata = ?
		WHERE id = ? AND version = ?
	`, quoteIdentifier(doc.Collection))

	result, err := r.db.ExecContext(ctx, query,
		dataJSON,
		time.Now(),
		metadataJSON,
		doc.ID,
		doc.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("optimistic lock error: document was modified by another process")
	}

	doc.Version++
	doc.UpdatedAt = time.Now()

	return nil
}

// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
func (r *MySQLRepository) UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	whereClause, args := r.buildWhereClause(filter)

	setClauses := []string{}
	for key, value := range update {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal update value: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("data = JSON_SET(data, '$.%s', CAST(? AS JSON))", key))
		args = append(args, valueJSON)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET %s, updated_at = NOW(6), version = version + 1
		%s
	`, quoteIdentifier(collection), strings.Join(setClauses, ", "), whereClause)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to update documents: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// Replace는 문서를 교체합니다
func (r *MySQLRepository) Replace(ctx context.Context, collection, id string, replacement *entity.Document) error {
	dataJSON, err := json.Marshal(replacement.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	metadataJSON, err := json.Marshal(replacement.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET data = ?, updated_at = ?, version = version + 1, metadata = ?
		WHERE id = ?
	`, quoteIdentifier(collection))

	result, err := r.db.ExecContext(ctx, query,
		dataJSON,
		time.Now(),
		metadataJSON,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to replace document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("document not found")
	}

	return nil
}

// Delete는 문서를 삭제합니다
func (r *MySQLRepository) Delete(ctx context.Context, collection, id string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s WHERE id = ?
	`, quoteIdentifier(collection))

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("document not found")
	}

	return nil
}

// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
func (r *MySQLRepository) DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		DELETE FROM %s
		%s
	`, quoteIdentifier(collection), whereClause)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// ===== 원자적 연산 (Atomic Operations) =====

// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다
func (r *MySQLRepository) FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// SELECT FOR UPDATE로 행 잠금
	query := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s
		WHERE id = ?
		FOR UPDATE
	`, quoteIdentifier(collection))

	var doc entity.Document
	var dataJSON, metadataJSON []byte

	err = tx.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&dataJSON,
		&doc.CreatedAt,
		&doc.UpdatedAt,
		&doc.Version,
		&metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	doc.Collection = collection
	if err := json.Unmarshal(dataJSON, &doc.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Apply updates
	for key, value := range update {
		doc.Data[key] = value
	}

	updatedDataJSON, err := json.Marshal(doc.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated data: %w", err)
	}

	updateQuery := fmt.Sprintf(`
		UPDATE %s
		SET data = ?, updated_at = ?, version = version + 1
		WHERE id = ?
	`, quoteIdentifier(collection))

	_, err = tx.ExecContext(ctx, updateQuery, updatedDataJSON, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	doc.Version++
	doc.UpdatedAt = time.Now()

	return &doc, nil
}

// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
func (r *MySQLRepository) FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 먼저 기존 문서 조회
	selectQuery := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s
		WHERE id = ?
		FOR UPDATE
	`, quoteIdentifier(collection))

	var doc entity.Document
	var dataJSON, metadataJSON []byte

	err = tx.QueryRowContext(ctx, selectQuery, id).Scan(
		&doc.ID,
		&dataJSON,
		&doc.CreatedAt,
		&doc.UpdatedAt,
		&doc.Version,
		&metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	replacementDataJSON, err := json.Marshal(replacement.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal replacement data: %w", err)
	}

	replacementMetadataJSON, err := json.Marshal(replacement.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal replacement metadata: %w", err)
	}

	updateQuery := fmt.Sprintf(`
		UPDATE %s
		SET data = ?, updated_at = ?, version = version + 1, metadata = ?
		WHERE id = ?
	`, quoteIdentifier(collection))

	_, err = tx.ExecContext(ctx, updateQuery, replacementDataJSON, time.Now(), replacementMetadataJSON, id)
	if err != nil {
		return nil, fmt.Errorf("failed to replace document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 업데이트된 문서 반환
	doc.Data = replacement.Data
	doc.Metadata = replacement.Metadata
	doc.Version++
	doc.UpdatedAt = time.Now()
	doc.Collection = collection

	return &doc, nil
}

// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
func (r *MySQLRepository) FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 먼저 문서 조회
	selectQuery := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s
		WHERE id = ?
	`, quoteIdentifier(collection))

	var doc entity.Document
	var dataJSON, metadataJSON []byte

	err = tx.QueryRowContext(ctx, selectQuery, id).Scan(
		&doc.ID,
		&dataJSON,
		&doc.CreatedAt,
		&doc.UpdatedAt,
		&doc.Version,
		&metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	doc.Collection = collection
	if err := json.Unmarshal(dataJSON, &doc.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// 삭제
	deleteQuery := fmt.Sprintf(`
		DELETE FROM %s WHERE id = ?
	`, quoteIdentifier(collection))

	_, err = tx.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &doc, nil
}

// Upsert는 문서가 없으면 생성하고 있으면 업데이트합니다
func (r *MySQLRepository) Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error) {
	if err := r.ensureTableExists(ctx, collection); err != nil {
		return "", fmt.Errorf("failed to ensure table exists: %w", err)
	}

	id, ok := filter["_id"].(string)
	if !ok {
		id, ok = filter["id"].(string)
		if !ok {
			return "", errors.New("upsert requires 'id' or '_id' in filter")
		}
	}

	updateDataJSON, err := json.Marshal(update)
	if err != nil {
		return "", fmt.Errorf("failed to marshal update data: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, data, created_at, updated_at, version, metadata)
		VALUES (?, ?, ?, ?, 1, '{}')
		ON DUPLICATE KEY UPDATE
			data = VALUES(data),
			updated_at = VALUES(updated_at),
			version = version + 1
	`, quoteIdentifier(collection))

	now := time.Now()
	_, err = r.db.ExecContext(ctx, query, id, updateDataJSON, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to upsert document: %w", err)
	}

	return id, nil
}

// ===== Helper methods =====

func (r *MySQLRepository) buildWhereClause(filter map[string]interface{}) (string, []interface{}) {
	if len(filter) == 0 {
		return "", nil
	}

	conditions := []string{}
	args := []interface{}{}

	for key, value := range filter {
		if key == "_id" || key == "id" {
			conditions = append(conditions, "id = ?")
			args = append(args, value)
		} else {
			// JSON 필드 검색
			valueJSON, _ := json.Marshal(value)
			conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = CAST(? AS JSON)", key))
			args = append(args, string(valueJSON))
		}
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

func (r *MySQLRepository) buildOrderBy(sort map[string]int) string {
	if len(sort) == 0 {
		return ""
	}

	orders := []string{}
	for key, direction := range sort {
		dir := "ASC"
		if direction == -1 {
			dir = "DESC"
		}
		if key == "_id" || key == "id" {
			orders = append(orders, fmt.Sprintf("id %s", dir))
		} else {
			orders = append(orders, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') %s", key, dir))
		}
	}

	return "ORDER BY " + strings.Join(orders, ", ")
}

func (r *MySQLRepository) buildLimit(limit int64) string {
	if limit <= 0 {
		return ""
	}
	return fmt.Sprintf("LIMIT %d", limit)
}

func (r *MySQLRepository) buildOffset(offset int64) string {
	if offset <= 0 {
		return ""
	}
	return fmt.Sprintf("OFFSET %d", offset)
}

func (r *MySQLRepository) scanDocuments(rows *sql.Rows, collection string) ([]*entity.Document, error) {
	documents := []*entity.Document{}

	for rows.Next() {
		var doc entity.Document
		var dataJSON, metadataJSON []byte

		err := rows.Scan(
			&doc.ID,
			&dataJSON,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&doc.Version,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		doc.Collection = collection

		if err := json.Unmarshal(dataJSON, &doc.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		documents = append(documents, &doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return documents, nil
}

// ===== 집계 (Aggregation) =====

// Aggregate는 집계 파이프라인을 실행합니다 (제한적 지원)
func (r *MySQLRepository) Aggregate(ctx context.Context, collection string, pipeline []bson.M) ([]map[string]interface{}, error) {
	return nil, errors.New("aggregate is not fully supported in MySQL implementation")
}

// Distinct는 고유한 값을 조회합니다
func (r *MySQLRepository) Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT DISTINCT JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s')) as value
		FROM %s
		%s
	`, field, quoteIdentifier(collection), whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct values: %w", err)
	}
	defer rows.Close()

	values := []interface{}{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("failed to scan value: %w", err)
		}
		values = append(values, value)
	}

	return values, nil
}

// Count는 문서 개수를 반환합니다
func (r *MySQLRepository) Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s %s
	`, quoteIdentifier(collection), whereClause)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// EstimatedDocumentCount는 컬렉션의 추정 문서 개수를 반환합니다
func (r *MySQLRepository) EstimatedDocumentCount(ctx context.Context, collection string) (int64, error) {
	query := `
		SELECT TABLE_ROWS
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = ?
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, collection).Scan(&count)
	if err != nil {
		// 테이블이 없거나 통계가 없으면 정확한 카운트 반환
		return r.Count(ctx, collection, nil)
	}

	return count, nil
}

// ===== 벌크 작업 (Bulk Operations) =====

// BulkWrite는 여러 작업을 한 번에 실행합니다
func (r *MySQLRepository) BulkWrite(ctx context.Context, operations []*repository.BulkOperation) (*repository.BulkResult, error) {
	if len(operations) == 0 {
		return &repository.BulkResult{}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result := &repository.BulkResult{
		UpsertedIDs: make(map[int]interface{}),
	}

	for i, op := range operations {
		switch op.Type {
		case "insert":
			if err := r.ensureTableExists(ctx, op.Collection); err != nil {
				return nil, fmt.Errorf("failed to ensure table exists: %w", err)
			}

			dataJSON, err := json.Marshal(op.Document.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal data: %w", err)
			}

			metadataJSON, err := json.Marshal(op.Document.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}

			query := fmt.Sprintf(`
				INSERT INTO %s (id, data, created_at, updated_at, version, metadata)
				VALUES (?, ?, ?, ?, ?, ?)
			`, quoteIdentifier(op.Collection))

			_, err = tx.ExecContext(ctx, query,
				op.Document.ID,
				dataJSON,
				op.Document.CreatedAt,
				op.Document.UpdatedAt,
				op.Document.Version,
				metadataJSON,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert document: %w", err)
			}

			result.InsertedCount++

		case "update":
			whereClause, args := r.buildWhereClause(op.Filter)

			setClauses := []string{}
			for key, value := range op.Update {
				valueJSON, err := json.Marshal(value)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal update value: %w", err)
				}
				setClauses = append(setClauses, fmt.Sprintf("data = JSON_SET(data, '$.%s', CAST(? AS JSON))", key))
				args = append(args, valueJSON)
			}

			query := fmt.Sprintf(`
				UPDATE %s
				SET %s, updated_at = NOW(6), version = version + 1
				%s
			`, quoteIdentifier(op.Collection), strings.Join(setClauses, ", "), whereClause)

			res, err := tx.ExecContext(ctx, query, args...)
			if err != nil {
				return nil, fmt.Errorf("failed to update documents: %w", err)
			}

			affected, _ := res.RowsAffected()
			result.MatchedCount += affected
			result.ModifiedCount += affected

		case "delete":
			whereClause, args := r.buildWhereClause(op.Filter)

			query := fmt.Sprintf(`
				DELETE FROM %s %s
			`, quoteIdentifier(op.Collection), whereClause)

			res, err := tx.ExecContext(ctx, query, args...)
			if err != nil {
				return nil, fmt.Errorf("failed to delete documents: %w", err)
			}

			affected, _ := res.RowsAffected()
			result.DeletedCount += affected

		case "replace":
			if op.Document == nil {
				return nil, errors.New("replace operation requires a document")
			}

			dataJSON, err := json.Marshal(op.Document.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal data: %w", err)
			}

			metadataJSON, err := json.Marshal(op.Document.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}

			query := fmt.Sprintf(`
				UPDATE %s
				SET data = ?, updated_at = ?, version = version + 1, metadata = ?
				WHERE id = ?
			`, quoteIdentifier(op.Collection))

			res, err := tx.ExecContext(ctx, query,
				dataJSON,
				time.Now(),
				metadataJSON,
				op.ReplaceOneID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to replace document: %w", err)
			}

			affected, _ := res.RowsAffected()
			result.MatchedCount += affected
			result.ModifiedCount += affected
		}

		result.UpsertedIDs[i] = i
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// ===== 인덱스 관리 (Index Management) =====

// CreateIndex는 단일 인덱스를 생성합니다
func (r *MySQLRepository) CreateIndex(ctx context.Context, collection string, model repository.IndexModel) (string, error) {
	indexName := model.Options.Name
	if indexName == "" {
		indexName = fmt.Sprintf("idx_%s_%v", collection, time.Now().Unix())
	}

	// JSON 필드에 대한 인덱스 생성 (Generated Column 사용)
	indexKeys := []string{}
	for key := range model.Keys {
		if key == "_id" || key == "id" {
			indexKeys = append(indexKeys, "id")
		} else {
			// MySQL에서는 JSON 필드에 직접 인덱스를 생성할 수 없으므로 generated column 필요
			indexKeys = append(indexKeys, fmt.Sprintf("(CAST(JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s')) AS CHAR(255)))", key))
		}
	}

	unique := ""
	if model.Options.Unique != nil && *model.Options.Unique {
		unique = "UNIQUE"
	}

	query := fmt.Sprintf(`
		CREATE %s INDEX %s ON %s %s
	`, unique, quoteIdentifier(indexName), quoteIdentifier(collection), "("+strings.Join(indexKeys, ", ")+")")

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to create index: %w", err)
	}

	return indexName, nil
}

// CreateIndexes는 여러 인덱스를 생성합니다
func (r *MySQLRepository) CreateIndexes(ctx context.Context, collection string, models []repository.IndexModel) ([]string, error) {
	indexNames := []string{}

	for _, model := range models {
		indexName, err := r.CreateIndex(ctx, collection, model)
		if err != nil {
			return indexNames, fmt.Errorf("failed to create index: %w", err)
		}
		indexNames = append(indexNames, indexName)
	}

	return indexNames, nil
}

// DropIndex는 인덱스를 삭제합니다
func (r *MySQLRepository) DropIndex(ctx context.Context, collection, indexName string) error {
	query := fmt.Sprintf(`DROP INDEX %s ON %s`, quoteIdentifier(indexName), quoteIdentifier(collection))

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	return nil
}

// ListIndexes는 컬렉션의 인덱스 목록을 반환합니다
func (r *MySQLRepository) ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	query := `
		SELECT DISTINCT INDEX_NAME, COLUMN_NAME, NON_UNIQUE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = ?
	`

	rows, err := r.db.QueryContext(ctx, query, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer rows.Close()

	indexes := []map[string]interface{}{}
	for rows.Next() {
		var indexName, columnName string
		var nonUnique int
		if err := rows.Scan(&indexName, &columnName, &nonUnique); err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		indexes = append(indexes, map[string]interface{}{
			"name":       indexName,
			"column":     columnName,
			"non_unique": nonUnique,
		})
	}

	return indexes, nil
}

// ===== 컬렉션 관리 (Collection Management) =====

// CreateCollection은 컬렉션을 생성합니다
func (r *MySQLRepository) CreateCollection(ctx context.Context, name string) error {
	return r.ensureTableExists(ctx, name)
}

// DropCollection은 컬렉션을 삭제합니다
func (r *MySQLRepository) DropCollection(ctx context.Context, name string) error {
	query := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, quoteIdentifier(name))

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	return nil
}

// RenameCollection은 컬렉션 이름을 변경합니다
func (r *MySQLRepository) RenameCollection(ctx context.Context, oldName, newName string) error {
	query := fmt.Sprintf(`RENAME TABLE %s TO %s`, quoteIdentifier(oldName), quoteIdentifier(newName))

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to rename collection: %w", err)
	}

	return nil
}

// ListCollections는 데이터베이스의 컬렉션 목록을 반환합니다
func (r *MySQLRepository) ListCollections(ctx context.Context) ([]string, error) {
	query := `
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_TYPE = 'BASE TABLE'
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer rows.Close()

	collections := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan collection name: %w", err)
		}
		collections = append(collections, name)
	}

	return collections, nil
}

// CollectionExists는 컬렉션이 존재하는지 확인합니다
func (r *MySQLRepository) CollectionExists(ctx context.Context, name string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = ?
		AND TABLE_TYPE = 'BASE TABLE'
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}

	return count > 0, nil
}

// ===== Change Streams =====

// Watch는 컬렉션의 변경 사항을 실시간으로 감지합니다
func (r *MySQLRepository) Watch(ctx context.Context, collection string, pipeline []bson.M) (*mongo.ChangeStream, error) {
	return nil, errors.New("watch is not supported in MySQL implementation")
}

// ===== 트랜잭션 (Transaction) =====

// WithTransaction은 트랜잭션 내에서 함수를 실행합니다
func (r *MySQLRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ===== Raw Query Execution =====

// ExecuteRawQuery는 데이터베이스별 raw query를 실행합니다
func (r *MySQLRepository) ExecuteRawQuery(ctx context.Context, query interface{}) (interface{}, error) {
	sqlQuery, ok := query.(string)
	if !ok {
		return nil, errors.New("query must be a string for MySQL")
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute raw query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	results := []map[string]interface{}{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

// ExecuteRawQueryWithResult는 raw query를 실행하고 결과를 특정 타입으로 반환합니다
func (r *MySQLRepository) ExecuteRawQueryWithResult(ctx context.Context, query interface{}, result interface{}) error {
	data, err := r.ExecuteRawQuery(ctx, query)
	if err != nil {
		return err
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(dataJSON, result); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

// ===== 헬스체크 =====

// HealthCheck는 저장소의 상태를 확인합니다
func (r *MySQLRepository) HealthCheck(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
