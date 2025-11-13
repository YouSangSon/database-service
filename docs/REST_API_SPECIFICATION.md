# REST API 완벽 명세서

Database Service - 범용 데이터베이스 REST API
지원 DB: MongoDB, PostgreSQL, MySQL, Cassandra, Elasticsearch, Vitess

## 📋 목차
1. [공통 사항](#공통-사항)
2. [기본 CRUD API](#기본-crud-api)
3. [쿼리 & 검색 API](#쿼리--검색-api)
4. [원자적 연산 API](#원자적-연산-api)
5. [집계 API](#집계-api)
6. [벌크 작업 API](#벌크-작업-api)
7. [인덱스 관리 API](#인덱스-관리-api)
8. [컬렉션 관리 API](#컬렉션-관리-api)
9. [트랜잭션 API](#트랜잭션-api)
10. [Raw Query API](#raw-query-api)
11. [헬스체크 & 모니터링](#헬스체크--모니터링)

---

## 공통 사항

### Base URL
```
http://localhost:8080/api/v1
```

### 공통 헤더

모든 요청에 다음 헤더가 필요합니다:

```http
Content-Type: application/json
X-Database-Type: mongodb|postgresql|mysql|cassandra|elasticsearch|vitess
```

### 데이터베이스 선택

**X-Database-Type** 헤더로 사용할 데이터베이스를 선택합니다:
- `mongodb` - MongoDB 7.0
- `postgresql` - PostgreSQL 16
- `mysql` - MySQL 8.0
- `cassandra` - Cassandra 4.1
- `elasticsearch` - Elasticsearch 8.11
- `vitess` - Vitess

### 공통 응답 형식

#### 성공 응답
```json
{
  "success": true,
  "data": { ... },
  "message": "Operation completed successfully"
}
```

#### 에러 응답
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Error description",
    "details": { ... }
  }
}
```

### HTTP 상태 코드
- `200 OK` - 성공
- `201 Created` - 생성 성공
- `400 Bad Request` - 잘못된 요청
- `404 Not Found` - 리소스 없음
- `409 Conflict` - 충돌 (예: 낙관적 잠금 실패)
- `500 Internal Server Error` - 서버 에러

---

## 기본 CRUD API

### 1. 문서 생성 (Create)

**POST** `/documents`

단일 문서를 생성합니다.

#### Headers
```http
Content-Type: application/json
X-Database-Type: mongodb
```

#### Request Body
```json
{
  "collection": "users",
  "data": {
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30,
    "tags": ["developer", "golang"]
  }
}
```

#### Response (201 Created)
```json
{
  "success": true,
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "collection": "users",
    "data": {
      "name": "John Doe",
      "email": "john@example.com",
      "age": 30,
      "tags": ["developer", "golang"]
    },
    "created_at": "2025-11-12T10:30:00Z",
    "updated_at": "2025-11-12T10:30:00Z",
    "version": 1,
    "metadata": {}
  },
  "message": "Document created successfully"
}
```

---

### 2. 문서 조회 (Read)

**GET** `/documents/{collection}/{id}`

ID로 단일 문서를 조회합니다.

#### Path Parameters
- `collection` (string, required) - 컬렉션/테이블 이름
- `id` (string, required) - 문서 ID

#### Headers
```http
X-Database-Type: mongodb
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "collection": "users",
    "data": {
      "name": "John Doe",
      "email": "john@example.com",
      "age": 30,
      "tags": ["developer", "golang"]
    },
    "created_at": "2025-11-12T10:30:00Z",
    "updated_at": "2025-11-12T10:30:00Z",
    "version": 1,
    "metadata": {}
  }
}
```

#### Error Response (404 Not Found)
```json
{
  "success": false,
  "error": {
    "code": "DOCUMENT_NOT_FOUND",
    "message": "Document not found",
    "details": {
      "collection": "users",
      "id": "507f1f77bcf86cd799439011"
    }
  }
}
```

---

### 3. 문서 업데이트 (Update)

**PUT** `/documents/{collection}/{id}`

기존 문서를 업데이트합니다 (낙관적 잠금 포함).

#### Path Parameters
- `collection` (string, required)
- `id` (string, required)

#### Headers
```http
Content-Type: application/json
X-Database-Type: mongodb
```

#### Request Body
```json
{
  "data": {
    "name": "Jane Doe",
    "age": 31
  },
  "version": 1
}
```

**주의**: `version` 필드는 낙관적 잠금을 위해 필수입니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "collection": "users",
    "data": {
      "name": "Jane Doe",
      "email": "john@example.com",
      "age": 31,
      "tags": ["developer", "golang"]
    },
    "created_at": "2025-11-12T10:30:00Z",
    "updated_at": "2025-11-12T10:35:00Z",
    "version": 2,
    "metadata": {}
  },
  "message": "Document updated successfully"
}
```

#### Error Response (409 Conflict)
```json
{
  "success": false,
  "error": {
    "code": "OPTIMISTIC_LOCK_ERROR",
    "message": "Document was modified by another process",
    "details": {
      "expected_version": 1,
      "current_version": 2
    }
  }
}
```

---

### 4. 문서 교체 (Replace)

**PUT** `/documents/{collection}/{id}/replace`

문서를 완전히 교체합니다.

#### Request Body
```json
{
  "data": {
    "name": "Completely New Data",
    "status": "active"
  },
  "metadata": {
    "source": "api"
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Document replaced successfully"
}
```

---

### 5. 문서 삭제 (Delete)

**DELETE** `/documents/{collection}/{id}`

문서를 삭제합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Document deleted successfully"
}
```

---

## 쿼리 & 검색 API

### 6. 문서 목록 조회 (List)

**GET** `/documents/{collection}`

필터, 정렬, 페이징을 사용하여 문서 목록을 조회합니다.

#### Query Parameters
- `filter` (string, optional) - JSON 형식의 필터 (URL encoded)
- `sort` (string, optional) - 정렬 필드와 순서 (예: `name:asc,age:desc`)
- `limit` (integer, optional) - 최대 결과 수 (기본값: 10)
- `offset` (integer, optional) - 건너뛸 문서 수 (기본값: 0)
- `projection` (string, optional) - 반환할 필드 (쉼표로 구분)

#### Example Request
```http
GET /api/v1/documents/users?filter={"age":{"$gte":25}}&sort=name:asc&limit=10&offset=0
X-Database-Type: mongodb
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "documents": [
      {
        "id": "507f1f77bcf86cd799439011",
        "collection": "users",
        "data": {
          "name": "Alice",
          "age": 28
        },
        "created_at": "2025-11-12T10:30:00Z",
        "updated_at": "2025-11-12T10:30:00Z",
        "version": 1
      },
      {
        "id": "507f1f77bcf86cd799439012",
        "collection": "users",
        "data": {
          "name": "Bob",
          "age": 35
        },
        "created_at": "2025-11-12T10:31:00Z",
        "updated_at": "2025-11-12T10:31:00Z",
        "version": 1
      }
    ],
    "pagination": {
      "total": 150,
      "limit": 10,
      "offset": 0,
      "has_more": true
    }
  }
}
```

---

### 7. 문서 검색 (Search)

**POST** `/documents/{collection}/search`

복잡한 검색 쿼리를 실행합니다.

#### Request Body
```json
{
  "filter": {
    "age": {"$gte": 25},
    "tags": {"$in": ["developer", "engineer"]}
  },
  "sort": {
    "name": 1,
    "age": -1
  },
  "limit": 20,
  "offset": 0,
  "projection": {
    "name": 1,
    "email": 1,
    "age": 1
  }
}
```

#### Response (200 OK)
동일한 형식으로 반환

---

### 8. 문서 개수 세기 (Count)

**POST** `/documents/{collection}/count`

필터 조건과 일치하는 문서 개수를 반환합니다.

#### Request Body
```json
{
  "filter": {
    "age": {"$gte": 25}
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "count": 150,
    "collection": "users"
  }
}
```

---

### 9. 예상 문서 개수 (Estimated Count)

**GET** `/documents/{collection}/count/estimate`

컬렉션의 예상 문서 개수를 빠르게 반환합니다 (정확하지 않을 수 있음).

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "estimated_count": 1500,
    "collection": "users"
  }
}
```

---

## 원자적 연산 API

### 10. 찾아서 업데이트 (Find and Update)

**POST** `/documents/{collection}/{id}/find-and-update`

문서를 찾아서 업데이트하고 업데이트된 문서를 반환합니다 (비관적 잠금).

#### Request Body
```json
{
  "update": {
    "age": 32,
    "status": "active"
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "collection": "users",
    "data": {
      "name": "John Doe",
      "age": 32,
      "status": "active"
    },
    "version": 3
  },
  "message": "Document found and updated"
}
```

---

### 11. 찾아서 교체 (Find and Replace)

**POST** `/documents/{collection}/{id}/find-and-replace`

문서를 찾아서 교체하고 교체된 문서를 반환합니다.

#### Request Body
```json
{
  "replacement": {
    "name": "New Name",
    "status": "inactive"
  }
}
```

#### Response (200 OK)
동일한 형식

---

### 12. 찾아서 삭제 (Find and Delete)

**POST** `/documents/{collection}/{id}/find-and-delete`

문서를 찾아서 삭제하고 삭제된 문서를 반환합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "collection": "users",
    "data": {
      "name": "John Doe",
      "age": 30
    }
  },
  "message": "Document found and deleted"
}
```

---

### 13. Upsert

**POST** `/documents/{collection}/upsert`

문서가 없으면 생성하고, 있으면 업데이트합니다.

#### Request Body
```json
{
  "filter": {
    "id": "user123"
  },
  "update": {
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "id": "user123",
    "operation": "updated"
  },
  "message": "Document upserted successfully"
}
```

---

## 집계 API

### 14. 집계 파이프라인 (Aggregate)

**POST** `/documents/{collection}/aggregate`

MongoDB 스타일의 집계 파이프라인을 실행합니다.

**지원 DB**: MongoDB, Vitess (제한적)

#### Request Body
```json
{
  "pipeline": [
    {
      "$match": {
        "age": {"$gte": 25}
      }
    },
    {
      "$group": {
        "_id": "$age",
        "count": {"$sum": 1},
        "avg_age": {"$avg": "$age"}
      }
    },
    {
      "$sort": {
        "count": -1
      }
    },
    {
      "$limit": 10
    }
  ]
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "_id": 30,
        "count": 45,
        "avg_age": 30
      },
      {
        "_id": 28,
        "count": 32,
        "avg_age": 28
      }
    ]
  }
}
```

---

### 15. 고유 값 조회 (Distinct)

**POST** `/documents/{collection}/distinct`

특정 필드의 고유한 값을 조회합니다.

#### Request Body
```json
{
  "field": "age",
  "filter": {
    "status": "active"
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "field": "age",
    "values": [25, 28, 30, 32, 35, 40]
  }
}
```

---

## 벌크 작업 API

### 16. 여러 문서 생성 (Bulk Insert)

**POST** `/documents/bulk/insert`

여러 문서를 한 번에 생성합니다.

#### Request Body
```json
{
  "collection": "users",
  "documents": [
    {
      "data": {
        "name": "User 1",
        "age": 25
      }
    },
    {
      "data": {
        "name": "User 2",
        "age": 30
      }
    },
    {
      "data": {
        "name": "User 3",
        "age": 35
      }
    }
  ]
}
```

#### Response (201 Created)
```json
{
  "success": true,
  "data": {
    "inserted_count": 3,
    "inserted_ids": [
      "507f1f77bcf86cd799439011",
      "507f1f77bcf86cd799439012",
      "507f1f77bcf86cd799439013"
    ]
  },
  "message": "Documents inserted successfully"
}
```

---

### 17. 여러 문서 업데이트 (Update Many)

**POST** `/documents/{collection}/update-many`

필터와 일치하는 여러 문서를 업데이트합니다.

#### Request Body
```json
{
  "filter": {
    "age": {"$lt": 30}
  },
  "update": {
    "status": "young",
    "discount": 10
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "matched_count": 45,
    "modified_count": 45
  },
  "message": "Documents updated successfully"
}
```

---

### 18. 여러 문서 삭제 (Delete Many)

**POST** `/documents/{collection}/delete-many`

필터와 일치하는 여러 문서를 삭제합니다.

#### Request Body
```json
{
  "filter": {
    "status": "inactive",
    "last_login": {"$lt": "2024-01-01"}
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "deleted_count": 150
  },
  "message": "Documents deleted successfully"
}
```

---

### 19. 벌크 쓰기 (Bulk Write)

**POST** `/documents/bulk/write`

여러 작업(insert, update, delete, replace)을 한 번에 실행합니다.

#### Request Body
```json
{
  "operations": [
    {
      "type": "insert",
      "collection": "users",
      "document": {
        "data": {"name": "New User", "age": 25}
      }
    },
    {
      "type": "update",
      "collection": "users",
      "filter": {"id": "user123"},
      "update": {"age": 31}
    },
    {
      "type": "delete",
      "collection": "users",
      "filter": {"id": "user456"}
    },
    {
      "type": "replace",
      "collection": "users",
      "id": "user789",
      "document": {
        "data": {"name": "Replaced User", "status": "active"}
      }
    }
  ]
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "inserted_count": 1,
    "matched_count": 1,
    "modified_count": 1,
    "deleted_count": 1,
    "upserted_count": 0,
    "upserted_ids": {}
  },
  "message": "Bulk operations completed"
}
```

---

## 인덱스 관리 API

### 20. 인덱스 생성 (Create Index)

**POST** `/indexes/{collection}`

단일 인덱스를 생성합니다.

#### Request Body
```json
{
  "keys": {
    "email": 1,
    "age": -1
  },
  "options": {
    "name": "idx_email_age",
    "unique": true,
    "background": true,
    "sparse": false
  }
}
```

**키 방향**:
- `1`: 오름차순
- `-1`: 내림차순

#### Response (201 Created)
```json
{
  "success": true,
  "data": {
    "index_name": "idx_email_age",
    "collection": "users"
  },
  "message": "Index created successfully"
}
```

---

### 21. 여러 인덱스 생성 (Create Indexes)

**POST** `/indexes/{collection}/bulk`

여러 인덱스를 한 번에 생성합니다.

#### Request Body
```json
{
  "indexes": [
    {
      "keys": {"email": 1},
      "options": {"name": "idx_email", "unique": true}
    },
    {
      "keys": {"age": 1},
      "options": {"name": "idx_age"}
    },
    {
      "keys": {"created_at": -1},
      "options": {"name": "idx_created_at"}
    }
  ]
}
```

#### Response (201 Created)
```json
{
  "success": true,
  "data": {
    "index_names": ["idx_email", "idx_age", "idx_created_at"],
    "created_count": 3
  },
  "message": "Indexes created successfully"
}
```

---

### 22. 인덱스 삭제 (Drop Index)

**DELETE** `/indexes/{collection}/{index_name}`

특정 인덱스를 삭제합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Index dropped successfully"
}
```

