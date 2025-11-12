package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/infrastructure/messaging/kafka"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"github.com/YouSangSon/database-service/internal/pkg/vault"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.uber.org/zap"
)

// MongoDBCommandRepository는 MongoDB 기반 쓰기 전용 저장소입니다 (CQRS Write Side)
// Primary 노드에만 연결하여 쓰기 작업을 처리합니다
type MongoDBCommandRepository struct {
	client        *mongo.Client
	database      *mongo.Database
	metrics       *metrics.Metrics
	cdcPublisher  *kafka.CDCPublisher
	vaultClient   *vault.Client
	cdcEnabled    bool
	writeOptions  *repository.WriteOptions
}

// CommandConfig는 쓰기 저장소 설정입니다
type CommandConfig struct {
	URI              string
	Database         string
	MaxPoolSize      uint64
	MinPoolSize      uint64
	MaxConnecting    uint64
	ConnectTimeout   time.Duration
	Timeout          time.Duration
	WriteConcern     string // "majority", "1", "2"
	RetryWrites      bool
	CDCEnabled       bool
	CDCPublisher     *kafka.CDCPublisher
	VaultClient      *vault.Client
}

// NewMongoDBCommandRepository는 새로운 MongoDB 쓰기 저장소를 생성합니다
func NewMongoDBCommandRepository(ctx context.Context, cfg *CommandConfig) (repository.DocumentCommandRepository, error) {
	logger.Info(ctx, "initializing MongoDB command repository",
		zap.String("database", cfg.Database),
		zap.String("write_concern", cfg.WriteConcern),
	)

	// Write Concern 설정
	var wc *writeconcern.WriteConcern
	switch cfg.WriteConcern {
	case "majority":
		wc = writeconcern.New(writeconcern.WMajority())
	case "1":
		wc = writeconcern.New(writeconcern.W(1))
	case "2":
		wc = writeconcern.New(writeconcern.W(2))
	default:
		wc = writeconcern.New(writeconcern.WMajority())
	}

	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxConnecting(cfg.MaxConnecting).
		SetServerSelectionTimeout(cfg.ConnectTimeout).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetSocketTimeout(cfg.Timeout).
		SetReadPreference(readpref.Primary()). // 쓰기는 항상 Primary
		SetWriteConcern(wc).
		SetRetryWrites(cfg.RetryWrites)

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB primary: %w", err)
	}

	// Primary 노드 연결 확인
	if err := client.Ping(connectCtx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB primary: %w", err)
	}

	database := client.Database(cfg.Database)

	logger.Info(ctx, "MongoDB command repository initialized successfully")

	return &MongoDBCommandRepository{
		client:       client,
		database:     database,
		metrics:      metrics.GetMetrics(),
		cdcPublisher: cfg.CDCPublisher,
		vaultClient:  cfg.VaultClient,
		cdcEnabled:   cfg.CDCEnabled,
		writeOptions: &repository.WriteOptions{
			WriteConcern: cfg.WriteConcern,
			RetryWrites:  cfg.RetryWrites,
			PublishCDC:   cfg.CDCEnabled,
		},
	}, nil
}

// Save는 문서를 저장합니다
func (r *MongoDBCommandRepository) Save(ctx context.Context, doc *entity.Document) error {
	start := time.Now()
	collection := doc.Collection()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("save", collection, "success", duration)
		logger.Debug(ctx, "document saved",
			zap.String("collection", collection),
			zap.String("id", doc.ID()),
			zap.Duration("duration", duration),
		)
	}()

	model := &documentModel{
		Collection: doc.Collection(),
		Data:       doc.Data(),
		Version:    doc.Version(),
		CreatedAt:  doc.CreatedAt(),
		UpdatedAt:  doc.UpdatedAt(),
	}

	coll := r.database.Collection(collection)
	result, err := coll.InsertOne(ctx, model)
	if err != nil {
		r.metrics.RecordDBOperation("save", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to save document",
			zap.String("collection", collection),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save document: %w", err)
	}

	// ID 설정
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		doc.SetID(oid.Hex())
	}

	// CDC 이벤트 발행
	if r.cdcEnabled && r.cdcPublisher != nil {
		if err := r.cdcPublisher.PublishDocumentCreated(ctx, doc); err != nil {
			logger.Warn(ctx, "failed to publish CDC event", zap.Error(err))
			// CDC 실패해도 저장은 성공으로 처리
		}
	}

	return nil
}

