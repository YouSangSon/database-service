# Database Service - Enterprise Edition

엔터프라이즈급 Go 기반의 범용 데이터베이스 서비스입니다. DDD(Domain-Driven Design), TDD(Test-Driven Development), Clean Architecture를 적용하여 대규모 프로덕션 환경에서 사용할 수 있도록 설계되었습니다.

## 🚀 주요 특징

### 아키텍처
- ✅ **DDD + Clean Architecture**: 도메인 주도 설계 및 4계층 아키텍처 (Domain, Application, Infrastructure, Interface)
- ✅ **Repository Pattern**: 데이터베이스 추상화를 통한 구현체 교체 가능
- ✅ **RepositoryManager Pattern**: 동적 멀티 데이터베이스 선택 및 관리
- ✅ **Dynamic Database Selection**: `X-Database-Type` 헤더로 런타임에 데이터베이스 선택
- ✅ **이벤트 기반**: Kafka CDC를 통한 확장 가능한 이벤트 기반 아키텍처
- ✅ **SOLID 원칙**: 의존성 역전, 단일 책임 등 객체지향 설계 원칙 준수

### 데이터베이스 지원 (6개)

**현재 활성화:**
- ✅ **MongoDB**: 30+ 고급 메서드 지원 (집계, 벌크 연산, 원자적 연산, 인덱스 관리, Change Streams 등)
  - 상태: **운영 중** (기본 데이터베이스)

**설정에서 활성화 가능:**
- ⚙️ **PostgreSQL**: 30+ 고급 메서드 지원 (JSONB 기반 유연한 문서 저장, 트랜잭션, 인덱스 관리)
  - 상태: 리포지토리 구현 완료, `configs/config.yaml`에서 `postgresql.enabled: true` 설정 후 사용
- ⚙️ **MySQL**: 30+ 고급 메서드 지원 (JSON 타입 지원, 트랜잭션, 인덱스 관리)
  - 상태: 리포지토리 구현 완료, `configs/config.yaml`에서 `mysql.enabled: true` 설정 후 사용
- ⚙️ **Cassandra**: 20+ 메서드 지원 (분산 NoSQL, CQL, LWT)
  - 상태: 리포지토리 구현 완료, `configs/config.yaml`에서 `cassandra.enabled: true` 설정 후 사용
- ⚙️ **Elasticsearch**: 25+ 메서드 지원 (전문 검색, 집계, 인덱싱)
  - 상태: 리포지토리 구현 완료, `configs/config.yaml`에서 `elasticsearch.enabled: true` 설정 후 사용
- ⚙️ **Vitess**: 30+ 고급 메서드 지원 (MongoDB와 동일한 인터페이스로 SQL 구현)
  - 상태: 리포지토리 구현 완료, `configs/config.yaml`에서 `vitess.enabled: true` 설정 후 사용

**공통 기능:**
- ✅ **36개 REST API 엔드포인트**: 모든 데이터베이스에서 동일한 API 사용
- ✅ **동적 선택**: `X-Database-Type` 헤더로 요청별 데이터베이스 선택
- ✅ **RepositoryManager**: 멀티 데이터베이스 동시 실행 및 관리
- ✅ **Raw Query 실행**: 각 DB별 네이티브 쿼리 실행 지원

### 인프라스트럭처
- ✅ **Redis 확장 기능**: 캐싱, Pub/Sub, Rate Limiting, Distributed Lock, Counter
- ✅ **Kafka CDC**: 데이터 변경 이벤트 자동 발행 (documents.created, documents.updated, documents.deleted)
- ✅ **HashiCorp Vault**: 동적 자격증명, 정적 시크릿, Transit 암호화 통합

### 프로토콜
- ✅ **REST API**: Gin 프레임워크 기반 HTTP/HTTPS API
- ✅ **gRPC**: Protocol Buffers 기반 고성능 RPC
- ✅ **이중 서버**: HTTP(8080)와 gRPC(9090) 동시 실행

### 확장성 & 성능
- ✅ **Kubernetes 네이티브**: HPA(Horizontal Pod Autoscaler) 기반 자동 스케일링
- ✅ **멀티 Pod 지원**: 3-10개 Pod 자동 확장 (CPU 70%, Memory 80% 기준)
- ✅ **동시성 처리**: Goroutine 및 Context 기반 동시 요청 처리
- ✅ **연결 풀링**: 6개 DB 모두 연결 풀 최적화
- ✅ **분산 캐싱**: Redis 기반 캐시 히트율 향상

