package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config는 애플리케이션 전체 설정입니다
type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Server        ServerConfig        `mapstructure:"server"`
	MongoDB       MongoDBConfig       `mapstructure:"mongodb"`
	PostgreSQL    PostgreSQLConfig    `mapstructure:"postgresql"`
	MySQL         MySQLConfig         `mapstructure:"mysql"`
	Cassandra     CassandraConfig     `mapstructure:"cassandra"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	Vitess        VitessConfig        `mapstructure:"vitess"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Kafka         KafkaConfig         `mapstructure:"kafka"`
	Vault         VaultConfig         `mapstructure:"vault"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

// AppConfig는 애플리케이션 기본 설정입니다
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`
}

// ServerConfig는 서버 설정입니다
type ServerConfig struct {
	HTTP HTTPServerConfig `mapstructure:"http"`
	GRPC GRPCServerConfig `mapstructure:"grpc"`
}

// HTTPServerConfig는 HTTP 서버 설정입니다
type HTTPServerConfig struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	MaxRequestSize    int64         `mapstructure:"max_request_size"`
	EnableCORS        bool          `mapstructure:"enable_cors"`
	AllowedOrigins    []string      `mapstructure:"allowed_origins"`
}

// GRPCServerConfig는 gRPC 서버 설정입니다
type GRPCServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	MaxRecvMsgSize  int           `mapstructure:"max_recv_msg_size"`
	MaxSendMsgSize  int           `mapstructure:"max_send_msg_size"`
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
	EnableReflection bool          `mapstructure:"enable_reflection"`
}

// MongoDBConfig는 MongoDB 설정입니다
type MongoDBConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	URI             string        `mapstructure:"uri"`
	Host            string        `mapstructure:"host"`
	Database        string        `mapstructure:"database"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	MaxPoolSize     uint64        `mapstructure:"max_pool_size"`
	MinPoolSize     uint64        `mapstructure:"min_pool_size"`
	MaxConnecting   uint64        `mapstructure:"max_connecting"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	Timeout         time.Duration `mapstructure:"timeout"`
	UseVault        bool          `mapstructure:"use_vault"`
	VaultPath       string        `mapstructure:"vault_path"`
}

// PostgreSQLConfig는 PostgreSQL 설정입니다
type PostgreSQLConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	UseVault        bool          `mapstructure:"use_vault"`
	VaultPath       string        `mapstructure:"vault_path"`
}

// MySQLConfig는 MySQL 설정입니다
type MySQLConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	Charset         string        `mapstructure:"charset"`
	ParseTime       bool          `mapstructure:"parse_time"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	UseVault        bool          `mapstructure:"use_vault"`
	VaultPath       string        `mapstructure:"vault_path"`
}

// CassandraConfig는 Cassandra 설정입니다
type CassandraConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	Hosts       []string `mapstructure:"hosts"`
	Port        int      `mapstructure:"port"`
	Keyspace    string   `mapstructure:"keyspace"`
	Username    string   `mapstructure:"username"`
	Password    string   `mapstructure:"password"`
	Consistency string   `mapstructure:"consistency"`
	NumConns    int      `mapstructure:"num_conns"`
	Timeout     time.Duration `mapstructure:"timeout"`
	UseVault    bool     `mapstructure:"use_vault"`
	VaultPath   string   `mapstructure:"vault_path"`
}

// ElasticsearchConfig는 Elasticsearch 설정입니다
type ElasticsearchConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
	APIKey    string   `mapstructure:"api_key"`
	CloudID   string   `mapstructure:"cloud_id"`
	MaxRetries int     `mapstructure:"max_retries"`
	UseVault  bool     `mapstructure:"use_vault"`
	VaultPath string   `mapstructure:"vault_path"`
}

// VitessConfig는 Vitess 설정입니다
type VitessConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Keyspace        string        `mapstructure:"keyspace"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	UseVault        bool          `mapstructure:"use_vault"`
	VaultPath       string        `mapstructure:"vault_path"`
}