// SaveMany는 여러 문서를 한 번에 저장합니다
func (r *MongoDBCommandRepository) SaveMany(ctx context.Context, docs []*entity.Document) error {
	if len(docs) == 0 {
		return nil
	}

	start := time.Now()
	collection := docs[0].Collection()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("save_many", collection, "success", duration)
		logger.Debug(ctx, "documents saved",
			zap.String("collection", collection),
			zap.Int("count", len(docs)),
			zap.Duration("duration", duration),
		)
	}()

	// 모델 변환
	models := make([]interface{}, len(docs))
	for i, doc := range docs {
		models[i] = &documentModel{
			Collection: doc.Collection(),
			Data:       doc.Data(),
			Version:    doc.Version(),
			CreatedAt:  doc.CreatedAt(),
			UpdatedAt:  doc.UpdatedAt(),
		}
	}

	coll := r.database.Collection(collection)
	opts := options.InsertMany().SetOrdered(false) // 에러 발생해도 계속 진행

	results, err := coll.InsertMany(ctx, models, opts)
	if err != nil {
		// Bulk insert는 부분 성공 가능
		bulkErr, ok := err.(mongo.BulkWriteException)
		if ok {
			logger.Warn(ctx, "bulk insert partially succeeded",
				zap.Int("inserted", len(bulkErr.WriteErrors)),
				zap.Int("total", len(docs)),
			)
		} else {
			r.metrics.RecordDBOperation("save_many", collection, "error", time.Since(start))
			return fmt.Errorf("failed to save documents: %w", err)
		}
	}

	// ID 설정
	for i, id := range results.InsertedIDs {
		if oid, ok := id.(primitive.ObjectID); ok {
			docs[i].SetID(oid.Hex())
		}
	}

	// CDC 이벤트 발행 (배치)
	if r.cdcEnabled && r.cdcPublisher != nil {
		for _, doc := range docs {
			if err := r.cdcPublisher.PublishDocumentCreated(ctx, doc); err != nil {
				logger.Warn(ctx, "failed to publish CDC event", zap.Error(err))
			}
		}
	}

	return nil
}

// Update는 문서를 업데이트합니다 (낙관적 잠금 포함)
func (r *MongoDBCommandRepository) Update(ctx context.Context, doc *entity.Document) error {
	start := time.Now()
	collection := doc.Collection()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("update", collection, "success", duration)
	}()

	objectID, err := primitive.ObjectIDFromHex(doc.ID())
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)

	// 낙관적 잠금: 현재 버전과 일치하는 문서만 업데이트
	filter := bson.M{
		"_id":     objectID,
		"version": doc.Version() - 1,
	}

	update := bson.M{
		"$set": bson.M{
			"data":       doc.Data(),
			"version":    doc.Version(),
			"updated_at": doc.UpdatedAt(),
		},
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		r.metrics.RecordDBOperation("update", collection, "error", time.Since(start))
		return fmt.Errorf("failed to update document: %w", err)
	}

	if result.MatchedCount == 0 {
		r.metrics.RecordDBOperation("update", collection, "conflict", time.Since(start))
		return entity.ErrVersionConflict
	}

	// CDC 이벤트 발행
	if r.cdcEnabled && r.cdcPublisher != nil {
		if err := r.cdcPublisher.PublishDocumentUpdated(ctx, doc); err != nil {
			logger.Warn(ctx, "failed to publish CDC event", zap.Error(err))
		}
	}

	return nil
}

// UpdateMany는 필터와 일치하는 여러 문서를 업데이트합니다
func (r *MongoDBCommandRepository) UpdateMany(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("update_many", collection, "success", duration)
	}()

	coll := r.database.Collection(collection)

	var bsonFilter bson.M
	if filter != nil {
		bsonFilter = bson.M(filter)
	} else {
		bsonFilter = bson.M{}
	}

	bsonUpdate := bson.M{"$set": update}
	bsonUpdate["$set"].(bson.M)["updated_at"] = time.Now()

	result, err := coll.UpdateMany(ctx, bsonFilter, bsonUpdate)
	if err != nil {
		r.metrics.RecordDBOperation("update_many", collection, "error", time.Since(start))
		return 0, fmt.Errorf("failed to update documents: %w", err)
	}

	logger.Info(ctx, "documents updated",
		zap.String("collection", collection),
		zap.Int64("modified_count", result.ModifiedCount),
	)

	return result.ModifiedCount, nil
}

// Replace는 문서를 완전히 교체합니다
func (r *MongoDBCommandRepository) Replace(ctx context.Context, collection, id string, replacement *entity.Document) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("replace", collection, "success", duration)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	model := &documentModel{
		ID:         objectID,
		Collection: collection,
		Data:       replacement.Data(),
		Version:    replacement.Version(),
		CreatedAt:  replacement.CreatedAt(),
		UpdatedAt:  time.Now(),
	}

	result, err := coll.ReplaceOne(ctx, filter, model)
	if err != nil {
		r.metrics.RecordDBOperation("replace", collection, "error", time.Since(start))
		return fmt.Errorf("failed to replace document: %w", err)
	}

	if result.MatchedCount == 0 {
		return entity.ErrDocumentNotFound
	}

	// CDC 이벤트 발행
	if r.cdcEnabled && r.cdcPublisher != nil {
		if err := r.cdcPublisher.PublishDocumentUpdated(ctx, replacement); err != nil {
			logger.Warn(ctx, "failed to publish CDC event", zap.Error(err))
		}
	}

	return nil
}

