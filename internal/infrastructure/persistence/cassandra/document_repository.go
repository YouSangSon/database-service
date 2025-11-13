package cassandra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/gocql/gocql"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CassandraRepository는 Cassandra 기반 문서 저장소입니다
type CassandraRepository struct {
	session *gocql.Session
	keyspace string
}

// NewCassandraRepository는 Cassandra 저장소를 생성합니다
func NewCassandraRepository(session *gocql.Session, keyspace string) repository.DocumentRepository {
	return &CassandraRepository{
		session:  session,
		keyspace: keyspace,
	}
}

// ensureTableExists는 컬렉션(테이블)이 존재하는지 확인하고 없으면 생성합니다
func (r *CassandraRepository) ensureTableExists(ctx context.Context, collection string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id text PRIMARY KEY,
			data text,
			created_at timestamp,
			updated_at timestamp,
			version int,
			metadata text
		)
	`, r.keyspace, collection)

	return r.session.Query(query).WithContext(ctx).Exec()
}

// ===== 기본 CRUD =====

// Save는 문서를 저장합니다
func (r *CassandraRepository) Save(ctx context.Context, doc *entity.Document) error {
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
		INSERT INTO %s.%s (id, data, created_at, updated_at, version, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, r.keyspace, doc.Collection)

	return r.session.Query(query,
		doc.ID,
		string(dataJSON),
		doc.CreatedAt,
		doc.UpdatedAt,
		doc.Version,
		string(metadataJSON),
	).WithContext(ctx).Exec()
}

// SaveMany는 여러 문서를 한 번에 저장합니다 (Batch)
func (r *CassandraRepository) SaveMany(ctx context.Context, docs []*entity.Document) error {
	if len(docs) == 0 {
		return nil
	}

	collection := docs[0].Collection
	if err := r.ensureTableExists(ctx, collection); err != nil {
		return fmt.Errorf("failed to ensure table exists: %w", err)
	}

	batch := r.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	queryStr := fmt.Sprintf(`
		INSERT INTO %s.%s (id, data, created_at, updated_at, version, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, r.keyspace, collection)

	for _, doc := range docs {
		dataJSON, err := json.Marshal(doc.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}

		metadataJSON, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		batch.Query(queryStr,
			doc.ID,
			string(dataJSON),
			doc.CreatedAt,
			doc.UpdatedAt,
			doc.Version,
			string(metadataJSON),
		)
	}

	return r.session.ExecuteBatch(batch)
}

// FindByID는 ID로 문서를 조회합니다
func (r *CassandraRepository) FindByID(ctx context.Context, collection, id string) (*entity.Document, error) {
	query := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s.%s
		WHERE id = ?
	`, r.keyspace, collection)

	var doc entity.Document
	var dataStr, metadataStr string

	err := r.session.Query(query, id).WithContext(ctx).Scan(
		&doc.ID,
		&dataStr,
		&doc.CreatedAt,
		&doc.UpdatedAt,
		&doc.Version,
		&metadataStr,
	)
	if err == gocql.ErrNotFound {
		return nil, errors.New("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	doc.Collection = collection

	if err := json.Unmarshal([]byte(dataStr), &doc.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	if err := json.Unmarshal([]byte(metadataStr), &doc.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &doc, nil
}

// FindAll은 컬렉션의 모든 문서를 조회합니다
// 주의: Cassandra에서는 ALLOW FILTERING이 필요하거나 Secondary Index가 필요합니다
func (r *CassandraRepository) FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s.%s
		%s
		ALLOW FILTERING
	`, r.keyspace, collection, whereClause)

	iter := r.session.Query(query, args...).WithContext(ctx).Iter()
	defer iter.Close()

	return r.scanDocuments(iter, collection)
}