// RedisConfig는 Redis 설정입니다
type RedisConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db"`
	MaxRetries      int           `mapstructure:"max_retries"`
	PoolSize        int           `mapstructure:"pool_size"`
	MinIdleConns    int           `mapstructure:"min_idle_conns"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	UseVault        bool          `mapstructure:"use_vault"`
	VaultPath       string        `mapstructure:"vault_path"`
	EnablePubSub    bool          `mapstructure:"enable_pubsub"`
	PubSubChannels  []string      `mapstructure:"pubsub_channels"`
}

// KafkaConfig는 Kafka 설정입니다
type KafkaConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	Brokers         []string `mapstructure:"brokers"`
	Version         string   `mapstructure:"version"`
	ClientID        string   `mapstructure:"client_id"`
	Producer        KafkaProducerConfig `mapstructure:"producer"`
	Consumer        KafkaConsumerConfig `mapstructure:"consumer"`
	EnableCDC       bool     `mapstructure:"enable_cdc"`
	CDCTopics       KafkaCDCTopics `mapstructure:"cdc_topics"`
}

// KafkaProducerConfig는 Kafka Producer 설정입니다
type KafkaProducerConfig struct {
	MaxMessageBytes   int           `mapstructure:"max_message_bytes"`
	RequiredAcks      int16         `mapstructure:"required_acks"`
	Timeout           time.Duration `mapstructure:"timeout"`
	Compression       string        `mapstructure:"compression"`
	MaxRetries        int           `mapstructure:"max_retries"`
	RetryBackoff      time.Duration `mapstructure:"retry_backoff"`
	EnableIdempotent  bool          `mapstructure:"enable_idempotent"`
}

// KafkaConsumerConfig는 Kafka Consumer 설정입니다
type KafkaConsumerConfig struct {
	GroupID               string        `mapstructure:"group_id"`
	AutoOffsetReset       string        `mapstructure:"auto_offset_reset"`
	EnableAutoCommit      bool          `mapstructure:"enable_auto_commit"`
	AutoCommitInterval    time.Duration `mapstructure:"auto_commit_interval"`
	SessionTimeout        time.Duration `mapstructure:"session_timeout"`
	HeartbeatInterval     time.Duration `mapstructure:"heartbeat_interval"`
	MaxProcessingTime     time.Duration `mapstructure:"max_processing_time"`
}

// KafkaCDCTopics는 CDC 토픽 설정입니다
type KafkaCDCTopics struct {
	DocumentCreated string `mapstructure:"document_created"`
	DocumentUpdated string `mapstructure:"document_updated"`
	DocumentDeleted string `mapstructure:"document_deleted"`
}

// VaultConfig는 Vault 설정입니다
type VaultConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	Address           string        `mapstructure:"address"`
	Token             string        `mapstructure:"token"`
	AuthMethod        string        `mapstructure:"auth_method"`
	RoleID            string        `mapstructure:"role_id"`
	SecretID          string        `mapstructure:"secret_id"`
	K8sRole           string        `mapstructure:"k8s_role"`
	Namespace         string        `mapstructure:"namespace"`
	TLS               VaultTLSConfig `mapstructure:"tls"`
	Paths             VaultPaths    `mapstructure:"paths"`
	Renewal           VaultRenewal  `mapstructure:"renewal"`
	Cache             VaultCache    `mapstructure:"cache"`
}

// VaultTLSConfig는 Vault TLS 설정입니다
type VaultTLSConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	SkipVerify    bool   `mapstructure:"skip_verify"`
	CACert        string `mapstructure:"ca_cert"`
	ClientCert    string `mapstructure:"client_cert"`
	ClientKey     string `mapstructure:"client_key"`
}

// VaultPaths는 Vault 경로 설정입니다
type VaultPaths struct {
	MongoDB       string `mapstructure:"mongodb"`
	PostgreSQL    string `mapstructure:"postgresql"`
	MySQL         string `mapstructure:"mysql"`
	Cassandra     string `mapstructure:"cassandra"`
	Elasticsearch string `mapstructure:"elasticsearch"`
	Vitess        string `mapstructure:"vitess"`
	Redis         string `mapstructure:"redis"`
	Secrets       string `mapstructure:"secrets"`
	Transit       string `mapstructure:"transit"`
}

// VaultRenewal는 Vault 갱신 설정입니다
type VaultRenewal struct {
	Interval          time.Duration `mapstructure:"interval"`
	RenewBeforeExpiry time.Duration `mapstructure:"renew_before_expiry"`
	MaxRetries        int           `mapstructure:"max_retries"`
	RetryInterval     time.Duration `mapstructure:"retry_interval"`
}

// VaultCache는 Vault 캐시 설정입니다
type VaultCache struct {
	Enabled bool          `mapstructure:"enabled"`
	TTL     time.Duration `mapstructure:"ttl"`
}

// ObservabilityConfig는 관찰성 설정입니다
type ObservabilityConfig struct {
	Logging LoggingConfig `mapstructure:"logging"`
	Tracing TracingConfig `mapstructure:"tracing"`
	Metrics MetricsConfig `mapstructure:"metrics"`
}

// LoggingConfig는 로깅 설정입니다
type LoggingConfig struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`
	Output      string `mapstructure:"output"`
	Development bool   `mapstructure:"development"`
}

