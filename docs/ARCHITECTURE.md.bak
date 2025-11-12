# Architecture Overview

이 문서는 데이터베이스 서비스의 전체 아키텍처를 설명합니다.

## 시스템 구성도

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Applications                       │
│                    (REST, gRPC, WebSocket)                       │
└───────────────────────┬─────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────┐
│                      Load Balancer                               │
│                    (Kubernetes Ingress)                          │
└───────────────────────┬─────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────┐
│                  Database Service (Multiple Pods)                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  HTTP Server (Port 8080)  │  gRPC Server (Port 9090)     │  │
│  ├──────────────────────────────────────────────────────────┤  │
│  │              Middleware / Interceptors                    │  │
│  │  - Logging   - Tracing   - Metrics   - Recovery          │  │
│  ├──────────────────────────────────────────────────────────┤  │
│  │                   Use Case Layer                          │  │
│  │  - Document CRUD   - Event Publishing   - Caching        │  │
│  ├──────────────────────────────────────────────────────────┤  │
│  │                 Domain Layer (DDD)                        │  │
│  │  - Entities   - Value Objects   - Domain Services        │  │
│  ├──────────────────────────────────────────────────────────┤  │
│  │               Infrastructure Layer                        │  │
│  │  - MongoDB Repo   - Vitess Repo   - Redis Cache          │  │
│  │  - Kafka Producer   - Vault Client                       │  │
│  └──────────────────────────────────────────────────────────┘  │
└───────────────────────┬─────────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┬───────────────┬─────────────┐
        │               │               │               │             │
┌───────▼──────┐ ┌──────▼──────┐ ┌─────▼─────┐ ┌──────▼─────┐ ┌────▼────┐
│   MongoDB    │ │   Vitess    │ │   Redis   │ │   Kafka    │ │  Vault  │
│   Cluster    │ │   Cluster   │ │  Cluster  │ │  Cluster   │ │ Server  │
└──────────────┘ └─────────────┘ └───────────┘ └────────────┘ └─────────┘
```

## 폴더 구조

```
database-service/
├── cmd/                           # 애플리케이션 엔트리포인트
│   ├── api/                      # HTTP 서버
│   └── grpc/                     # gRPC 서버
├── configs/                       # 설정 파일
│   ├── config.yaml               # 개발 환경 설정
│   ├── config.production.yaml    # 프로덕션 환경 설정
│   ├── vault-config.yaml         # Vault 설정
│   └── vault-setup.sh            # Vault 초기 설정 스크립트
├── internal/                      # 내부 패키지
│   ├── config/                   # 설정 로더 (Viper)
│   │   └── config.go
│   ├── domain/                   # 도메인 레이어 (DDD)
│   │   ├── entity/              # 엔티티
│   │   │   └── document.go
│   │   └── repository/          # 레포지토리 인터페이스
│   │       └── document_repository.go
│   ├── application/              # 애플리케이션 레이어
│   │   └── usecase/             # 유스케이스
│   │       └── document_usecase.go
│   ├── infrastructure/           # 인프라스트럭처 레이어
│   │   ├── persistence/         # 영속성
│   │   │   ├── mongodb/        # MongoDB 구현
│   │   │   │   ├── document_repository.go
│   │   │   │   ├── aggregation_operations.go
│   │   │   │   ├── atomic_operations.go
│   │   │   │   ├── bulk_operations.go
│   │   │   │   ├── index_operations.go
│   │   │   │   ├── collection_operations.go
│   │   │   │   ├── query_operations.go
│   │   │   │   └── change_streams.go
│   │   │   └── vitess/         # Vitess 구현
│   │   │       └── vitess_repository.go
│   │   ├── cache/              # 캐싱
│   │   │   ├── redis.go
│   │   │   └── redis_extended.go  # Pub/Sub, Rate Limiting, Lock
│   │   └── messaging/          # 메시징
│   │       └── kafka/
│   │           └── producer.go     # CDC, Event Sourcing
│   ├── interfaces/               # 인터페이스 레이어
│   │   ├── http/                # HTTP 핸들러
│   │   │   ├── handler/
│   │   │   └── middleware/     # HTTP 미들웨어
│   │   └── grpc/                # gRPC 핸들러
│   │       ├── handler/
│   │       └── interceptor/    # gRPC 인터셉터
│   └── pkg/                      # 공통 패키지
│       ├── logger/              # 구조화된 로깅 (Zap)
│       ├── metrics/             # 메트릭 (Prometheus)
│       ├── tracing/             # 분산 추적 (OpenTelemetry)
│       ├── errors/              # 에러 처리
│       └── vault/               # Vault 클라이언트
│           ├── client.go       # 클라이언트 및 인증
│           ├── config.go       # 설정
│           ├── secrets.go      # 시크릿 관리
│           ├── encryption.go   # Transit Engine
│           └── database.go     # DB 자격증명 관리
├── examples/                     # 예시 코드
│   ├── vault_example.go
│   └── complete_example.go
├── docs/                         # 문서
│   ├── ARCHITECTURE.md
│   └── VAULT_INTEGRATION.md
├── deployments/                  # 배포 관련
│   └── kubernetes/              # Kubernetes 매니페스트
└── go.mod                        # Go 모듈

