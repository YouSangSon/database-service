# Architecture Overview

이 문서는 데이터베이스 서비스의 전체 아키텍처를 설명합니다.

## 시스템 구성도

```mermaid
graph TB
    subgraph "Client Applications"
        REST[REST Clients]
        GRPC[gRPC Clients]
        WS[WebSocket Clients]
    end

    subgraph "Load Balancer"
        LB[Kubernetes Ingress<br/>Load Balancer]
    end

    subgraph "Database Service Pods"
        subgraph "Pod 1"
            HTTP1[HTTP Server<br/>Port 8080]
            GRPC1[gRPC Server<br/>Port 9090]
        end
        subgraph "Pod 2"
            HTTP2[HTTP Server<br/>Port 8080]
            GRPC2[gRPC Server<br/>Port 9090]
        end
        subgraph "Pod N"
            HTTPN[HTTP Server<br/>Port 8080]
            GRPCN[gRPC Server<br/>Port 9090]
        end
    end

    subgraph "Data Layer"
        MONGO[(MongoDB<br/>Cluster)]
        VITESS[(Vitess<br/>Cluster)]
        REDIS[(Redis<br/>Cluster)]
        KAFKA[Kafka<br/>Cluster]
        VAULT[Vault<br/>Server]
    end

    REST --> LB
    GRPC --> LB
    WS --> LB

    LB --> HTTP1
    LB --> HTTP2
    LB --> HTTPN
    LB --> GRPC1
    LB --> GRPC2
    LB --> GRPCN

    HTTP1 --> MONGO
    HTTP1 --> VITESS
    HTTP1 --> REDIS
    HTTP1 --> KAFKA
    HTTP1 --> VAULT

    GRPC1 --> MONGO
    GRPC1 --> VITESS
    GRPC1 --> REDIS
    GRPC1 --> KAFKA
    GRPC1 --> VAULT

    style REST fill:#e1f5ff
    style GRPC fill:#e1f5ff
    style LB fill:#fff4e6
    style MONGO fill:#c8e6c9
    style VITESS fill:#c8e6c9
    style REDIS fill:#ffccbc
    style KAFKA fill:#f8bbd0
    style VAULT fill:#d1c4e9
```

## 계층 아키텍처 (Clean Architecture + DDD)

```mermaid
graph TD
    subgraph "Interface Layer"
        HTTP[HTTP Handlers]
        GRPC[gRPC Handlers]
        MW[Middleware/<br/>Interceptors]
    end

    subgraph "Application Layer"
        UC[Use Cases]
        EV[Event Publishers]
        CACHE[Caching Logic]
    end

    subgraph "Domain Layer"
        ENT[Entities]
        VO[Value Objects]
        REPO_IF[Repository<br/>Interfaces]
        DS[Domain Services]
    end

    subgraph "Infrastructure Layer"
        MONGO_R[MongoDB<br/>Repository]
        VITESS_R[Vitess<br/>Repository]
        REDIS_C[Redis<br/>Cache]
        KAFKA_P[Kafka<br/>Producer]
        VAULT_C[Vault<br/>Client]
    end

    HTTP --> MW
    GRPC --> MW
    MW --> UC
    UC --> EV
    UC --> CACHE
    UC --> ENT
    UC --> REPO_IF
    REPO_IF --> MONGO_R
    REPO_IF --> VITESS_R
    ENT --> DS
    CACHE --> REDIS_C
    EV --> KAFKA_P
    UC --> VAULT_C

    style HTTP fill:#e3f2fd
    style GRPC fill:#e3f2fd
    style UC fill:#fff3e0
    style ENT fill:#f3e5f5
    style REPO_IF fill:#f3e5f5
    style MONGO_R fill:#e8f5e9
    style VITESS_R fill:#e8f5e9
    style REDIS_C fill:#fce4ec
```

## 주요 컴포넌트

### 1. Domain Layer (DDD)

```mermaid
classDiagram
    class Document {
        -id string
        -collection string
        -data map
        -version int
        -createdAt time
        -updatedAt time
        +ID() string
        +Collection() string
        +Data() map
        +Version() int
        +IncrementVersion()
        +Validate() error
    }

    class DocumentRepository {
        <<interface>>
        +Save(doc)
        +SaveMany(docs)
        +FindByID(id)
        +FindAll(filter)
        +FindWithOptions(filter, opts)
        +Update(doc)
        +UpdateMany(filter, update)
        +Delete(id)
        +DeleteMany(filter)
        +FindAndUpdate(id, update)
        +FindOneAndReplace(id, replacement)
        +FindOneAndDelete(id)
        +Upsert(filter, update)
        +Aggregate(pipeline)
        +Distinct(field, filter)
        +Count(filter)
        +BulkWrite(operations)
        +CreateIndex(model)
        +WithTransaction(fn)
    }

    class MongoDBRepository {
        -client *mongo.Client
        -database *mongo.Database
        +Save(doc)
        +FindByID(id)
        +Aggregate(pipeline)
        +...30+ methods
    }

    class VitessRepository {
        -db *sql.DB
        -keyspace string
        +Save(doc)
        +FindByID(id)
        +Aggregate(pipeline)
        +...30+ methods
    }

    DocumentRepository <|.. MongoDBRepository
    DocumentRepository <|.. VitessRepository
    Document --> DocumentRepository
```