### 보안
- ✅ **Vault 동적 자격증명**: MongoDB, Vitess 사용자 자동 생성/로테이션/삭제
- ✅ **Vault Transit 암호화**: 민감 데이터 암호화/복호화 (Encryption as a Service)
- ✅ **인증 방식**: Token, AppRole, Kubernetes Service Account
- ✅ **자동 Lease 갱신**: 자격증명 TTL 만료 전 자동 갱신

### 관찰성 (Observability)
- ✅ **구조화된 로깅**: Zap logger 기반 JSON 구조화 로그
- ✅ **분산 추적**: OpenTelemetry + Jaeger 통합
- ✅ **메트릭 수집**: Prometheus 메트릭 (요청률, 에러율, 지연시간, 캐시 히트율 등)
- ✅ **AlertManager**: 100+ 알림 규칙, Slack/Email/PagerDuty 통합
- ✅ **Grafana Dashboards**: 실시간 모니터링 대시보드, Auto-provisioning

### 안정성
- ✅ **Circuit Breaker**: 장애 전파 방지
- ✅ **Retry Logic**: Exponential backoff 재시도
- ✅ **Graceful Shutdown**: 안전한 서비스 종료 (15초 대기)
- ✅ **Health Checks**: Liveness & Readiness 프로브

### CI/CD
- ✅ **GitLab CI/CD**: 자동화된 빌드, 테스트, 배포 파이프라인
- ✅ **단계별 배포**: Development → Staging → Production
- ✅ **Docker 멀티스테이지 빌드**: 최적화된 컨테이너 이미지 생성

## 📁 프로젝트 구조

