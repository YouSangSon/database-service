# Vault Integration Guide

이 문서는 데이터베이스 서비스의 HashiCorp Vault 통합에 대한 완벽한 가이드입니다.

## 목차

- [개요](#개요)
- [기능](#기능)
- [Vault 설정](#vault-설정)
- [애플리케이션 설정](#애플리케이션-설정)
- [사용 방법](#사용-방법)
- [보안 모범 사례](#보안-모범-사례)

## 개요

HashiCorp Vault와의 통합을 통해 다음과 같은 보안 기능을 제공합니다:

- **동적 자격증명**: MongoDB 자격증명을 동적으로 생성하고 자동 갱신
- **정적 시크릿 관리**: API 키, 설정 값 등의 안전한 저장
- **암호화/복호화**: Transit Engine을 통한 데이터 암호화
- **자동 리스 갱신**: 자격증명 자동 갱신으로 서비스 중단 없음
- **다양한 인증 방법**: Token, AppRole, Kubernetes 인증 지원

## 기능

### 1. 동적 자격증명 (Dynamic Credentials)

MongoDB 데이터베이스 자격증명을 동적으로 생성하고 관리합니다.

```go
credManager := vault.NewDatabaseCredentialsManager(client)
creds, err := credManager.GetMongoDBCredentials(ctx)

// 자동 갱신 시작
credManager.StartAutoRenewal(ctx)
```

**장점:**
- 자격증명이 자동으로 로테이션됨
- 리스 만료 전 자동 갱신
- 서비스 중단 없이 자격증명 변경

### 2. 정적 시크릿 관리 (Static Secrets)

API 키, JWT 시크릿 등의 정적 시크릿을 안전하게 저장합니다.

```go
// 시크릿 저장
secrets := map[string]interface{}{
    "api_key": "your_api_key",
    "jwt_secret": "your_jwt_secret",
}
client.PutSecret(ctx, "secret/data/app", secrets)

// 시크릿 읽기
metadata, err := client.GetSecret(ctx, "secret/data/app")
apiKey := metadata.Data["api_key"].(string)
```

### 3. 암호화/복호화 (Transit Engine)

민감한 데이터를 암호화하여 저장하고 필요할 때 복호화합니다.

```go
// 암호화
ciphertext, err := client.EncryptString(ctx, "my-key", "sensitive data")

// 복호화
plaintext, err := client.DecryptString(ctx, "my-key", ciphertext)
```

**지원 기능:**
- 데이터 암호화/복호화
- 서명 생성 및 검증
- 해시 생성
- 데이터 키 생성 (Envelope Encryption)

### 4. 자동 리스 갱신 (Automatic Lease Renewal)

동적 자격증명의 리스를 자동으로 갱신합니다.

```go
// 전역 자동 갱신
client.StartRenewal(ctx)

// MongoDB 자격증명 자동 갱신
credManager.StartAutoRenewal(ctx)
```

## Vault 설정

### 1. Vault 서버 설치 및 시작

```bash
# Vault 다운로드 및 설치
wget https://releases.hashicorp.com/vault/1.15.0/vault_1.15.0_linux_amd64.zip
unzip vault_1.15.0_linux_amd64.zip
sudo mv vault /usr/local/bin/

# 개발 모드로 시작 (프로덕션에서는 사용하지 마세요!)
vault server -dev

# 프로덕션 모드
vault server -config=/etc/vault/config.hcl
```

### 2. Vault 초기 설정

제공된 스크립트를 사용하여 Vault를 설정합니다:

```bash
chmod +x configs/vault-setup.sh
./configs/vault-setup.sh
```

이 스크립트는 다음을 수행합니다:
- Secrets Engine 활성화 (KV v2, Database, Transit)
- MongoDB 동적 자격증명 설정
- Redis 정적 자격증명 저장
- Transit 암호화 키 생성
- Policy 생성
- AppRole 생성

### 3. 수동 설정 (선택사항)

#### MongoDB 동적 자격증명 설정

```bash
# Database Secrets Engine 활성화
vault secrets enable database

# MongoDB 연결 설정
vault write database/config/mongodb \
    plugin_name=mongodb-database-plugin \
    allowed_roles="mongodb-role" \
    connection_url="mongodb://{{username}}:{{password}}@localhost:27017/admin" \
    username="admin" \
    password="admin123"

# Role 생성
vault write database/roles/mongodb-role \
    db_name=mongodb \
    creation_statements='{ "db": "admin", "roles": [{ "role": "readWrite" }] }' \
    default_ttl="1h" \
    max_ttl="24h"
```

#### Transit Engine 설정

```bash
# Transit Secrets Engine 활성화
vault secrets enable transit

# 암호화 키 생성
vault write -f transit/keys/database-encryption
vault write -f transit/keys/app-data-key
```

## 애플리케이션 설정

### 1. 환경변수 설정

```bash
# Token 인증
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="your-vault-token"

# AppRole 인증
export VAULT_ADDR="http://localhost:8200"
export VAULT_ROLE_ID="your-role-id"
export VAULT_SECRET_ID="your-secret-id"
```

### 2. 설정 파일

`configs/vault-config.yaml`:

```yaml
vault:
  address: "http://localhost:8200"
  auth_method: "token"
  token: "${VAULT_TOKEN}"

  paths:
    mongodb: "database/creds/mongodb-role"
    redis: "secret/data/redis"
    secrets: "secret/data/app"
    transit: "transit"

  renewal:
    interval: "15m"
    renew_before_expiry: "5m"
    max_retries: 3
    retry_interval: "5s"

  cache:
    enabled: true
    ttl: "5m"
```

## 사용 방법

### 기본 사용법

```go
package main

import (
    "context"
    "github.com/YouSangSon/database-service/internal/pkg/vault"
)

func main() {
    ctx := context.Background()

    // 1. Vault 클라이언트 생성
    config := vault.DefaultConfig()
    config.Address = "http://localhost:8200"
    config.Token = "root"

    client, err := vault.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 2. MongoDB 자격증명 가져오기
    credManager := vault.NewDatabaseCredentialsManager(client)
    creds, err := credManager.GetMongoDBCredentials(ctx)
    if err != nil {
        panic(err)
    }

    // 3. 자동 갱신 시작
    credManager.StartAutoRenewal(ctx)

    // 4. MongoDB 연결
    // ... MongoDB 연결 로직
}
```

### 암호화 사용법

```go
// 민감한 데이터 암호화
plaintext := "credit card number: 1234-5678-9012-3456"
ciphertext, err := client.EncryptString(ctx, "database-encryption", plaintext)

// 데이터베이스에 암호화된 데이터 저장
// ...

// 필요할 때 복호화
decrypted, err := client.DecryptString(ctx, "database-encryption", ciphertext)
```

### Kubernetes 환경에서 사용

```yaml
# deployment.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: database-service
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: database-service
spec:
  template:
    spec:
      serviceAccountName: database-service
      containers:
      - name: app
        env:
        - name: VAULT_ADDR
          value: "https://vault.example.com:8200"
        - name: VAULT_AUTH_METHOD
          value: "kubernetes"
        - name: VAULT_K8S_ROLE
          value: "database-service"
```

애플리케이션 코드:

```go
config := vault.DefaultConfig()
config.Address = os.Getenv("VAULT_ADDR")
config.AuthMethod = "kubernetes"
config.K8sRole = "database-service"

client, err := vault.NewClient(config)
```

## 보안 모범 사례

### 1. 인증

✅ **권장사항:**
- 프로덕션에서는 Token 인증 대신 AppRole 또는 Kubernetes 인증 사용
- Token은 최소 권한 원칙에 따라 제한된 Policy 적용
- Token TTL을 짧게 설정하고 자동 갱신 사용

❌ **피해야 할 것:**
- Root Token을 애플리케이션에서 직접 사용
- Token을 코드에 하드코딩
- 환경변수에 Token을 평문으로 저장 (Kubernetes Secrets 사용)

### 2. TLS/SSL

프로덕션 환경에서는 반드시 TLS 활성화:

```yaml
vault:
  address: "https://vault.example.com:8200"
  tls:
    enabled: true
    skip_verify: false
    ca_cert: "/etc/vault/tls/ca.crt"
    client_cert: "/etc/vault/tls/client.crt"
    client_key: "/etc/vault/tls/client.key"
```

### 3. 자격증명 로테이션

- MongoDB 자격증명의 TTL을 적절히 설정 (1-24시간)
- 자동 갱신을 반드시 활성화
- 리스 만료 전 충분한 시간(5분)을 두고 갱신

### 4. 로깅 및 모니터링

```go
// 로깅 레벨 설정
logger.SetLevel("info") // 프로덕션
logger.SetLevel("debug") // 개발

// Health Check 정기적 수행
if err := client.HealthCheck(ctx); err != nil {
    logger.Error("vault health check failed", zap.Error(err))
    // 알림 전송
}
```

### 5. 캐싱

```yaml
vault:
  cache:
    enabled: true
    ttl: "5m"  # 너무 길면 보안 위험, 너무 짧으면 성능 저하
```

### 6. 권한 최소화

Policy 예시:

```hcl
# 최소 권한 Policy
path "database/creds/mongodb-role" {
  capabilities = ["read"]
}

path "secret/data/app" {
  capabilities = ["read"]
}

path "transit/encrypt/database-encryption" {
  capabilities = ["update"]
}

path "transit/decrypt/database-encryption" {
  capabilities = ["update"]
}
```

## 트러블슈팅

### 1. "vault is sealed" 에러

```bash
vault operator unseal
```

### 2. 인증 실패

```bash
# Token 확인
vault token lookup

# Policy 확인
vault token capabilities <path>
```

### 3. 동적 자격증명 생성 실패

```bash
# MongoDB 연결 확인
vault read database/config/mongodb

# Role 확인
vault read database/roles/mongodb-role
```

### 4. 리스 갱신 실패

로그 확인:
```
failed to renew lease: lease not found
```

해결: 새로운 자격증명 발급
```go
credManager.RevokeCredentials(ctx)
creds, _ := credManager.GetMongoDBCredentials(ctx)
```

## 참고 자료

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [Vault Go Client](https://github.com/hashicorp/vault/tree/main/api)
- [Database Secrets Engine](https://www.vaultproject.io/docs/secrets/databases)
- [Transit Secrets Engine](https://www.vaultproject.io/docs/secrets/transit)

## 지원

문제가 발생하면 다음을 확인하세요:
1. Vault 서버 상태 및 로그
2. 애플리케이션 로그
3. 네트워크 연결
4. 인증 및 권한 설정

추가 지원이 필요하면 이슈를 생성해주세요.