---

### 23. 인덱스 목록 조회 (List Indexes)

**GET** `/indexes/{collection}`

컬렉션의 모든 인덱스를 조회합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "indexes": [
      {
        "name": "_id_",
        "keys": {"_id": 1},
        "unique": true
      },
      {
        "name": "idx_email",
        "keys": {"email": 1},
        "unique": true
      },
      {
        "name": "idx_age",
        "keys": {"age": 1},
        "unique": false
      }
    ]
  }
}
```

---

## 컬렉션 관리 API

### 24. 컬렉션 생성 (Create Collection)

**POST** `/collections`

새 컬렉션/테이블을 생성합니다.

#### Request Body
```json
{
  "name": "products",
  "options": {
    "capped": false,
    "size": 0,
    "max": 0
  }
}
```

#### Response (201 Created)
```json
{
  "success": true,
  "data": {
    "collection": "products"
  },
  "message": "Collection created successfully"
}
```

---

### 25. 컬렉션 삭제 (Drop Collection)

**DELETE** `/collections/{collection}`

컬렉션/테이블을 삭제합니다.

**주의**: 모든 데이터가 삭제됩니다!

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Collection dropped successfully"
}
```

---

### 26. 컬렉션 이름 변경 (Rename Collection)

**POST** `/collections/{old_name}/rename`