```
.
├── cmd/                                  # 애플리케이션 진입점
│   ├── api/                              # REST API 서버 (포트 8080)
│   │   ├── main.go                       # 메인 진입점 (MongoDB 활성화)
│   │   └── main_complete.go              # 6개 DB 모두 초기화 예제
│   └── grpc/                             # gRPC 서버 (포트 9090)
├── internal/
│   ├── domain/                           # 도메인 레이어 (DDD)
│   │   ├── entity/                       # 도메인 엔티티 (Document)
│   │   ├── repository/                   # 리포지토리 인터페이스
│   │   └── valueobject/                  # 값 객체
│   ├── application/                      # 애플리케이션 레이어
│   │   ├── usecase/                      # 유즈케이스 (비즈니스 로직)
│   │   └── dto/                          # 데이터 전송 객체
│   ├── infrastructure/                   # 인프라 레이어
│   │   ├── persistence/                  # 영속성
│   │   │   ├── repository_manager.go     # RepositoryManager (멀티 DB 관리)
│   │   │   ├── mongodb/                  # MongoDB 구현 (30+ 메서드)
│   │   │   ├── postgresql/               # PostgreSQL 구현 (30+ 메서드)
│   │   │   ├── mysql/                    # MySQL 구현 (30+ 메서드)
│   │   │   ├── cassandra/                # Cassandra 구현 (20+ 메서드)
│   │   │   ├── elasticsearch/            # Elasticsearch 구현 (25+ 메서드)
│   │   │   └── vitess/                   # Vitess 구현 (30+ 메서드)
│   │   ├── cache/                        # Redis 캐시 및 확장 기능
│   │   ├── messaging/                    # Kafka 메시징
│   │   └── monitoring/                   # 모니터링 (메트릭, 추적)
│   ├── interfaces/                       # 인터페이스 레이어
│   │   ├── http/                         # HTTP 핸들러 (Gin)
│   │   │   ├── middleware/               # HTTP 미들웨어
│   │   │   │   ├── database.go           # X-Database-Type 헤더 처리
│   │   │   │   └── ...                   # 로깅, 추적, 메트릭
│   │   │   └── router/                   # 라우터 (36개 엔드포인트)
│   │   └── grpc/                         # gRPC 핸들러
│   │       └── interceptor/              # gRPC 인터셉터
│   ├── config/                           # 설정 관리 (Viper)
│   └── pkg/                              # 공통 유틸리티
│       ├── logger/                       # Zap 로거
│       ├── vault/                        # Vault 클라이언트
│       ├── metrics/                      # Prometheus 메트릭
│       ├── tracing/                      # OpenTelemetry 추적
│       ├── circuitbreaker/               # Circuit Breaker
│       └── retry/                        # Retry 로직
├── configs/                              # 설정 파일
│   ├── config.yaml                       # 기본 설정
│   ├── config_local.yaml                 # 로컬 개발 설정
│   ├── prometheus/                       # Prometheus 설정
│   │   ├── prometheus.yml                # Prometheus 메인 설정
│   │   ├── alert_rules.yml               # 알림 규칙 (100+ rules)
│   │   └── alertmanager.yml              # AlertManager 설정
│   └── grafana/                          # Grafana 설정
│       ├── dashboards/                   # 대시보드 JSON
│       └── provisioning/                 # Auto-provisioning 설정
├── deployments/
│   └── kubernetes/                       # Kubernetes 매니페스트
│       ├── deployment.yaml               # Deployment (HPA 지원)
│       ├── service.yaml                  # Service (LoadBalancer)
│       ├── ingress.yaml                  # Ingress
│       └── hpa.yaml                      # HPA (3-10 replicas)
├── docs/                                 # 문서
│   ├── ARCHITECTURE.md                   # 아키텍처 가이드 (Mermaid 다이어그램)
│   ├── CLIENT_INTEGRATION.md             # 클라이언트 통합 가이드 (Go, Python, Node.js, Java)
│   ├── REST_API_SPECIFICATION.md         # REST API 완벽 명세서 (36개 엔드포인트)
│   ├── QUICKSTART.md                     # 빠른 시작 가이드
│   └── VAULT_INTEGRATION.md              # Vault 통합 가이드
├── test/                                 # 테스트
│   ├── integration/                      # 통합 테스트 (Testcontainers)
│   ├── e2e/                              # E2E 테스트 (HTTP API)
│   ├── benchmark/                        # 벤치마크 테스트
│   └── load/                             # 부하 테스트 (k6)
├── scripts/                              # 자동화 스크립트
│   ├── backup.sh                         # 백업 스크립트
│   └── restore.sh                        # 복원 스크립트
├── proto/                                # gRPC 프로토콜 정의
├── Dockerfile.http                       # HTTP 서버 Dockerfile
├── Dockerfile.grpc                       # gRPC 서버 Dockerfile
├── .gitlab-ci.yml                        # GitLab CI/CD 파이프라인
└── docker-compose.yml                    # 로컬 개발용 Docker Compose (11 services)
```

## 🛠️ 기술 스택

### 언어 & 프레임워크
- **Go**: 1.25.4
- **Gin**: HTTP 웹 프레임워크
- **gRPC**: Protocol Buffers 기반 RPC
- **Viper**: 설정 관리

### 데이터베이스 (6개)
- **MongoDB**: 7.0 (NoSQL 문서 데이터베이스)
- **PostgreSQL**: 16 (관계형 DB, JSONB 지원)
- **MySQL**: 8.0 (관계형 DB, JSON 지원)
- **Cassandra**: 4.1 (분산 NoSQL, 컬럼 패밀리)
- **Elasticsearch**: 8.11 (검색 엔진, 문서 저장소)
- **Vitess**: MySQL 호환 분산 데이터베이스
- **Redis**: 7.0 (캐시, Pub/Sub, Lock, Counter)

### 인프라
- **Kafka**: 이벤트 스트리밍 플랫폼
- **HashiCorp Vault**: 시크릿 관리 및 암호화

### 관찰성
- **Zap**: 구조화된 로깅
- **OpenTelemetry**: 분산 추적
- **Jaeger**: 추적 백엔드
- **Prometheus**: 메트릭 수집
- **Grafana**: 메트릭 시각화

### 컨테이너 & 오케스트레이션
- **Docker**: 컨테이너화
- **Kubernetes**: 컨테이너 오케스트레이션
- **GitLab CI/CD**: 자동화 파이프라인

### 테스팅
- **Go testing**: 유닛 테스트
- **Testify**: 테스트 어설션
- **Testcontainers**: Docker 기반 통합 테스트
- **k6**: 부하 테스트 및 성능 측정

