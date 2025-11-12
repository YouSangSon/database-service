package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB는 MongoDB 데이터베이스 구현체입니다
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	config   *database.Config
}

// NewMongoDB는 새로운 MongoDB 인스턴스를 생성합니다
func NewMongoDB(config *database.Config) *MongoDB {
	return &MongoDB{
		config: config,
	}
}

// Connect는 MongoDB에 연결합니다
func (m *MongoDB) Connect(ctx context.Context) error {
	connectionString := fmt.Sprintf("mongodb://%s:%s@%s:%d",
		m.config.Username,
		m.config.Password,
		m.config.Host,
		m.config.Port,
	)

	// Username이 없으면 인증 없이 연결
	if m.config.Username == "" {
		connectionString = fmt.Sprintf("mongodb://%s:%d", m.config.Host, m.config.Port)
	}

	clientOptions := options.Client().ApplyURI(connectionString)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// 연결 확인
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client
	m.database = client.Database(m.config.Database)

	return nil
}

// Disconnect는 MongoDB 연결을 종료합니다
func (m *MongoDB) Disconnect(ctx context.Context) error {
	if m.client == nil {
		return nil
	}

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	return nil
}

// Ping은 MongoDB 연결 상태를 확인합니다
func (m *MongoDB) Ping(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("client is not connected")
	}

	if err := m.client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return nil
}

// Create는 새로운 문서를 생성합니다
func (m *MongoDB) Create(ctx context.Context, collection string, document interface{}) (string, error) {
	coll := m.database.Collection(collection)

	// 문서에 타임스탬프 추가
	doc := bson.M{
		"data":       document,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	result, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}

	id := result.InsertedID.(primitive.ObjectID).Hex()
	return id, nil
}

// Read는 ID로 문서를 조회합니다
func (m *MongoDB) Read(ctx context.Context, collection string, id string, result interface{}) error {
	coll := m.database.Collection(collection)

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	filter := bson.M{"_id": objectID}

	if err := coll.FindOne(ctx, filter).Decode(result); err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("document not found")
		}
		return fmt.Errorf("failed to read document: %w", err)
	}

	return nil
}

// Update는 기존 문서를 업데이트합니다
func (m *MongoDB) Update(ctx context.Context, collection string, id string, update interface{}) error {
	coll := m.database.Collection(collection)

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	filter := bson.M{"_id": objectID}
	updateDoc := bson.M{
		"$set": bson.M{
			"data":       update,
			"updated_at": time.Now(),
		},
	}

	result, err := coll.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// Delete는 문서를 삭제합니다
func (m *MongoDB) Delete(ctx context.Context, collection string, id string) error {
	coll := m.database.Collection(collection)

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	filter := bson.M{"_id": objectID}

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// List는 컬렉션의 모든 문서를 조회합니다
func (m *MongoDB) List(ctx context.Context, collection string, filter interface{}, results interface{}) error {
	coll := m.database.Collection(collection)

	var bsonFilter bson.M
	if filter != nil {
		bsonFilter = convertToBSON(filter)
	} else {
		bsonFilter = bson.M{}
	}

	cursor, err := coll.Find(ctx, bsonFilter)
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		return fmt.Errorf("failed to decode documents: %w", err)
	}

	return nil
}

// Query는 커스텀 쿼리를 실행합니다
func (m *MongoDB) Query(ctx context.Context, collection string, query interface{}, results interface{}) error {
	coll := m.database.Collection(collection)

	bsonQuery := convertToBSON(query)

	cursor, err := coll.Find(ctx, bsonQuery)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		return fmt.Errorf("failed to decode query results: %w", err)
	}

	return nil
}

// convertToBSON은 map[string]interface{}를 bson.M으로 변환합니다
func convertToBSON(data interface{}) bson.M {
	if m, ok := data.(map[string]interface{}); ok {
		return bson.M(m)
	}
	if m, ok := data.(bson.M); ok {
		return m
	}
	return bson.M{}
}
