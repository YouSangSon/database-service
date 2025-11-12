package vault

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// Encrypt는 데이터를 암호화합니다 (Transit Engine)
// keyName: Transit 엔진의 암호화 키 이름
// plaintext: 암호화할 평문 데이터
func (c *Client) Encrypt(ctx context.Context, keyName string, plaintext []byte) (string, error) {
	if keyName == "" {
		return "", fmt.Errorf("key name is required")
	}

	// 평문을 Base64로 인코딩
	encodedPlaintext := base64.StdEncoding.EncodeToString(plaintext)

	// Transit 엔진으로 암호화
	path := fmt.Sprintf("%s/encrypt/%s", c.config.TransitPath, keyName)
	data := map[string]interface{}{
		"plaintext": encodedPlaintext,
	}

	secret, err := c.client.Logical().Write(path, data)
	if err != nil {
		logger.Error(ctx, "failed to encrypt data",
			logger.Field("key_name", keyName),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	if secret == nil || secret.Data["ciphertext"] == nil {
		return "", fmt.Errorf("encryption returned no ciphertext")
	}

	ciphertext := secret.Data["ciphertext"].(string)

	logger.Debug(ctx, "data encrypted successfully",
		logger.Field("key_name", keyName),
		logger.Field("plaintext_length", len(plaintext)),
	)

	return ciphertext, nil
}

// Decrypt는 데이터를 복호화합니다 (Transit Engine)
// keyName: Transit 엔진의 암호화 키 이름
// ciphertext: 암호화된 데이터
func (c *Client) Decrypt(ctx context.Context, keyName string, ciphertext string) ([]byte, error) {
	if keyName == "" {
		return nil, fmt.Errorf("key name is required")
	}

	if ciphertext == "" {
		return nil, fmt.Errorf("ciphertext is required")
	}

	// Transit 엔진으로 복호화
	path := fmt.Sprintf("%s/decrypt/%s", c.config.TransitPath, keyName)
	data := map[string]interface{}{
		"ciphertext": ciphertext,
	}

	secret, err := c.client.Logical().Write(path, data)
	if err != nil {
		logger.Error(ctx, "failed to decrypt data",
			logger.Field("key_name", keyName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	if secret == nil || secret.Data["plaintext"] == nil {
		return nil, fmt.Errorf("decryption returned no plaintext")
	}

	// Base64 디코딩
	encodedPlaintext := secret.Data["plaintext"].(string)
	plaintext, err := base64.StdEncoding.DecodeString(encodedPlaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode plaintext: %w", err)
	}

	logger.Debug(ctx, "data decrypted successfully",
		logger.Field("key_name", keyName),
		logger.Field("plaintext_length", len(plaintext)),
	)

	return plaintext, nil
}

// EncryptString는 문자열을 암호화합니다
func (c *Client) EncryptString(ctx context.Context, keyName string, plaintext string) (string, error) {
	return c.Encrypt(ctx, keyName, []byte(plaintext))
}

// DecryptString는 암호화된 문자열을 복호화합니다
func (c *Client) DecryptString(ctx context.Context, keyName string, ciphertext string) (string, error) {
	plaintext, err := c.Decrypt(ctx, keyName, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// Hash는 데이터의 해시를 생성합니다 (Transit Engine)
// algorithm: sha2-256, sha2-512 등
func (c *Client) Hash(ctx context.Context, keyName string, algorithm string, data []byte) (string, error) {
	if keyName == "" {
		return "", fmt.Errorf("key name is required")
	}

	// 데이터를 Base64로 인코딩
	encodedData := base64.StdEncoding.EncodeToString(data)

	// Transit 엔진으로 해싱
	path := fmt.Sprintf("%s/hash/%s", c.config.TransitPath, keyName)
	requestData := map[string]interface{}{
		"input":     encodedData,
		"algorithm": algorithm,
	}

	secret, err := c.client.Logical().Write(path, requestData)
	if err != nil {
		logger.Error(ctx, "failed to hash data",
			logger.Field("key_name", keyName),
			logger.Field("algorithm", algorithm),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to hash data: %w", err)
	}

	if secret == nil || secret.Data["sum"] == nil {
		return "", fmt.Errorf("hashing returned no sum")
	}

	hash := secret.Data["sum"].(string)

	logger.Debug(ctx, "data hashed successfully",
		logger.Field("key_name", keyName),
		logger.Field("algorithm", algorithm),
	)

	return hash, nil
}

// Sign는 데이터에 서명합니다 (Transit Engine)
func (c *Client) Sign(ctx context.Context, keyName string, data []byte) (string, error) {
	if keyName == "" {
		return "", fmt.Errorf("key name is required")
	}

	// 데이터를 Base64로 인코딩
	encodedData := base64.StdEncoding.EncodeToString(data)

	// Transit 엔진으로 서명
	path := fmt.Sprintf("%s/sign/%s", c.config.TransitPath, keyName)
	requestData := map[string]interface{}{
		"input": encodedData,
	}

	secret, err := c.client.Logical().Write(path, requestData)
	if err != nil {
		logger.Error(ctx, "failed to sign data",
			logger.Field("key_name", keyName),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to sign data: %w", err)
	}

	if secret == nil || secret.Data["signature"] == nil {
		return "", fmt.Errorf("signing returned no signature")
	}

	signature := secret.Data["signature"].(string)

	logger.Debug(ctx, "data signed successfully",
		logger.Field("key_name", keyName),
	)

	return signature, nil
}

// Verify는 서명을 검증합니다 (Transit Engine)
func (c *Client) Verify(ctx context.Context, keyName string, data []byte, signature string) (bool, error) {
	if keyName == "" {
		return false, fmt.Errorf("key name is required")
	}

	// 데이터를 Base64로 인코딩
	encodedData := base64.StdEncoding.EncodeToString(data)

	// Transit 엔진으로 검증
	path := fmt.Sprintf("%s/verify/%s", c.config.TransitPath, keyName)
	requestData := map[string]interface{}{
		"input":     encodedData,
		"signature": signature,
	}

	secret, err := c.client.Logical().Write(path, requestData)
	if err != nil {
		logger.Error(ctx, "failed to verify signature",
			logger.Field("key_name", keyName),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}

	if secret == nil || secret.Data["valid"] == nil {
		return false, fmt.Errorf("verification returned no result")
	}

	valid := secret.Data["valid"].(bool)

	logger.Debug(ctx, "signature verified",
		logger.Field("key_name", keyName),
		logger.Field("valid", valid),
	)

	return valid, nil
}

// GenerateDataKey는 데이터 암호화 키를 생성합니다 (Transit Engine)
// 반환값: plaintext key, encrypted key, error
func (c *Client) GenerateDataKey(ctx context.Context, keyName string, bits int) ([]byte, string, error) {
	if keyName == "" {
		return nil, "", fmt.Errorf("key name is required")
	}

	// Transit 엔진으로 데이터 키 생성
	path := fmt.Sprintf("%s/datakey/plaintext/%s", c.config.TransitPath, keyName)
	requestData := map[string]interface{}{}

	if bits > 0 {
		requestData["bits"] = bits
	}

	secret, err := c.client.Logical().Write(path, requestData)
	if err != nil {
		logger.Error(ctx, "failed to generate data key",
			logger.Field("key_name", keyName),
			zap.Error(err),
		)
		return nil, "", fmt.Errorf("failed to generate data key: %w", err)
	}

	if secret == nil || secret.Data["plaintext"] == nil || secret.Data["ciphertext"] == nil {
		return nil, "", fmt.Errorf("data key generation returned incomplete data")
	}

	// Base64 디코딩
	encodedPlaintext := secret.Data["plaintext"].(string)
	plaintext, err := base64.StdEncoding.DecodeString(encodedPlaintext)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode plaintext key: %w", err)
	}

	ciphertext := secret.Data["ciphertext"].(string)

	logger.Info(ctx, "data key generated successfully",
		logger.Field("key_name", keyName),
		logger.Field("key_length", len(plaintext)),
	)

	return plaintext, ciphertext, nil
}