## 🚀 시작하기

### 필요 사항

- Go 1.25.4+
- Docker & Docker Compose
- Protocol Buffers 컴파일러 (protoc)
- Make (선택 사항)
- Kubernetes 클러스터 (프로덕션 배포)
- GitLab Runner (CI/CD)

### 로컬 개발

#### 1. 저장소 클론
```bash
git clone https://github.com/YouSangSon/database-service.git
cd database-service
```

#### 2. 의존성 설치
```bash
go mod download
go mod verify
```

#### 3. 로컬 설정 파일 준비
```bash
# configs/config_local.yaml 파일 확인 및 수정
# Vault, Kafka는 로컬에서 비활성화 가능
```

#### 4. 로컬 인프라 실행 (Docker Compose)
```bash
# MongoDB, Redis 실행
docker-compose up -d mongodb redis

# 전체 스택 실행 (15개 서비스: 6개 DB + Redis + Kafka + Vault + Prometheus + AlertManager + Grafana + App 등)
docker-compose up -d

# 모든 서비스 확인
docker-compose ps
```

#### 5. 애플리케이션 실행

```bash
# HTTP 서버 실행 (포트 8080)
go run cmd/api/main.go --config=./configs/config_local.yaml

# gRPC 서버 실행 (다른 터미널, 포트 9090)
go run cmd/grpc/main.go --config=./configs/config_local.yaml
```

### Docker로 실행

```bash
# Docker 이미지 빌드
docker build -t database-service-http:latest -f Dockerfile.http .
docker build -t database-service-grpc:latest -f Dockerfile.grpc .

# 컨테이너 실행
docker run -d -p 8080:8080 \
  -e APP_MONGODB_URI=mongodb://mongodb:27017 \
  database-service-http:latest

docker run -d -p 9090:9090 \
  -e APP_MONGODB_URI=mongodb://mongodb:27017 \
  database-service-grpc:latest
```

### Kubernetes 배포

```bash
# Namespace 생성
kubectl create namespace production

# Secret 생성 (Vault 자격증명, DB 연결 정보)
kubectl create secret generic db-credentials \
  --from-literal=mongodb-uri='mongodb://...' \
  --from-literal=vault-token='...' \
  -n production

# 서비스 배포
kubectl apply -f deployments/kubernetes/service.yaml
kubectl apply -f deployments/kubernetes/deployment.yaml
kubectl apply -f deployments/kubernetes/ingress.yaml
kubectl apply -f deployments/kubernetes/hpa.yaml

# 상태 확인
kubectl get pods -n production
kubectl get svc -n production
kubectl get hpa -n production
```

## 📖 API 사용법

### REST API

기본 엔드포인트: `http://localhost:8080`

#### 헬스체크
```bash
curl http://localhost:8080/health
```

#### 문서 생성 (MongoDB - 기본값)
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

#### 동적 데이터베이스 선택 (X-Database-Type 헤더)

다른 데이터베이스를 사용하려면 `X-Database-Type` 헤더를 추가하세요:

```bash
# PostgreSQL 사용
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: postgresql" \
  -d '{
    "collection": "users",
    "data": {"name": "John Doe", "email": "john@example.com"}
  }'

# MySQL 사용
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: mysql" \
  -d '{
    "collection": "users",
    "data": {"name": "Jane Doe", "email": "jane@example.com"}
  }'

# Elasticsearch 사용 (전문 검색)
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: elasticsearch" \
  -d '{
    "collection": "logs",
    "data": {"message": "User logged in", "level": "info"}
  }'
```

**지원 데이터베이스 타입:**
- `mongodb` (기본값)
- `postgresql`
- `mysql`
- `cassandra`
- `elasticsearch`
- `vitess`

> ⚠️ **참고**: 데이터베이스를 사용하기 전에 `configs/config.yaml`에서 해당 데이터베이스를 활성화해야 합니다.

#### 문서 조회
```bash
# MongoDB에서 조회 (기본값)
curl http://localhost:8080/api/v1/documents/users/{id}

# PostgreSQL에서 조회
curl http://localhost:8080/api/v1/documents/users/{id} \
  -H "X-Database-Type: postgresql"
```

