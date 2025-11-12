package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// Producer는 Kafka 프로듀서입니다
type Producer struct {
	producer sarama.SyncProducer
	async    sarama.AsyncProducer
	config   *ProducerConfig
}

// ProducerConfig는 프로듀서 설정입니다
type ProducerConfig struct {
	Brokers          []string
	ClientID         string
	MaxMessageBytes  int
	RequiredAcks     sarama.RequiredAcks
	Compression      sarama.CompressionCodec
	MaxRetries       int
	RetryBackoff     time.Duration
	EnableIdempotent bool
	UseAsync         bool
}

// NewProducer는 새로운 Kafka 프로듀서를 생성합니다
func NewProducer(cfg *ProducerConfig) (*Producer, error) {
	config := sarama.NewConfig()
	config.ClientID = cfg.ClientID
	config.Producer.RequiredAcks = cfg.RequiredAcks
	config.Producer.Compression = cfg.Compression
	config.Producer.MaxMessageBytes = cfg.MaxMessageBytes
	config.Producer.Retry.Max = cfg.MaxRetries
	config.Producer.Retry.Backoff = cfg.RetryBackoff
	config.Producer.Idempotent = cfg.EnableIdempotent
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	// 버전 설정
	config.Version = sarama.V3_6_0_0

	p := &Producer{
		config: cfg,
	}

	var err error
	if cfg.UseAsync {
		p.async, err = sarama.NewAsyncProducer(cfg.Brokers, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create async producer: %w", err)
		}

		// 에러 및 성공 메시지 처리
		go p.handleAsyncResults()
	} else {
		p.producer, err = sarama.NewSyncProducer(cfg.Brokers, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create sync producer: %w", err)
		}
	}

	logger.Info(context.Background(), "kafka producer initialized",
		logger.Field("brokers", cfg.Brokers),
		logger.Field("client_id", cfg.ClientID),
		logger.Field("async", cfg.UseAsync),
	)

	return p, nil
}

// PublishEvent는 이벤트를 발행합니다
func (p *Producer) PublishEvent(ctx context.Context, topic string, key string, event interface{}) error {
	// 이벤트를 JSON으로 변환
	eventJSON, err := json.Marshal(event)
	if err != nil {
		logger.Error(ctx, "failed to marshal event",
			logger.Field("topic", topic),
			zap.Error(err),
		)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(eventJSON),
		Timestamp: time.Now(),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event_time"),
				Value: []byte(time.Now().Format(time.RFC3339)),
			},
		},
	}

	if p.config.UseAsync {
		// 비동기 전송
		p.async.Input() <- msg
		logger.Debug(ctx, "event sent asynchronously",
			logger.Field("topic", topic),
			logger.Field("key", key),
		)
		return nil
	}

	// 동기 전송
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		logger.Error(ctx, "failed to send event",
			logger.Field("topic", topic),
			logger.Field("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send event: %w", err)
	}

	logger.Info(ctx, "event published successfully",
		logger.Field("topic", topic),
		logger.Field("key", key),
		logger.Field("partition", partition),
		logger.Field("offset", offset),
	)

	return nil
}

// handleAsyncResults는 비동기 프로듀서의 결과를 처리합니다
func (p *Producer) handleAsyncResults() {
	for {
		select {
		case success := <-p.async.Successes():
			logger.Debug(context.Background(), "async event published",
				logger.Field("topic", success.Topic),
				logger.Field("partition", success.Partition),
				logger.Field("offset", success.Offset),
			)

		case err := <-p.async.Errors():
			logger.Error(context.Background(), "async publish failed",
				logger.Field("topic", err.Msg.Topic),
				zap.Error(err.Err),
			)
		}
	}
}

// Close는 프로듀서를 종료합니다
func (p *Producer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	if p.async != nil {
		return p.async.Close()
	}
	return nil
}

// Event Types

// DocumentEvent는 문서 이벤트 기본 구조입니다
type DocumentEvent struct {
	EventID     string                 `json:"event_id"`
	EventType   string                 `json:"event_type"`
	Timestamp   time.Time              `json:"timestamp"`
	DocumentID  string                 `json:"document_id"`
	Collection  string                 `json:"collection"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Version     int                    `json:"version"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// DocumentCreatedEvent는 문서 생성 이벤트입니다
type DocumentCreatedEvent struct {
	DocumentEvent
}

// DocumentUpdatedEvent는 문서 업데이트 이벤트입니다
type DocumentUpdatedEvent struct {
	DocumentEvent
	PreviousVersion int                    `json:"previous_version"`
	Changes         map[string]interface{} `json:"changes,omitempty"`
}

// DocumentDeletedEvent는 문서 삭제 이벤트입니다
type DocumentDeletedEvent struct {
	DocumentEvent
	DeletedAt time.Time `json:"deleted_at"`
}

// CDCPublisher는 Change Data Capture 이벤트를 발행합니다
type CDCPublisher struct {
	producer     *Producer
	topicCreated string
	topicUpdated string
	topicDeleted string
}

// NewCDCPublisher는 새로운 CDC 발행자를 생성합니다
func NewCDCPublisher(producer *Producer, topicCreated, topicUpdated, topicDeleted string) *CDCPublisher {
	return &CDCPublisher{
		producer:     producer,
		topicCreated: topicCreated,
		topicUpdated: topicUpdated,
		topicDeleted: topicDeleted,
	}
}

// PublishDocumentCreated는 문서 생성 이벤트를 발행합니다
func (c *CDCPublisher) PublishDocumentCreated(ctx context.Context, docID, collection string, data map[string]interface{}, version int) error {
	event := DocumentCreatedEvent{
		DocumentEvent: DocumentEvent{
			EventID:    fmt.Sprintf("%s-%d", docID, time.Now().UnixNano()),
			EventType:  "document.created",
			Timestamp:  time.Now(),
			DocumentID: docID,
			Collection: collection,
			Data:       data,
			Version:    version,
		},
	}

	return c.producer.PublishEvent(ctx, c.topicCreated, docID, event)
}

// PublishDocumentUpdated는 문서 업데이트 이벤트를 발행합니다
func (c *CDCPublisher) PublishDocumentUpdated(ctx context.Context, docID, collection string, data map[string]interface{}, version, previousVersion int, changes map[string]interface{}) error {
	event := DocumentUpdatedEvent{
		DocumentEvent: DocumentEvent{
			EventID:    fmt.Sprintf("%s-%d", docID, time.Now().UnixNano()),
			EventType:  "document.updated",
			Timestamp:  time.Now(),
			DocumentID: docID,
			Collection: collection,
			Data:       data,
			Version:    version,
		},
		PreviousVersion: previousVersion,
		Changes:         changes,
	}

	return c.producer.PublishEvent(ctx, c.topicUpdated, docID, event)
}

// PublishDocumentDeleted는 문서 삭제 이벤트를 발행합니다
func (c *CDCPublisher) PublishDocumentDeleted(ctx context.Context, docID, collection string, version int) error {
	event := DocumentDeletedEvent{
		DocumentEvent: DocumentEvent{
			EventID:    fmt.Sprintf("%s-%d", docID, time.Now().UnixNano()),
			EventType:  "document.deleted",
			Timestamp:  time.Now(),
			DocumentID: docID,
			Collection: collection,
			Version:    version,
		},
		DeletedAt: time.Now(),
	}

	return c.producer.PublishEvent(ctx, c.topicDeleted, docID, event)
}
