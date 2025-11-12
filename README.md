# Database Service - Enterprise Edition

엔터프라이즈급 Go 기반의 범용 데이터베이스 서비스입니다. DDD(Domain-Driven Design), TDD(Test-Driven Development), Clean Architecture를 적용하여 대규모 프로덕션 환경에서 사용할 수 있도록 설계되었습니다.

## 🚀 주요 특징

### 아키텍처
- ✅ **DDD + Clean Architecture**: 도메인 주도 설계 및 4계층 아키텍처 (Domain, Application, Infrastructure, Interface)
- ✅ **Repository Pattern**: 데이터베이스 추상화를 통한 구현체 교체 가능
- ✅ **이벤트 기반**: Kafka CDC를 통한 확장 가능한 이벤트 기반 아키텍처
- ✅ **SOLID 원칙**: 의존성 역전, 단일 책임 등 객체지향 설계 원칙 준수

### 데이터베이스 지원
- ✅ **MongoDB**: 30+ 고급 메서드 지원 (집계, 벌크 연산, 원자적 연산, 인덱스 관리, Change Streams 등)
- ✅ **Vitess**: 30+ 고급 메서드 지원 (MongoDB와 동일한 인터페이스로 SQL 구현)
- ✅ **Raw Query 실행**: MongoDB RunCommand, Vitess SQL 직접 실행 지원
- 🔜 **향후 지원 예정**: PostgreSQL, MySQL 등

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
- ✅ **연결 풀링**: MongoDB, Vitess, Redis 연결 풀 최적화
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
- ✅ **대시보드**: Grafana 대시보드 지원

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
│   │   │   ├── mongodb/                  # MongoDB 구현 (30+ 메서드)
│   │   │   └── vitess/                   # Vitess 구현 (30+ 메서드)
│   │   ├── cache/                        # Redis 캐시 및 확장 기능
│   │   ├── messaging/                    # Kafka 메시징
│   │   └── monitoring/                   # 모니터링 (메트릭, 추적)
│   ├── interfaces/                       # 인터페이스 레이어
│   │   ├── http/                         # HTTP 핸들러 (Gin)
│   │   │   └── middleware/               # HTTP 미들웨어 (로깅, 추적, 메트릭)
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
│   └── config_local.yaml                 # 로컬 개발 설정
├── deployments/
│   └── kubernetes/                       # Kubernetes 매니페스트
│       ├── deployment.yaml               # Deployment (HPA 지원)
│       ├── service.yaml                  # Service (LoadBalancer)
│       ├── ingress.yaml                  # Ingress
│       └── hpa.yaml                      # HPA (3-10 replicas)
├── docs/                                 # 문서
│   ├── ARCHITECTURE.md                   # 아키텍처 가이드 (Mermaid 다이어그램)
│   └── VAULT_INTEGRATION.md              # Vault 통합 가이드
├── proto/                                # gRPC 프로토콜 정의
├── Dockerfile.http                       # HTTP 서버 Dockerfile
├── Dockerfile.grpc                       # gRPC 서버 Dockerfile
├── .gitlab-ci.yml                        # GitLab CI/CD 파이프라인
└── docker-compose.yml                    # 로컬 개발용 Docker Compose
```

## 🛠️ 기술 스택

### 언어 & 프레임워크
- **Go**: 1.25.4
- **Gin**: HTTP 웹 프레임워크
- **gRPC**: Protocol Buffers 기반 RPC
- **Viper**: 설정 관리

### 데이터베이스
- **MongoDB**: 7.0 (NoSQL 문서 데이터베이스)
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

#### 문서 생성 (MongoDB)
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

#### 문서 조회
```bash
curl http://localhost:8080/api/v1/documents/users/{id}
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

```bash
# 유닛 테스트
go test -v ./...

# 통합 테스트 (MongoDB, Redis 필요)
go test -v -tags=integration ./...

# 커버리지 리포트
go test -v -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out

# 벤치마크
go test -bench=. -benchmem ./...
```

## 📊 관찰성

### 메트릭 (Prometheus)

`http://localhost:9091/metrics` 엔드포인트에서 수집:

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

- [x] MongoDB 지원 (30+ 메서드)
- [x] Vitess 지원 (30+ 메서드)
- [x] Kafka CDC
- [x] HashiCorp Vault 통합
- [x] Redis 확장 기능
- [x] GitLab CI/CD 파이프라인
- [ ] PostgreSQL 네이티브 지원
- [ ] MySQL 네이티브 지원
- [ ] GraphQL API
- [ ] Event Sourcing
- [ ] CQRS 패턴
- [ ] Service Mesh (Istio) 통합
- [ ] Multi-tenancy 지원
- [ ] WebSocket 실시간 알림

## 📚 참고 문서

- [Architecture Guide](./docs/ARCHITECTURE.md) - 전체 아키텍처 및 Mermaid 다이어그램
- [Vault Integration Guide](./docs/VAULT_INTEGRATION.md) - Vault 연동 상세 가이드
- [Logging Guide](./internal/pkg/logger/LOGGING_GUIDE.md) - 로깅 가이드
- [GitLab CI/CD Configuration](./.gitlab-ci.yml) - CI/CD 파이프라인 설정
- [Kubernetes Deployments](./deployments/kubernetes/) - Kubernetes 매니페스트

## 📞 연락처

- GitHub: [@YouSangSon](https://github.com/YouSangSon)
- Issues: [GitHub Issues](https://github.com/YouSangSon/database-service/issues)