#### 문서 업데이트
```bash
curl -X PUT http://localhost:8080/api/v1/documents/users/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Jane Doe",
    "age": 31
  }'
```

#### 문서 삭제
```bash
curl -X DELETE http://localhost:8080/api/v1/documents/users/{id}
```

#### 문서 목록 조회 (필터링, 정렬, 페이징)
```bash
curl "http://localhost:8080/api/v1/documents/users?limit=10&offset=0&sort=created_at:-1"
```

#### 집계 쿼리 (MongoDB)
```bash
curl -X POST http://localhost:8080/api/v1/documents/users/aggregate \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline": [
      {"$match": {"age": {"$gte": 25}}},
      {"$group": {"_id": "$age", "count": {"$sum": 1}}},
      {"$sort": {"count": -1}}
    ]
  }'
```

#### Raw Query 실행 (MongoDB)
```bash
curl -X POST http://localhost:8080/api/v1/documents/raw-query \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "listCollections": 1
    }
  }'
```

### gRPC

gRPC 서버는 `localhost:9090`에서 실행됩니다.

#### grpcurl 사용 예제

```bash
# 서비스 목록 조회
grpcurl -plaintext localhost:9090 list

# 헬스체크
grpcurl -plaintext localhost:9090 database.DatabaseService/HealthCheck

# 문서 생성
grpcurl -plaintext -d '{
  "collection": "users",
  "data": {
    "name": "John Doe",
    "email": "john@example.com"
  }
}' localhost:9090 database.DatabaseService/Create

# 문서 조회
grpcurl -plaintext -d '{
  "collection": "users",
  "id": "your-document-id"
}' localhost:9090 database.DatabaseService/Read

# 집계 쿼리
grpcurl -plaintext -d '{
  "collection": "users",
  "pipeline": "[{\"$match\": {\"age\": {\"$gte\": 25}}}]"
}' localhost:9090 database.DatabaseService/Aggregate
```

## 🔧 설정

### 환경변수 (GitLab CI/CD)

GitLab CI/CD 프로젝트 변수 설정:

```bash
# 애플리케이션
APP_NAME=database-service
APP_VERSION=1.0.0
APP_ENVIRONMENT=production

# MongoDB
APP_MONGODB_ENABLED=true
APP_MONGODB_URI=mongodb://mongodb-cluster:27017

# Vitess
APP_VITESS_ENABLED=true
APP_VITESS_HOST=vtgate
APP_VITESS_PORT=15306

# Redis
APP_REDIS_ENABLED=true
APP_REDIS_HOST=redis-cluster
APP_REDIS_PORT=6379

# Kafka
APP_KAFKA_ENABLED=true
APP_KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092

# Vault
APP_VAULT_ENABLED=true
APP_VAULT_ADDRESS=https://vault.production.svc.cluster.local:8200
APP_VAULT_AUTH_METHOD=kubernetes
APP_VAULT_K8S_ROLE=database-service

# Docker Registry
CI_REGISTRY=registry.gitlab.com
CI_REGISTRY_USER=<your-username>
CI_REGISTRY_PASSWORD=<your-token>

# Kubernetes
KUBE_CONTEXT=production-cluster
KUBE_NAMESPACE=production
```

### 로컬 개발 설정 (config_local.yaml)

```yaml
app:
  name: "database-service"
  version: "1.0.0-local"
  environment: "local"
  debug: true

mongodb:
  enabled: true
  uri: "mongodb://localhost:27017"
  use_vault: false

vitess:
  enabled: false  # 로컬에서는 비활성화

redis:
  enabled: true
  host: "localhost"
  port: 6379

kafka:
  enabled: false  # 로컬에서는 비활성화

vault:
  enabled: false  # 로컬에서는 비활성화
```

## 🧪 테스트

### 유닛 테스트
```bash
# 전체 유닛 테스트
go test -v ./...

# 커버리지 리포트
go test -v -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### 통합 테스트 (Testcontainers)
```bash
# Docker가 실행 중이어야 합니다
go test -v -tags=integration ./test/integration/...