컬렉션/테이블 이름을 변경합니다.

**지원 DB**: MongoDB, PostgreSQL, MySQL (Cassandra, Elasticsearch 미지원)

#### Request Body
```json
{
  "new_name": "users_v2"
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "old_name": "users",
    "new_name": "users_v2"
  },
  "message": "Collection renamed successfully"
}
```

---

### 27. 컬렉션 목록 조회 (List Collections)

**GET** `/collections`

데이터베이스의 모든 컬렉션/테이블을 조회합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "collections": [
      "users",
      "products",
      "orders",
      "payments"
    ],
    "count": 4
  }
}
```

---

### 28. 컬렉션 존재 확인 (Check Collection Exists)

**GET** `/collections/{collection}/exists`

컬렉션/테이블이 존재하는지 확인합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "collection": "users",
    "exists": true
  }
}
```

---

## 트랜잭션 API

### 29. 트랜잭션 실행 (Execute Transaction)

**POST** `/transactions/execute`

트랜잭션 내에서 여러 작업을 원자적으로 실행합니다.

**지원 DB**: MongoDB, PostgreSQL, MySQL

#### Request Body
```json
{
  "operations": [
    {
      "type": "insert",
      "collection": "orders",
      "document": {
        "data": {
          "user_id": "user123",
          "product_id": "prod456",
          "amount": 100
        }
      }
    },
    {
      "type": "update",
      "collection": "users",
      "filter": {"id": "user123"},
      "update": {"balance": {"$inc": -100}}
    },
    {
      "type": "update",
      "collection": "products",
      "filter": {"id": "prod456"},
      "update": {"stock": {"$inc": -1}}
    }
  ]
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "transaction_id": "txn_abc123",
    "operations_count": 3,
    "status": "committed"
  },
  "message": "Transaction completed successfully"
}
```

