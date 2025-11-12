package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// Consumer는 Kafka 컨슈머입니다
type Consumer struct {
	consumer sarama.ConsumerGroup
	config   *ConsumerConfig
	handlers map[string]MessageHandler
	mu       sync.RWMutex
	ready    chan bool
}

// ConsumerConfig는 컨슈머 설정입니다
type ConsumerConfig struct {
	Brokers       []string
	GroupID       string
	Topics        []string
	InitialOffset string // "oldest" or "newest"
	SessionTimeout time.Duration
	HeartbeatInterval time.Duration
}

// MessageHandler는 메시지 핸들러 함수 타입입니다
type MessageHandler func(ctx context.Context, msg *sarama.ConsumerMessage) error

// NewConsumer는 새로운 Kafka 컨슈머를 생성합니다
func NewConsumer(cfg *ConsumerConfig) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V3_6_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Return.Errors = true

	// Set initial offset
	if cfg.InitialOffset == "oldest" {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	// Session timeout and heartbeat
	if cfg.SessionTimeout > 0 {
		config.Consumer.Group.Session.Timeout = cfg.SessionTimeout
	} else {
		config.Consumer.Group.Session.Timeout = 10 * time.Second
	}

	if cfg.HeartbeatInterval > 0 {
		config.Consumer.Group.Heartbeat.Interval = cfg.HeartbeatInterval
	} else {
		config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	}

	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	consumer := &Consumer{
		consumer: consumerGroup,
		config:   cfg,
		handlers: make(map[string]MessageHandler),
		ready:    make(chan bool),
	}

	logger.Info(context.Background(), "kafka consumer initialized",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("group_id", cfg.GroupID),
		zap.Strings("topics", cfg.Topics),
	)

	return consumer, nil
}

// RegisterHandler는 특정 토픽에 대한 메시지 핸들러를 등록합니다
func (c *Consumer) RegisterHandler(topic string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = handler
	logger.Info(context.Background(), "registered handler for topic",
		zap.String("topic", topic),
	)
}

// Start는 컨슈머를 시작합니다
func (c *Consumer) Start(ctx context.Context) error {
	// Handle errors
	go func() {
		for err := range c.consumer.Errors() {
			logger.Error(ctx, "consumer error", zap.Error(err))
		}
	}()

	// Start consuming
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			// Consumer group session
			handler := &consumerGroupHandler{
				consumer: c,
				ready:    c.ready,
			}

			if err := c.consumer.Consume(ctx, c.config.Topics, handler); err != nil {
				logger.Error(ctx, "error from consumer", zap.Error(err))
			}

			// Check if context was cancelled
			if ctx.Err() != nil {
				logger.Info(ctx, "consumer context cancelled")
				return
			}

			c.ready = make(chan bool)
		}
	}()

	// Wait for consumer to be ready
	<-c.ready
	logger.Info(ctx, "kafka consumer started and ready")

	// Wait for context cancellation
	<-ctx.Done()
	logger.Info(ctx, "shutting down kafka consumer")

	wg.Wait()

	if err := c.consumer.Close(); err != nil {
		logger.Error(ctx, "error closing consumer", zap.Error(err))
		return err
	}

	logger.Info(ctx, "kafka consumer shut down successfully")
	return nil
}

// Close는 컨슈머를 종료합니다
func (c *Consumer) Close() error {
	return c.consumer.Close()
}

// consumerGroupHandler는 sarama.ConsumerGroupHandler를 구현합니다
type consumerGroupHandler struct {
	consumer *Consumer
	ready    chan bool
}