# MongoDB 통합 테스트
go test -v -tags=integration ./test/integration/ -run TestMongoDBIntegration
```

### E2E 테스트
```bash
# 서비스가 실행 중이어야 합니다 (localhost:8080)
go test -v -tags=e2e ./test/e2e/...
```

### 벤치마크 테스트
```bash
# 벤치마크 실행
go test -bench=. -benchmem ./test/benchmark/...

# 특정 벤치마크만 실행
go test -bench=BenchmarkCreateDocument -benchmem ./test/benchmark/
```

### 부하 테스트 (k6)
```bash
# k6 설치 필요: brew install k6 (macOS) 또는 https://k6.io/docs/get-started/installation/

# 기본 부하 테스트 실행
cd test/load
./run-load-test.sh

# 특정 테스트 실행
k6 run database-service-load-test.js

# 스트레스 테스트
k6 run scenarios/stress-test.js
```

## 📊 관찰성

### Prometheus 메트릭

Prometheus는 `http://localhost:9090`에서 실행되며, 애플리케이션 메트릭은 `http://localhost:9091/metrics`에서 수집합니다.

#### 수집 메트릭
- `http_requests_total`: HTTP 요청 총 수
- `http_request_duration_seconds`: HTTP 요청 지속 시간 (P50, P95, P99)
- `grpc_requests_total`: gRPC 요청 총 수
- `grpc_request_duration_seconds`: gRPC 요청 지속 시간
- `db_operations_total`: DB 작업 총 수 (operation, collection 레이블)
- `db_operation_duration_seconds`: DB 작업 지속 시간
- `cache_hits_total`: 캐시 히트 수
- `cache_misses_total`: 캐시 미스 수
- `kafka_messages_published_total`: Kafka 메시지 발행 수
- `vault_lease_renewals_total`: Vault Lease 갱신 수

### AlertManager

AlertManager는 `http://localhost:9093`에서 실행됩니다.

#### 알림 규칙 (100+ rules)
- **서비스 가용성**: 서비스 다운, 높은 에러율 감지
- **API 성능**: 높은 지연시간, 느린 응답 시간
- **데이터베이스 건강**: MongoDB/Vitess 연결 실패, 높은 쿼리 지연
- **캐시 건강**: Redis 연결 실패, 낮은 캐시 히트율
- **시스템 리소스**: CPU/메모리 사용률, 디스크 공간 부족
- **비즈니스 메트릭**: 높은 문서 생성 실패율, 비정상적인 트래픽 패턴
- **보안**: 높은 인증 실패율, 비정상적인 API 요청

#### 알림 채널
```yaml
# Slack 알림
slack_configs:
  - channel: '#alerts'
    api_url: 'your-webhook-url'

# Email 알림
email_configs:
  - to: 'team@example.com'
    from: 'alertmanager@example.com'

# PagerDuty 알림 (Critical만)
pagerduty_configs:
  - service_key: 'your-service-key'
```

### Grafana 대시보드

Grafana는 `http://localhost:3000`에서 실행됩니다 (기본 로그인: admin/admin).

#### 자동 프로비저닝된 대시보드
- **Database Service Overview**: 서비스 상태, CPU/메모리, 요청률, 지연시간, 에러율
- **실시간 업데이트**: 5초마다 자동 새로고침
- **시간 범위**: 기본 15분

```bash
# Grafana UI 접속
http://localhost:3000

# 대시보드 경로
Dashboards → Database Service → Database Service Overview
```

### 로깅 (Zap)

구조화된 JSON 로그:

```json
{
  "level": "info",
  "timestamp": "2025-11-12T08:30:00.000Z",
  "msg": "document created",
  "trace_id": "abc123def456",
  "span_id": "789ghi012jkl",
  "collection": "users",
  "document_id": "507f1f77bcf86cd799439011",
  "duration_ms": 15.3
}
```

### 분산 추적 (Jaeger)

```bash
# Jaeger UI 접속
http://localhost:16686

# 추적 검색
# - 서비스: database-service
# - 작업: /api/v1/documents, CreateDocument, etc.
```

## 🔒 보안

### Vault 통합

자세한 내용은 [VAULT_INTEGRATION.md](./docs/VAULT_INTEGRATION.md) 참조

- **동적 자격증명**: MongoDB, Vitess 사용자 자동 생성/삭제 (TTL: 1-24시간)
- **정적 시크릿**: Redis 비밀번호, API 키 등
- **Transit 암호화**: 민감 데이터 암호화/복호화 (AES-256-GCM)
- **자동 Lease 갱신**: TTL 만료 3분 전 자동 갱신

