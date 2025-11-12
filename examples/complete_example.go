package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/YouSangSon/database-service/internal/config"
	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/infrastructure/cache"
	"github.com/YouSangSon/database-service/internal/infrastructure/messaging/kafka"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/mongodb"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/vitess"
	"github.com/YouSangSon/database-service/internal/pkg/vault"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// ===== 1. 설정 로드 =====
	fmt.Println("=== Loading Configuration ===")

	cfg, err := config.LoadConfig("./configs", "config")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	fmt.Printf("App: %s v%s (%s)\n", cfg.App.Name, cfg.App.Version, cfg.App.Environment)
	fmt.Printf("MongoDB Enabled: %v\n", cfg.MongoDB.Enabled)
	fmt.Printf("Vitess Enabled: %v\n", cfg.Vitess.Enabled)
	fmt.Printf("Kafka Enabled: %v\n", cfg.Kafka.Enabled)
	fmt.Printf("Vault Enabled: %v\n", cfg.Vault.Enabled)

	// ===== 2. Vault 초기화 (옵션) =====
	var vaultClient *vault.Client
	if cfg.Vault.Enabled {
		fmt.Println("\n=== Initializing Vault ===")

		vaultConfig := &vault.Config{
			Address:           cfg.Vault.Address,
			Token:             cfg.Vault.Token,
			AuthMethod:        cfg.Vault.AuthMethod,
			RoleID:            cfg.Vault.RoleID,
			SecretID:          cfg.Vault.SecretID,
			K8sRole:           cfg.Vault.K8sRole,
			Namespace:         cfg.Vault.Namespace,
			TLSEnabled:        cfg.Vault.TLS.Enabled,
			TLSSkipVerify:     cfg.Vault.TLS.SkipVerify,
			CACert:            cfg.Vault.TLS.CACert,
			ClientCert:        cfg.Vault.TLS.ClientCert,
			ClientKey:         cfg.Vault.TLS.ClientKey,
			MongoDBPath:       cfg.Vault.Paths.MongoDB,
			RedisPath:         cfg.Vault.Paths.Redis,
			SecretsPath:       cfg.Vault.Paths.Secrets,
			TransitPath:       cfg.Vault.Paths.Transit,
			RenewInterval:     cfg.Vault.Renewal.Interval,
			RenewBeforeExpiry: cfg.Vault.Renewal.RenewBeforeExpiry,
			MaxRetries:        cfg.Vault.Renewal.MaxRetries,
			RetryInterval:     cfg.Vault.Renewal.RetryInterval,
			CacheEnabled:      cfg.Vault.Cache.Enabled,
			CacheTTL:          cfg.Vault.Cache.TTL,
		}

		vaultClient, err = vault.NewClient(vaultConfig)
		if err != nil {
			log.Fatalf("Failed to create vault client: %v", err)
		}
		defer vaultClient.Close()

		// Health check
		if err := vaultClient.HealthCheck(ctx); err != nil {
			log.Fatalf("Vault health check failed: %v", err)
		}

		fmt.Println("✓ Vault initialized")
	}

	// ===== 3. MongoDB 초기화 =====
	if cfg.MongoDB.Enabled {
		fmt.Println("\n=== Initializing MongoDB ===")

		mongoURI := cfg.MongoDB.URI

		// Vault에서 자격증명 가져오기
		if cfg.MongoDB.UseVault && vaultClient != nil {
			username, password, err := vaultClient.GetMongoDBCredentials(ctx)
			if err != nil {
				log.Fatalf("Failed to get MongoDB credentials: %v", err)
			}
			mongoURI = fmt.Sprintf("mongodb://%s:%s@localhost:27017/%s",
				username, password, cfg.MongoDB.Database)
		}

		mongoConfig := &mongodb.Config{
			URI:             mongoURI,
			Database:        cfg.MongoDB.Database,
			MaxPoolSize:     cfg.MongoDB.MaxPoolSize,
			MinPoolSize:     cfg.MongoDB.MinPoolSize,
			MaxConnecting:   cfg.MongoDB.MaxConnecting,
			ConnectTimeout:  cfg.MongoDB.ConnectTimeout,
			Timeout:         cfg.MongoDB.Timeout,
		}

		mongoRepo, err := mongodb.NewDocumentRepository(mongoConfig)
		if err != nil {
			log.Fatalf("Failed to create MongoDB repository: %v", err)
		}

		// 테스트 문서 생성
		doc := entity.NewDocument("test_collection", map[string]interface{}{
			"name":  "Test Document",
			"value": 12345,
		})

		if err := mongoRepo.Save(ctx, doc); err != nil {
			log.Printf("Failed to save document: %v", err)
		} else {
			fmt.Printf("✓ MongoDB document saved: %s\n", doc.ID())
		}
	}

	// ===== 4. Vitess 초기화 =====
	if cfg.Vitess.Enabled {
		fmt.Println("\n=== Initializing Vitess ===")

		username := cfg.Vitess.Username
		password := cfg.Vitess.Password

		// Vault에서 자격증명 가져오기
		if cfg.Vitess.UseVault && vaultClient != nil {
			username, password, err = vaultClient.GetVitessCredentials(ctx, cfg.Vault.Paths.Vitess)
			if err != nil {
				log.Fatalf("Failed to get Vitess credentials: %v", err)
			}
		}

		vitessConfig := &vitess.Config{
			Host:            cfg.Vitess.Host,
			Port:            cfg.Vitess.Port,
			Keyspace:        cfg.Vitess.Keyspace,
			Username:        username,
			Password:        password,
			MaxOpenConns:    cfg.Vitess.MaxOpenConns,
			MaxIdleConns:    cfg.Vitess.MaxIdleConns,
			ConnMaxLifetime: cfg.Vitess.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.Vitess.ConnMaxIdleTime,
		}

		vitessRepo, err := vitess.NewVitessRepository(vitessConfig)
		if err != nil {
			log.Printf("Vitess not available: %v", err)
		} else {
			// 테스트 문서 생성
			doc := entity.NewDocument("test_vitess", map[string]interface{}{
				"name": "Vitess Test",
				"ts":   time.Now().Unix(),
			})

			if err := vitessRepo.Save(ctx, doc); err != nil {
				log.Printf("Failed to save to Vitess: %v", err)
			} else {
				fmt.Printf("✓ Vitess document saved: %s\n", doc.ID())
			}
		}
	}

	// ===== 5. Redis 초기화 =====
	if cfg.Redis.Enabled {
		fmt.Println("\n=== Initializing Redis ===")

		password := cfg.Redis.Password

		// Vault에서 비밀번호 가져오기
		if cfg.Redis.UseVault && vaultClient != nil {
			password, err = vaultClient.GetRedisCredentials(ctx)
			if err != nil {
				log.Printf("Failed to get Redis credentials: %v", err)
			}
		}

		redisClient := redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password:     password,
			DB:           cfg.Redis.DB,
			MaxRetries:   cfg.Redis.MaxRetries,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
		})

		// 연결 확인
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Fatalf("Redis ping failed: %v", err)
		}

		fmt.Println("✓ Redis connected")

		// Redis 확장 기능 데모
		redisExt := cache.NewRedisExtended(redisClient)

		// Rate Limiting
		rateLimiter := redisExt.NewRateLimiter("api")
		allowed, _ := rateLimiter.Allow(ctx, "user:123", 100, time.Minute)
		fmt.Printf("Rate limit allowed: %v\n", allowed)

		// Distributed Lock
		lock := redisExt.NewDistributedLock("resource:123", 30*time.Second)
		acquired, _ := lock.Acquire(ctx)
		if acquired {
			fmt.Println("Lock acquired")
			defer lock.Release(ctx)
		}

		// Pub/Sub (optional)
		if cfg.Redis.EnablePubSub {
			err := redisExt.Publish(ctx, "documents.events", map[string]interface{}{
				"event": "test",
				"ts":    time.Now().Unix(),
			})
			if err == nil {
				fmt.Println("✓ Message published to Redis")
			}
		}
	}

	// ===== 6. Kafka 초기화 =====
	if cfg.Kafka.Enabled {
		fmt.Println("\n=== Initializing Kafka ===")

		producerConfig := &kafka.ProducerConfig{
			Brokers:          cfg.Kafka.Brokers,
			ClientID:         cfg.Kafka.ClientID,
			MaxMessageBytes:  cfg.Kafka.Producer.MaxMessageBytes,
			RequiredAcks:     sarama.RequiredAcks(cfg.Kafka.Producer.RequiredAcks),
			Compression:      parseCompression(cfg.Kafka.Producer.Compression),
			MaxRetries:       cfg.Kafka.Producer.MaxRetries,
			RetryBackoff:     cfg.Kafka.Producer.RetryBackoff,
			EnableIdempotent: cfg.Kafka.Producer.EnableIdempotent,
			UseAsync:         false,
		}

		producer, err := kafka.NewProducer(producerConfig)
		if err != nil {
			log.Printf("Failed to create Kafka producer: %v", err)
		} else {
			defer producer.Close()

			fmt.Println("✓ Kafka producer initialized")

			// CDC Publisher
			if cfg.Kafka.EnableCDC {
				cdcPublisher := kafka.NewCDCPublisher(
					producer,
					cfg.Kafka.CDCTopics.DocumentCreated,
					cfg.Kafka.CDCTopics.DocumentUpdated,
					cfg.Kafka.CDCTopics.DocumentDeleted,
				)

				// 테스트 이벤트 발행
				err := cdcPublisher.PublishDocumentCreated(ctx, "doc123", "users", map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
				}, 1)

				if err == nil {
					fmt.Println("✓ CDC event published to Kafka")
				}
			}
		}
	}

	// ===== 7. 암호화 데모 (Vault) =====
	if vaultClient != nil {
		fmt.Println("\n=== Vault Encryption Demo ===")

		sensitiveData := "Credit Card: 1234-5678-9012-3456"
		encrypted, err := vaultClient.EncryptString(ctx, "database-encryption", sensitiveData)
		if err != nil {
			log.Printf("Encryption failed: %v", err)
		} else {
			fmt.Printf("Encrypted: %s\n", encrypted[:50]+"...")

			decrypted, err := vaultClient.DecryptString(ctx, "database-encryption", encrypted)
			if err != nil {
				log.Printf("Decryption failed: %v", err)
			} else {
				fmt.Printf("Decrypted: %s\n", decrypted)
				fmt.Println("✓ Encryption/Decryption successful")
			}
		}
	}

	fmt.Println("\n=== All Services Initialized Successfully ===")
}

func parseCompression(compression string) sarama.CompressionCodec {
	switch compression {
	case "gzip":
		return sarama.CompressionGZIP
	case "snappy":
		return sarama.CompressionSnappy
	case "lz4":
		return sarama.CompressionLZ4
	case "zstd":
		return sarama.CompressionZSTD
	default:
		return sarama.CompressionNone
	}
}

// Import sarama for RequiredAcks
import "github.com/IBM/sarama"