// TracingConfig는 분산 추적 설정입니다
type TracingConfig struct {
	Enabled         bool    `mapstructure:"enabled"`
	ServiceName     string  `mapstructure:"service_name"`
	JaegerEndpoint  string  `mapstructure:"jaeger_endpoint"`
	SamplingRate    float64 `mapstructure:"sampling_rate"`
}

// MetricsConfig는 메트릭 설정입니다
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// LoadConfig는 설정 파일을 로드합니다
func LoadConfig(configPath string, configName string) (*Config, error) {
	v := viper.New()

	// 설정 파일 경로 및 이름 설정
	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	if configName != "" {
		v.SetConfigName(configName)
	} else {
		v.SetConfigName("config")
	}

	v.SetConfigType("yaml")

	// 환경변수 바인딩
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 설정 파일 읽기
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 설정 구조체로 언마샬
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 환경변수로 민감한 값 오버라이드
	overrideFromEnv(&config)

	return &config, nil
}

// overrideFromEnv는 환경변수로 민감한 설정을 오버라이드합니다
// GitLab CI/CD 프로젝트 변수를 지원합니다
func overrideFromEnv(config *Config) {
	// Vault 설정
	if val := viper.GetString("VAULT_TOKEN"); val != "" {
		config.Vault.Token = val
	}
	if val := viper.GetString("VAULT_ADDRESS"); val != "" {
		config.Vault.Address = val
	}
	if val := viper.GetString("VAULT_ROLE_ID"); val != "" {
		config.Vault.RoleID = val
	}
	if val := viper.GetString("VAULT_SECRET_ID"); val != "" {
		config.Vault.SecretID = val
	}
	if val := viper.GetString("VAULT_NAMESPACE"); val != "" {
		config.Vault.Namespace = val
	}

	// MongoDB 설정
	if val := viper.GetString("MONGODB_URI"); val != "" {
		config.MongoDB.URI = val
	}
	if val := viper.GetString("MONGODB_HOST"); val != "" {
		config.MongoDB.Host = val
	}
	if val := viper.GetString("MONGODB_DATABASE"); val != "" {
		config.MongoDB.Database = val
	}
	if val := viper.GetString("MONGODB_USERNAME"); val != "" {
		config.MongoDB.Username = val
	}
	if val := viper.GetString("MONGODB_PASSWORD"); val != "" {
		config.MongoDB.Password = val
	}

	// PostgreSQL 설정
	if val := viper.GetString("POSTGRESQL_HOST"); val != "" {
		config.PostgreSQL.Host = val
	}
	if val := viper.GetInt("POSTGRESQL_PORT"); val != 0 {
		config.PostgreSQL.Port = val
	}
	if val := viper.GetString("POSTGRESQL_USER"); val != "" {
		config.PostgreSQL.User = val
	}
	if val := viper.GetString("POSTGRESQL_PASSWORD"); val != "" {
		config.PostgreSQL.Password = val
	}
	if val := viper.GetString("POSTGRESQL_DATABASE"); val != "" {
		config.PostgreSQL.Database = val
	}

	// MySQL 설정
	if val := viper.GetString("MYSQL_HOST"); val != "" {
		config.MySQL.Host = val
	}
	if val := viper.GetInt("MYSQL_PORT"); val != 0 {
		config.MySQL.Port = val
	}
	if val := viper.GetString("MYSQL_USER"); val != "" {
		config.MySQL.User = val
	}
	if val := viper.GetString("MYSQL_PASSWORD"); val != "" {
		config.MySQL.Password = val
	}
	if val := viper.GetString("MYSQL_DATABASE"); val != "" {
		config.MySQL.Database = val
	}

	// Cassandra 설정
	if val := viper.GetString("CASSANDRA_HOSTS"); val != "" {
		hosts := strings.Split(val, ",")
		config.Cassandra.Hosts = hosts
	}
	if val := viper.GetInt("CASSANDRA_PORT"); val != 0 {
		config.Cassandra.Port = val
	}
	if val := viper.GetString("CASSANDRA_KEYSPACE"); val != "" {
		config.Cassandra.Keyspace = val
	}
	if val := viper.GetString("CASSANDRA_USERNAME"); val != "" {
		config.Cassandra.Username = val
	}
	if val := viper.GetString("CASSANDRA_PASSWORD"); val != "" {
		config.Cassandra.Password = val
	}

	// Elasticsearch 설정
	if val := viper.GetString("ELASTICSEARCH_ADDRESSES"); val != "" {
		addresses := strings.Split(val, ",")
		config.Elasticsearch.Addresses = addresses
	}
	if val := viper.GetString("ELASTICSEARCH_USERNAME"); val != "" {
		config.Elasticsearch.Username = val
	}
	if val := viper.GetString("ELASTICSEARCH_PASSWORD"); val != "" {
		config.Elasticsearch.Password = val
	}
	if val := viper.GetString("ELASTICSEARCH_API_KEY"); val != "" {
		config.Elasticsearch.APIKey = val
	}

	// Vitess 설정
	if val := viper.GetString("VITESS_HOST"); val != "" {
		config.Vitess.Host = val
	}
	if val := viper.GetString("VITESS_USERNAME"); val != "" {
		config.Vitess.Username = val
	}
	if val := viper.GetString("VITESS_PASSWORD"); val != "" {
		config.Vitess.Password = val
	}
	if val := viper.GetString("VITESS_KEYSPACE"); val != "" {
		config.Vitess.Keyspace = val
	}

	// Redis 설정
	if val := viper.GetString("REDIS_HOST"); val != "" {
		config.Redis.Host = val
	}
	if val := viper.GetString("REDIS_PASSWORD"); val != "" {
		config.Redis.Password = val
	}

	// Kafka 설정
	if val := viper.GetString("KAFKA_BROKERS"); val != "" {
		brokers := strings.Split(val, ",")
		config.Kafka.Brokers = brokers
	}
	if val := viper.GetString("KAFKA_CLIENT_ID"); val != "" {
		config.Kafka.ClientID = val
	}

	// 애플리케이션 설정 (GitLab CI/CD 변수)
	if val := viper.GetString("APP_ENVIRONMENT"); val != "" {
		config.App.Environment = val
	}
	if val := viper.GetString("APP_VERSION"); val != "" {
		config.App.Version = val
	}
	if val := viper.GetString("CI_COMMIT_TAG"); val != "" {
		// GitLab CI/CD 태그가 있으면 버전으로 사용
		config.App.Version = val
	}
	if val := viper.GetString("CI_ENVIRONMENT_NAME"); val != "" {
		// GitLab CI/CD 환경 이름 사용
		config.App.Environment = val
	}

	// Observability 설정
	if val := viper.GetString("JAEGER_ENDPOINT"); val != "" {
		config.Observability.Tracing.JaegerEndpoint = val
	}
	if val := viper.GetString("LOG_LEVEL"); val != "" {
		config.Observability.Logging.Level = val
	}
}

