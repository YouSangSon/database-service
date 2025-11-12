package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// DocumentRepository는 MongoDB 기반 문서 저장소입니다
type DocumentRepository struct {
	client   *mongo.Client
	database *mongo.Database
	metrics  *metrics.Metrics
}

// documentModel은 MongoDB에 저장되는 문서 모델입니다
type documentModel struct {
	ID         primitive.ObjectID     `bson:"_id,omitempty"`
	Collection string                 `bson:"collection"`
	Data       map[string]interface{} `bson:"data"`
	Version    int                    `bson:"version"`
	CreatedAt  time.Time              `bson:"created_at"`
	UpdatedAt  time.Time              `bson:"updated_at"`
}

// Config는 MongoDB 설정입니다
type Config struct {
	URI            string
	Database       string
	MaxPoolSize    uint64
	MinPoolSize    uint64
	MaxConnecting  uint64
	ConnectTimeout time.Duration
	Timeout        time.Duration
}

// NewDocumentRepository는 새로운 MongoDB 문서 저장소를 생성합니다
func NewDocumentRepository(cfg *Config) (repository.DocumentRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxConnecting(cfg.MaxConnecting).
		SetServerSelectionTimeout(cfg.ConnectTimeout).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetSocketTimeout(cfg.Timeout).
		SetReadPreference(readpref.Primary())

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// 연결 확인
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)

	return &DocumentRepository{
		client:   client,
		database: database,
		metrics:  metrics.GetMetrics(),
	}, nil
}

// Save는 문서를 저장합니다
func (r *DocumentRepository) Save(ctx context.Context, doc *entity.Document) error {
	start := time.Now()
	collection := doc.Collection()

	defer func() {
		duration := time.Since(start)
		status := "success"
		r.metrics.RecordDBOperation("save", collection, status, duration)
		logger.Debug(ctx, "document saved",
			zap.String("collection", collection),
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

	return nil
}

// FindByID는 ID로 문서를 조회합니다
func (r *DocumentRepository) FindByID(ctx context.Context, collection, id string) (*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find", collection, "success", duration)
		logger.Debug(ctx, "document found",
			zap.String("collection", collection),
			zap.String("id", id),
			zap.Duration("duration", duration),
		)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	var model documentModel
	if err := coll.FindOne(ctx, filter).Decode(&model); err != nil {
		if err == mongo.ErrNoDocuments {
			r.metrics.RecordDBOperation("find", collection, "not_found", time.Since(start))
			return nil, entity.ErrDocumentNotFound
		}
		r.metrics.RecordDBOperation("find", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find document",
			zap.String("collection", collection),
			zap.String("id", id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	doc := entity.ReconstructDocument(
		model.ID.Hex(),
		model.Collection,
		model.Data,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)

	return doc, nil
}

// Update는 문서를 업데이트합니다 (낙관적 잠금 포함)
func (r *DocumentRepository) Update(ctx context.Context, doc *entity.Document) error {
	start := time.Now()
	collection := doc.Collection()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("update", collection, "success", duration)
		logger.Debug(ctx, "document updated",
			zap.String("collection", collection),
			zap.String("id", doc.ID()),
			zap.Duration("duration", duration),
		)
	}()

	objectID, err := primitive.ObjectIDFromHex(doc.ID())
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)

	// 낙관적 잠금: 현재 버전과 일치하는 문서만 업데이트
	filter := bson.M{
		"_id":     objectID,
		"version": doc.Version() - 1, // 업데이트 전 버전
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
		logger.Error(ctx, "failed to update document",
			zap.String("collection", collection),
			zap.String("id", doc.ID()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update document: %w", err)
	}

	if result.MatchedCount == 0 {
		r.metrics.RecordDBOperation("update", collection, "conflict", time.Since(start))
		return entity.ErrVersionConflict
	}

	return nil
}

// Delete는 문서를 삭제합니다
func (r *DocumentRepository) Delete(ctx context.Context, collection, id string) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("delete", collection, "success", duration)
		logger.Debug(ctx, "document deleted",
			zap.String("collection", collection),
			zap.String("id", id),
			zap.Duration("duration", duration),
		)
	}()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	coll := r.database.Collection(collection)
	filter := bson.M{"_id": objectID}

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		r.metrics.RecordDBOperation("delete", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to delete document",
			zap.String("collection", collection),
			zap.String("id", id),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.DeletedCount == 0 {
		r.metrics.RecordDBOperation("delete", collection, "not_found", time.Since(start))
		return entity.ErrDocumentNotFound
	}

	return nil
}

// FindAll은 컬렉션의 모든 문서를 조회합니다
func (r *DocumentRepository) FindAll(ctx context.Context, collection string, filter map[string]interface{}) ([]*entity.Document, error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		r.metrics.RecordDBOperation("find_all", collection, "success", duration)
		logger.Debug(ctx, "documents found",
			zap.String("collection", collection),
			zap.Duration("duration", duration),
		)
	}()

	coll := r.database.Collection(collection)

	var bsonFilter bson.M
	if filter != nil {
		bsonFilter = bson.M(filter)
	} else {
		bsonFilter = bson.M{}
	}

	cursor, err := coll.Find(ctx, bsonFilter)
	if err != nil {
		r.metrics.RecordDBOperation("find_all", collection, "error", time.Since(start))
		logger.Error(ctx, "failed to find documents",
			zap.String("collection", collection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*entity.Document
	for cursor.Next(ctx) {
		var model documentModel
		if err := cursor.Decode(&model); err != nil {
			logger.Warn(ctx, "failed to decode document", zap.Error(err))
			continue
		}

		doc := entity.ReconstructDocument(
			model.ID.Hex(),
			model.Collection,
			model.Data,
			model.Version,
			model.CreatedAt,
			model.UpdatedAt,
		)
		documents = append(documents, doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return documents, nil
}

// Count는 문서 개수를 반환합니다
func (r *DocumentRepository) Count(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	coll := r.database.Collection(collection)

	var bsonFilter bson.M
	if filter != nil {
		bsonFilter = bson.M(filter)
	} else {
		bsonFilter = bson.M{}
	}

	count, err := coll.CountDocuments(ctx, bsonFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// HealthCheck는 저장소의 상태를 확인합니다
func (r *DocumentRepository) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx, readpref.Primary())
}

// Close는 MongoDB 연결을 종료합니다
func (r *DocumentRepository) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}