### Kubernetes 보안

- **RBAC**: ServiceAccount 기반 접근 제어
- **Network Policies**: Pod 간 통신 제한
- **Secrets**: 민감 정보 Kubernetes Secrets 저장
- **TLS/mTLS**: 통신 암호화 (Istio/Linkerd)

## 💾 백업 & 복원

### 자동 백업

백업 스크립트는 MongoDB, Redis, 애플리케이션 데이터를 자동으로 백업합니다.

```bash
# 백업 실행
./scripts/backup.sh

# 백업 내용
# - MongoDB 데이터 (mongodump)
# - Redis 데이터 (dump.rdb)
# - 애플리케이션 설정 파일 (configs/)
```

#### 백업 저장 위치
```
./backups/
├── mongodb_20250112_153045.tar.gz
├── redis_20250112_153045.tar.gz
├── appdata_20250112_153045.tar.gz
└── backup_20250112_153045_manifest.txt
```

#### 자동 보관 정책
- 백업 보관 기간: 7일
- 7일 이상 된 백업 자동 삭제
- Timestamped 파일명으로 버전 관리

### 복원

```bash
# 사용 가능한 백업 목록 확인
./scripts/restore.sh

# 특정 백업으로 복원
./scripts/restore.sh 20250112_153045

# 복원 프로세스
# 1. 백업 정보 표시
# 2. 확인 프롬프트 (yes/no)
# 3. MongoDB 복원 (mongorestore)
# 4. Redis 복원 (dump.rdb 교체)
# 5. 애플리케이션 데이터 복원
```

⚠️ **주의사항**:
- 복원 시 현재 데이터가 모두 삭제됩니다
- 프로덕션 환경에서는 반드시 백업 후 복원하세요
- Redis는 복원 중 재시작됩니다

### Cron 자동 백업 설정

```bash
# crontab 편집
crontab -e

# 매일 새벽 3시에 백업 실행
0 3 * * * /path/to/database-service/scripts/backup.sh >> /var/log/db-backup.log 2>&1
```

## 📈 성능 & 확장성

### HPA (Horizontal Pod Autoscaler)