// Setup is run at the beginning of a new session
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := session.Context()

	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// Log message received
			logger.Debug(ctx, "message received",
				zap.String("topic", message.Topic),
				zap.Int32("partition", message.Partition),
				zap.Int64("offset", message.Offset),
				zap.String("key", string(message.Key)),
			)

			// Get handler for this topic
			h.consumer.mu.RLock()
			handler, exists := h.consumer.handlers[message.Topic]
			h.consumer.mu.RUnlock()

			if !exists {
				logger.Warn(ctx, "no handler registered for topic",
					zap.String("topic", message.Topic),
				)
				session.MarkMessage(message, "")
				continue
			}

			// Process message
			if err := handler(ctx, message); err != nil {
				logger.Error(ctx, "error processing message",
					zap.String("topic", message.Topic),
					zap.Int64("offset", message.Offset),
					zap.Error(err),
				)
				// Optionally: implement retry logic or dead letter queue here
			} else {
				logger.Debug(ctx, "message processed successfully",
					zap.String("topic", message.Topic),
					zap.Int64("offset", message.Offset),
				)
			}

			// Mark message as consumed
			session.MarkMessage(message, "")

		case <-ctx.Done():
			return nil
		}
	}
}

// CDCConsumer는 Change Data Capture 이벤트를 처리하는 컨슈머입니다
type CDCConsumer struct {
	consumer *Consumer
	handlers *CDCHandlers
}

// CDCHandlers는 CDC 이벤트 핸들러들입니다
type CDCHandlers struct {
	OnDocumentCreated func(ctx context.Context, event *DocumentCreatedEvent) error
	OnDocumentUpdated func(ctx context.Context, event *DocumentUpdatedEvent) error
	OnDocumentDeleted func(ctx context.Context, event *DocumentDeletedEvent) error
}

// NewCDCConsumer는 새로운 CDC 컨슈머를 생성합니다
func NewCDCConsumer(cfg *ConsumerConfig, handlers *CDCHandlers) (*CDCConsumer, error) {
	consumer, err := NewConsumer(cfg)
	if err != nil {
		return nil, err
	}

	cdcConsumer := &CDCConsumer{
		consumer: consumer,
		handlers: handlers,
	}

	// Register handlers for each topic
	if handlers.OnDocumentCreated != nil {
		consumer.RegisterHandler(cfg.Topics[0], cdcConsumer.handleCreatedEvent)
	}
	if handlers.OnDocumentUpdated != nil && len(cfg.Topics) > 1 {
		consumer.RegisterHandler(cfg.Topics[1], cdcConsumer.handleUpdatedEvent)
	}
	if handlers.OnDocumentDeleted != nil && len(cfg.Topics) > 2 {
		consumer.RegisterHandler(cfg.Topics[2], cdcConsumer.handleDeletedEvent)
	}

	return cdcConsumer, nil
}

// handleCreatedEvent handles document.created events
func (c *CDCConsumer) handleCreatedEvent(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event DocumentCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal created event: %w", err)
	}

	logger.Info(ctx, "processing document.created event",
		zap.String("event_id", event.EventID),
		zap.String("document_id", event.DocumentID),
		zap.String("collection", event.Collection),
	)

	if c.handlers.OnDocumentCreated != nil {
		return c.handlers.OnDocumentCreated(ctx, &event)
	}

	return nil
}

// handleUpdatedEvent handles document.updated events
func (c *CDCConsumer) handleUpdatedEvent(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event DocumentUpdatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal updated event: %w", err)
	}

	logger.Info(ctx, "processing document.updated event",
		zap.String("event_id", event.EventID),
		zap.String("document_id", event.DocumentID),
		zap.String("collection", event.Collection),
		zap.Int("version", event.Version),
	)

	if c.handlers.OnDocumentUpdated != nil {
		return c.handlers.OnDocumentUpdated(ctx, &event)
	}

	return nil
}

// handleDeletedEvent handles document.deleted events
func (c *CDCConsumer) handleDeletedEvent(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event DocumentDeletedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal deleted event: %w", err)
	}

	logger.Info(ctx, "processing document.deleted event",
		zap.String("event_id", event.EventID),
		zap.String("document_id", event.DocumentID),
		zap.String("collection", event.Collection),
	)

	if c.handlers.OnDocumentDeleted != nil {
		return c.handlers.OnDocumentDeleted(ctx, &event)
	}

	return nil
}

// Start starts the CDC consumer
func (c *CDCConsumer) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

// Close closes the CDC consumer
func (c *CDCConsumer) Close() error {
	return c.consumer.Close()
}
