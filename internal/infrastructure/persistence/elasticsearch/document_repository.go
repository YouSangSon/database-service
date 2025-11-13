package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ElasticsearchRepository는 Elasticsearch 기반 문서 저장소입니다
type ElasticsearchRepository struct {
	client *elasticsearch.Client
}

// NewElasticsearchRepository는 Elasticsearch 저장소를 생성합니다
func NewElasticsearchRepository(client *elasticsearch.Client) repository.DocumentRepository {
	return &ElasticsearchRepository{client: client}
}

// ensureIndexExists는 인덱스가 존재하는지 확인하고 없으면 생성합니다
func (r *ElasticsearchRepository) ensureIndexExists(ctx context.Context, index string) error {
	res, err := r.client.Indices.Exists([]string{index})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil // 인덱스가 이미 존재
	}

	// 인덱스 생성
	mapping := `{
		"mappings": {
			"properties": {
				"id": {"type": "keyword"},
				"data": {"type": "object", "enabled": true},
				"created_at": {"type": "date"},
				"updated_at": {"type": "date"},
				"version": {"type": "integer"},
				"metadata": {"type": "object", "enabled": true}
			}
		}
	}`

	res, err = r.client.Indices.Create(
		index,
		r.client.Indices.Create.WithContext(ctx),
		r.client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	return nil
}

// ===== 기본 CRUD =====

// Save는 문서를 저장합니다
func (r *ElasticsearchRepository) Save(ctx context.Context, doc *entity.Document) error {
	if err := r.ensureIndexExists(ctx, doc.Collection); err != nil {
		return fmt.Errorf("failed to ensure index exists: %w", err)
	}

	// Elasticsearch 문서 생성
	esDoc := map[string]interface{}{
		"id":         doc.ID,
		"data":       doc.Data,
		"created_at": doc.CreatedAt,
		"updated_at": doc.UpdatedAt,
		"version":    doc.Version,
		"metadata":   doc.Metadata,
	}

	docJSON, err := json.Marshal(esDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	res, err := r.client.Index(
		doc.Collection,
		bytes.NewReader(docJSON),
		r.client.Index.WithContext(ctx),
		r.client.Index.WithDocumentID(doc.ID),
		r.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index document: %s", res.String())
	}

	return nil
}

// SaveMany는 여러 문서를 한 번에 저장합니다
func (r *ElasticsearchRepository) SaveMany(ctx context.Context, docs []*entity.Document) error {
	if len(docs) == 0 {
		return nil
	}

	collection := docs[0].Collection
	if err := r.ensureIndexExists(ctx, collection); err != nil {
		return fmt.Errorf("failed to ensure index exists: %w", err)
	}

	// Bulk API 사용
	var buf bytes.Buffer
	for _, doc := range docs {
		// Action line
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": collection,
				"_id":    doc.ID,
			},
		}
		metaJSON, _ := json.Marshal(meta)
		buf.Write(metaJSON)
		buf.WriteByte('\n')

		// Document line
		esDoc := map[string]interface{}{
			"id":         doc.ID,
			"data":       doc.Data,
			"created_at": doc.CreatedAt,
			"updated_at": doc.UpdatedAt,
			"version":    doc.Version,
			"metadata":   doc.Metadata,
		}
		docJSON, _ := json.Marshal(esDoc)
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	res, err := r.client.Bulk(
		bytes.NewReader(buf.Bytes()),
		r.client.Bulk.WithContext(ctx),
		r.client.Bulk.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to bulk index documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to bulk index documents: %s", res.String())
	}

	return nil
}

// FindByID는 ID로 문서를 조회합니다
func (r *ElasticsearchRepository) FindByID(ctx context.Context, collection, id string) (*entity.Document, error) {
	res, err := r.client.Get(
		collection,
		id,
		r.client.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, errors.New("document not found")
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get document: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	source, ok := result["_source"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid document format")
	}

	return r.parseDocument(source, collection)
}

// FindAll은 컬렉션의 모든 문서를 조회합니다
func (r *ElasticsearchRepository) FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error) {
	query := r.buildQuery(filter)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(collection),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithSize(10000), // 기본 최대 크기
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to search documents: %s", res.String())
	}

	return r.parseSearchResponse(res.Body, collection)
}