#### Error Response (트랜잭션 롤백)
```json
{
  "success": false,
  "error": {
    "code": "TRANSACTION_FAILED",
    "message": "Transaction rolled back due to error",
    "details": {
      "failed_operation": 2,
      "reason": "Insufficient stock"
    }
  }
}
```

---

## Raw Query API

### 30. Raw Query 실행 (Execute Raw Query)

**POST** `/query/raw`

데이터베이스별 네이티브 쿼리를 직접 실행합니다.

#### Request Body (MongoDB)
```json
{
  "query": {
    "listCollections": 1
  }
}
```

#### Request Body (PostgreSQL)
```json
{
  "query": "SELECT * FROM users WHERE age > 25 LIMIT 10"
}
```

#### Request Body (MySQL)
```json
{
  "query": "SELECT COUNT(*) as total FROM users WHERE status = 'active'"
}
```

#### Request Body (Cassandra)
```json
{
  "query": "SELECT * FROM users WHERE id = 'user123'"
}
```

#### Request Body (Elasticsearch)
```json
{
  "query": {
    "query": {
      "match": {
        "name": "john"
      }
    }
  }
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "results": [ ... ],
    "execution_time_ms": 15
  }
}
```

---

### 31. Raw Query (타입 지정)

**POST** `/query/raw/typed`