// FindWithOptions는 옵션을 사용하여 문서를 조회합니다
func (r *CassandraRepository) FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *repository.FindOptions) ([]*entity.Document, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT id, data, created_at, updated_at, version, metadata
		FROM %s.%s
		%s
		%s
		%s
		ALLOW FILTERING
	`, r.keyspace, collection,
		whereClause,
		r.buildOrderBy(opts.Sort),
		r.buildLimit(opts.Limit),
	)

	iter := r.session.Query(query, args...).WithContext(ctx).Iter()
	defer iter.Close()

	documents, err := r.scanDocuments(iter, collection)
	if err != nil {
		return nil, err
	}

	// Cassandra는 OFFSET을 지원하지 않으므로 메모리에서 스킵
	if opts.Skip > 0 {
		if int64(len(documents)) > opts.Skip {
			documents = documents[opts.Skip:]
		} else {
			documents = []*entity.Document{}
		}
	}

	return documents, nil
}

// Update는 문서를 업데이트합니다
func (r *CassandraRepository) Update(ctx context.Context, doc *entity.Document) error {
	dataJSON, err := json.Marshal(doc.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Cassandra에서는 LWT (Lightweight Transaction) 사용
	query := fmt.Sprintf(`
		UPDATE %s.%s
		SET data = ?, updated_at = ?, version = ?, metadata = ?
		WHERE id = ?
		IF version = ?
	`, r.keyspace, doc.Collection)

	applied, err := r.session.Query(query,
		string(dataJSON),
		time.Now(),
		doc.Version+1,
		string(metadataJSON),
		doc.ID,
		doc.Version,
	).WithContext(ctx).ScanCAS(&doc.Version) // CAS = Compare And Swap

	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	if !applied {
		return errors.New("optimistic lock error: document was modified by another process")
	}

	doc.Version++
	doc.UpdatedAt = time.Now()

	return nil
}

// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
func (r *CassandraRepository) UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	// Cassandra에서는 batch update가 제한적이므로
	// 먼저 조회한 후 하나씩 업데이트
	docs, err := r.FindAll(ctx, collection, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to find documents: %w", err)
	}

	var count int64
	for _, doc := range docs {
		for key, value := range update {
			doc.Data[key] = value
		}

		if err := r.Update(ctx, doc); err != nil {
			// 일부 실패해도 계속 진행
			continue
		}
		count++
	}

	return count, nil
}

// Replace는 문서를 교체합니다
func (r *CassandraRepository) Replace(ctx context.Context, collection, id string, replacement *entity.Document) error {
	dataJSON, err := json.Marshal(replacement.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	metadataJSON, err := json.Marshal(replacement.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		UPDATE %s.%s
		SET data = ?, updated_at = ?, version = version + 1, metadata = ?
		WHERE id = ?
	`, r.keyspace, collection)

	return r.session.Query(query,
		string(dataJSON),
		time.Now(),
		string(metadataJSON),
		id,
	).WithContext(ctx).Exec()
}

// Delete는 문서를 삭제합니다
func (r *CassandraRepository) Delete(ctx context.Context, collection, id string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.%s WHERE id = ?
	`, r.keyspace, collection)

	return r.session.Query(query, id).WithContext(ctx).Exec()
}

// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
func (r *CassandraRepository) DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	// Cassandra에서는 먼저 조회한 후 삭제
	docs, err := r.FindAll(ctx, collection, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to find documents: %w", err)
	}

	batch := r.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	queryStr := fmt.Sprintf(`
		DELETE FROM %s.%s WHERE id = ?
	`, r.keyspace, collection)

	for _, doc := range docs {
		batch.Query(queryStr, doc.ID)
	}

	if err := r.session.ExecuteBatch(batch); err != nil {
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}

	return int64(len(docs)), nil
}

// ===== 원자적 연산 (Atomic Operations) =====

// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다
func (r *CassandraRepository) FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error) {
	// 먼저 조회
	doc, err := r.FindByID(ctx, collection, id)
	if err != nil {
		return nil, err
	}

	// 업데이트 적용
	for key, value := range update {
		doc.Data[key] = value
	}

	// 업데이트
	if err := r.Update(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
func (r *CassandraRepository) FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error) {
	if err := r.Replace(ctx, collection, id, replacement); err != nil {
		return nil, err
	}

	return r.FindByID(ctx, collection, id)
}

// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
func (r *CassandraRepository) FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error) {
	doc, err := r.FindByID(ctx, collection, id)
	if err != nil {
		return nil, err
	}

	if err := r.Delete(ctx, collection, id); err != nil {
		return nil, err
	}

	return doc, nil
}

// Upsert는 문서가 없으면 생성하고 있으면 업데이트합니다
func (r *CassandraRepository) Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error) {
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
		INSERT INTO %s.%s (id, data, created_at, updated_at, version, metadata)
		VALUES (?, ?, ?, ?, 1, '{}')
	`, r.keyspace, collection)

	now := time.Now()
	err = r.session.Query(query, id, string(updateDataJSON), now, now).WithContext(ctx).Exec()
	if err != nil {
		return "", fmt.Errorf("failed to upsert document: %w", err)
	}

	return id, nil
}

// ===== Helper methods =====