```yaml
# 자동 스케일링 설정
minReplicas: 3
maxReplicas: 10
metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 벤치마크 (단일 Pod, 4 vCPU, 8GB RAM)

- **처리량**: ~10,000 req/s (Read), ~5,000 req/s (Write)
- **지연시간**: P50: 5ms, P95: 15ms, P99: 30ms
- **동시 연결**: 1,000+ 동시 연결
- **메모리**: ~256MB (일반 부하), ~512MB (고부하)
- **캐시 히트율**: ~85% (Redis)

## 🗂️ 아키텍처

자세한 아키텍처는 [ARCHITECTURE.md](./docs/ARCHITECTURE.md) 참조

### 계층 구조

1. **Interface Layer**: HTTP/gRPC 핸들러, 미들웨어/인터셉터
2. **Application Layer**: 유즈케이스 (비즈니스 로직), 이벤트 발행, 캐싱
3. **Domain Layer**: 엔티티, 값 객체, 리포지토리 인터페이스, 도메인 서비스
4. **Infrastructure Layer**: MongoDB/Vitess 리포지토리, Redis 캐시, Kafka 프로듀서, Vault 클라이언트

### MongoDB 고급 연산 (30+ 메서드)

- **기본 CRUD**: Save, FindByID, Update, Delete, FindAll, Count
- **쿼리 연산**: FindWithOptions (Sort, Limit, Skip, Projection), Upsert, Replace
- **벌크 연산**: SaveMany, UpdateMany, DeleteMany, BulkWrite
- **원자적 연산**: FindAndUpdate, FindOneAndReplace, FindOneAndDelete
- **집계**: Aggregate, Distinct, EstimatedDocumentCount
- **인덱스 관리**: CreateIndex, CreateIndexes, DropIndex, ListIndexes
- **컬렉션 관리**: CreateCollection, DropCollection, RenameCollection, ListCollections
- **Change Streams**: Watch, WatchWithResumeToken
- **Raw Query**: ExecuteRawQuery, ExecuteRawQueryWithResult, RunAggregateCommand, GetCollectionStats, GetDatabaseStats

### Vitess 고급 연산 (30+ 메서드)

MongoDB와 동일한 인터페이스를 SQL로 구현:

- **기본 CRUD**: INSERT, SELECT, UPDATE, DELETE
- **쿼리 연산**: JSON_EXTRACT 기반 복잡한 쿼리
- **벌크 연산**: 트랜잭션 기반 배치 처리
- **원자적 연산**: SELECT FOR UPDATE (비관적 잠금)
- **집계**: GROUP BY, COUNT, DISTINCT를 활용한 SQL 집계
- **인덱스 관리**: ALTER TABLE을 통한 인덱스 관리
- **컬렉션 관리**: 논리적 컬렉션 (collection 필드 사용)
- **Raw Query**: ExecuteRawQuery, ExecutePreparedQuery, ExecuteBatch

## 🚀 CI/CD 파이프라인

GitLab CI/CD 파이프라인 단계:

1. **Lint**: golangci-lint 코드 품질 검사
2. **Test**: 유닛 테스트 + 통합 테스트 (MongoDB, Redis)
3. **Build**: Go 바이너리 빌드 (HTTP, gRPC)
4. **Docker**: Docker 이미지 빌드 및 레지스트리 푸시
5. **Deploy**: Kubernetes 배포
   - `develop` 브랜치 → Development 환경 (자동)
   - `main` 브랜치 → Staging 환경 (수동)
   - `tags` → Production 환경 (수동)

## 🤝 기여

Pull Request를 환영합니다! 다음 가이드라인을 따라주세요:

1. 기능 브랜치 생성 (`git checkout -b feature/amazing-feature`)
2. 변경사항 커밋 (`git commit -m 'Add amazing feature'`)
3. 테스트 작성 및 통과 확인 (`go test ./...`)
4. 브랜치 푸시 (`git push origin feature/amazing-feature`)
5. Pull Request 생성

### 코드 스타일

- `gofmt` 및 `golangci-lint` 사용
- 구조화된 로깅 (Zap) 사용
- 테스트 커버리지 80% 이상 유지
- DDD 및 Clean Architecture 패턴 준수

## 📝 라이선스

MIT License

## 🔮 로드맵

### ✅ 완료
- [x] MongoDB 지원 (30+ 메서드)
- [x] PostgreSQL 지원 (30+ 메서드, JSONB)
- [x] MySQL 지원 (30+ 메서드, JSON)
- [x] Cassandra 지원 (20+ 메서드)
- [x] Elasticsearch 지원 (25+ 메서드)
- [x] Vitess 지원 (30+ 메서드)
- [x] Kafka CDC
- [x] HashiCorp Vault 통합
- [x] Redis 확장 기능
- [x] GitLab CI/CD 파이프라인
- [x] Prometheus AlertManager (100+ 알림 규칙)
- [x] Grafana Dashboards (Auto-provisioning)
- [x] 부하 테스트 (k6 기반)
- [x] 백업/복원 자동화
- [x] 통합 테스트 (Testcontainers)
- [x] E2E 테스트
- [x] 벤치마크 테스트

### 🔜 향후 계획
- [ ] GraphQL API
- [ ] Event Sourcing
- [ ] CQRS 패턴
- [ ] Service Mesh (Istio) 통합
- [ ] WebSocket 실시간 알림
- [ ] Multi-tenancy 지원

## 📚 참고 문서

- [Architecture Guide](./docs/ARCHITECTURE.md) - 전체 아키텍처 및 Mermaid 다이어그램
- [Vault Integration Guide](./docs/VAULT_INTEGRATION.md) - Vault 연동 상세 가이드
- [Logging Guide](./internal/pkg/logger/LOGGING_GUIDE.md) - 로깅 가이드
- [GitLab CI/CD Configuration](./.gitlab-ci.yml) - CI/CD 파이프라인 설정
- [Kubernetes Deployments](./deployments/kubernetes/) - Kubernetes 매니페스트

## 📞 연락처

- GitHub: [@YouSangSon](https://github.com/YouSangSon)
- Issues: [GitHub Issues](https://github.com/YouSangSon/database-service/issues)
