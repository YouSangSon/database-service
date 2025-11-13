# Quick Start Guide

이 가이드는 Database Service를 5분 안에 로컬에서 실행하는 방법을 안내합니다.

## 📋 사전 요구사항

- Docker 20.10+
- Docker Compose 2.0+
- 8GB+ RAM (모든 서비스 실행 시)
- 포트: 3000, 6379, 8080, 8090, 8200, 9090, 9091, 16686, 27017, 29092

## 🚀 30초 안에 시작하기

```bash
# 1. 저장소 클론
git clone https://github.com/YouSangSon/database-service.git
cd database-service

# 2. 전체 스택 실행
docker-compose up -d

# 3. 로그 확인
docker-compose logs -f api
```

## 🎯 서비스 접속

실행 후 다음 서비스들에 접속할 수 있습니다:

| 서비스 | URL | 설명 | 기본 인증 |
|--------|-----|------|----------|
| **HTTP API** | http://localhost:8080 | REST API 엔드포인트 | - |
| **gRPC API** | localhost:9090 | gRPC 엔드포인트 | - |
| **API Swagger** | http://localhost:8080/swagger/index.html | API 문서 (구현 예정) | - |
| **Prometheus** | http://localhost:9090 | 메트릭 조회 | - |
| **Grafana** | http://localhost:3000 | 대시보드 | admin / admin |
| **Jaeger UI** | http://localhost:16686 | 분산 추적 | - |
| **Kafka UI** | http://localhost:8090 | Kafka 토픽 모니터링 | - |
| **Vault UI** | http://localhost:8200 | 시크릿 관리 | Token: dev-only-token |

## 📝 API 테스트

### Health Check
```bash
curl http://localhost:8080/health
```

### 문서 생성 (MongoDB - 기본값)
```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "collection": "users",
    "data": {
      "name": "John Doe",
      "email": "john@example.com",
      "age": 30
    }
  }'
```

응답:
```json
{
  "id": "507f1f77bcf86cd799439011",
  "collection": "users",
  "data": {
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  },
  "created_at": "2025-11-12T08:30:00Z",
  "updated_at": "2025-11-12T08:30:00Z",
  "version": 1
}
```

### 다른 데이터베이스 사용 (동적 선택)

`X-Database-Type` 헤더를 추가하여 다른 데이터베이스를 사용할 수 있습니다:

```bash
# PostgreSQL 사용
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: postgresql" \
  -d '{
    "collection": "users",
    "data": {
      "name": "Jane Doe",
      "email": "jane@example.com"
    }
  }'

# MySQL 사용
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: mysql" \
  -d '{
    "collection": "products",
    "data": {
      "name": "Product A",
      "price": 99.99
    }
  }'
```

**지원 데이터베이스:**
- `mongodb` (기본값, 현재 활성화)
- `postgresql` (설정에서 활성화 필요)
- `mysql` (설정에서 활성화 필요)
- `cassandra` (설정에서 활성화 필요)
- `elasticsearch` (설정에서 활성화 필요)
- `vitess` (설정에서 활성화 필요)

> ⚠️ **참고**: 다른 데이터베이스를 사용하려면 `configs/config.yaml`에서 해당 데이터베이스를 활성화해야 합니다.

### 문서 조회
```bash
# ID는 생성 시 반환된 값 사용
curl http://localhost:8080/api/v1/documents/users/{id}
```

### 문서 목록 조회
```bash
curl "http://localhost:8080/api/v1/documents/users?limit=10&offset=0"
```

### 문서 업데이트
```bash
curl -X PUT http://localhost:8080/api/v1/documents/users/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "age": 31
  }'
```

### 문서 삭제
```bash
curl -X DELETE http://localhost:8080/api/v1/documents/users/{id}
```

## 🔍 관찰성 (Observability)

### 1. Jaeger로 요청 추적 확인

1. 브라우저에서 http://localhost:16686 열기
2. Service: `database-service` 선택
3. Operation: `DocumentUseCase.CreateDocument` 또는 다른 작업 선택
4. "Find Traces" 클릭
5. Trace를 클릭하여 상세 타임라인 확인

**확인 가능한 정보:**
- 요청 전체 실행 시간
- 각 레이어별 소요 시간 (Handler → UseCase → Repository → DB)
- MongoDB 쿼리 실행 시간
- Redis 캐시 조회 시간
- 에러 발생 위치

### 2. Prometheus로 메트릭 조회

1. 브라우저에서 http://localhost:9090 열기
2. Graph 탭에서 메트릭 쿼리:

```promql
# HTTP 요청률 (초당 요청 수)
rate(http_requests_total[5m])

# HTTP P95 레이턴시
histogram_quantile(0.95, http_request_duration_seconds_bucket)

# 데이터베이스 작업 에러율
rate(db_operations_total{status="error"}[5m])

# 캐시 히트율
rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))
```

### 3. Grafana 대시보드 (구성 예정)

1. http://localhost:3000 접속
2. 로그인: admin / admin
3. Configuration → Data Sources → Add Prometheus
   - URL: http://prometheus:9090
   - Save & Test
4. 대시보드 생성 또는 import

## 📊 Kafka 이벤트 확인

### Kafka UI에서 이벤트 확인