func (r *CassandraRepository) buildWhereClause(filter map[string]interface{}) (string, []interface{}) {
	if len(filter) == 0 {
		return "", nil
	}

	conditions := []string{}
	args := []interface{}{}

	for key, value := range filter {
		if key == "_id" || key == "id" {
			conditions = append(conditions, "id = ?")
			args = append(args, value)
		}
		// Cassandra에서 JSON 필드 검색은 복잡하므로 id만 지원
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

func (r *CassandraRepository) buildOrderBy(sort map[string]int) string {
	if len(sort) == 0 {
		return ""
	}

	orders := []string{}
	for key, direction := range sort {
		dir := "ASC"
		if direction == -1 {
			dir = "DESC"
		}
		orders = append(orders, fmt.Sprintf("%s %s", key, dir))
	}

	return "ORDER BY " + strings.Join(orders, ", ")
}

func (r *CassandraRepository) buildLimit(limit int64) string {
	if limit <= 0 {
		return ""
	}
	return fmt.Sprintf("LIMIT %d", limit)
}

func (r *CassandraRepository) scanDocuments(iter *gocql.Iter, collection string) ([]*entity.Document, error) {
	documents := []*entity.Document{}

	var id, dataStr, metadataStr string
	var createdAt, updatedAt time.Time
	var version int

	for iter.Scan(&id, &dataStr, &createdAt, &updatedAt, &version, &metadataStr) {
		doc := &entity.Document{
			ID:         id,
			Collection: collection,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			Version:    version,
		}

		if err := json.Unmarshal([]byte(dataStr), &doc.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataStr), &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		documents = append(documents, doc)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return documents, nil
}

// ===== 집계 (Aggregation) =====

// Aggregate는 집계 파이프라인을 실행합니다
func (r *CassandraRepository) Aggregate(ctx context.Context, collection string, pipeline []bson.M) ([]map[string]interface{}, error) {
	return nil, errors.New("aggregate is not supported in Cassandra implementation")
}

// Distinct는 고유한 값을 조회합니다
func (r *CassandraRepository) Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error) {
	// Cassandra에서는 DISTINCT가 제한적입니다
	return nil, errors.New("distinct is not fully supported in Cassandra implementation")
}

// Count는 문서 개수를 반환합니다
func (r *CassandraRepository) Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.%s %s
	`, r.keyspace, collection, whereClause)

	var count int64
	err := r.session.Query(query, args...).WithContext(ctx).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// EstimatedDocumentCount는 컬렉션의 추정 문서 개수를 반환합니다
func (r *CassandraRepository) EstimatedDocumentCount(ctx context.Context, collection string) (int64, error) {
	// Cassandra에서는 정확한 COUNT를 사용
	return r.Count(ctx, collection, nil)
}

// ===== 벌크 작업 (Bulk Operations) =====

// BulkWrite는 여러 작업을 한 번에 실행합니다
func (r *CassandraRepository) BulkWrite(ctx context.Context, operations []*repository.BulkOperation) (*repository.BulkResult, error) {
	if len(operations) == 0 {
		return &repository.BulkResult{}, nil
	}

	batch := r.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
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

			queryStr := fmt.Sprintf(`
				INSERT INTO %s.%s (id, data, created_at, updated_at, version, metadata)
				VALUES (?, ?, ?, ?, ?, ?)
			`, r.keyspace, op.Collection)

			batch.Query(queryStr,
				op.Document.ID,
				string(dataJSON),
				op.Document.CreatedAt,
				op.Document.UpdatedAt,
				op.Document.Version,
				string(metadataJSON),
			)

			result.InsertedCount++

		case "delete":
			queryStr := fmt.Sprintf(`
				DELETE FROM %s.%s WHERE id = ?
			`, r.keyspace, op.Collection)

			// Filter에서 ID 추출
			if id, ok := op.Filter["id"]; ok {
				batch.Query(queryStr, id)
				result.DeletedCount++
			}
		}

		result.UpsertedIDs[i] = i
	}

	if err := r.session.ExecuteBatch(batch); err != nil {
		return nil, fmt.Errorf("failed to execute batch: %w", err)
	}

	return result, nil
}

// ===== 인덱스 관리 (Index Management) =====

// CreateIndex는 단일 인덱스를 생성합니다
func (r *CassandraRepository) CreateIndex(ctx context.Context, collection string, model repository.IndexModel) (string, error) {
	indexName := model.Options.Name
	if indexName == "" {
		indexName = fmt.Sprintf("idx_%s_%v", collection, time.Now().Unix())
	}

	// Cassandra에서는 단일 컬럼만 인덱싱 가능
	var indexKey string
	for key := range model.Keys {
		indexKey = key
		break
	}

	query := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s ON %s.%s (%s)
	`, indexName, r.keyspace, collection, indexKey)

	return indexName, r.session.Query(query).WithContext(ctx).Exec()
}