Raw query를 실행하고 결과를 특정 타입으로 반환합니다.

#### Request Body
```json
{
  "query": "SELECT id, name, email FROM users WHERE age > 25",
  "result_type": "array"
}
```

**result_type**:
- `array` - 배열로 반환
- `object` - 단일 객체로 반환
- `map` - Map 형식으로 반환

#### Response (200 OK)
```json
{
  "success": true,
  "data": [
    {"id": "1", "name": "John", "email": "john@example.com"},
    {"id": "2", "name": "Jane", "email": "jane@example.com"}
  ]
}
```

---

## 헬스체크 & 모니터링

### 32. 헬스체크 (Health Check)

**GET** `/health`

서비스 전체 상태를 확인합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime_seconds": 86400,
    "databases": {
      "mongodb": "connected",
      "postgresql": "connected",
      "mysql": "connected",
      "cassandra": "connected",
      "elasticsearch": "connected",
      "vitess": "disconnected"
    },
    "cache": {
      "redis": "connected"
    },
    "messaging": {
      "kafka": "connected"
    }
  }
}
```

---

### 33. 데이터베이스 헬스체크

**GET** `/health/database/{db_type}`

특정 데이터베이스의 상태를 확인합니다.

#### Path Parameters
- `db_type`: `mongodb`, `postgresql`, `mysql`, `cassandra`, `elasticsearch`, `vitess`

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "database": "mongodb",
    "status": "healthy",
    "latency_ms": 5,
    "connections": {
      "active": 10,
      "idle": 5,
      "max": 100
    }
  }
}
```