1. http://localhost:8090 접속
2. Topics 탭 선택
3. 다음 토픽 확인:
   - `documents.created` - 문서 생성 이벤트
   - `documents.updated` - 문서 업데이트 이벤트
   - `documents.deleted` - 문서 삭제 이벤트
4. Messages 탭에서 실제 이벤트 데이터 확인

### Kafka CLI로 이벤트 소비

```bash
# Kafka 컨테이너 접속
docker exec -it database-service-kafka bash

# 토픽 목록 확인
kafka-topics --list --bootstrap-server localhost:9092

# 문서 생성 이벤트 소비
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic documents.created \
  --from-beginning \
  --property print.key=true \
  --property key.separator=":"
```

이벤트 형식:
```json
{
  "event_id": "507f1f77bcf86cd799439011-1699780200000000000",
  "event_type": "document.created",
  "timestamp": "2025-11-12T08:30:00Z",
  "document_id": "507f1f77bcf86cd799439011",
  "collection": "users",
  "data": {
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  },
  "version": 1
}
```

## 🔐 Vault 시크릿 확인

### Vault UI 접속

1. http://localhost:8200 접속
2. Token으로 로그인: `dev-only-token`
3. Secrets 탭에서 KV 엔진 탐색

### Vault CLI 사용

```bash
# Vault 컨테이너 접속
docker exec -it database-service-vault sh

# 환경변수 설정
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='dev-only-token'

# 시크릿 읽기
vault kv get secret/production/app

# 시크릿 쓰기
vault kv put secret/production/app \
  api_key="test-api-key" \
  jwt_secret="test-jwt-secret"
```

## 🐛 트러블슈팅

### 서비스가 시작되지 않는 경우

```bash
# 로그 확인
docker-compose logs api
docker-compose logs mongodb
docker-compose logs redis

# 서비스 재시작
docker-compose restart api

# 전체 재시작
docker-compose down
docker-compose up -d
```

### 포트 충돌

이미 사용 중인 포트가 있다면 `docker-compose.yml`에서 포트 매핑 수정:

```yaml
services:
  api:
    ports:
      - "8081:8080"  # 호스트 포트를 8081로 변경
```

### MongoDB 연결 실패

```bash
# MongoDB 헬스체크 확인
docker-compose ps mongodb

# MongoDB 로그 확인
docker-compose logs mongodb

# MongoDB 직접 접속 테스트
docker exec -it database-service-mongodb mongosh -u admin -p password
```

### Redis 연결 실패

```bash
# Redis 테스트
docker exec -it database-service-redis redis-cli -a redispassword ping

# 응답: PONG
```

### Kafka 연결 실패

```bash
# Kafka 브로커 확인
docker exec -it database-service-kafka kafka-broker-api-versions \
  --bootstrap-server localhost:9092

# Zookeeper 확인
docker exec -it database-service-zookeeper \
  zkServer.sh status
```

## 🧹 정리

### 서비스 중지 (데이터 유지)

```bash
docker-compose stop
```

### 서비스 완전 삭제 (데이터 포함)

```bash
docker-compose down -v
```

### 특정 서비스만 실행

```bash
# MongoDB와 API만 실행
docker-compose up -d mongodb api

# Redis와 Kafka 추가
docker-compose up -d redis kafka
```

## 📚 다음 단계

1. **테스트 실행**: [Testing Guide](../test/README.md)
2. **프로덕션 배포**: [Deployment Guide](./DEPLOYMENT.md)
3. **아키텍처 이해**: [Architecture Guide](./ARCHITECTURE.md)
4. **Vault 설정**: [Vault Integration](./VAULT_INTEGRATION.md)
5. **고도화**: [Enhancement Recommendations](./ENHANCEMENT_RECOMMENDATIONS.md)

## 💡 팁

### 성능 최적화

로컬 개발 시 일부 서비스만 실행하여 리소스 절약:

```bash
# 필수 서비스만 (MongoDB, API)
docker-compose up -d mongodb api

# 관찰성 추가 (Jaeger)
docker-compose up -d jaeger

# 메시징 추가 (Kafka)
docker-compose up -d zookeeper kafka
```

### 실시간 로그 확인

```bash
# API 서버 로그만
docker-compose logs -f api

# 여러 서비스 동시 확인
docker-compose logs -f api mongodb redis
```

### 환경변수 오버라이드

```bash
# 특정 환경변수 변경
APP_LOG_LEVEL=debug docker-compose up -d api
```

## 🔗 유용한 명령어 모음

```bash
# 모든 컨테이너 상태 확인
docker-compose ps

# 리소스 사용량 확인
docker stats

# 네트워크 확인
docker network ls
docker network inspect database-service_database-service-network

# 볼륨 확인
docker volume ls
docker volume inspect database-service_mongodb_data

# 컨테이너 내부 접속
docker exec -it database-service-api sh
docker exec -it database-service-mongodb mongosh -u admin -p password

# 빌드 캐시 제거 후 재빌드
docker-compose build --no-cache
docker-compose up -d --force-recreate
```

## 📞 도움 필요 시

- **GitHub Issues**: https://github.com/YouSangSon/database-service/issues
- **Documentation**: [README.md](../README.md)
- **Architecture**: [ARCHITECTURE.md](./ARCHITECTURE.md)
