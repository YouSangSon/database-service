package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config는 애플리케이션 설정입니다
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig는 서버 설정입니다
type ServerConfig struct {
	APIPort  int
	GRPCPort int
}

// DatabaseConfig는 데이터베이스 설정입니다
type DatabaseConfig struct {
	Type     string
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// Load는 환경변수에서 설정을 로드합니다
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			APIPort:  getEnvAsInt("API_PORT", 8080),
			GRPCPort: getEnvAsInt("GRPC_PORT", 50051),
		},
		Database: DatabaseConfig{
			Type:     getEnv("DB_TYPE", "mongodb"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 27017),
			Username: getEnv("DB_USERNAME", ""),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_DATABASE", "testdb"),
		},
	}

	return config, nil
}

// getEnv는 환경변수 값을 가져오거나 기본값을 반환합니다
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt는 환경변수 값을 정수로 가져오거나 기본값을 반환합니다
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		fmt.Printf("Warning: invalid value for %s, using default %d\n", key, defaultValue)
		return defaultValue
	}

	return value
}