---

### 34. 메트릭 조회

**GET** `/metrics`

Prometheus 형식의 메트릭을 반환합니다.

#### Response (200 OK, text/plain)
```
# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",endpoint="/documents"} 1500
http_requests_total{method="POST",endpoint="/documents"} 500

# HELP db_operations_total Total database operations
# TYPE db_operations_total counter
db_operations_total{database="mongodb",operation="find"} 5000
db_operations_total{database="postgresql",operation="insert"} 1000
```

---

## 추가 엔드포인트

### 35. 데이터베이스 통계

**GET** `/stats/database/{db_type}`

데이터베이스 통계를 조회합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "database": "mongodb",
    "collections_count": 10,
    "total_documents": 150000,
    "total_size_bytes": 5242880,
    "indexes_count": 25,
    "avg_query_time_ms": 12
  }
}
```

---

### 36. 컬렉션 통계

**GET** `/stats/collection/{collection}`

특정 컬렉션의 통계를 조회합니다.

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "collection": "users",
    "document_count": 15000,
    "size_bytes": 524288,
    "avg_document_size_bytes": 35,
    "indexes": [
      {
        "name": "idx_email",
        "size_bytes": 102400
      }
    ]
  }
}
```

---

## 에러 코드 목록

| 코드 | 설명 |
|------|------|
| `DOCUMENT_NOT_FOUND` | 문서를 찾을 수 없음 |
| `COLLECTION_NOT_FOUND` | 컬렉션이 존재하지 않음 |
| `INVALID_REQUEST` | 잘못된 요청 |
| `VALIDATION_ERROR` | 유효성 검증 실패 |
| `OPTIMISTIC_LOCK_ERROR` | 낙관적 잠금 충돌 |
| `TRANSACTION_FAILED` | 트랜잭션 실패 |
| `DATABASE_ERROR` | 데이터베이스 에러 |
| `CONNECTION_ERROR` | 연결 실패 |
| `UNSUPPORTED_OPERATION` | 지원되지 않는 작업 |
| `DUPLICATE_KEY` | 중복 키 에러 |
| `INDEX_ERROR` | 인덱스 작업 실패 |
| `QUERY_TIMEOUT` | 쿼리 타임아웃 |
| `PERMISSION_DENIED` | 권한 없음 |

---

## 요청 예제 모음

### cURL 예제

#### 1. MongoDB에 문서 생성
```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: mongodb" \
  -d '{
    "collection": "users",
    "data": {
      "name": "John Doe",
      "email": "john@example.com",
      "age": 30
    }
  }'
```

#### 2. PostgreSQL에서 문서 조회
```bash
curl -X GET http://localhost:8080/api/v1/documents/users/507f1f77bcf86cd799439011 \
  -H "X-Database-Type: postgresql"
```

#### 3. MySQL에 여러 문서 생성
```bash
curl -X POST http://localhost:8080/api/v1/documents/bulk/insert \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: mysql" \
  -d '{
    "collection": "products",
    "documents": [
      {"data": {"name": "Product 1", "price": 100}},
      {"data": {"name": "Product 2", "price": 200}}
    ]
  }'
```

#### 4. Cassandra에서 검색
```bash
curl -X POST http://localhost:8080/api/v1/documents/orders/search \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: cassandra" \
  -d '{
    "filter": {"status": "completed"},
    "limit": 10
  }'
```

#### 5. Elasticsearch로 집계
```bash
curl -X POST http://localhost:8080/api/v1/documents/logs/aggregate \
  -H "Content-Type: application/json" \
  -H "X-Database-Type: elasticsearch" \
  -d '{
    "pipeline": [
      {"$match": {"level": "error"}},
      {"$group": {"_id": "$service", "count": {"$sum": 1}}}
    ]
  }'
```

---

## 페이징 전략

### Offset-based Pagination (기본)
```http
GET /api/v1/documents/users?limit=10&offset=20
```

### Cursor-based Pagination (선택)
```http
GET /api/v1/documents/users?limit=10&cursor=eyJpZCI6IjUwN2YxZjc3In0=
```

