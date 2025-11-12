#!/bin/bash
# Vault 초기 설정 스크립트
# 이 스크립트는 Vault를 설정하고 데이터베이스 서비스에 필요한 시크릿을 생성합니다

set -e

VAULT_ADDR="${VAULT_ADDR:-http://localhost:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-root}"

echo "=== Vault Setup for Database Service ==="
echo "Vault Address: $VAULT_ADDR"
echo ""

# Vault CLI 설정
export VAULT_ADDR
export VAULT_TOKEN

# 1. Secrets Engine 활성화
echo "1. Enabling secrets engines..."

# KV v2 Secrets Engine (정적 시크릿)
vault secrets enable -path=secret kv-v2 2>/dev/null || echo "KV v2 already enabled"

# Database Secrets Engine (동적 자격증명)
vault secrets enable database 2>/dev/null || echo "Database secrets engine already enabled"

# Transit Secrets Engine (암호화/복호화)
vault secrets enable transit 2>/dev/null || echo "Transit secrets engine already enabled"

echo "✓ Secrets engines enabled"
echo ""

# 2. MongoDB 동적 자격증명 설정
echo "2. Configuring MongoDB dynamic credentials..."

# MongoDB 연결 설정
vault write database/config/mongodb \
    plugin_name=mongodb-database-plugin \
    allowed_roles="mongodb-role" \
    connection_url="mongodb://{{username}}:{{password}}@localhost:27017/admin" \
    username="admin" \
    password="admin123"

# MongoDB Role 생성 (TTL 1시간)
vault write database/roles/mongodb-role \
    db_name=mongodb \
    creation_statements='{ "db": "admin", "roles": [{ "role": "readWrite" }, {"role": "read", "db": "database_service"}] }' \
    default_ttl="1h" \
    max_ttl="24h"

echo "✓ MongoDB dynamic credentials configured"
echo ""

# 3. Redis 정적 자격증명 저장
echo "3. Storing Redis static credentials..."

vault kv put secret/redis \
    password="redis_password_123" \
    host="localhost" \
    port="6379"

echo "✓ Redis credentials stored"
echo ""

# 4. 애플리케이션 시크릿 저장
echo "4. Storing application secrets..."

vault kv put secret/app \
    api_key="your_api_key_here" \
    jwt_secret="your_jwt_secret_here" \
    encryption_key="your_encryption_key_here"

echo "✓ Application secrets stored"
echo ""

# 5. Transit 암호화 키 생성
echo "5. Creating Transit encryption keys..."

vault write -f transit/keys/database-encryption
vault write -f transit/keys/app-data-key

echo "✓ Transit encryption keys created"
echo ""

# 6. Policy 생성
echo "6. Creating Vault policies..."

# Database Service Policy
cat <<EOF | vault policy write database-service -
# MongoDB 동적 자격증명 읽기
path "database/creds/mongodb-role" {
  capabilities = ["read"]
}

# Redis 자격증명 읽기
path "secret/data/redis" {
  capabilities = ["read"]
}

# 애플리케이션 시크릿 읽기/쓰기
path "secret/data/app" {
  capabilities = ["read", "write", "delete"]
}

# Transit 암호화/복호화
path "transit/encrypt/database-encryption" {
  capabilities = ["update"]
}

path "transit/decrypt/database-encryption" {
  capabilities = ["update"]
}

path "transit/encrypt/app-data-key" {
  capabilities = ["update"]
}

path "transit/decrypt/app-data-key" {
  capabilities = ["update"]
}

# 데이터 키 생성
path "transit/datakey/plaintext/*" {
  capabilities = ["update"]
}

# 리스 갱신
path "sys/leases/renew" {
  capabilities = ["update"]
}

# 리스 취소
path "sys/leases/revoke" {
  capabilities = ["update"]
}
EOF

echo "✓ Vault policy created"
echo ""

# 7. AppRole 생성 (선택사항)
echo "7. Creating AppRole (optional)..."

vault auth enable approle 2>/dev/null || echo "AppRole already enabled"

vault write auth/approle/role/database-service \
    token_policies="database-service" \
    token_ttl=1h \
    token_max_ttl=4h

ROLE_ID=$(vault read -field=role_id auth/approle/role/database-service/role-id)
SECRET_ID=$(vault write -field=secret_id -f auth/approle/role/database-service/secret-id)

echo "✓ AppRole created"
echo ""
echo "Role ID: $ROLE_ID"
echo "Secret ID: $SECRET_ID"
echo ""

# 8. 테스트
echo "8. Testing configuration..."

# MongoDB 자격증명 테스트
echo "Testing MongoDB credentials..."
vault read database/creds/mongodb-role

# Redis 자격증명 테스트
echo "Testing Redis credentials..."
vault kv get secret/redis

# Transit 암호화 테스트
echo "Testing Transit encryption..."
CIPHERTEXT=$(vault write -field=ciphertext transit/encrypt/database-encryption plaintext=$(echo -n "test data" | base64))
echo "Encrypted: $CIPHERTEXT"
PLAINTEXT=$(vault write -field=plaintext transit/decrypt/database-encryption ciphertext=$CIPHERTEXT | base64 -d)
echo "Decrypted: $PLAINTEXT"

echo ""
echo "=== Vault Setup Complete ==="
echo ""
echo "Next steps:"
echo "1. Export environment variables:"
echo "   export VAULT_ADDR=$VAULT_ADDR"
echo "   export VAULT_TOKEN=$VAULT_TOKEN"
echo "   # OR for AppRole:"
echo "   export VAULT_ROLE_ID=$ROLE_ID"
echo "   export VAULT_SECRET_ID=$SECRET_ID"
echo ""
echo "2. Update your application config with Vault settings"
echo "3. Start your application"
echo ""
