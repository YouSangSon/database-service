# Database Service

확장 가능한 Go 기반의 데이터베이스 서비스입니다. REST API와 gRPC를 통해 여러 데이터베이스에 대한 CRUD 작업을 제공합니다.

## 특징

- ✅ **확장 가능한 아키텍처**: 데이터베이스 추상화 레이어를 통해 여러 데이터베이스 지원
- ✅ **현재 지원**: MongoDB
- 🔜 **향후 지원 예정**: PostgreSQL, MySQL, Redis 등
- ✅ **이중 프로토콜**: REST API와 gRPC 동시 지원
- ✅ **범용 CRUD**: 모든 컬렉션/테이블에 대한 범용 CRUD 작업
- ✅ **Docker 지원**: Docker Compose를 통한 쉬운 배포

## 프로젝트 구조

```
.
├── cmd/
│   ├── api/          # REST API 서버
│   └── grpc/         # gRPC 서버
├── internal/
│   ├── database/     # 데이터베이스 추상화 레이어
│   │   └── mongodb/  # MongoDB 구현
│   ├── models/       # 데이터 모델
│   ├── service/      # 비즈니스 로직
│   ├── handler/      # HTTP 핸들러
│   └── grpc_handler/ # gRPC 핸들러
├── proto/            # gRPC proto 파일
├── config/           # 설정 관리
└── docker-compose.yml
```

## 시작하기

### 필요 사항

- Go 1.21+
- Docker & Docker Compose
- Protocol Buffers 컴파일러 (protoc)
- Make

### 설치

1. 저장소 클론:
```bash
git clone https://github.com/YouSangSon/database-service.git
cd database-service
```

2. 의존성 설치:
```bash
make deps
```

3. Proto 파일 컴파일 (gRPC를 사용할 경우):
```bash
make proto
```

### Docker로 실행

가장 간단한 방법은 Docker Compose를 사용하는 것입니다:

```bash
# 모든 서비스 시작 (MongoDB, API, gRPC)
docker-compose up -d

# 로그 확인
docker-compose logs -f

# 서비스 중지
docker-compose down
```

### 로컬에서 실행

1. MongoDB 실행:
```bash
docker run -d -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password \
  --name mongodb \
  mongo:7.0
```

2. 환경변수 설정:
```bash
cp .env.example .env
# .env 파일을 필요에 따라 수정
```

3. API 서버 실행:
```bash
make run-api
# 또는
go run cmd/api/main.go
```

4. gRPC 서버 실행 (다른 터미널에서):
```bash
make run-grpc
# 또는
go run cmd/grpc/main.go
```

## API 사용법

### REST API

기본 엔드포인트: `http://localhost:8080`

#### 헬스체크
```bash
curl http://localhost:8080/health
```

#### 문서 생성
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
    "email": "jane@example.com"
  }'
```

#### 문서 삭제
```bash
curl -X DELETE http://localhost:8080/api/v1/documents/users/{id}
```

#### 문서 목록 조회
```bash
curl http://localhost:8080/api/v1/documents/users
```

### gRPC

gRPC 서버는 `localhost:50051`에서 실행됩니다.

#### grpcurl 사용 예제

```bash
# 서비스 목록 조회
grpcurl -plaintext localhost:50051 list

# 헬스체크
grpcurl -plaintext localhost:50051 database.DatabaseService/HealthCheck

# 문서 생성
grpcurl -plaintext -d '{
  "collection": "users",
  "data": {
    "name": "John Doe",
    "email": "john@example.com"
  }
}' localhost:50051 database.DatabaseService/Create

# 문서 조회
grpcurl -plaintext -d '{
  "collection": "users",
  "id": "your-document-id"
}' localhost:50051 database.DatabaseService/Read
```

## 환경변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `API_PORT` | REST API 서버 포트 | 8080 |
| `GRPC_PORT` | gRPC 서버 포트 | 50051 |
| `DB_TYPE` | 데이터베이스 타입 | mongodb |
| `DB_HOST` | 데이터베이스 호스트 | localhost |
| `DB_PORT` | 데이터베이스 포트 | 27017 |
| `DB_USERNAME` | 데이터베이스 사용자명 | - |
| `DB_PASSWORD` | 데이터베이스 비밀번호 | - |
| `DB_DATABASE` | 데이터베이스 이름 | testdb |

## 개발

### 테스트 실행
```bash
make test
```

### 빌드
```bash
make build
```

### 클린업
```bash
make clean
```

## 새로운 데이터베이스 추가하기

1. `internal/database/` 아래에 새 디렉토리 생성 (예: `postgresql/`)
2. `database.Database` 인터페이스 구현
3. `cmd/api/main.go`와 `cmd/grpc/main.go`의 switch 문에 새 케이스 추가

예제:
```go
// internal/database/postgresql/postgresql.go
package postgresql

import (
    "github.com/YouSangSon/database-service/internal/database"
)

type PostgreSQL struct {
    // implementation
}

func NewPostgreSQL(config *database.Config) *PostgreSQL {
    return &PostgreSQL{}
}

// Implement all database.Database interface methods
func (p *PostgreSQL) Connect(ctx context.Context) error { ... }
func (p *PostgreSQL) Disconnect(ctx context.Context) error { ... }
// ... etc
```

## 아키텍처

이 프로젝트는 클린 아키텍처 원칙을 따릅니다:

1. **데이터베이스 추상화 레이어**: 모든 데이터베이스 구현체가 따라야 할 공통 인터페이스
2. **서비스 레이어**: 비즈니스 로직을 처리하고 데이터베이스 추상화를 사용
3. **핸들러 레이어**: HTTP/gRPC 요청을 받아 서비스 레이어를 호출
4. **설정 관리**: 환경변수를 통한 중앙 집중식 설정

## 라이선스

MIT License

## 기여

Pull Request를 환영합니다! 주요 변경사항은 먼저 이슈를 열어 논의해주세요.
