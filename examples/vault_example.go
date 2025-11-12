package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/vault"
)

func main() {
	ctx := context.Background()

	// Vault 클라이언트 설정
	config := vault.DefaultConfig()
	config.Address = "http://localhost:8200"
	config.Token = "root" // 프로덕션에서는 환경변수에서 읽기
	config.MongoDBPath = "database/creds/mongodb-role"

	// Vault 클라이언트 생성
	client, err := vault.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create vault client: %v", err)
	}
	defer client.Close()

	// Health Check
	if err := client.HealthCheck(ctx); err != nil {
		log.Fatalf("Vault health check failed: %v", err)
	}
	fmt.Println("✓ Vault health check passed")

	// ========================================
	// 1. MongoDB 동적 자격증명 사용
	// ========================================
	fmt.Println("\n=== MongoDB Dynamic Credentials ===")

	// MongoDB 자격증명 관리자 생성
	credManager := vault.NewDatabaseCredentialsManager(client)
	defer credManager.Close(ctx)

	// 자격증명 가져오기
	creds, err := credManager.GetMongoDBCredentials(ctx)
	if err != nil {
		log.Fatalf("Failed to get MongoDB credentials: %v", err)
	}

	fmt.Printf("Username: %s\n", creds.Username)
	fmt.Printf("Password: %s\n", creds.Password)
	fmt.Printf("Expires At: %s\n", creds.ExpiresAt.Format(time.RFC3339))

	// 연결 문자열 생성
	connStr, err := credManager.GetConnectionString(ctx, "localhost", 27017, "database_service")
	if err != nil {
		log.Fatalf("Failed to get connection string: %v", err)
	}
	fmt.Printf("Connection String: %s\n", connStr)

	// 자동 갱신 시작
	credManager.StartAutoRenewal(ctx)
	fmt.Println("✓ Auto renewal started")

	// ========================================
	// 2. 정적 시크릿 관리
	// ========================================
	fmt.Println("\n=== Static Secrets ===")

	// 시크릿 저장
	appSecrets := map[string]interface{}{
		"api_key":    "my_api_key_12345",
		"jwt_secret": "my_jwt_secret",
		"database":   "production",
	}

	err = client.PutSecret(ctx, "secret/data/myapp", appSecrets)
	if err != nil {
		log.Fatalf("Failed to put secret: %v", err)
	}
	fmt.Println("✓ Secret stored")

	// 시크릿 읽기
	secretMeta, err := client.GetSecret(ctx, "secret/data/myapp")
	if err != nil {
		log.Fatalf("Failed to get secret: %v", err)
	}

	fmt.Printf("API Key: %s\n", secretMeta.Data["api_key"])
	fmt.Printf("JWT Secret: %s\n", secretMeta.Data["jwt_secret"])
	fmt.Printf("Database: %s\n", secretMeta.Data["database"])

	// ========================================
	// 3. 암호화/복호화 (Transit Engine)
	// ========================================
	fmt.Println("\n=== Encryption/Decryption ===")

	// 문자열 암호화
	plaintext := "sensitive data that needs encryption"
	ciphertext, err := client.EncryptString(ctx, "database-encryption", plaintext)
	if err != nil {
		log.Fatalf("Failed to encrypt: %v", err)
	}
	fmt.Printf("Plaintext: %s\n", plaintext)
	fmt.Printf("Ciphertext: %s\n", ciphertext)

	// 복호화
	decrypted, err := client.DecryptString(ctx, "database-encryption", ciphertext)
	if err != nil {
		log.Fatalf("Failed to decrypt: %v", err)
	}
	fmt.Printf("Decrypted: %s\n", decrypted)

	if plaintext == decrypted {
		fmt.Println("✓ Encryption/Decryption successful")
	}

	// ========================================
	// 4. 데이터 키 생성
	// ========================================
	fmt.Println("\n=== Data Key Generation ===")

	plaintextKey, encryptedKey, err := client.GenerateDataKey(ctx, "app-data-key", 256)
	if err != nil {
		log.Fatalf("Failed to generate data key: %v", err)
	}

	fmt.Printf("Plaintext Key (first 16 bytes): %x\n", plaintextKey[:16])
	fmt.Printf("Encrypted Key: %s\n", encryptedKey)
	fmt.Println("✓ Data key generated")

	// ========================================
	// 5. 서명 및 검증
	// ========================================
	fmt.Println("\n=== Sign and Verify ===")

	data := []byte("important message")

	// 서명 생성
	signature, err := client.Sign(ctx, "database-encryption", data)
	if err != nil {
		log.Fatalf("Failed to sign: %v", err)
	}
	fmt.Printf("Data: %s\n", string(data))
	fmt.Printf("Signature: %s\n", signature)

	// 서명 검증
	valid, err := client.Verify(ctx, "database-encryption", data, signature)
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}
	fmt.Printf("Signature Valid: %v\n", valid)

	if valid {
		fmt.Println("✓ Signature verification successful")
	}

	// ========================================
	// 6. Redis 자격증명
	// ========================================
	fmt.Println("\n=== Redis Credentials ===")

	redisPassword, err := client.GetRedisCredentials(ctx)
	if err != nil {
		log.Fatalf("Failed to get redis credentials: %v", err)
	}
	fmt.Printf("Redis Password: %s\n", redisPassword)

	// ========================================
	// 7. 자동 갱신 데모
	// ========================================
	fmt.Println("\n=== Auto Renewal Demo ===")

	// 자동 갱신 시작
	client.StartRenewal(ctx)
	fmt.Println("✓ Auto renewal started")

	// 5초 대기 (갱신 동작 확인)
	fmt.Println("Waiting for 5 seconds...")
	time.Sleep(5 * time.Second)

	// 자동 갱신 중지
	client.StopRenewal()
	fmt.Println("✓ Auto renewal stopped")

	// ========================================
	// 완료
	// ========================================
	fmt.Println("\n=== All Examples Completed Successfully ===")
}

// MongoDB 연결 예시 함수
func connectToMongoDBWithVault(ctx context.Context) error {
	config := vault.DefaultConfig()
	config.Address = "http://localhost:8200"
	config.Token = "root"

	client, err := vault.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}
	defer client.Close()

	credManager := vault.NewDatabaseCredentialsManager(client)
	defer credManager.Close(ctx)

	// 자동 갱신 시작
	credManager.StartAutoRenewal(ctx)

	// MongoDB 연결 문자열 가져오기
	connStr, err := credManager.GetConnectionString(ctx, "localhost", 27017, "database_service")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	fmt.Printf("MongoDB Connection String: %s\n", connStr)

	// 여기서 MongoDB 연결 로직 수행
	// client, err := mongo.Connect(ctx, options.Client().ApplyURI(connStr))

	return nil
}