// FindWithOptions는 옵션을 사용하여 문서를 조회합니다
func (r *ElasticsearchRepository) FindWithOptions(ctx context.Context, collection string, filter map[string]interface{}, opts *repository.FindOptions) ([]*entity.Document, error) {
	query := r.buildQueryWithOptions(filter, opts)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(collection),
		r.client.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to search documents: %s", res.String())
	}

	return r.parseSearchResponse(res.Body, collection)
}

// Update는 문서를 업데이트합니다
func (r *ElasticsearchRepository) Update(ctx context.Context, doc *entity.Document) error {
	// Optimistic locking with version
	script := map[string]interface{}{
		"script": map[string]interface{}{
			"source": "if (ctx._source.version == params.expected_version) { ctx._source.data = params.data; ctx._source.updated_at = params.updated_at; ctx._source.version = params.new_version; ctx._source.metadata = params.metadata; } else { ctx.op = 'noop'; }",
			"params": map[string]interface{}{
				"expected_version": doc.Version,
				"new_version":      doc.Version + 1,
				"data":             doc.Data,
				"updated_at":       doc.UpdatedAt,
				"metadata":         doc.Metadata,
			},
		},
	}

	scriptJSON, err := json.Marshal(script)
	if err != nil {
		return fmt.Errorf("failed to marshal script: %w", err)
	}

	res, err := r.client.Update(
		doc.Collection,
		doc.ID,
		bytes.NewReader(scriptJSON),
		r.client.Update.WithContext(ctx),
		r.client.Update.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update document: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result["result"] == "noop" {
		return errors.New("optimistic lock error: document was modified by another process")
	}

	doc.Version++
	return nil
}

// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
func (r *ElasticsearchRepository) UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	query := r.buildQuery(filter)

	// Update by query
	script := map[string]interface{}{
		"query":  query["query"],
		"script": r.buildUpdateScript(update),
	}

	scriptJSON, err := json.Marshal(script)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal script: %w", err)
	}

	res, err := r.client.UpdateByQuery(
		[]string{collection},
		r.client.UpdateByQuery.WithContext(ctx),
		r.client.UpdateByQuery.WithBody(bytes.NewReader(scriptJSON)),
		r.client.UpdateByQuery.WithRefresh(true),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to update documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("failed to update documents: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	updated, ok := result["updated"].(float64)
	if !ok {
		return 0, nil
	}

	return int64(updated), nil
}

// Replace는 문서를 교체합니다
func (r *ElasticsearchRepository) Replace(ctx context.Context, collection, id string, replacement *entity.Document) error {
	esDoc := map[string]interface{}{
		"id":         id,
		"data":       replacement.Data,
		"created_at": replacement.CreatedAt,
		"updated_at": replacement.UpdatedAt,
		"version":    replacement.Version,
		"metadata":   replacement.Metadata,
	}

	docJSON, err := json.Marshal(esDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	res, err := r.client.Index(
		collection,
		bytes.NewReader(docJSON),
		r.client.Index.WithContext(ctx),
		r.client.Index.WithDocumentID(id),
		r.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to replace document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to replace document: %s", res.String())
	}

	return nil
}

// Delete는 문서를 삭제합니다
func (r *ElasticsearchRepository) Delete(ctx context.Context, collection, id string) error {
	res, err := r.client.Delete(
		collection,
		id,
		r.client.Delete.WithContext(ctx),
		r.client.Delete.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return errors.New("document not found")
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete document: %s", res.String())
	}

	return nil
}

// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
func (r *ElasticsearchRepository) DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	query := r.buildQuery(filter)

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := r.client.DeleteByQuery(
		[]string{collection},
		bytes.NewReader(queryJSON),
		r.client.DeleteByQuery.WithContext(ctx),
		r.client.DeleteByQuery.WithRefresh(true),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("failed to delete documents: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	deleted, ok := result["deleted"].(float64)
	if !ok {
		return 0, nil
	}

	return int64(deleted), nil
}

// ===== Helper Methods =====

func (r *ElasticsearchRepository) buildQuery(filter map[string]interface{}) map[string]interface{} {
	if len(filter) == 0 {
		return map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
	}

	must := []map[string]interface{}{}
	for key, value := range filter {
		if key == "_id" || key == "id" {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{
					"id": value,
				},
			})
		} else {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{
					fmt.Sprintf("data.%s", key): value,
				},
			})
		}
	}

	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
	}
}