### 2. 문서 생성 흐름

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Middleware
    participant Handler
    participant UseCase
    participant Repository
    participant MongoDB
    participant Kafka

    Client->>Middleware: POST /documents
    Middleware->>Middleware: Generate Request ID
    Middleware->>Middleware: Start Tracing
    Middleware->>Handler: Forward Request

    Handler->>Handler: Validate Request
    Handler->>UseCase: Create(ctx, data)

    UseCase->>UseCase: Create Entity
    UseCase->>Repository: Save(doc)
    Repository->>MongoDB: InsertOne(doc)
    MongoDB-->>Repository: InsertedID
    Repository-->>UseCase: Document

    UseCase->>Kafka: PublishDocumentCreated(event)
    Kafka-->>UseCase: Ack

    UseCase-->>Handler: Response
    Handler-->>Middleware: Response
    Middleware->>Middleware: Log & Record Metrics
    Middleware-->>Client: 201 Created
```

### 3. 캐시 조회 흐름

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Handler
    participant UseCase
    participant Cache
    participant Repository
    participant MongoDB

    Client->>Handler: GET /documents/:id
    Handler->>UseCase: GetByID(ctx, id)

    UseCase->>Cache: Get(key)

    alt Cache Hit
        Cache-->>UseCase: Document
        UseCase-->>Handler: Response
        Handler-->>Client: 200 OK (from cache)
    else Cache Miss
        Cache-->>UseCase: Nil
        UseCase->>Repository: FindByID(id)
        Repository->>MongoDB: FindOne(id)
        MongoDB-->>Repository: Document
        Repository-->>UseCase: Document
        UseCase->>Cache: Set(key, doc, TTL)
        Cache-->>UseCase: OK
        UseCase-->>Handler: Response
        Handler-->>Client: 200 OK (from DB)
    end
```

## 인프라스트럭처

### MongoDB 고급 연산

30+ 메서드를 지원하는 MongoDB 구현:

- **기본 CRUD**: Save, FindByID, Update, Delete, FindAll, Count
- **쿼리 연산**: FindWithOptions (Sort, Limit, Skip, Projection), Upsert, Replace
- **벌크 연산**: SaveMany, UpdateMany, DeleteMany, BulkWrite
- **원자적 연산**: FindAndUpdate, FindOneAndReplace, FindOneAndDelete
- **집계**: Aggregate, Distinct, EstimatedDocumentCount
- **인덱스 관리**: CreateIndex, CreateIndexes, DropIndex, ListIndexes
- **컬렉션 관리**: CreateCollection, DropCollection, RenameCollection, ListCollections
- **Change Streams**: Watch, WatchWithResumeToken

### Vitess 고급 연산

MongoDB와 동일한 30+ 메서드를 SQL로 구현:

- **기본 CRUD**: INSERT, SELECT, UPDATE, DELETE
- **쿼리 연산**: JSON_EXTRACT를 활용한 복잡한 쿼리
- **벌크 연산**: 트랜잭션 기반 배치 처리
- **원자적 연산**: SELECT FOR UPDATE (비관적 잠금)
- **집계**: GROUP BY, COUNT, DISTINCT를 활용한 SQL 집계
- **인덱스 관리**: ALTER TABLE을 통한 인덱스 관리
- **컬렉션 관리**: 논리적 컬렉션 (collection 필드 사용)

### Redis 확장 기능

```mermaid
graph TD
    subgraph "Redis Extended"
        A[Basic Cache<br/>Get, Set, Delete]
        B[Pub/Sub<br/>Publisher/Subscriber]
        C[Rate Limiting<br/>Token Bucket]
        D[Distributed Lock<br/>Acquire/Release]
        E[Distributed Counter<br/>Incr/Decr]
    end

    A --> REDIS[(Redis Cluster)]
    B --> REDIS
    C --> REDIS
    D --> REDIS
    E --> REDIS

    style REDIS fill:#ffccbc
```

### Kafka CDC

