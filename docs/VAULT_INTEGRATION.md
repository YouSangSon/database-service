# Vault Integration Guide

HashiCorp Vault와의 통합 가이드입니다. 이 문서는 Vault를 사용하여 데이터베이스 서비스의 보안을 강화하는 방법을 설명합니다.

## 목차

1. [개요](#개요)
2. [인증 방식](#인증-방식)
3. [동적 자격증명](#동적-자격증명)
4. [정적 시크릿](#정적-시크릿)
5. [Transit 암호화](#transit-암호화)
6. [Lease 관리](#lease-관리)
7. [설정](#설정)
8. [예제](#예제)

## 개요

```mermaid
graph TB
    subgraph "Application"
        APP[Database Service]
    end

    subgraph "Vault"
        AUTH[Authentication]
        DB_ENGINE[Database Engine]
        KV_ENGINE[KV Secrets Engine]
        TRANSIT_ENGINE[Transit Engine]
    end

    subgraph "Backends"
        MONGO[(MongoDB)]
        VITESS[(Vitess)]
        REDIS[(Redis)]
    end

    APP -->|1. Authenticate| AUTH
    AUTH -->|2. Token| APP

    APP -->|3. Get MongoDB Creds| DB_ENGINE
    DB_ENGINE -->|4. Dynamic Creds| APP
    APP -->|5. Connect| MONGO

    APP -->|6. Get Vitess Creds| DB_ENGINE
    DB_ENGINE -->|7. Dynamic Creds| APP
    APP -->|8. Connect| VITESS

    APP -->|9. Get Redis Password| KV_ENGINE
    KV_ENGINE -->|10. Static Secret| APP
    APP -->|11. Connect| REDIS

    APP -->|12. Encrypt Data| TRANSIT_ENGINE
    TRANSIT_ENGINE -->|13. Encrypted| APP

    style APP fill:#90caf9,color:#fff
    style AUTH fill:#ffb74d,color:#000
    style DB_ENGINE fill:#81c784,color:#fff
    style KV_ENGINE fill:#f06292,color:#fff
    style TRANSIT_ENGINE fill:#9575cd,color:#fff
```

**주요 기능:**
- **인증**: Token, AppRole, Kubernetes Service Account
- **동적 자격증명**: MongoDB, Vitess (자동 생성, 로테이션, 만료)
- **정적 시크릿**: API Keys, Redis Password
- **Transit 암호화**: 민감한 데이터 암호화/복호화
- **Lease 관리**: 자동 갱신 및 재시도

## 인증 방식

### 1. Token 인증 (개발 환경)

```mermaid
sequenceDiagram
    autonumber
    participant App
    participant Vault

    App->>Vault: Authenticate with Token
    Vault->>Vault: Validate Token
    Vault-->>App: Authenticated (Client)

    App->>Vault: Request Secret
    Vault-->>App: Secret Data
```

**사용 시나리오:**
- 로컬 개발
- 테스트 환경
- 빠른 프로토타이핑

**설정:**
```go
vault:
  enabled: true
  address: "http://localhost:8200"
  token: "dev-only-token"
  auth_method: "token"
```

### 2. AppRole 인증 (프로덕션)

```mermaid
sequenceDiagram
    autonumber
    participant App
    participant Vault

    App->>Vault: Login with RoleID + SecretID
    Vault->>Vault: Validate Credentials
    Vault->>Vault: Generate Token
    Vault-->>App: Access Token (TTL)

    loop Token Renewal
        App->>Vault: Renew Token
        Vault-->>App: Token Renewed
    end

    App->>Vault: Request Secret (with Token)
    Vault-->>App: Secret Data
```

**사용 시나리오:**
- 프로덕션 환경
- CI/CD 파이프라인
- 자동화된 배포

**설정:**
```go
vault:
  enabled: true
  address: "https://vault.production.svc.cluster.local:8200"
  auth_method: "approle"
  role_id: "${VAULT_ROLE_ID}"     # GitLab CI/CD Variable
  secret_id: "${VAULT_SECRET_ID}" # GitLab CI/CD Variable
```

### 3. Kubernetes Service Account 인증

```mermaid
sequenceDiagram
    autonumber
    participant Pod
    participant K8s API
    participant Vault

    Note over Pod: Read ServiceAccount Token<br/>from /var/run/secrets/...

    Pod->>K8s API: Verify JWT Token
    K8s API-->>Pod: Token Verified

    Pod->>Vault: Login with JWT + Role
    Vault->>K8s API: Validate JWT
    K8s API-->>Vault: Valid
    Vault->>Vault: Generate Token
    Vault-->>Pod: Access Token

    Pod->>Vault: Request Secret
    Vault-->>Pod: Secret Data
```

**사용 시나리오:**
- Kubernetes 프로덕션 환경
- 자동 인증 (no manual credentials)
- 보안 강화

**설정:**
```go
vault:
  enabled: true
  address: "https://vault.production.svc.cluster.local:8200"
  auth_method: "kubernetes"
  k8s_role: "database-service"
  namespace: "production"
```

## 동적 자격증명

### MongoDB 동적 자격증명

```mermaid
sequenceDiagram
    autonumber
    participant App
    participant Vault
    participant MongoDB

    App->>Vault: Request MongoDB Credentials<br/>(path: database/creds/mongodb-role)

    Vault->>MongoDB: CREATE USER 'v-token-xyz-...'<br/>WITH ROLES ['readWrite']
    MongoDB-->>Vault: User Created

    Vault-->>App: {<br/>  username: "v-token-xyz-..."<br/>  password: "A1b2C3d4..."<br/>  ttl: 1h<br/>  lease_id: "database/creds/..."<br/>}

    App->>MongoDB: Connect(username, password)
    MongoDB-->>App: Connected

    Note over App,Vault: Before TTL expires...

    App->>Vault: Renew Lease (lease_id)
    Vault-->>App: Lease Extended (new TTL: 1h)

    Note over App,Vault: After max TTL or revoke...

    App->>Vault: Revoke Lease
    Vault->>MongoDB: DROP USER 'v-token-xyz-...'
    MongoDB-->>Vault: User Deleted
    Vault-->>App: Lease Revoked
```

**특징:**
- **자동 생성**: Vault가 사용자를 자동으로 생성
- **TTL**: 1-24시간 (설정 가능)
- **자동 로테이션**: TTL 만료 전 자동 갱신
- **자동 삭제**: Lease 만료 또는 취소 시 사용자 삭제

**Vault 설정 (관리자):**
```bash
# MongoDB 연결 설정
vault write database/config/mongodb \
  plugin_name=mongodb-database-plugin \
  allowed_roles="mongodb-role" \
  connection_url="mongodb://{{username}}:{{password}}@mongodb:27017/admin" \
  username="vault-admin" \
  password="vault-password"

# Role 설정
vault write database/roles/mongodb-role \
  db_name=mongodb \
  creation_statements='{ "db": "database_service", "roles": [{ "role": "readWrite" }] }' \
  default_ttl="1h" \
  max_ttl="24h"
```

### Vitess 동적 자격증명

```mermaid
sequenceDiagram
    autonumber
    participant App
    participant Vault
    participant Vitess

    App->>Vault: Request Vitess Credentials<br/>(path: database/creds/vitess-role)

    Vault->>Vitess: CREATE USER 'v-token-abc-...'<br/>IDENTIFIED BY '...'
    Vault->>Vitess: GRANT SELECT, INSERT, UPDATE, DELETE<br/>ON keyspace.* TO 'v-token-abc-...'
    Vitess-->>Vault: User Created & Granted

    Vault-->>App: {<br/>  username: "v-token-abc-..."<br/>  password: "X9y8Z7..."<br/>  ttl: 2h<br/>  lease_id: "database/creds/..."<br/>}

    App->>Vitess: Connect(username, password)
    Vitess-->>App: Connected

    Note over App,Vault: Auto-renewal...

    App->>Vault: Renew Lease
    Vault-->>App: Lease Extended
```

**특징:**
- MySQL 프로토콜 사용
- 동적 사용자 생성 및 권한 부여
- 자동 정리 (revoke 시 DROP USER)

**Vault 설정 (관리자):**
```bash
# Vitess 연결 설정
vault write database/config/vitess \
  plugin_name=mysql-database-plugin \
  allowed_roles="vitess-role" \
  connection_url="{{username}}:{{password}}@tcp(vtgate:15306)/" \
  username="vault-admin" \
  password="vault-password"

# Role 설정
vault write database/roles/vitess-role \
  db_name=vitess \
  creation_statements="CREATE USER '{{name}}'@'%' IDENTIFIED BY '{{password}}';GRANT SELECT, INSERT, UPDATE, DELETE ON commerce.* TO '{{name}}'@'%';" \
  default_ttl="2h" \
  max_ttl="24h"
```

## 정적 시크릿

### KV Secrets Engine

```mermaid
graph LR
    subgraph "Vault KV Engine v2"
        S1[secret/data/production/redis<br/>{password: "..."}]
        S2[secret/data/production/app<br/>{api_key: "...", jwt_secret: "..."}]
        S3[secret/data/production/external<br/>{third_party_api: "..."}]
    end

    APP[Application] -->|Get Secret| S1
    APP -->|Get Secret| S2
    APP -->|Get Secret| S3

    style APP fill:#90caf9,color:#fff
    style S1 fill:#f06292,color:#fff
    style S2 fill:#f06292,color:#fff
    style S3 fill:#f06292,color:#fff
```

**사용 사례:**
- Redis 비밀번호
- API 키
- JWT Secret
- 외부 서비스 토큰

**코드 예제:**
```go
// Redis 비밀번호 가져오기
password, err := vaultClient.GetRedisCredentials(ctx)
if err != nil {
    log.Fatalf("Failed to get Redis password: %v", err)
}

// 애플리케이션 시크릿 가져오기
secrets, err := vaultClient.GetStaticSecret(ctx, "secret/data/production/app")
if err != nil {
    log.Fatalf("Failed to get secrets: %v", err)
}

apiKey := secrets["api_key"].(string)
jwtSecret := secrets["jwt_secret"].(string)
```

## Transit 암호화

### Encrypt/Decrypt Flow

```mermaid
sequenceDiagram
    autonumber
    participant App
    participant Vault
    participant Database

    Note over App: Sensitive Data:<br/>"Credit Card: 1234-5678-9012-3456"

    App->>Vault: POST /transit/encrypt/database-encryption<br/>{plaintext: base64(data)}
    Vault->>Vault: Encrypt with AES-256-GCM
    Vault-->>App: {ciphertext: "vault:v1:abcd..."}

    App->>Database: INSERT INTO documents<br/>(data = "vault:v1:abcd...")
    Database-->>App: OK

    Note over App: Later... retrieval

    App->>Database: SELECT data FROM documents
    Database-->>App: "vault:v1:abcd..."

    App->>Vault: POST /transit/decrypt/database-encryption<br/>{ciphertext: "vault:v1:abcd..."}
    Vault->>Vault: Decrypt with Key
    Vault-->>App: {plaintext: base64(data)}

    Note over App: Decrypted:<br/>"Credit Card: 1234-5678-9012-3456"
```

**특징:**
- **Encryption as a Service**: 애플리케이션이 암호화 키를 직접 보유하지 않음
- **Key Rotation**: 자동 키 로테이션 (ciphertext는 버전 포함)
- **고성능**: Vault가 암호화 전용으로 최적화
- **Convergent Encryption**: 동일한 plaintext → 동일한 ciphertext (캐싱 가능)

**코드 예제:**
```go
// 암호화
sensitiveData := "Credit Card: 1234-5678-9012-3456"
encrypted, err := vaultClient.EncryptString(ctx, "database-encryption", sensitiveData)
if err != nil {
    return err
}
// encrypted: "vault:v1:8SDd3WHDOjf7..."

// 데이터베이스에 저장
doc.Data["credit_card"] = encrypted
repo.Save(ctx, doc)

// 복호화
encrypted := doc.Data["credit_card"].(string)
decrypted, err := vaultClient.DecryptString(ctx, "database-encryption", encrypted)
if err != nil {
    return err
}
// decrypted: "Credit Card: 1234-5678-9012-3456"
```

### Key Rotation

```mermaid
sequenceDiagram
    autonumber
    participant Admin
    participant Vault
    participant App

    Admin->>Vault: Rotate Encryption Key<br/>(transit/keys/database-encryption/rotate)
    Vault->>Vault: Create New Key Version (v2)
    Vault-->>Admin: Key Rotated (now at v2)

    Note over Vault: Old ciphertexts (v1) still valid

    App->>Vault: Encrypt (uses latest key v2)
    Vault-->>App: vault:v2:newcipher...

    App->>Vault: Decrypt (vault:v1:oldcipher...)
    Vault->>Vault: Use Key Version 1
    Vault-->>App: Plaintext (still works!)

    Note over App,Vault: Re-wrap to new version

    App->>Vault: POST /transit/rewrap<br/>(vault:v1:oldcipher...)
    Vault->>Vault: Decrypt with v1, Encrypt with v2
    Vault-->>App: vault:v2:newcipher...
```

**Key Rotation 전략:**
1. **정기 로테이션**: 매월 또는 분기별
2. **침해 대응**: 키 노출 시 즉시 로테이션
3. **Re-wrap**: 점진적으로 old version → new version

## Lease 관리

### Automatic Renewal

```mermaid
graph TD
    A[Application Start] --> B{Vault Enabled?}
    B -->|Yes| C[Authenticate]
    B -->|No| Z[Skip Vault]

    C --> D[Get Dynamic Credentials]
    D --> E[Start Renewal Goroutine]

    E --> F{Check TTL}
    F -->|TTL > Renew Threshold| G[Sleep]
    G --> F

    F -->|TTL < Renew Threshold| H[Renew Lease]
    H --> I{Renewal Success?}
    I -->|Yes| J[Update TTL]
    J --> F

    I -->|No| K[Retry with Backoff]
    K --> L{Max Retries?}
    L -->|No| H
    L -->|Yes| M[Alert & Fail]

    style A fill:#90caf9,color:#fff
    style C fill:#ffb74d,color:#000
    style E fill:#81c784,color:#fff
    style H fill:#ffd54f,color:#000
    style M fill:#ff8a65,color:#fff
```

**Renewal 설정:**
```yaml
vault:
  renewal:
    interval: 10m              # 갱신 체크 주기
    renew_before_expiry: 3m    # TTL 만료 3분 전에 갱신
    max_retries: 5             # 최대 재시도 횟수
    retry_interval: 3s         # 재시도 간격
```

**자동 갱신 로직:**
```go
// Lease 자동 갱신 (백그라운드 고루틴)
func (c *Client) startRenewal(ctx context.Context, secret *api.Secret) {
    go func() {
        ticker := time.NewTicker(c.config.RenewInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                ttl := time.Duration(secret.LeaseDuration) * time.Second
                renewThreshold := c.config.RenewBeforeExpiry

                if ttl < renewThreshold {
                    // Renew lease
                    newSecret, err := c.client.Sys().Renew(secret.LeaseID, 0)
                    if err != nil {
                        log.Error("Failed to renew lease", zap.Error(err))
                        // Retry logic...
                        continue
                    }
                    secret = newSecret
                    log.Info("Lease renewed", zap.Duration("new_ttl", ttl))
                }
            }
        }
    }()
}
```

## 설정

### 개발 환경 (config.yaml)

```yaml
vault:
  enabled: false                      # Vault 비활성화 (로컬 개발)
  address: "http://localhost:8200"
  token: "dev-only-token"
  auth_method: "token"
```

### 프로덕션 환경 (config.production.yaml)

```yaml
vault:
  enabled: true
  address: "https://vault.production.svc.cluster.local:8200"
  auth_method: "kubernetes"           # Kubernetes Service Account
  k8s_role: "database-service"
  namespace: "production"

  tls:
    enabled: true
    skip_verify: false
    ca_cert: "/etc/vault/tls/ca.crt"
    client_cert: "/etc/vault/tls/client.crt"
    client_key: "/etc/vault/tls/client.key"

  paths:
    mongodb: "database/creds/production-mongodb"
    vitess: "database/creds/production-vitess"
    redis: "secret/data/production/redis"
    secrets: "secret/data/production/app"
    transit: "transit"

  renewal:
    interval: 10m
    renew_before_expiry: 3m
    max_retries: 5
    retry_interval: 3s

  cache:
    enabled: true
    ttl: 3m
```

### GitLab CI/CD 변수

```bash
# Vault 설정
VAULT_ADDRESS=https://vault.example.com:8200
VAULT_ROLE_ID=<your-role-id>
VAULT_SECRET_ID=<your-secret-id>
VAULT_NAMESPACE=production

# 또는 환경변수로 직접 전달
APP_VAULT_ADDRESS=https://vault.example.com:8200
APP_VAULT_ROLE_ID=<your-role-id>
APP_VAULT_SECRET_ID=<your-secret-id>
```

## 예제

### Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/YouSangSon/database-service/internal/config"
    "github.com/YouSangSon/database-service/internal/pkg/vault"
)

func main() {
    ctx := context.Background()

    // 1. Load configuration
    cfg, err := config.LoadConfig("./configs", "config")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 2. Initialize Vault client
    vaultConfig := &vault.Config{
        Address:           cfg.Vault.Address,
        Token:             cfg.Vault.Token,
        AuthMethod:        cfg.Vault.AuthMethod,
        RoleID:            cfg.Vault.RoleID,
        SecretID:          cfg.Vault.SecretID,
        K8sRole:           cfg.Vault.K8sRole,
        MongoDBPath:       cfg.Vault.Paths.MongoDB,
        RenewInterval:     cfg.Vault.Renewal.Interval,
        RenewBeforeExpiry: cfg.Vault.Renewal.RenewBeforeExpiry,
    }

    vaultClient, err := vault.NewClient(vaultConfig)
    if err != nil {
        log.Fatalf("Failed to create vault client: %v", err)
    }
    defer vaultClient.Close()

    // 3. Health check
    if err := vaultClient.HealthCheck(ctx); err != nil {
        log.Fatalf("Vault health check failed: %v", err)
    }
    fmt.Println("✓ Vault health check passed")

    // 4. Get MongoDB dynamic credentials
    username, password, err := vaultClient.GetMongoDBCredentials(ctx)
    if err != nil {
        log.Fatalf("Failed to get MongoDB credentials: %v", err)
    }
    fmt.Printf("✓ MongoDB credentials: %s / %s\n", username, "***")

    // 5. Get Vitess dynamic credentials
    vUsername, vPassword, err := vaultClient.GetVitessCredentials(ctx, cfg.Vault.Paths.Vitess)
    if err != nil {
        log.Fatalf("Failed to get Vitess credentials: %v", err)
    }
    fmt.Printf("✓ Vitess credentials: %s / %s\n", vUsername, "***")

    // 6. Get Redis static secret
    redisPassword, err := vaultClient.GetRedisCredentials(ctx)
    if err != nil {
        log.Fatalf("Failed to get Redis password: %v", err)
    }
    fmt.Printf("✓ Redis password: %s\n", "***")

    // 7. Encrypt sensitive data
    sensitiveData := "Credit Card: 1234-5678-9012-3456"
    encrypted, err := vaultClient.EncryptString(ctx, "database-encryption", sensitiveData)
    if err != nil {
        log.Fatalf("Failed to encrypt data: %v", err)
    }
    fmt.Printf("✓ Encrypted: %s...\n", encrypted[:50])

    // 8. Decrypt sensitive data
    decrypted, err := vaultClient.DecryptString(ctx, "database-encryption", encrypted)
    if err != nil {
        log.Fatalf("Failed to decrypt data: %v", err)
    }
    fmt.Printf("✓ Decrypted: %s\n", decrypted)

    fmt.Println("\n=== All Vault operations completed successfully ===")
}
```

### 실행:
```bash
# Local development (Vault disabled)
go run examples/vault_example.go

# With Vault enabled
export VAULT_TOKEN=your-token
go run examples/vault_example.go
```

## 참고 자료

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [Vault Database Secrets Engine](https://www.vaultproject.io/docs/secrets/databases)
- [Vault Transit Secrets Engine](https://www.vaultproject.io/docs/secrets/transit)
- [Kubernetes Auth Method](https://www.vaultproject.io/docs/auth/kubernetes)
- [Architecture Guide](./ARCHITECTURE.md)