func (r *ElasticsearchRepository) buildQueryWithOptions(filter map[string]interface{}, opts *repository.FindOptions) map[string]interface{} {
	query := r.buildQuery(filter)

	// Sort
	if len(opts.Sort) > 0 {
		sort := []map[string]interface{}{}
		for key, direction := range opts.Sort {
			order := "asc"
			if direction == -1 {
				order = "desc"
			}
			if key == "_id" || key == "id" {
				sort = append(sort, map[string]interface{}{
					"id": map[string]interface{}{"order": order},
				})
			} else {
				sort = append(sort, map[string]interface{}{
					fmt.Sprintf("data.%s.keyword", key): map[string]interface{}{"order": order},
				})
			}
		}
		query["sort"] = sort
	}

	// Limit (size in Elasticsearch)
	if opts.Limit > 0 {
		query["size"] = opts.Limit
	}

	// Skip (from in Elasticsearch)
	if opts.Skip > 0 {
		query["from"] = opts.Skip
	}

	return query
}

func (r *ElasticsearchRepository) buildUpdateScript(update map[string]interface{}) map[string]interface{} {
	params := make(map[string]interface{})
	scriptParts := []string{}

	for key, value := range update {
		paramKey := strings.ReplaceAll(key, ".", "_")
		params[paramKey] = value
		scriptParts = append(scriptParts, fmt.Sprintf("ctx._source.data.%s = params.%s", key, paramKey))
	}

	scriptParts = append(scriptParts, "ctx._source.version++")
	scriptParts = append(scriptParts, "ctx._source.updated_at = params.updated_at")
	params["updated_at"] = "now"

	return map[string]interface{}{
		"source": strings.Join(scriptParts, "; "),
		"params": params,
		"lang":   "painless",
	}
}

func (r *ElasticsearchRepository) parseDocument(source map[string]interface{}, collection string) (*entity.Document, error) {
	doc := &entity.Document{
		Collection: collection,
	}

	if id, ok := source["id"].(string); ok {
		doc.ID = id
	}

	if data, ok := source["data"].(map[string]interface{}); ok {
		doc.Data = data
	}

	if createdAt, ok := source["created_at"].(string); ok {
		// Parse timestamp
		doc.CreatedAt, _ = parseTimestamp(createdAt)
	}

	if updatedAt, ok := source["updated_at"].(string); ok {
		doc.UpdatedAt, _ = parseTimestamp(updatedAt)
	}

	if version, ok := source["version"].(float64); ok {
		doc.Version = int(version)
	}

	if metadata, ok := source["metadata"].(map[string]interface{}); ok {
		doc.Metadata = metadata
	}

	return doc, nil
}