// Delete는 문서를 삭제합니다
func (r *MongoDBCommandRepository) Delete(ctx context.Context, collection, id string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("delete", collection, "success", duration)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	// CDC를 위해 삭제 전 문서 조회
	var deletedDoc *entity.Document
	if r.cdcEnabled && r.cdcPublisher != nil {
		coll := r.database.Collection(collection)
		var model documentModel
		err := coll.FindOne(ctx, bson.M{"_id": objectID}).Decode(&model)
		if err == nil {
			deletedDoc = entity.ReconstructDocument(
				model.ID.Hex(),
				model.Collection,
				model.Data,
				model.Version,
				model.CreatedAt,
				model.UpdatedAt,
			)
		}
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		r.metrics.RecordDBOperation("delete", collection, "error", time.Since(start))
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.DeletedCount == 0 {
		r.metrics.RecordDBOperation("delete", collection, "not_found", time.Since(start))
		return entity.ErrDocumentNotFound
	}

	// CDC 이벤트 발행
	if deletedDoc != nil && r.cdcPublisher != nil {
		if err := r.cdcPublisher.PublishDocumentDeleted(ctx, deletedDoc); err != nil {
			logger.Warn(ctx, "failed to publish CDC event", zap.Error(err))
		}
	}

	return nil
}

// DeleteMany는 필터와 일치하는 여러 문서를 삭제합니다
func (r *MongoDBCommandRepository) DeleteMany(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("delete_many", collection, "success", duration)
	}()

	coll := r.database.Collection(collection)

	var bsonFilter bson.M
	if filter != nil {
		bsonFilter = bson.M(filter)
	} else {
		bsonFilter = bson.M{}
	}

	result, err := coll.DeleteMany(ctx, bsonFilter)
	if err != nil {
		r.metrics.RecordDBOperation("delete_many", collection, "error", time.Since(start))
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}

	logger.Info(ctx, "documents deleted",
		zap.String("collection", collection),
		zap.Int64("deleted_count", result.DeletedCount),
	)

	return result.DeletedCount, nil
}

// FindAndUpdate는 문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다
func (r *MongoDBCommandRepository) FindAndUpdate(ctx context.Context, collection, id string, update map[string]interface{}) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_and_update", collection, "success", duration)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	bsonUpdate := bson.M{"$set": update}
	bsonUpdate["$set"].(bson.M)["updated_at"] = time.Now()
	bsonUpdate["$inc"] = bson.M{"version": 1} // 버전 증가

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var model documentModel
	err = coll.FindOneAndUpdate(ctx, filter, bsonUpdate, opts).Decode(&model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, entity.ErrDocumentNotFound
		}
		r.metrics.RecordDBOperation("find_and_update", collection, "error", time.Since(start))
		return nil, fmt.Errorf("failed to find and update document: %w", err)
	}

	doc := entity.ReconstructDocument(
		model.ID.Hex(),
		model.Collection,
		model.Data,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)

	// CDC 이벤트 발행
	if r.cdcEnabled && r.cdcPublisher != nil {
		if err := r.cdcPublisher.PublishDocumentUpdated(ctx, doc); err != nil {
			logger.Warn(ctx, "failed to publish CDC event", zap.Error(err))
		}
	}

	return doc, nil
}

// FindOneAndReplace는 문서를 찾아서 교체하고 교체된 문서를 반환합니다
func (r *MongoDBCommandRepository) FindOneAndReplace(ctx context.Context, collection, id string, replacement *entity.Document) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_and_replace", collection, "success", duration)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	model := &documentModel{
		ID:         objectID,
		Collection: collection,
		Data:       replacement.Data(),
		Version:    replacement.Version() + 1,
		CreatedAt:  replacement.CreatedAt(),
		UpdatedAt:  time.Now(),
	}

	opts := options.FindOneAndReplace().SetReturnDocument(options.After)

	var resultModel documentModel
	err = coll.FindOneAndReplace(ctx, filter, model, opts).Decode(&resultModel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, entity.ErrDocumentNotFound
		}
		r.metrics.RecordDBOperation("find_and_replace", collection, "error", time.Since(start))
		return nil, fmt.Errorf("failed to find and replace document: %w", err)
	}

	doc := entity.ReconstructDocument(
		resultModel.ID.Hex(),
		resultModel.Collection,
		resultModel.Data,
		resultModel.Version,
		resultModel.CreatedAt,
		resultModel.UpdatedAt,
	)

	return doc, nil
}