// Validate는 설정을 검증합니다
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}

	if c.Server.HTTP.Port <= 0 {
		return fmt.Errorf("server.http.port must be positive")
	}

	if c.Server.GRPC.Port <= 0 {
		return fmt.Errorf("server.grpc.port must be positive")
	}

	if c.MongoDB.Enabled {
		if !c.MongoDB.UseVault && c.MongoDB.URI == "" {
			return fmt.Errorf("mongodb.uri is required when vault is not used")
		}
	}

	if c.PostgreSQL.Enabled {
		if !c.PostgreSQL.UseVault && (c.PostgreSQL.Host == "" || c.PostgreSQL.Database == "") {
			return fmt.Errorf("postgresql.host and postgresql.database are required when vault is not used")
		}
	}

	if c.MySQL.Enabled {
		if !c.MySQL.UseVault && (c.MySQL.Host == "" || c.MySQL.Database == "") {
			return fmt.Errorf("mysql.host and mysql.database are required when vault is not used")
		}
	}

	if c.Cassandra.Enabled {
		if !c.Cassandra.UseVault && (len(c.Cassandra.Hosts) == 0 || c.Cassandra.Keyspace == "") {
			return fmt.Errorf("cassandra.hosts and cassandra.keyspace are required when vault is not used")
		}
	}

	if c.Elasticsearch.Enabled {
		if !c.Elasticsearch.UseVault && len(c.Elasticsearch.Addresses) == 0 {
			return fmt.Errorf("elasticsearch.addresses is required when vault is not used")
		}
	}

	if c.Vitess.Enabled {
		if !c.Vitess.UseVault && (c.Vitess.Host == "" || c.Vitess.Keyspace == "") {
			return fmt.Errorf("vitess.host and vitess.keyspace are required when vault is not used")
		}
	}

	if c.Redis.Enabled {
		if c.Redis.Host == "" {
			return fmt.Errorf("redis.host is required")
		}
	}

	if c.Kafka.Enabled {
		if len(c.Kafka.Brokers) == 0 {
			return fmt.Errorf("kafka.brokers is required")
		}
	}

	if c.Vault.Enabled {
		if c.Vault.Address == "" {
			return fmt.Errorf("vault.address is required")
		}
		if c.Vault.AuthMethod == "token" && c.Vault.Token == "" {
			return fmt.Errorf("vault.token is required for token auth")
		}
	}

	return nil
}