```

## 주요 컴포넌트

### 1. Domain Layer (DDD)

**엔티티:**
- `Document`: 범용 문서 엔티티
- 버전 관리 (낙관적 잠금)
- 불변성 보장

**레포지토리 인터페이스:**
- 데이터베이스 독립적 인터페이스
- MongoDB, Vitess 모두 동일한 인터페이스 구현
- 30+ 메서드 지원

### 2. Application Layer

**유스케이스:**
- 비즈니스 로직 처리
- 트랜잭션 관리
- 이벤트 발행
- 캐싱 전략

### 3. Infrastructure Layer

**MongoDB:**
- 기본 CRUD
- 집계 (Aggregate, Distinct)
- 벌크 작업
- 인덱스 관리
- Change Streams
- 트랜잭션

**Vitess:**
- MySQL 프로토콜 사용
- 수평 확장
- 샤딩 지원
- 트랜잭션

**Redis:**
- 기본 캐싱
- Pub/Sub
- Rate Limiting
- Distributed Lock
- Distributed Counter

**Kafka:**
- CDC (Change Data Capture)
- 이벤트 소싱
- 비동기 처리
- 이벤트 재생

**Vault:**
- 동적 자격증명 (MongoDB, Vitess)
- 정적 시크릿
- Transit 암호화
- 자동 리스 갱신

### 4. Interface Layer

**HTTP API:**
- RESTful API
- Gin 프레임워크
- Middleware stack:
  - Request ID
  - Logging
  - Tracing
  - Metrics
  - Recovery
  - CORS

**gRPC API:**
- Protocol Buffers
- Interceptor stack:
  - Logging
  - Tracing
  - Metrics
  - Recovery
  - Error Handling

### 5. Observability

**Logging:**
- Zap (구조화된 로깅)
- 일관된 필드 명명
- 컨텍스트 전파

**Tracing:**
- OpenTelemetry
- Jaeger
- 분산 추적
- Span 전파

**Metrics:**
- Prometheus
- 자동 메트릭 수집
- 커스텀 메트릭

## 데이터 흐름

### 1. 문서 생성 (Create)

```
1. Client Request (HTTP/gRPC)
   ↓
2. Middleware/Interceptor
   - Request ID 생성
   - 로깅 시작
   - Tracing 시작
   ↓
3. Handler
   - 요청 검증
   - UseCase 호출
   ↓
4. UseCase
   - 엔티티 생성
   - Repository 호출
   ↓
5. Repository
   - MongoDB/Vitess에 저장
   - 버전 설정
   ↓
6. Event Publishing (optional)
   - Kafka로 CDC 이벤트 발행
   ↓
7. Response
   - 클라이언트에 응답
   - 로깅 완료
   - 메트릭 기록