// FindOneAndDelete는 문서를 찾아서 삭제하고 삭제된 문서를 반환합니다
func (r *MongoDBCommandRepository) FindOneAndDelete(ctx context.Context, collection, id string) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_and_delete", collection, "success", duration)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	var model documentModel
	err = coll.FindOneAndDelete(ctx, filter).Decode(&model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, entity.ErrDocumentNotFound
		}
		r.metrics.RecordDBOperation("find_and_delete", collection, "error", time.Since(start))
		return nil, fmt.Errorf("failed to find and delete document: %w", err)
	}

	doc := entity.ReconstructDocument(
		model.ID.Hex(),
		model.Collection,
		model.Data,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)

	// CDC 이벤트 발행
	if r.cdcEnabled && r.cdcPublisher != nil {
		if err := r.cdcPublisher.PublishDocumentDeleted(ctx, doc); err != nil {
			logger.Warn(ctx, "failed to publish CDC event", zap.Error(err))
		}
	}

	return doc, nil
}

// Upsert는 문서가 없으면 생성하고 있으면 업데이트합니다
func (r *MongoDBCommandRepository) Upsert(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) (string, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("upsert", collection, "success", duration)
	}()

	coll := r.database.Collection(collection)

	bsonFilter := bson.M(filter)
	bsonUpdate := bson.M{"$set": update}
	bsonUpdate["$set"].(bson.M)["updated_at"] = time.Now()
	bsonUpdate["$setOnInsert"] = bson.M{"created_at": time.Now(), "version": 1}

	opts := options.Update().SetUpsert(true)

	result, err := coll.UpdateOne(ctx, bsonFilter, bsonUpdate, opts)
	if err != nil {
		r.metrics.RecordDBOperation("upsert", collection, "error", time.Since(start))
		return "", fmt.Errorf("failed to upsert document: %w", err)
	}

	var id string
	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			id = oid.Hex()
		}
	}

	return id, nil
}

// Note: BulkWrite, CreateIndex, CreateIndexes, DropIndex, CreateCollection,
// DropCollection, RenameCollection, WithTransaction, EnableChangeDataCapture,
// DisableChangeDataCapture, ExecuteWriteCommand 메서드들은 기존 파일들을 참조하여 구현하거나
// 별도 파일로 분리할 수 있습니다.

// Placeholder methods for interface compliance
func (r *MongoDBCommandRepository) BulkWrite(ctx context.Context, operations []*repository.BulkOperation) (*repository.BulkResult, error) {
	// TODO: Implement using bulk_operations.go
	return nil, fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) CreateIndex(ctx context.Context, collection string, model repository.IndexModel) (string, error) {
	// TODO: Implement using index_operations.go
	return "", fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) CreateIndexes(ctx context.Context, collection string, models []repository.IndexModel) ([]string, error) {
	// TODO: Implement using index_operations.go
	return nil, fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) DropIndex(ctx context.Context, collection, indexName string) error {
	// TODO: Implement using index_operations.go
	return fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) CreateCollection(ctx context.Context, name string) error {
	// TODO: Implement using collection_operations.go
	return fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) DropCollection(ctx context.Context, name string) error {
	// TODO: Implement using collection_operations.go
	return fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) RenameCollection(ctx context.Context, oldName, newName string) error {
	// TODO: Implement using collection_operations.go
	return fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// TODO: Implement using bulk_and_transaction_operations.go
	return fmt.Errorf("not implemented yet")
}

func (r *MongoDBCommandRepository) EnableChangeDataCapture(ctx context.Context, collections []string) error {
	// TODO: Implement using change_streams.go
	r.cdcEnabled = true
	return nil
}

func (r *MongoDBCommandRepository) DisableChangeDataCapture(ctx context.Context) error {
	r.cdcEnabled = false
	return nil
}

func (r *MongoDBCommandRepository) ExecuteWriteCommand(ctx context.Context, command interface{}) (interface{}, error) {
	// TODO: Implement using raw_query_operations.go
	return nil, fmt.Errorf("not implemented yet")
}

// HealthCheck는 쓰기 저장소의 상태를 확인합니다
func (r *MongoDBCommandRepository) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx, readpref.Primary())
}

// Close는 MongoDB 연결을 종료합니다
func (r *MongoDBCommandRepository) Close(ctx context.Context) error {
	logger.Info(ctx, "closing MongoDB command repository connection")
	return r.client.Disconnect(ctx)
}