응답에 `next_cursor` 포함:
```json
{
  "data": { ... },
  "pagination": {
    "next_cursor": "eyJpZCI6IjYwOGYyZjg4In0=",
    "has_more": true
  }
}
```

---

## Rate Limiting

API Rate Limit: **1000 requests/minute**

응답 헤더:
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1699876543
```

Rate limit 초과 시:
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests",
    "retry_after_seconds": 30
  }
}
```

---

## 전체 엔드포인트 요약

| # | Method | Endpoint | 설명 |
|---|--------|----------|------|
| 1 | POST | `/documents` | 문서 생성 |
| 2 | GET | `/documents/{collection}/{id}` | 문서 조회 |
| 3 | PUT | `/documents/{collection}/{id}` | 문서 업데이트 |
| 4 | PUT | `/documents/{collection}/{id}/replace` | 문서 교체 |
| 5 | DELETE | `/documents/{collection}/{id}` | 문서 삭제 |
| 6 | GET | `/documents/{collection}` | 문서 목록 조회 |
| 7 | POST | `/documents/{collection}/search` | 문서 검색 |
| 8 | POST | `/documents/{collection}/count` | 문서 개수 |
| 9 | GET | `/documents/{collection}/count/estimate` | 예상 문서 개수 |
| 10 | POST | `/documents/{collection}/{id}/find-and-update` | 찾아서 업데이트 |
| 11 | POST | `/documents/{collection}/{id}/find-and-replace` | 찾아서 교체 |
| 12 | POST | `/documents/{collection}/{id}/find-and-delete` | 찾아서 삭제 |
| 13 | POST | `/documents/{collection}/upsert` | Upsert |
| 14 | POST | `/documents/{collection}/aggregate` | 집계 |
| 15 | POST | `/documents/{collection}/distinct` | 고유 값 조회 |
| 16 | POST | `/documents/bulk/insert` | 벌크 삽입 |
| 17 | POST | `/documents/{collection}/update-many` | 여러 문서 업데이트 |
| 18 | POST | `/documents/{collection}/delete-many` | 여러 문서 삭제 |
| 19 | POST | `/documents/bulk/write` | 벌크 쓰기 |
| 20 | POST | `/indexes/{collection}` | 인덱스 생성 |
| 21 | POST | `/indexes/{collection}/bulk` | 여러 인덱스 생성 |
| 22 | DELETE | `/indexes/{collection}/{index_name}` | 인덱스 삭제 |
| 23 | GET | `/indexes/{collection}` | 인덱스 목록 |
| 24 | POST | `/collections` | 컬렉션 생성 |
| 25 | DELETE | `/collections/{collection}` | 컬렉션 삭제 |
| 26 | POST | `/collections/{old_name}/rename` | 컬렉션 이름 변경 |
| 27 | GET | `/collections` | 컬렉션 목록 |
| 28 | GET | `/collections/{collection}/exists` | 컬렉션 존재 확인 |
| 29 | POST | `/transactions/execute` | 트랜잭션 실행 |
| 30 | POST | `/query/raw` | Raw Query 실행 |
| 31 | POST | `/query/raw/typed` | Raw Query (타입 지정) |
| 32 | GET | `/health` | 전체 헬스체크 |
| 33 | GET | `/health/database/{db_type}` | DB 헬스체크 |
| 34 | GET | `/metrics` | 메트릭 조회 |
| 35 | GET | `/stats/database/{db_type}` | DB 통계 |
| 36 | GET | `/stats/collection/{collection}` | 컬렉션 통계 |

**총 36개 엔드포인트**

---

## 다음 단계

이 명세서를 바탕으로 다음을 구현합니다:

1. ✅ REST API 핸들러 구현
2. ✅ 요청/응답 DTO 정의
3. ✅ 라우터 설정
4. ✅ 에러 핸들링
5. ✅ 유효성 검증
6. ✅ 미들웨어 (로깅, 메트릭, Rate Limiting)
7. ✅ API 문서 생성 (Swagger/OpenAPI)

모든 엔드포인트는 6개 데이터베이스를 지원하며, `X-Database-Type` 헤더로 선택합니다!
