# gRPC Client Integration Guide

Database Service의 gRPC API를 사용하는 클라이언트 통합 가이드입니다.

## 📋 목차

1. [gRPC 개요](#grpc-개요)
2. [REST API vs gRPC](#rest-api-vs-grpc)
3. [프로토콜 정의](#프로토콜-정의)
4. [코드 생성](#코드-생성)
5. [언어별 클라이언트](#언어별-클라이언트)
6. [메타데이터로 데이터베이스 선택](#메타데이터로-데이터베이스-선택)
7. [성능 최적화](#성능-최적화)
8. [에러 핸들링](#에러-핸들링)
9. [베스트 프랙티스](#베스트-프랙티스)

---

## gRPC 개요

### gRPC란?

gRPC는 Google이 개발한 고성능 RPC(Remote Procedure Call) 프레임워크입니다.

**주요 특징:**
- **HTTP/2 기반**: 멀티플렉싱, 서버 푸시, 헤더 압축
- **Protocol Buffers**: 효율적인 직렬화 (JSON 대비 3-10배 빠름)
- **양방향 스트리밍**: 클라이언트-서버 간 실시간 통신
- **언어 중립적**: 다양한 언어 지원 (Go, Python, Java, C++, Node.js 등)
- **코드 생성**: .proto 파일로부터 자동 클라이언트/서버 코드 생성

### Database Service gRPC 정보

- **서버 주소**: `database-service:9090` (Kubernetes) 또는 `localhost:9090` (로컬)
- **프로토콜**: HTTP/2
- **인코딩**: Protocol Buffers v3
- **TLS**: 프로덕션에서 활성화 권장

---

## REST API vs gRPC

### 비교표

| 항목 | REST API | gRPC |
|------|----------|------|
| **프로토콜** | HTTP/1.1 | HTTP/2 |
| **데이터 포맷** | JSON (텍스트) | Protocol Buffers (바이너리) |
| **성능** | 기준 | **3-10배 빠름** |
| **페이로드 크기** | 기준 | **30-50% 작음** |
| **스트리밍** | 제한적 (SSE, WebSocket) | **양방향 스트리밍 네이티브 지원** |
| **브라우저 지원** | ✅ 네이티브 | ⚠️ gRPC-Web 필요 |
| **코드 생성** | 수동 또는 OpenAPI | ✅ 자동 생성 |
| **학습 곡선** | 낮음 | 중간 |
| **디버깅** | 쉬움 (curl, Postman) | 중간 (grpcurl, BloomRPC) |

### 언제 gRPC를 사용해야 하나?

**gRPC 사용 권장:**
- ✅ 마이크로서비스 간 통신 (Service-to-Service)
- ✅ 높은 성능이 필요한 경우
- ✅ 실시간 양방향 통신 (스트리밍)
- ✅ 다국어 클라이언트 개발 (자동 코드 생성)
- ✅ 모바일 앱 (배터리 절약, 데이터 절약)

**REST API 사용 권장:**
- ✅ 웹 브라우저 직접 호출
- ✅ 외부 API 노출 (Public API)
- ✅ 간단한 CRUD 작업
- ✅ 디버깅/테스트 용이성이 중요한 경우

---

## 프로토콜 정의

### database.proto

Database Service의 gRPC 서비스 정의:

```protobuf
syntax = "proto3";

package database;

option go_package = "github.com/YouSangSon/database-service/proto/pb";

import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";

service DatabaseService {
  // Create는 새로운 문서를 생성합니다
  rpc Create(CreateRequest) returns (CreateResponse);

  // Read는 ID로 문서를 조회합니다
  rpc Read(ReadRequest) returns (ReadResponse);

  // Update는 기존 문서를 업데이트합니다
  rpc Update(UpdateRequest) returns (UpdateResponse);

  // Delete는 문서를 삭제합니다
  rpc Delete(DeleteRequest) returns (DeleteResponse);

  // List는 문서 목록을 조회합니다
  rpc List(ListRequest) returns (ListResponse);

  // HealthCheck는 서비스 상태를 확인합니다
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

// CreateRequest는 문서 생성 요청입니다
message CreateRequest {
  string collection = 1;
  google.protobuf.Struct data = 2;
}

// CreateResponse는 문서 생성 응답입니다
message CreateResponse {
  string id = 1;
  google.protobuf.Timestamp created = 2;
}

// ReadRequest는 문서 조회 요청입니다
message ReadRequest {
  string collection = 1;
  string id = 2;
}

// ReadResponse는 문서 조회 응답입니다
message ReadResponse {
  string id = 1;
  google.protobuf.Struct data = 2;
  google.protobuf.Timestamp created_at = 3;
  google.protobuf.Timestamp updated_at = 4;
}

// UpdateRequest는 문서 업데이트 요청입니다
message UpdateRequest {
  string collection = 1;
  string id = 2;
  google.protobuf.Struct data = 3;
}

// UpdateResponse는 문서 업데이트 응답입니다
message UpdateResponse {
  bool success = 1;
  string message = 2;
}

// DeleteRequest는 문서 삭제 요청입니다
message DeleteRequest {
  string collection = 1;
  string id = 2;
}

// DeleteResponse는 문서 삭제 응답입니다
message DeleteResponse {
  bool success = 1;
  string message = 2;
}

// ListRequest는 문서 목록 조회 요청입니다
message ListRequest {
  string collection = 1;
  google.protobuf.Struct filter = 2;
  int32 limit = 3;
  int32 skip = 4;
}

// ListResponse는 문서 목록 조회 응답입니다
message ListResponse {
  repeated Document documents = 1;
  int32 total = 2;
}

// Document는 문서 모델입니다
message Document {
  string id = 1;
  google.protobuf.Struct data = 2;
  google.protobuf.Timestamp created_at = 3;
  google.protobuf.Timestamp updated_at = 4;
}

// HealthCheckRequest는 헬스체크 요청입니다
message HealthCheckRequest {}

// HealthCheckResponse는 헬스체크 응답입니다
message HealthCheckResponse {
  bool healthy = 1;
  string message = 2;
}
```

---

## 코드 생성

### 필요 도구 설치

#### 1. Protocol Buffers 컴파일러 (protoc)

```bash
# macOS
brew install protobuf

# Ubuntu/Debian
apt-get install -y protobuf-compiler

# 버전 확인
protoc --version  # libprotoc 3.x 이상
```

#### 2. 언어별 플러그인 설치

**Go:**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Python:**
```bash
pip install grpcio grpcio-tools
```

**Java:**
```bash
# Maven/Gradle에서 자동 처리
```

**Node.js:**
```bash
npm install -g grpc-tools
```

### proto 파일 다운로드

```bash
# Repository에서 다운로드
curl -O https://raw.githubusercontent.com/YouSangSon/database-service/main/proto/database.proto

# 또는 git clone
git clone https://github.com/YouSangSon/database-service.git
cd database-service/proto
```

### 코드 생성 명령어

#### Go

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       database.proto
```

생성 파일:
- `database.pb.go` - 메시지 타입
- `database_grpc.pb.go` - gRPC 클라이언트/서버 스텁

#### Python

```bash
python -m grpc_tools.protoc -I. \
       --python_out=. \
       --grpc_python_out=. \
       database.proto
```

생성 파일:
- `database_pb2.py` - 메시지 타입
- `database_pb2_grpc.py` - gRPC 클라이언트/서버 스텁

#### Java

```bash
protoc --java_out=src/main/java \
       --grpc-java_out=src/main/java \
       --plugin=protoc-gen-grpc-java=$(which protoc-gen-grpc-java) \
       database.proto
```

#### Node.js

```bash
grpc_tools_node_protoc --js_out=import_style=commonjs,binary:. \
                       --grpc_out=grpc_js:. \
                       database.proto
```

---

## 언어별 클라이언트

### 1. Go 클라이언트

#### 기본 클라이언트

```go
package main

import (
    "context"
    "log"
    "time"

    pb "github.com/YouSangSon/database-service/proto/pb"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
    "google.golang.org/protobuf/types/known/structpb"
)

// DatabaseClient는 gRPC 클라이언트 래퍼입니다
type DatabaseClient struct {
    client pb.DatabaseServiceClient
    conn   *grpc.ClientConn
}

// NewDatabaseClient는 새로운 gRPC 클라이언트를 생성합니다
func NewDatabaseClient(address string) (*DatabaseClient, error) {
    // TLS 비활성화 (개발 환경)
    conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, err
    }

    client := pb.NewDatabaseServiceClient(conn)

    return &DatabaseClient{
        client: client,
        conn:   conn,
    }, nil
}

// Close는 연결을 종료합니다
func (c *DatabaseClient) Close() error {
    return c.conn.Close()
}

// Create는 문서를 생성합니다
func (c *DatabaseClient) Create(ctx context.Context, collection string, data map[string]interface{}) (*pb.CreateResponse, error) {
    // map을 structpb.Struct로 변환
    structData, err := structpb.NewStruct(data)
    if err != nil {
        return nil, err
    }

    req := &pb.CreateRequest{
        Collection: collection,
        Data:       structData,
    }

    return c.client.Create(ctx, req)
}

// Read는 문서를 조회합니다
func (c *DatabaseClient) Read(ctx context.Context, collection, id string) (*pb.ReadResponse, error) {
    req := &pb.ReadRequest{
        Collection: collection,
        Id:         id,
    }

    return c.client.Read(ctx, req)
}

// Update는 문서를 업데이트합니다
func (c *DatabaseClient) Update(ctx context.Context, collection, id string, data map[string]interface{}) (*pb.UpdateResponse, error) {
    structData, err := structpb.NewStruct(data)
    if err != nil {
        return nil, err
    }

    req := &pb.UpdateRequest{
        Collection: collection,
        Id:         id,
        Data:       structData,
    }

    return c.client.Update(ctx, req)
}

// Delete는 문서를 삭제합니다
func (c *DatabaseClient) Delete(ctx context.Context, collection, id string) (*pb.DeleteResponse, error) {
    req := &pb.DeleteRequest{
        Collection: collection,
        Id:         id,
    }

    return c.client.Delete(ctx, req)
}

// List는 문서 목록을 조회합니다
func (c *DatabaseClient) List(ctx context.Context, collection string, filter map[string]interface{}, limit, skip int32) (*pb.ListResponse, error) {
    var structFilter *structpb.Struct
    if filter != nil {
        var err error
        structFilter, err = structpb.NewStruct(filter)
        if err != nil {
            return nil, err
        }
    }

    req := &pb.ListRequest{
        Collection: collection,
        Filter:     structFilter,
        Limit:      limit,
        Skip:       skip,
    }

    return c.client.List(ctx, req)
}

// HealthCheck는 서비스 상태를 확인합니다
func (c *DatabaseClient) HealthCheck(ctx context.Context) (*pb.HealthCheckResponse, error) {
    req := &pb.HealthCheckRequest{}
    return c.client.HealthCheck(ctx, req)
}

// CreateWithDatabase는 특정 데이터베이스를 지정하여 문서를 생성합니다
func (c *DatabaseClient) CreateWithDatabase(ctx context.Context, dbType, collection string, data map[string]interface{}) (*pb.CreateResponse, error) {
    // 메타데이터에 데이터베이스 타입 추가
    md := metadata.New(map[string]string{
        "x-database-type": dbType,
    })
    ctx = metadata.NewOutgoingContext(ctx, md)

    return c.Create(ctx, collection, data)
}
```

#### 사용 예제

```go
func main() {
    // 클라이언트 생성
    client, err := NewDatabaseClient("localhost:9090")
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 문서 생성 (MongoDB - 기본값)
    createResp, err := client.Create(ctx, "users", map[string]interface{}{
        "name":  "John Doe",
        "email": "john@example.com",
        "age":   30,
    })
    if err != nil {
        log.Fatalf("Create failed: %v", err)
    }
    log.Printf("Document created with ID: %s", createResp.Id)

    // PostgreSQL에 문서 생성
    createResp, err = client.CreateWithDatabase(ctx, "postgresql", "users", map[string]interface{}{
        "name":  "Jane Doe",
        "email": "jane@example.com",
    })
    if err != nil {
        log.Fatalf("Create failed: %v", err)
    }
    log.Printf("Document created in PostgreSQL with ID: %s", createResp.Id)

    // 문서 조회
    readResp, err := client.Read(ctx, "users", createResp.Id)
    if err != nil {
        log.Fatalf("Read failed: %v", err)
    }
    log.Printf("Document data: %v", readResp.Data.AsMap())

    // 문서 업데이트
    updateResp, err := client.Update(ctx, "users", createResp.Id, map[string]interface{}{
        "age": 31,
    })
    if err != nil {
        log.Fatalf("Update failed: %v", err)
    }
    log.Printf("Update result: %v", updateResp.Success)

    // 문서 목록 조회
    listResp, err := client.List(ctx, "users", map[string]interface{}{
        "age": map[string]interface{}{"$gte": 25},
    }, 10, 0)
    if err != nil {
        log.Fatalf("List failed: %v", err)
    }
    log.Printf("Found %d documents", listResp.Total)

    // 헬스체크
    healthResp, err := client.HealthCheck(ctx)
    if err != nil {
        log.Fatalf("HealthCheck failed: %v", err)
    }
    log.Printf("Health status: %v - %s", healthResp.Healthy, healthResp.Message)
}
```

### 2. Python 클라이언트

```python
import grpc
from google.protobuf import struct_pb2
import database_pb2
import database_pb2_grpc

class DatabaseClient:
    """gRPC Database Service 클라이언트"""

    def __init__(self, address: str):
        """
        클라이언트 초기화

        Args:
            address: gRPC 서버 주소 (예: localhost:9090)
        """
        self.channel = grpc.insecure_channel(address)
        self.stub = database_pb2_grpc.DatabaseServiceStub(self.channel)

    def close(self):
        """연결 종료"""
        self.channel.close()

    def _dict_to_struct(self, data: dict) -> struct_pb2.Struct:
        """Python dict를 protobuf Struct로 변환"""
        struct = struct_pb2.Struct()
        struct.update(data)
        return struct

    def create(self, collection: str, data: dict, db_type: str = "mongodb") -> database_pb2.CreateResponse:
        """
        문서 생성

        Args:
            collection: 컬렉션 이름
            data: 문서 데이터
            db_type: 데이터베이스 타입 (mongodb, postgresql 등)

        Returns:
            CreateResponse: 생성 결과
        """
        request = database_pb2.CreateRequest(
            collection=collection,
            data=self._dict_to_struct(data)
        )

        # 메타데이터에 데이터베이스 타입 추가
        metadata = [('x-database-type', db_type)]

        return self.stub.Create(request, metadata=metadata)

    def read(self, collection: str, doc_id: str, db_type: str = "mongodb") -> database_pb2.ReadResponse:
        """
        문서 조회

        Args:
            collection: 컬렉션 이름
            doc_id: 문서 ID
            db_type: 데이터베이스 타입

        Returns:
            ReadResponse: 문서 데이터
        """
        request = database_pb2.ReadRequest(
            collection=collection,
            id=doc_id
        )

        metadata = [('x-database-type', db_type)]

        return self.stub.Read(request, metadata=metadata)

    def update(self, collection: str, doc_id: str, data: dict, db_type: str = "mongodb") -> database_pb2.UpdateResponse:
        """
        문서 업데이트

        Args:
            collection: 컬렉션 이름
            doc_id: 문서 ID
            data: 업데이트할 데이터
            db_type: 데이터베이스 타입

        Returns:
            UpdateResponse: 업데이트 결과
        """
        request = database_pb2.UpdateRequest(
            collection=collection,
            id=doc_id,
            data=self._dict_to_struct(data)
        )

        metadata = [('x-database-type', db_type)]

        return self.stub.Update(request, metadata=metadata)

    def delete(self, collection: str, doc_id: str, db_type: str = "mongodb") -> database_pb2.DeleteResponse:
        """
        문서 삭제

        Args:
            collection: 컬렉션 이름
            doc_id: 문서 ID
            db_type: 데이터베이스 타입

        Returns:
            DeleteResponse: 삭제 결과
        """
        request = database_pb2.DeleteRequest(
            collection=collection,
            id=doc_id
        )

        metadata = [('x-database-type', db_type)]

        return self.stub.Delete(request, metadata=metadata)

    def list_documents(self, collection: str, filter_dict: dict = None,
                      limit: int = 10, skip: int = 0,
                      db_type: str = "mongodb") -> database_pb2.ListResponse:
        """
        문서 목록 조회

        Args:
            collection: 컬렉션 이름
            filter_dict: 필터 조건
            limit: 최대 결과 개수
            skip: 건너뛸 개수
            db_type: 데이터베이스 타입

        Returns:
            ListResponse: 문서 목록
        """
        request = database_pb2.ListRequest(
            collection=collection,
            limit=limit,
            skip=skip
        )

        if filter_dict:
            request.filter.CopyFrom(self._dict_to_struct(filter_dict))

        metadata = [('x-database-type', db_type)]

        return self.stub.List(request, metadata=metadata)

    def health_check(self) -> database_pb2.HealthCheckResponse:
        """
        헬스체크

        Returns:
            HealthCheckResponse: 헬스 상태
        """
        request = database_pb2.HealthCheckRequest()
        return self.stub.HealthCheck(request)


# 사용 예제
if __name__ == "__main__":
    # 클라이언트 생성
    client = DatabaseClient("localhost:9090")

    try:
        # 문서 생성 (MongoDB)
        create_response = client.create("users", {
            "name": "John Doe",
            "email": "john@example.com",
            "age": 30
        })
        print(f"Document created with ID: {create_response.id}")

        # 문서 생성 (PostgreSQL)
        create_response_pg = client.create("users", {
            "name": "Jane Doe",
            "email": "jane@example.com"
        }, db_type="postgresql")
        print(f"Document created in PostgreSQL with ID: {create_response_pg.id}")

        # 문서 조회
        read_response = client.read("users", create_response.id)
        print(f"Document data: {dict(read_response.data)}")

        # 문서 업데이트
        update_response = client.update("users", create_response.id, {"age": 31})
        print(f"Update result: {update_response.success}")

        # 문서 목록 조회
        list_response = client.list_documents("users", limit=10)
        print(f"Found {list_response.total} documents")

        # 헬스체크
        health_response = client.health_check()
        print(f"Health status: {health_response.healthy} - {health_response.message}")

    finally:
        client.close()
```

### 3. Node.js 클라이언트

```javascript
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const path = require('path');

// proto 파일 로드
const PROTO_PATH = path.join(__dirname, 'database.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const database = protoDescriptor.database;

class DatabaseClient {
  /**
   * gRPC Database Service 클라이언트
   * @param {string} address - gRPC 서버 주소
   */
  constructor(address) {
    this.client = new database.DatabaseService(
      address,
      grpc.credentials.createInsecure()
    );
  }

  /**
   * 문서 생성
   * @param {string} collection - 컬렉션 이름
   * @param {object} data - 문서 데이터
   * @param {string} dbType - 데이터베이스 타입
   * @returns {Promise<object>} 생성 결과
   */
  create(collection, data, dbType = 'mongodb') {
    return new Promise((resolve, reject) => {
      const metadata = new grpc.Metadata();
      metadata.add('x-database-type', dbType);

      this.client.Create(
        { collection, data },
        metadata,
        (error, response) => {
          if (error) {
            reject(error);
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * 문서 조회
   * @param {string} collection - 컬렉션 이름
   * @param {string} id - 문서 ID
   * @param {string} dbType - 데이터베이스 타입
   * @returns {Promise<object>} 문서 데이터
   */
  read(collection, id, dbType = 'mongodb') {
    return new Promise((resolve, reject) => {
      const metadata = new grpc.Metadata();
      metadata.add('x-database-type', dbType);

      this.client.Read(
        { collection, id },
        metadata,
        (error, response) => {
          if (error) {
            reject(error);
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * 문서 업데이트
   * @param {string} collection - 컬렉션 이름
   * @param {string} id - 문서 ID
   * @param {object} data - 업데이트할 데이터
   * @param {string} dbType - 데이터베이스 타입
   * @returns {Promise<object>} 업데이트 결과
   */
  update(collection, id, data, dbType = 'mongodb') {
    return new Promise((resolve, reject) => {
      const metadata = new grpc.Metadata();
      metadata.add('x-database-type', dbType);

      this.client.Update(
        { collection, id, data },
        metadata,
        (error, response) => {
          if (error) {
            reject(error);
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * 문서 삭제
   * @param {string} collection - 컬렉션 이름
   * @param {string} id - 문서 ID
   * @param {string} dbType - 데이터베이스 타입
   * @returns {Promise<object>} 삭제 결과
   */
  delete(collection, id, dbType = 'mongodb') {
    return new Promise((resolve, reject) => {
      const metadata = new grpc.Metadata();
      metadata.add('x-database-type', dbType);

      this.client.Delete(
        { collection, id },
        metadata,
        (error, response) => {
          if (error) {
            reject(error);
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * 문서 목록 조회
   * @param {string} collection - 컬렉션 이름
   * @param {object} filter - 필터 조건
   * @param {number} limit - 최대 결과 개수
   * @param {number} skip - 건너뛸 개수
   * @param {string} dbType - 데이터베이스 타입
   * @returns {Promise<object>} 문서 목록
   */
  list(collection, filter = {}, limit = 10, skip = 0, dbType = 'mongodb') {
    return new Promise((resolve, reject) => {
      const metadata = new grpc.Metadata();
      metadata.add('x-database-type', dbType);

      this.client.List(
        { collection, filter, limit, skip },
        metadata,
        (error, response) => {
          if (error) {
            reject(error);
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * 헬스체크
   * @returns {Promise<object>} 헬스 상태
   */
  healthCheck() {
    return new Promise((resolve, reject) => {
      this.client.HealthCheck({}, (error, response) => {
        if (error) {
          reject(error);
        } else {
          resolve(response);
        }
      });
    });
  }
}

// 사용 예제
async function main() {
  const client = new DatabaseClient('localhost:9090');

  try {
    // 문서 생성 (MongoDB)
    const createResult = await client.create('users', {
      name: 'John Doe',
      email: 'john@example.com',
      age: 30
    });
    console.log(`Document created with ID: ${createResult.id}`);

    // 문서 생성 (PostgreSQL)
    const createResultPg = await client.create('users', {
      name: 'Jane Doe',
      email: 'jane@example.com'
    }, 'postgresql');
    console.log(`Document created in PostgreSQL with ID: ${createResultPg.id}`);

    // 문서 조회
    const readResult = await client.read('users', createResult.id);
    console.log('Document data:', readResult.data);

    // 문서 업데이트
    const updateResult = await client.update('users', createResult.id, { age: 31 });
    console.log('Update result:', updateResult.success);

    // 문서 목록 조회
    const listResult = await client.list('users', {}, 10, 0);
    console.log(`Found ${listResult.total} documents`);

    // 헬스체크
    const healthResult = await client.healthCheck();
    console.log(`Health status: ${healthResult.healthy} - ${healthResult.message}`);

  } catch (error) {
    console.error('Error:', error.message);
  }
}

module.exports = DatabaseClient;

// 실행
if (require.main === module) {
  main();
}
```

### 4. Java 클라이언트

```java
package com.example.database;

import com.google.protobuf.Struct;
import com.google.protobuf.Value;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.stub.MetadataUtils;

import java.util.Map;
import java.util.concurrent.TimeUnit;

public class DatabaseClient {
    private final ManagedChannel channel;
    private final DatabaseServiceGrpc.DatabaseServiceBlockingStub blockingStub;

    private static final Metadata.Key<String> DATABASE_TYPE_KEY =
        Metadata.Key.of("x-database-type", Metadata.ASCII_STRING_MARSHALLER);

    /**
     * gRPC Database Service 클라이언트
     *
     * @param host 서버 호스트
     * @param port 서버 포트
     */
    public DatabaseClient(String host, int port) {
        this.channel = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext()
                .build();
        this.blockingStub = DatabaseServiceGrpc.newBlockingStub(channel);
    }

    /**
     * 연결 종료
     */
    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }

    /**
     * Map을 Struct로 변환
     */
    private Struct mapToStruct(Map<String, Object> map) {
        Struct.Builder structBuilder = Struct.newBuilder();
        for (Map.Entry<String, Object> entry : map.entrySet()) {
            structBuilder.putFields(entry.getKey(),
                Value.newBuilder().setStringValue(entry.getValue().toString()).build());
        }
        return structBuilder.build();
    }

    /**
     * 문서 생성
     *
     * @param collection 컬렉션 이름
     * @param data 문서 데이터
     * @param dbType 데이터베이스 타입
     * @return 생성 결과
     */
    public CreateResponse create(String collection, Map<String, Object> data, String dbType) {
        Metadata metadata = new Metadata();
        metadata.put(DATABASE_TYPE_KEY, dbType);

        DatabaseServiceGrpc.DatabaseServiceBlockingStub stub =
            MetadataUtils.attachHeaders(blockingStub, metadata);

        CreateRequest request = CreateRequest.newBuilder()
                .setCollection(collection)
                .setData(mapToStruct(data))
                .build();

        return stub.create(request);
    }

    /**
     * 문서 조회
     *
     * @param collection 컬렉션 이름
     * @param id 문서 ID
     * @param dbType 데이터베이스 타입
     * @return 문서 데이터
     */
    public ReadResponse read(String collection, String id, String dbType) {
        Metadata metadata = new Metadata();
        metadata.put(DATABASE_TYPE_KEY, dbType);

        DatabaseServiceGrpc.DatabaseServiceBlockingStub stub =
            MetadataUtils.attachHeaders(blockingStub, metadata);

        ReadRequest request = ReadRequest.newBuilder()
                .setCollection(collection)
                .setId(id)
                .build();

        return stub.read(request);
    }

    /**
     * 문서 업데이트
     *
     * @param collection 컬렉션 이름
     * @param id 문서 ID
     * @param data 업데이트할 데이터
     * @param dbType 데이터베이스 타입
     * @return 업데이트 결과
     */
    public UpdateResponse update(String collection, String id, Map<String, Object> data, String dbType) {
        Metadata metadata = new Metadata();
        metadata.put(DATABASE_TYPE_KEY, dbType);

        DatabaseServiceGrpc.DatabaseServiceBlockingStub stub =
            MetadataUtils.attachHeaders(blockingStub, metadata);

        UpdateRequest request = UpdateRequest.newBuilder()
                .setCollection(collection)
                .setId(id)
                .setData(mapToStruct(data))
                .build();

        return stub.update(request);
    }

    /**
     * 문서 삭제
     *
     * @param collection 컬렉션 이름
     * @param id 문서 ID
     * @param dbType 데이터베이스 타입
     * @return 삭제 결과
     */
    public DeleteResponse delete(String collection, String id, String dbType) {
        Metadata metadata = new Metadata();
        metadata.put(DATABASE_TYPE_KEY, dbType);

        DatabaseServiceGrpc.DatabaseServiceBlockingStub stub =
            MetadataUtils.attachHeaders(blockingStub, metadata);

        DeleteRequest request = DeleteRequest.newBuilder()
                .setCollection(collection)
                .setId(id)
                .build();

        return stub.delete(request);
    }

    /**
     * 문서 목록 조회
     *
     * @param collection 컬렉션 이름
     * @param filter 필터 조건
     * @param limit 최대 결과 개수
     * @param skip 건너뛸 개수
     * @param dbType 데이터베이스 타입
     * @return 문서 목록
     */
    public ListResponse list(String collection, Map<String, Object> filter, int limit, int skip, String dbType) {
        Metadata metadata = new Metadata();
        metadata.put(DATABASE_TYPE_KEY, dbType);

        DatabaseServiceGrpc.DatabaseServiceBlockingStub stub =
            MetadataUtils.attachHeaders(blockingStub, metadata);

        ListRequest.Builder requestBuilder = ListRequest.newBuilder()
                .setCollection(collection)
                .setLimit(limit)
                .setSkip(skip);

        if (filter != null && !filter.isEmpty()) {
            requestBuilder.setFilter(mapToStruct(filter));
        }

        return stub.list(requestBuilder.build());
    }

    /**
     * 헬스체크
     *
     * @return 헬스 상태
     */
    public HealthCheckResponse healthCheck() {
        HealthCheckRequest request = HealthCheckRequest.newBuilder().build();
        return blockingStub.healthCheck(request);
    }

    // 사용 예제
    public static void main(String[] args) throws Exception {
        DatabaseClient client = new DatabaseClient("localhost", 9090);

        try {
            // 문서 생성 (MongoDB)
            Map<String, Object> data = new HashMap<>();
            data.put("name", "John Doe");
            data.put("email", "john@example.com");
            data.put("age", "30");

            CreateResponse createResponse = client.create("users", data, "mongodb");
            System.out.println("Document created with ID: " + createResponse.getId());

            // 문서 생성 (PostgreSQL)
            CreateResponse createResponsePg = client.create("users", data, "postgresql");
            System.out.println("Document created in PostgreSQL with ID: " + createResponsePg.getId());

            // 문서 조회
            ReadResponse readResponse = client.read("users", createResponse.getId(), "mongodb");
            System.out.println("Document data: " + readResponse.getData());

            // 문서 업데이트
            Map<String, Object> updateData = new HashMap<>();
            updateData.put("age", "31");
            UpdateResponse updateResponse = client.update("users", createResponse.getId(), updateData, "mongodb");
            System.out.println("Update result: " + updateResponse.getSuccess());

            // 문서 목록 조회
            ListResponse listResponse = client.list("users", null, 10, 0, "mongodb");
            System.out.println("Found " + listResponse.getTotal() + " documents");

            // 헬스체크
            HealthCheckResponse healthResponse = client.healthCheck();
            System.out.println("Health status: " + healthResponse.getHealthy() + " - " + healthResponse.getMessage());

        } finally {
            client.shutdown();
        }
    }
}
```

---

## 메타데이터로 데이터베이스 선택

gRPC에서 데이터베이스를 선택하려면 **메타데이터(Metadata)**를 사용합니다.

### 메타데이터 키

```
x-database-type: mongodb|postgresql|mysql|cassandra|elasticsearch|vitess
```

### 언어별 메타데이터 설정

#### Go

```go
import "google.golang.org/grpc/metadata"

// 메타데이터 생성
md := metadata.New(map[string]string{
    "x-database-type": "postgresql",
})

// Context에 메타데이터 추가
ctx = metadata.NewOutgoingContext(ctx, md)

// RPC 호출
resp, err := client.Create(ctx, req)
```

#### Python

```python
# 메타데이터 생성
metadata = [('x-database-type', 'postgresql')]

# RPC 호출 시 메타데이터 전달
response = stub.Create(request, metadata=metadata)
```

#### Node.js

```javascript
const grpc = require('@grpc/grpc-js');

// 메타데이터 생성
const metadata = new grpc.Metadata();
metadata.add('x-database-type', 'postgresql');

// RPC 호출 시 메타데이터 전달
client.Create({ collection, data }, metadata, callback);
```

#### Java

```java
import io.grpc.Metadata;
import io.grpc.stub.MetadataUtils;

// 메타데이터 생성
Metadata metadata = new Metadata();
Metadata.Key<String> key = Metadata.Key.of("x-database-type", Metadata.ASCII_STRING_MARSHALLER);
metadata.put(key, "postgresql");

// Stub에 메타데이터 첨부
DatabaseServiceBlockingStub stub = MetadataUtils.attachHeaders(blockingStub, metadata);

// RPC 호출
CreateResponse response = stub.create(request);
```

---

## 성능 최적화

### 1. Connection Pooling

gRPC는 HTTP/2 멀티플렉싱을 사용하므로, **하나의 연결로 여러 요청을 동시에 처리**할 수 있습니다.

```go
// ❌ 나쁜 예: 매번 새 연결 생성
func BadExample() {
    conn, _ := grpc.Dial("localhost:9090", grpc.WithInsecure())
    defer conn.Close()

    client := pb.NewDatabaseServiceClient(conn)
    client.Create(ctx, req)
}

// ✅ 좋은 예: 연결 재사용
var globalConn *grpc.ClientConn
var globalClient pb.DatabaseServiceClient

func init() {
    conn, _ := grpc.Dial("localhost:9090", grpc.WithInsecure())
    globalConn = conn
    globalClient = pb.NewDatabaseServiceClient(conn)
}

func GoodExample() {
    globalClient.Create(ctx, req)
}
```

### 2. Keepalive 설정

```go
import "google.golang.org/grpc/keepalive"

conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithInsecure(),
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                10 * time.Second,  // Keepalive ping 간격
        Timeout:             3 * time.Second,   // Ping 타임아웃
        PermitWithoutStream: true,              // 스트림 없어도 keepalive 허용
    }),
)
```

### 3. Compression

```go
import "google.golang.org/grpc/encoding/gzip"

// 압축 활성화
resp, err := client.Create(
    ctx,
    req,
    grpc.UseCompressor(gzip.Name),
)
```

### 4. 병렬 처리

```go
import "golang.org/x/sync/errgroup"

// 여러 문서를 병렬로 생성
func CreateConcurrently(ctx context.Context, client pb.DatabaseServiceClient, docs []*pb.CreateRequest) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, doc := range docs {
        doc := doc  // 클로저 변수 캡처
        g.Go(func() error {
            _, err := client.Create(ctx, doc)
            return err
        })
    }

    return g.Wait()
}
```

---

## 에러 핸들링

### gRPC 상태 코드

| 코드 | 의미 | 처리 방법 |
|------|------|----------|
| OK (0) | 성공 | 정상 처리 |
| CANCELLED (1) | 취소됨 | 재시도 안함 |
| INVALID_ARGUMENT (3) | 잘못된 인자 | 요청 수정 |
| NOT_FOUND (5) | 리소스 없음 | 존재 확인 |
| ALREADY_EXISTS (6) | 이미 존재 | 중복 확인 |
| PERMISSION_DENIED (7) | 권한 없음 | 인증 확인 |
| RESOURCE_EXHAUSTED (8) | 리소스 고갈 | Backoff 후 재시도 |
| UNAVAILABLE (14) | 서비스 불가 | Exponential backoff 재시도 |
| DEADLINE_EXCEEDED (4) | 타임아웃 | 타임아웃 증가 또는 재시도 |

### Go 에러 핸들링

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

resp, err := client.Create(ctx, req)
if err != nil {
    st, ok := status.FromError(err)
    if !ok {
        // 네트워크 에러 등
        return err
    }

    switch st.Code() {
    case codes.InvalidArgument:
        // 잘못된 요청 - 재시도 안함
        return fmt.Errorf("invalid request: %v", st.Message())
    case codes.NotFound:
        // 리소스 없음
        return fmt.Errorf("not found: %v", st.Message())
    case codes.Unavailable:
        // 서비스 불가 - 재시도
        time.Sleep(time.Second)
        return client.Create(ctx, req)  // 재시도
    default:
        return fmt.Errorf("grpc error: %v", st.Message())
    }
}
```

### Python 에러 핸들링

```python
import grpc

try:
    response = stub.Create(request)
except grpc.RpcError as e:
    code = e.code()

    if code == grpc.StatusCode.INVALID_ARGUMENT:
        print(f"Invalid request: {e.details()}")
    elif code == grpc.StatusCode.NOT_FOUND:
        print(f"Not found: {e.details()}")
    elif code == grpc.StatusCode.UNAVAILABLE:
        # 재시도
        time.sleep(1)
        response = stub.Create(request)
    else:
        print(f"gRPC error: {e.details()}")
```

---

## 베스트 프랙티스

### 1. TLS 사용 (프로덕션)

```go
import "google.golang.org/grpc/credentials"

// TLS 인증서 로드
creds, err := credentials.NewClientTLSFromFile("cert.pem", "")
if err != nil {
    log.Fatal(err)
}

// TLS 활성화
conn, err := grpc.Dial(
    "database-service:9090",
    grpc.WithTransportCredentials(creds),
)
```

### 2. Timeout 설정

```go
// RPC별 타임아웃
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Create(ctx, req)
```

### 3. Retry 로직

```go
func CreateWithRetry(ctx context.Context, client pb.DatabaseServiceClient, req *pb.CreateRequest, maxRetries int) (*pb.CreateResponse, error) {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        resp, err := client.Create(ctx, req)
        if err == nil {
            return resp, nil
        }

        lastErr = err

        // gRPC 상태 코드 확인
        st, ok := status.FromError(err)
        if !ok || st.Code() != codes.Unavailable {
            return nil, err  // 재시도 불가능한 에러
        }

        // Exponential backoff
        backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
        time.Sleep(backoff)
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### 4. Interceptor 사용

```go
// Logging Interceptor
func loggingInterceptor(
    ctx context.Context,
    method string,
    req interface{},
    reply interface{},
    cc *grpc.ClientConn,
    invoker grpc.UnaryInvoker,
    opts ...grpc.CallOption,
) error {
    start := time.Now()
    err := invoker(ctx, method, req, reply, cc, opts...)
    log.Printf("Method: %s, Duration: %v, Error: %v", method, time.Since(start), err)
    return err
}

// Interceptor 등록
conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithInsecure(),
    grpc.WithUnaryInterceptor(loggingInterceptor),
)
```

### 5. Health Check

```go
// 정기적인 헬스체크
func HealthCheckLoop(client pb.DatabaseServiceClient, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        resp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{})
        cancel()

        if err != nil || !resp.Healthy {
            log.Printf("Health check failed: %v", err)
        } else {
            log.Printf("Service healthy: %s", resp.Message)
        }
    }
}
```

---

## 도구

### grpcurl - CLI 도구

```bash
# 설치
brew install grpcurl  # macOS
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 서비스 목록 조회
grpcurl -plaintext localhost:9090 list

# 메서드 목록 조회
grpcurl -plaintext localhost:9090 list database.DatabaseService

# RPC 호출 (MongoDB)
grpcurl -plaintext -d '{
  "collection": "users",
  "data": {"name": "John", "age": 30}
}' localhost:9090 database.DatabaseService/Create

# RPC 호출 (PostgreSQL) - 메타데이터 사용
grpcurl -plaintext \
  -H 'x-database-type: postgresql' \
  -d '{"collection": "users", "data": {"name": "Jane"}}' \
  localhost:9090 database.DatabaseService/Create
```

### BloomRPC - GUI 도구

1. [BloomRPC](https://github.com/bloomrpc/bloomrpc) 다운로드
2. proto 파일 임포트
3. 서버 주소 입력: `localhost:9090`
4. Metadata에 `x-database-type: postgresql` 추가
5. 요청 실행

---

## 추가 리소스

- **REST API 가이드**: [CLIENT_INTEGRATION.md](./CLIENT_INTEGRATION.md)
- **API 명세서**: [REST_API_SPECIFICATION.md](./REST_API_SPECIFICATION.md)
- **빠른 시작**: [QUICKSTART.md](./QUICKSTART.md)
- **gRPC 공식 문서**: https://grpc.io/docs/

---

## 지원

질문이나 이슈가 있으면 GitHub Issues에 등록해주세요:
https://github.com/YouSangSon/database-service/issues