func (r *ElasticsearchRepository) parseSearchResponse(body io.Reader, collection string) ([]*entity.Document, error) {
	var response map[string]interface{}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	hits, ok := response["hits"].(map[string]interface{})
	if !ok {
		return []*entity.Document{}, nil
	}

	hitsList, ok := hits["hits"].([]interface{})
	if !ok {
		return []*entity.Document{}, nil
	}

	documents := []*entity.Document{}
	for _, hit := range hitsList {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		doc, err := r.parseDocument(source, collection)
		if err != nil {
			continue
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

// ===== Atomic Operations =====

// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다
func (r *ElasticsearchRepository) FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error) {
	doc, err := r.FindByID(ctx, collection, id)
	if err != nil {
		return nil, err
	}

	for key, value := range update {
		doc.Data[key] = value
	}

	if err := r.Update(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
func (r *ElasticsearchRepository) FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error) {
	if err := r.Replace(ctx, collection, id, replacement); err != nil {
		return nil, err
	}

	return r.FindByID(ctx, collection, id)
}

// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
func (r *ElasticsearchRepository) FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error) {
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
func (r *ElasticsearchRepository) Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error) {
	if err := r.ensureIndexExists(ctx, collection); err != nil {
		return "", fmt.Errorf("failed to ensure index exists: %w", err)
	}

	id, ok := filter["_id"].(string)
	if !ok {
		id, ok = filter["id"].(string)
		if !ok {
			return "", errors.New("upsert requires 'id' or '_id' in filter")
		}
	}

	updateScript := map[string]interface{}{
		"script": r.buildUpdateScript(update),
		"upsert": map[string]interface{}{
			"id":       id,
			"data":     update,
			"version":  1,
			"metadata": map[string]interface{}{},
		},
	}

	scriptJSON, err := json.Marshal(updateScript)
	if err != nil {
		return "", fmt.Errorf("failed to marshal script: %w", err)
	}

	res, err := r.client.Update(
		collection,
		id,
		bytes.NewReader(scriptJSON),
		r.client.Update.WithContext(ctx),
		r.client.Update.WithRefresh("true"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to upsert document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("failed to upsert document: %s", res.String())
	}

	return id, nil
}

// ===== Aggregation =====

// Aggregate는 집계 파이프라인을 실행합니다
func (r *ElasticsearchRepository) Aggregate(ctx context.Context, collection string, pipeline []bson.M) ([]map[string]interface{}, error) {
	// Elasticsearch aggregations은 MongoDB와 다르지만 유사하게 구현 가능
	// 여기서는 간단한 구현만 제공
	return nil, errors.New("aggregate with MongoDB pipeline is not directly supported in Elasticsearch")
}

// Distinct는 고유한 값을 조회합니다
func (r *ElasticsearchRepository) Distinct(ctx context.Context, collection, field string, filter map[string]interface{}) ([]interface{}, error) {
	query := r.buildQuery(filter)
	query["aggs"] = map[string]interface{}{
		"distinct_values": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": fmt.Sprintf("data.%s.keyword", field),
				"size":  10000,
			},
		},
	}
	query["size"] = 0

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(collection),
		r.client.Search.WithBody(bytes.NewReader(queryJSON)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer res.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	aggs, ok := response["aggregations"].(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	distinctValues, ok := aggs["distinct_values"].(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	buckets, ok := distinctValues["buckets"].([]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	values := []interface{}{}
	for _, bucket := range buckets {
		bucketMap, ok := bucket.(map[string]interface{})
		if !ok {
			continue
		}
		if key, ok := bucketMap["key"]; ok {
			values = append(values, key)
		}
	}

	return values, nil
}

// Count는 문서 개수를 반환합니다
func (r *ElasticsearchRepository) Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	query := r.buildQuery(filter)

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := r.client.Count(
		r.client.Count.WithContext(ctx),
		r.client.Count.WithIndex(collection),
		r.client.Count.WithBody(bytes.NewReader(queryJSON)),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}
	defer res.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	count, ok := response["count"].(float64)
	if !ok {
		return 0, nil
	}

	return int64(count), nil
}

// EstimatedDocumentCount는 컬렉션의 추정 문서 개수를 반환합니다
func (r *ElasticsearchRepository) EstimatedDocumentCount(ctx context.Context, collection string) (int64, error) {
	return r.Count(ctx, collection, nil)
}

// ===== Bulk Operations =====

// BulkWrite는 여러 작업을 한 번에 실행합니다
func (r *ElasticsearchRepository) BulkWrite(ctx context.Context, operations []*repository.BulkOperation) (*repository.BulkResult, error) {
	if len(operations) == 0 {
		return &repository.BulkResult{}, nil
	}

	var buf bytes.Buffer
	result := &repository.BulkResult{
		UpsertedIDs: make(map[int]interface{}),
	}

	for i, op := range operations {
		switch op.Type {
		case "insert":
			if err := r.ensureIndexExists(ctx, op.Collection); err != nil {
				return nil, fmt.Errorf("failed to ensure index exists: %w", err)
			}

			meta := map[string]interface{}{
				"index": map[string]interface{}{
					"_index": op.Collection,
					"_id":    op.Document.ID,
				},
			}
			metaJSON, _ := json.Marshal(meta)
			buf.Write(metaJSON)
			buf.WriteByte('\n')

			esDoc := map[string]interface{}{
				"id":         op.Document.ID,
				"data":       op.Document.Data,
				"created_at": op.Document.CreatedAt,
				"updated_at": op.Document.UpdatedAt,
				"version":    op.Document.Version,
				"metadata":   op.Document.Metadata,
			}
			docJSON, _ := json.Marshal(esDoc)
			buf.Write(docJSON)
			buf.WriteByte('\n')

			result.InsertedCount++

		case "update":
			meta := map[string]interface{}{
				"update": map[string]interface{}{
					"_index": op.Collection,
					"_id":    op.Filter["id"],
				},
			}
			metaJSON, _ := json.Marshal(meta)
			buf.Write(metaJSON)
			buf.WriteByte('\n')

			updateDoc := map[string]interface{}{
				"doc": op.Update,
			}
			docJSON, _ := json.Marshal(updateDoc)
			buf.Write(docJSON)
			buf.WriteByte('\n')

			result.ModifiedCount++

		case "delete":
			meta := map[string]interface{}{
				"delete": map[string]interface{}{
					"_index": op.Collection,
					"_id":    op.Filter["id"],
				},
			}
			metaJSON, _ := json.Marshal(meta)
			buf.Write(metaJSON)
			buf.WriteByte('\n')

			result.DeletedCount++
		}

		result.UpsertedIDs[i] = i
	}

	res, err := r.client.Bulk(
		bytes.NewReader(buf.Bytes()),
		r.client.Bulk.WithContext(ctx),
		r.client.Bulk.WithRefresh("true"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk write: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to bulk write: %s", res.String())
	}

	return result, nil
}

// ===== Index Management =====

// CreateIndex는 단일 인덱스를 생성합니다
func (r *ElasticsearchRepository) CreateIndex(ctx context.Context, collection string, model repository.IndexModel) (string, error) {
	// Elasticsearch는 자동 mapping 생성
	return "auto", nil
}

// CreateIndexes는 여러 인덱스를 생성합니다
func (r *ElasticsearchRepository) CreateIndexes(ctx context.Context, collection string, models []repository.IndexModel) ([]string, error) {
	return []string{"auto"}, nil
}

// DropIndex는 인덱스를 삭제합니다
func (r *ElasticsearchRepository) DropIndex(ctx context.Context, collection, indexName string) error {
	return errors.New("drop specific index not supported in Elasticsearch")
}

// ListIndexes는 컬렉션의 인덱스 목록을 반환합니다
func (r *ElasticsearchRepository) ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	res, err := r.client.Indices.GetMapping(
		r.client.Indices.GetMapping.WithContext(ctx),
		r.client.Indices.GetMapping.WithIndex(collection),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}
	defer res.Body.Close()

	var mapping map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&mapping); err != nil {
		return nil, fmt.Errorf("failed to decode mapping: %w", err)
	}

	return []map[string]interface{}{mapping}, nil
}

// ===== Collection Management =====

// CreateCollection은 컬렉션을 생성합니다
func (r *ElasticsearchRepository) CreateCollection(ctx context.Context, name string) error {
	return r.ensureIndexExists(ctx, name)
}

// DropCollection은 컬렉션을 삭제합니다
func (r *ElasticsearchRepository) DropCollection(ctx context.Context, name string) error {
	res, err := r.client.Indices.Delete(
		[]string{name},
		r.client.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to delete index: %s", res.String())
	}

	return nil
}

// RenameCollection은 컬렉션 이름을 변경합니다
func (r *ElasticsearchRepository) RenameCollection(ctx context.Context, oldName, newName string) error {
	// Elasticsearch에서는 reindex API 사용
	reindexBody := map[string]interface{}{
		"source": map[string]interface{}{
			"index": oldName,
		},
		"dest": map[string]interface{}{
			"index": newName,
		},
	}

	bodyJSON, err := json.Marshal(reindexBody)
	if err != nil {
		return fmt.Errorf("failed to marshal reindex body: %w", err)
	}

	req := esapi.ReindexRequest{
		Body: bytes.NewReader(bodyJSON),
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to reindex: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to reindex: %s", res.String())
	}

	// 기존 인덱스 삭제
	return r.DropCollection(ctx, oldName)
}

// ListCollections는 데이터베이스의 컬렉션 목록을 반환합니다
func (r *ElasticsearchRepository) ListCollections(ctx context.Context) ([]string, error) {
	res, err := r.client.Cat.Indices(
		r.client.Cat.Indices.WithContext(ctx),
		r.client.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list indices: %w", err)
	}
	defer res.Body.Close()

	var indices []map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	collections := []string{}
	for _, index := range indices {
		if indexName, ok := index["index"].(string); ok {
			// 시스템 인덱스 제외
			if !strings.HasPrefix(indexName, ".") {
				collections = append(collections, indexName)
			}
		}
	}

	return collections, nil
}

// CollectionExists는 컬렉션이 존재하는지 확인합니다
func (r *ElasticsearchRepository) CollectionExists(ctx context.Context, name string) (bool, error) {
	res, err := r.client.Indices.Exists(
		[]string{name},
		r.client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

// ===== Change Streams =====

// Watch는 컬렉션의 변경 사항을 실시간으로 감지합니다
func (r *ElasticsearchRepository) Watch(ctx context.Context, collection string, pipeline []bson.M) (*mongo.ChangeStream, error) {
	return nil, errors.New("watch is not supported in Elasticsearch implementation")
}

// ===== Transaction =====

// WithTransaction은 트랜잭션 내에서 함수를 실행합니다
func (r *ElasticsearchRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Elasticsearch는 트랜잭션을 지원하지 않습니다
	return fn(ctx)
}

// ===== Raw Query Execution =====

// ExecuteRawQuery는 데이터베이스별 raw query를 실행합니다
func (r *ElasticsearchRepository) ExecuteRawQuery(ctx context.Context, query interface{}) (interface{}, error) {
	queryMap, ok := query.(map[string]interface{})
	if !ok {
		return nil, errors.New("query must be a map for Elasticsearch")
	}

	queryJSON, err := json.Marshal(queryMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithBody(bytes.NewReader(queryJSON)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ExecuteRawQueryWithResult는 raw query를 실행하고 결과를 특정 타입으로 반환합니다
func (r *ElasticsearchRepository) ExecuteRawQueryWithResult(ctx context.Context, query interface{}, result interface{}) error {
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

// ===== Health Check =====

// HealthCheck는 저장소의 상태를 확인합니다
func (r *ElasticsearchRepository) HealthCheck(ctx context.Context) error {
	res, err := r.client.Ping(r.client.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch is not healthy: %s", res.String())
	}

	return nil
}

// ===== Helper Functions =====

func parseTimestamp(ts string) (time.Time, error) {
	// Try different timestamp formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, ts); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse timestamp: %s", ts)
}