```

### 2. 문서 조회 (Read with Cache)

```
1. Client Request
   ↓
2. UseCase
   - 캐시 확인 (Redis)
   ├─ Cache Hit → 즉시 반환
   └─ Cache Miss ↓
3. Repository
   - MongoDB/Vitess에서 조회
   ↓
4. 캐시 저장
   - Redis에 저장 (TTL 설정)
   ↓
5. Response
```

### 3. 이벤트 기반 아키텍처

```
Document Update
   ↓
Repository Save
   ↓
Kafka Producer → [documents.updated] Topic
   ↓
┌────────────────┬────────────────┬────────────────┐
│  Consumer 1    │  Consumer 2    │  Consumer 3    │
│  (Analytics)   │  (Audit Log)   │  (Notification)│
└────────────────┴────────────────┴────────────────┘
```

## 보안

### 1. Vault 통합

**동적 자격증명:**
- MongoDB: 1-24시간 TTL
- Vitess: 1-24시간 TTL
- 자동 로테이션
- 만료 전 자동 갱신

**정적 시크릿:**
- API 키
- JWT 시크릿
- Redis 비밀번호

**Transit 암호화:**
- 민감한 데이터 암호화
- 데이터베이스에 암호화된 상태로 저장
- 필요할 때만 복호화

### 2. 인증 & 인가

- Token 인증
- AppRole 인증 (프로덕션)
- Kubernetes Service Account (K8s)

### 3. TLS/SSL

- Vault 통신 암호화
- gRPC mTLS
- 데이터베이스 연결 암호화

## 확장성

### 1. 수평 확장

- Stateless 서비스
- Kubernetes HPA (Horizontal Pod Autoscaler)
- 3-10 replicas

### 2. 데이터베이스 확장

**MongoDB:**
- Replica Set
- Sharding
- Read Preference

**Vitess:**
- 자동 샤딩
- 수평 확장
- Query Routing

**Redis:**
- Cluster Mode
- Read Replicas
- Sentinel

### 3. 메시징 확장

**Kafka:**
- Partitioning
- Consumer Groups
- Replication

## 장애 복구

### 1. Circuit Breaker

- 외부 서비스 호출 보호
- 자동 복구
- Fallback 전략

### 2. Retry Logic

- 지수 백오프
- Jitter
- 최대 재시도 제한

### 3. Graceful Shutdown

- 진행 중인 요청 완료
- 연결 정리
- 리소스 해제

## 성능 최적화

### 1. 캐싱 전략

- Redis L1 캐시
- Application L2 캐시
- Cache-Aside 패턴

### 2. 연결 풀링

- MongoDB: 100 connections
- Vitess: 200 connections
- Redis: 200 connections

### 3. 벌크 작업

- Batch Insert
- Bulk Update
- Ordered=false (MongoDB)

## 모니터링 & 알림

### 1. 메트릭

- Request Rate
- Error Rate
- Latency (p50, p95, p99)
- DB Connection Pool
- Cache Hit Rate
- Kafka Lag

### 2. 로그 집계

- ELK Stack
- Structured Logging
- Correlation ID

### 3. 알림

- Prometheus Alertmanager
- Slack/PagerDuty 통합
- SLA 모니터링

## 배포

### 1. Kubernetes

```yaml
Deployment:
  - Replicas: 3-10 (HPA)
  - Resources:
      CPU: 500m-2000m
      Memory: 512Mi-2Gi
  - Health Checks:
      Liveness: /health
      Readiness: /ready

Service:
  - HTTP: 8080
  - gRPC: 9090
  - Metrics: 9091

Ingress:
  - TLS Termination
  - Load Balancing
  - Rate Limiting
```

### 2. CI/CD

- GitHub Actions
- Docker Build
- Kubernetes Deployment
- Automated Testing
- Canary Deployment

## 참고 자료

- [Vault Integration Guide](./VAULT_INTEGRATION.md)
- [API Documentation](./API.md)
- [Deployment Guide](./DEPLOYMENT.md)