```mermaid
graph LR
    subgraph "Event Flow"
        A[Document Update] --> B[Repository Save]
        B --> C[Kafka Producer]
        C --> D[documents.updated Topic]

        D --> E[Consumer 1<br/>Analytics]
        D --> F[Consumer 2<br/>Audit Log]
        D --> G[Consumer 3<br/>Notification]
        D --> H[Consumer 4<br/>Search Index]
    end

    style A fill:#e3f2fd
    style C fill:#f8bbd0
    style D fill:#f8bbd0
    style E fill:#c8e6c9
    style F fill:#c8e6c9
    style G fill:#c8e6c9
    style H fill:#c8e6c9
```

## 보안 (Vault Integration)

```mermaid
sequenceDiagram
    autonumber
    participant App
    participant Vault
    participant MongoDB
    participant Vitess

    App->>Vault: Authenticate (Token/AppRole/K8s SA)
    Vault-->>App: Access Token

    App->>Vault: Request MongoDB Credentials
    Vault->>Vault: Generate Dynamic Credentials
    Vault-->>App: {username, password, TTL}

    App->>MongoDB: Connect (username, password)
    MongoDB-->>App: Connection Established

    App->>Vault: Request Vitess Credentials
    Vault-->>App: {username, password, TTL}

    App->>Vitess: Connect (username, password)
    Vitess-->>App: Connection Established

    Note over App,Vault: Auto-renewal before expiry

    App->>Vault: Renew Lease
    Vault-->>App: Lease Extended
```

**Vault 기능:**
- 동적 자격증명 (MongoDB, Vitess)
- 정적 시크릿 (API Keys, Redis Password)
- Transit 암호화
- 자동 Lease 갱신

## 확장성

### Horizontal Pod Autoscaler

```mermaid
graph TB
    subgraph "Kubernetes HPA"
        M[Metrics Server]
        HPA[Horizontal Pod Autoscaler]
    end

    subgraph "Pods (3-10 replicas)"
        P1[Pod 1]
        P2[Pod 2]
        P3[Pod 3]
        PN[Pod N]
    end

    M -->|CPU/Memory| HPA
    HPA -->|Scale| P1
    HPA -->|Scale| P2
    HPA -->|Scale| P3
    HPA -->|Scale| PN

    style HPA fill:#fff3e0
```

## CI/CD 파이프라인

```mermaid
graph LR
    A[Git Push] --> B[Lint]
    B --> C[Unit Tests]
    C --> D[Integration Tests]
    D --> E[Build Binaries]
    E --> F[Build Docker Images]
    F --> G{Environment}

    G -->|Develop| H[Deploy to Dev]
    G -->|Main| I[Deploy to Staging]
    G -->|Tag| J[Deploy to Production]

    style A fill:#e3f2fd
    style B fill:#fff3e0
    style C fill:#fff3e0
    style E fill:#f3e5f5
    style F fill:#f3e5f5
    style H fill:#c8e6c9
    style I fill:#ffecb3
    style J fill:#ffccbc
```

**CI/CD 단계:**
1. **Lint**: golangci-lint 코드 품질 검사
2. **Test**: 유닛 테스트 + 통합 테스트
3. **Build**: Go 바이너리 빌드
4. **Docker**: 멀티스테이지 Dockerfile
5. **Deploy**: Kubernetes 배포 (Dev → Staging → Production)

## Observability

```mermaid
graph TB
    subgraph "Application"
        APP[Database Service]
    end

    subgraph "Observability Stack"
        LOG[Zap Logger<br/>Structured Logging]
        TRACE[OpenTelemetry<br/>Distributed Tracing]
        METRIC[Prometheus<br/>Metrics]
    end

    subgraph "Backends"
        ELK[ELK Stack]
        JAEGER[Jaeger]
        GRAFANA[Grafana]
    end

    APP --> LOG
    APP --> TRACE
    APP --> METRIC

    LOG --> ELK
    TRACE --> JAEGER
    METRIC --> GRAFANA

    style APP fill:#e3f2fd
    style LOG fill:#fff3e0
    style TRACE fill:#f3e5f5
    style METRIC fill:#ffecb3
```

**메트릭:**
- Request Rate, Error Rate
- Latency (p50, p95, p99)
- DB Connection Pool
- Cache Hit Rate
- Kafka Lag

## 참고 자료

- [Vault Integration Guide](./VAULT_INTEGRATION.md)
- [GitLab CI/CD Configuration](../.gitlab-ci.yml)
- [Kubernetes Deployments](../deployments/kubernetes/)
- [Configuration Guide](../configs/)
- [Complete Example](../examples/complete_example.go)