// CreateIndexes는 여러 인덱스를 생성합니다
func (r *CassandraRepository) CreateIndexes(ctx context.Context, collection string, models []repository.IndexModel) ([]string, error) {
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
func (r *CassandraRepository) DropIndex(ctx context.Context, collection, indexName string) error {
	query := fmt.Sprintf(`DROP INDEX IF EXISTS %s.%s`, r.keyspace, indexName)

	return r.session.Query(query).WithContext(ctx).Exec()
}

// ListIndexes는 컬렉션의 인덱스 목록을 반환합니다
func (r *CassandraRepository) ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	// Cassandra system tables 조회
	query := `
		SELECT index_name, column_name
		FROM system_schema.indexes
		WHERE keyspace_name = ? AND table_name = ?
	`

	iter := r.session.Query(query, r.keyspace, collection).WithContext(ctx).Iter()
	defer iter.Close()

	indexes := []map[string]interface{}{}
	var indexName, columnName string

	for iter.Scan(&indexName, &columnName) {
		indexes = append(indexes, map[string]interface{}{
			"name":   indexName,
			"column": columnName,
		})
	}

	return indexes, nil
}

// ===== 컬렉션 관리 (Collection Management) =====

// CreateCollection은 컬렉션을 생성합니다
func (r *CassandraRepository) CreateCollection(ctx context.Context, name string) error {
	return r.ensureTableExists(ctx, name)
}

// DropCollection은 컬렉션을 삭제합니다
func (r *CassandraRepository) DropCollection(ctx context.Context, name string) error {
	query := fmt.Sprintf(`DROP TABLE IF EXISTS %s.%s`, r.keyspace, name)

	return r.session.Query(query).WithContext(ctx).Exec()
}

// RenameCollection은 컬렉션 이름을 변경합니다
func (r *CassandraRepository) RenameCollection(ctx context.Context, oldName, newName string) error {
	// Cassandra에서는 테이블 이름 변경이 지원되지 않습니다
	return errors.New("rename collection is not supported in Cassandra")
}

// ListCollections는 데이터베이스의 컬렉션 목록을 반환합니다
func (r *CassandraRepository) ListCollections(ctx context.Context) ([]string, error) {
	query := `
		SELECT table_name
		FROM system_schema.tables
		WHERE keyspace_name = ?
	`

	iter := r.session.Query(query, r.keyspace).WithContext(ctx).Iter()
	defer iter.Close()

	collections := []string{}
	var name string

	for iter.Scan(&name) {
		collections = append(collections, name)
	}

	return collections, nil
}

// CollectionExists는 컬렉션이 존재하는지 확인합니다
func (r *CassandraRepository) CollectionExists(ctx context.Context, name string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM system_schema.tables
		WHERE keyspace_name = ? AND table_name = ?
	`

	var count int
	err := r.session.Query(query, r.keyspace, name).WithContext(ctx).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}

	return count > 0, nil
}

// ===== Change Streams =====

// Watch는 컬렉션의 변경 사항을 실시간으로 감지합니다
func (r *CassandraRepository) Watch(ctx context.Context, collection string, pipeline []bson.M) (*mongo.ChangeStream, error) {
	return nil, errors.New("watch is not supported in Cassandra implementation")
}

// ===== 트랜잭션 (Transaction) =====

// WithTransaction은 트랜잭션 내에서 함수를 실행합니다
func (r *CassandraRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Cassandra에서는 전통적인 트랜잭션이 제한적입니다
	// Batch를 사용하거나 LWT (Lightweight Transactions) 사용
	return fn(ctx)
}

// ===== Raw Query Execution =====

// ExecuteRawQuery는 데이터베이스별 raw query를 실행합니다
func (r *CassandraRepository) ExecuteRawQuery(ctx context.Context, query interface{}) (interface{}, error) {
	cqlQuery, ok := query.(string)
	if !ok {
		return nil, errors.New("query must be a string for Cassandra")
	}

	iter := r.session.Query(cqlQuery).WithContext(ctx).Iter()
	defer iter.Close()

	// 컬럼명 가져오기
	columns := iter.Columns()

	results := []map[string]interface{}{}
	row := make(map[string]interface{})

	for iter.MapScan(row) {
		result := make(map[string]interface{})
		for _, col := range columns {
			result[col.Name] = row[col.Name]
		}
		results = append(results, result)
		row = make(map[string]interface{})
	}

	return results, iter.Close()
}

// ExecuteRawQueryWithResult는 raw query를 실행하고 결과를 특정 타입으로 반환합니다
func (r *CassandraRepository) ExecuteRawQueryWithResult(ctx context.Context, query interface{}, result interface{}) error {
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
func (r *CassandraRepository) HealthCheck(ctx context.Context) error {
	query := "SELECT now() FROM system.local"
	var timestamp time.Time
	return r.session.Query(query).WithContext(ctx).Scan(&timestamp)
}
