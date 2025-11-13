# Client Integration Guide

다른 서비스에서 Database Service를 이용하기 위한 통합 가이드입니다.

## 📋 목차

1. [개요](#개요)
2. [기본 설정](#기본-설정)
3. [언어별 클라이언트 예제](#언어별-클라이언트-예제)
4. [인증 및 보안](#인증-및-보안)
5. [에러 핸들링](#에러-핸들링)
6. [Connection 관리](#connection-관리)
7. [베스트 프랙티스](#베스트-프랙티스)
8. [마이크로서비스 통합 패턴](#마이크로서비스-통합-패턴)

---

## 개요

### 서비스 정보

- **Base URL**: `http://database-service:8080/api/v1`
- **프로토콜**: REST API (HTTP/HTTPS), gRPC (9090)
- **인증**: Optional (현재는 인증 없음, 필요시 API Gateway에서 처리)
- **데이터 포맷**: JSON

### 지원 데이터베이스

다음 6개 데이터베이스를 지원하며, `X-Database-Type` 헤더로 선택합니다:

| Database | X-Database-Type | 설명 |
|----------|-----------------|------|
| MongoDB | `mongodb` | NoSQL 문서 데이터베이스 (기본값) |
| PostgreSQL | `postgresql` | 관계형 DB (JSONB 지원) |
| MySQL | `mysql` | 관계형 DB (JSON 지원) |
| Cassandra | `cassandra` | 분산 NoSQL |
| Elasticsearch | `elasticsearch` | 검색 엔진 |
| Vitess | `vitess` | MySQL 호환 분산 DB |

---

## 기본 설정

### 필수 헤더

모든 요청에 다음 헤더를 포함해야 합니다:

```http
Content-Type: application/json
X-Database-Type: mongodb
```

### 선택 헤더

```http
X-Request-ID: unique-request-id         # 요청 추적용 (권장)
X-Trace-ID: trace-id                    # 분산 추적용 (선택)
```

### 엔드포인트 구조

```
/api/v1/documents                       # 문서 생성, 목록 조회
/api/v1/documents/{collection}/{id}     # 문서 조회, 업데이트, 삭제
/api/v1/documents/{collection}/search   # 문서 검색
/api/v1/documents/{collection}/count    # 문서 개수
/health                                 # 헬스체크
```

---

## 언어별 클라이언트 예제

### 1. Go 클라이언트

#### 기본 클라이언트 구조

```go
package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// DatabaseServiceClient는 Database Service 클라이언트입니다
type DatabaseServiceClient struct {
    baseURL    string
    httpClient *http.Client
    dbType     string
}

// NewClient는 새로운 클라이언트를 생성합니다
func NewClient(baseURL string, dbType string) *DatabaseServiceClient {
    return &DatabaseServiceClient{
        baseURL: baseURL,
        dbType:  dbType,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
    }
}

// Document는 문서 구조입니다
type Document struct {
    ID         string                 `json:"id,omitempty"`
    Collection string                 `json:"collection"`
    Data       map[string]interface{} `json:"data"`
    Version    int                    `json:"version,omitempty"`
    CreatedAt  time.Time              `json:"created_at,omitempty"`
    UpdatedAt  time.Time              `json:"updated_at,omitempty"`
}

// CreateDocumentRequest는 문서 생성 요청입니다
type CreateDocumentRequest struct {
    Collection string                 `json:"collection"`
    Data       map[string]interface{} `json:"data"`
}

// CreateDocumentResponse는 문서 생성 응답입니다
type CreateDocumentResponse struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
}

// CreateDocument는 문서를 생성합니다
func (c *DatabaseServiceClient) CreateDocument(ctx context.Context, req *CreateDocumentRequest) (*CreateDocumentResponse, error) {
    url := fmt.Sprintf("%s/api/v1/documents", c.baseURL)

    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("X-Database-Type", c.dbType)

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
    }

    var result CreateDocumentResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &result, nil
}

// GetDocument는 문서를 조회합니다
func (c *DatabaseServiceClient) GetDocument(ctx context.Context, collection, id string) (*Document, error) {
    url := fmt.Sprintf("%s/api/v1/documents/%s/%s", c.baseURL, collection, id)

    httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("X-Database-Type", c.dbType)

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        if resp.StatusCode == http.StatusNotFound {
            return nil, fmt.Errorf("document not found")
        }
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
    }

    var doc Document
    if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &doc, nil
}

// UpdateDocument는 문서를 업데이트합니다
func (c *DatabaseServiceClient) UpdateDocument(ctx context.Context, collection, id string, data map[string]interface{}, version int) error {
    url := fmt.Sprintf("%s/api/v1/documents/%s/%s", c.baseURL, collection, id)

    reqBody := map[string]interface{}{
        "data":    data,
        "version": version,
    }

    body, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("X-Database-Type", c.dbType)

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
    }

    return nil
}

// DeleteDocument는 문서를 삭제합니다
func (c *DatabaseServiceClient) DeleteDocument(ctx context.Context, collection, id string) error {
    url := fmt.Sprintf("%s/api/v1/documents/%s/%s", c.baseURL, collection, id)

    httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("X-Database-Type", c.dbType)

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
    }

    return nil
}
```

#### 사용 예제

```go
package main

import (
    "context"
    "log"
    "time"
)

func main() {
    // 클라이언트 생성
    client := NewClient("http://database-service:8080", "mongodb")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 문서 생성
    createReq := &CreateDocumentRequest{
        Collection: "users",
        Data: map[string]interface{}{
            "name":  "John Doe",
            "email": "john@example.com",
            "age":   30,
        },
    }

    createResp, err := client.CreateDocument(ctx, createReq)
    if err != nil {
        log.Fatalf("Failed to create document: %v", err)
    }

    log.Printf("Document created with ID: %s", createResp.ID)

    // 문서 조회
    doc, err := client.GetDocument(ctx, "users", createResp.ID)
    if err != nil {
        log.Fatalf("Failed to get document: %v", err)
    }

    log.Printf("Document retrieved: %+v", doc)

    // 문서 업데이트
    err = client.UpdateDocument(ctx, "users", createResp.ID, map[string]interface{}{
        "age": 31,
    }, doc.Version)
    if err != nil {
        log.Fatalf("Failed to update document: %v", err)
    }

    log.Println("Document updated successfully")
}
```

### 2. Python 클라이언트

```python
import requests
from typing import Dict, Any, Optional
from datetime import datetime
import json

class DatabaseServiceClient:
    """Database Service 클라이언트"""

    def __init__(self, base_url: str, db_type: str = "mongodb", timeout: int = 30):
        """
        클라이언트 초기화

        Args:
            base_url: Database Service URL (예: http://database-service:8080)
            db_type: 데이터베이스 타입 (mongodb, postgresql, mysql 등)
            timeout: 요청 타임아웃 (초)
        """
        self.base_url = base_url.rstrip('/')
        self.db_type = db_type
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json',
            'X-Database-Type': db_type
        })

    def create_document(self, collection: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        문서 생성

        Args:
            collection: 컬렉션 이름
            data: 문서 데이터

        Returns:
            생성된 문서 정보 (id, created_at)

        Raises:
            requests.HTTPError: HTTP 에러 발생 시
        """
        url = f"{self.base_url}/api/v1/documents"
        payload = {
            "collection": collection,
            "data": data
        }

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()

        return response.json()

    def get_document(self, collection: str, doc_id: str) -> Dict[str, Any]:
        """
        문서 조회

        Args:
            collection: 컬렉션 이름
            doc_id: 문서 ID

        Returns:
            문서 데이터

        Raises:
            requests.HTTPError: HTTP 에러 발생 시
        """
        url = f"{self.base_url}/api/v1/documents/{collection}/{doc_id}"

        response = self.session.get(url, timeout=self.timeout)
        response.raise_for_status()

        return response.json()

    def update_document(self, collection: str, doc_id: str, data: Dict[str, Any], version: int) -> None:
        """
        문서 업데이트

        Args:
            collection: 컬렉션 이름
            doc_id: 문서 ID
            data: 업데이트할 데이터
            version: 문서 버전 (낙관적 잠금)

        Raises:
            requests.HTTPError: HTTP 에러 발생 시
        """
        url = f"{self.base_url}/api/v1/documents/{collection}/{doc_id}"
        payload = {
            "data": data,
            "version": version
        }

        response = self.session.put(url, json=payload, timeout=self.timeout)
        response.raise_for_status()

    def delete_document(self, collection: str, doc_id: str) -> None:
        """
        문서 삭제

        Args:
            collection: 컬렉션 이름
            doc_id: 문서 ID

        Raises:
            requests.HTTPError: HTTP 에러 발생 시
        """
        url = f"{self.base_url}/api/v1/documents/{collection}/{doc_id}"

        response = self.session.delete(url, timeout=self.timeout)
        response.raise_for_status()

    def search_documents(self, collection: str, filter_query: Dict[str, Any],
                        limit: int = 10, offset: int = 0,
                        sort: Optional[Dict[str, int]] = None) -> Dict[str, Any]:
        """
        문서 검색

        Args:
            collection: 컬렉션 이름
            filter_query: 필터 쿼리
            limit: 최대 결과 개수
            offset: 시작 위치
            sort: 정렬 조건

        Returns:
            검색 결과 (documents, total)
        """
        url = f"{self.base_url}/api/v1/documents/{collection}/search"
        payload = {
            "filter": filter_query,
            "limit": limit,
            "offset": offset
        }
        if sort:
            payload["sort"] = sort

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()

        return response.json()

    def count_documents(self, collection: str, filter_query: Optional[Dict[str, Any]] = None) -> int:
        """
        문서 개수 조회

        Args:
            collection: 컬렉션 이름
            filter_query: 필터 쿼리 (선택)

        Returns:
            문서 개수
        """
        url = f"{self.base_url}/api/v1/documents/{collection}/count"
        payload = {}
        if filter_query:
            payload["filter"] = filter_query

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()

        return response.json()["count"]

    def health_check(self) -> Dict[str, Any]:
        """
        헬스체크

        Returns:
            헬스 상태
        """
        url = f"{self.base_url}/health"

        response = self.session.get(url, timeout=self.timeout)
        response.raise_for_status()

        return response.json()

    def close(self):
        """세션 종료"""
        self.session.close()


# 사용 예제
if __name__ == "__main__":
    # 클라이언트 생성
    client = DatabaseServiceClient("http://database-service:8080", db_type="mongodb")

    try:
        # 문서 생성
        result = client.create_document("users", {
            "name": "John Doe",
            "email": "john@example.com",
            "age": 30
        })
        doc_id = result["id"]
        print(f"Document created with ID: {doc_id}")

        # 문서 조회
        doc = client.get_document("users", doc_id)
        print(f"Document retrieved: {doc}")

        # 문서 업데이트
        client.update_document("users", doc_id, {"age": 31}, doc["version"])
        print("Document updated successfully")

        # 문서 검색
        results = client.search_documents("users", {"age": {"$gte": 30}}, limit=10)
        print(f"Found {results['total']} documents")

        # 문서 개수
        count = client.count_documents("users", {"age": {"$gte": 30}})
        print(f"Total count: {count}")

        # 문서 삭제
        client.delete_document("users", doc_id)
        print("Document deleted successfully")

    finally:
        client.close()
```

### 3. Node.js 클라이언트

```javascript
const axios = require('axios');

class DatabaseServiceClient {
  /**
   * Database Service 클라이언트
   * @param {string} baseURL - Database Service URL
   * @param {string} dbType - 데이터베이스 타입
   * @param {number} timeout - 타임아웃 (ms)
   */
  constructor(baseURL, dbType = 'mongodb', timeout = 30000) {
    this.client = axios.create({
      baseURL: baseURL,
      timeout: timeout,
      headers: {
        'Content-Type': 'application/json',
        'X-Database-Type': dbType
      }
    });
  }

  /**
   * 문서 생성
   * @param {string} collection - 컬렉션 이름
   * @param {object} data - 문서 데이터
   * @returns {Promise<object>} 생성된 문서 정보
   */
  async createDocument(collection, data) {
    const response = await this.client.post('/api/v1/documents', {
      collection,
      data
    });
    return response.data;
  }

  /**
   * 문서 조회
   * @param {string} collection - 컬렉션 이름
   * @param {string} id - 문서 ID
   * @returns {Promise<object>} 문서 데이터
   */
  async getDocument(collection, id) {
    const response = await this.client.get(`/api/v1/documents/${collection}/${id}`);
    return response.data;
  }

  /**
   * 문서 업데이트
   * @param {string} collection - 컬렉션 이름
   * @param {string} id - 문서 ID
   * @param {object} data - 업데이트할 데이터
   * @param {number} version - 문서 버전
   */
  async updateDocument(collection, id, data, version) {
    await this.client.put(`/api/v1/documents/${collection}/${id}`, {
      data,
      version
    });
  }

  /**
   * 문서 삭제
   * @param {string} collection - 컬렉션 이름
   * @param {string} id - 문서 ID
   */
  async deleteDocument(collection, id) {
    await this.client.delete(`/api/v1/documents/${collection}/${id}`);
  }

  /**
   * 문서 검색
   * @param {string} collection - 컬렉션 이름
   * @param {object} filter - 필터 쿼리
   * @param {number} limit - 최대 결과 개수
   * @param {number} offset - 시작 위치
   * @param {object} sort - 정렬 조건
   * @returns {Promise<object>} 검색 결과
   */
  async searchDocuments(collection, filter, limit = 10, offset = 0, sort = null) {
    const payload = { filter, limit, offset };
    if (sort) {
      payload.sort = sort;
    }

    const response = await this.client.post(`/api/v1/documents/${collection}/search`, payload);
    return response.data;
  }

  /**
   * 문서 개수 조회
   * @param {string} collection - 컬렉션 이름
   * @param {object} filter - 필터 쿼리
   * @returns {Promise<number>} 문서 개수
   */
  async countDocuments(collection, filter = null) {
    const payload = filter ? { filter } : {};
    const response = await this.client.post(`/api/v1/documents/${collection}/count`, payload);
    return response.data.count;
  }

  /**
   * 헬스체크
   * @returns {Promise<object>} 헬스 상태
   */
  async healthCheck() {
    const response = await this.client.get('/health');
    return response.data;
  }
}

// 사용 예제
async function main() {
  const client = new DatabaseServiceClient('http://database-service:8080', 'mongodb');

  try {
    // 문서 생성
    const createResult = await client.createDocument('users', {
      name: 'John Doe',
      email: 'john@example.com',
      age: 30
    });
    const docId = createResult.id;
    console.log(`Document created with ID: ${docId}`);

    // 문서 조회
    const doc = await client.getDocument('users', docId);
    console.log('Document retrieved:', doc);

    // 문서 업데이트
    await client.updateDocument('users', docId, { age: 31 }, doc.version);
    console.log('Document updated successfully');

    // 문서 검색
    const searchResults = await client.searchDocuments('users', { age: { $gte: 30 } }, 10);
    console.log(`Found ${searchResults.total} documents`);

    // 문서 개수
    const count = await client.countDocuments('users', { age: { $gte: 30 } });
    console.log(`Total count: ${count}`);

    // 문서 삭제
    await client.deleteDocument('users', docId);
    console.log('Document deleted successfully');

  } catch (error) {
    console.error('Error:', error.message);
    if (error.response) {
      console.error('Response data:', error.response.data);
    }
  }
}

module.exports = DatabaseServiceClient;

// 실행
if (require.main === module) {
  main();
}
```

### 4. Java 클라이언트

```java
package com.example.client;

import com.fasterxml.jackson.databind.ObjectMapper;
import okhttp3.*;
import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class DatabaseServiceClient {
    private final String baseUrl;
    private final String dbType;
    private final OkHttpClient httpClient;
    private final ObjectMapper objectMapper;
    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");

    /**
     * Database Service 클라이언트
     *
     * @param baseUrl Database Service URL
     * @param dbType 데이터베이스 타입
     */
    public DatabaseServiceClient(String baseUrl, String dbType) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.dbType = dbType;
        this.objectMapper = new ObjectMapper();
        this.httpClient = new OkHttpClient.Builder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .writeTimeout(10, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .build();
    }

    /**
     * 문서 생성
     */
    public CreateDocumentResponse createDocument(String collection, Map<String, Object> data) throws IOException {
        String url = baseUrl + "/api/v1/documents";

        Map<String, Object> payload = new HashMap<>();
        payload.put("collection", collection);
        payload.put("data", data);

        String json = objectMapper.writeValueAsString(payload);
        RequestBody body = RequestBody.create(json, JSON);

        Request request = new Request.Builder()
                .url(url)
                .post(body)
                .header("Content-Type", "application/json")
                .header("X-Database-Type", dbType)
                .build();

        try (Response response = httpClient.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                throw new IOException("Unexpected code " + response + ", body: " + response.body().string());
            }

            return objectMapper.readValue(response.body().string(), CreateDocumentResponse.class);
        }
    }

    /**
     * 문서 조회
     */
    public DocumentResponse getDocument(String collection, String id) throws IOException {
        String url = String.format("%s/api/v1/documents/%s/%s", baseUrl, collection, id);

        Request request = new Request.Builder()
                .url(url)
                .get()
                .header("X-Database-Type", dbType)
                .build();

        try (Response response = httpClient.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                if (response.code() == 404) {
                    throw new IOException("Document not found");
                }
                throw new IOException("Unexpected code " + response + ", body: " + response.body().string());
            }

            return objectMapper.readValue(response.body().string(), DocumentResponse.class);
        }
    }

    /**
     * 문서 업데이트
     */
    public void updateDocument(String collection, String id, Map<String, Object> data, int version) throws IOException {
        String url = String.format("%s/api/v1/documents/%s/%s", baseUrl, collection, id);

        Map<String, Object> payload = new HashMap<>();
        payload.put("data", data);
        payload.put("version", version);

        String json = objectMapper.writeValueAsString(payload);
        RequestBody body = RequestBody.create(json, JSON);

        Request request = new Request.Builder()
                .url(url)
                .put(body)
                .header("Content-Type", "application/json")
                .header("X-Database-Type", dbType)
                .build();

        try (Response response = httpClient.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                throw new IOException("Unexpected code " + response + ", body: " + response.body().string());
            }
        }
    }

    /**
     * 문서 삭제
     */
    public void deleteDocument(String collection, String id) throws IOException {
        String url = String.format("%s/api/v1/documents/%s/%s", baseUrl, collection, id);

        Request request = new Request.Builder()
                .url(url)
                .delete()
                .header("X-Database-Type", dbType)
                .build();

        try (Response response = httpClient.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                throw new IOException("Unexpected code " + response + ", body: " + response.body().string());
            }
        }
    }

    // DTO classes
    public static class CreateDocumentResponse {
        public String id;
        public String createdAt;
    }

    public static class DocumentResponse {
        public String id;
        public String collection;
        public Map<String, Object> data;
        public int version;
        public String createdAt;
        public String updatedAt;
    }
}

// 사용 예제
public class Main {
    public static void main(String[] args) {
        DatabaseServiceClient client = new DatabaseServiceClient(
            "http://database-service:8080",
            "mongodb"
        );

        try {
            // 문서 생성
            Map<String, Object> data = new HashMap<>();
            data.put("name", "John Doe");
            data.put("email", "john@example.com");
            data.put("age", 30);

            var createResult = client.createDocument("users", data);
            System.out.println("Document created with ID: " + createResult.id);

            // 문서 조회
            var doc = client.getDocument("users", createResult.id);
            System.out.println("Document retrieved: " + doc.data);

            // 문서 업데이트
            Map<String, Object> updateData = new HashMap<>();
            updateData.put("age", 31);
            client.updateDocument("users", createResult.id, updateData, doc.version);
            System.out.println("Document updated successfully");

            // 문서 삭제
            client.deleteDocument("users", createResult.id);
            System.out.println("Document deleted successfully");

        } catch (IOException e) {
            e.printStackTrace();
        }
    }
}
```

---

## 인증 및 보안

### 현재 상태

현재 Database Service는 **인증이 없는 상태**입니다. 프로덕션 환경에서는 다음 중 하나를 구현해야 합니다.

### 권장 인증 방식

#### 1. API Gateway 레벨 인증 (권장)

```
[클라이언트] → [API Gateway (인증)] → [Database Service]
```

- Kong, Ambassador, Istio 등의 API Gateway 사용
- JWT 토큰 검증
- Rate Limiting
- IP 화이트리스트

#### 2. Service Mesh 레벨 인증

```yaml
# Istio PeerAuthentication 예제
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: database-service-mtls
spec:
  mtls:
    mode: STRICT
```

#### 3. API 키 기반 인증 (간단한 방식)

```go
// Go 예제
req.Header.Set("X-API-Key", "your-api-key")
```

```python
# Python 예제
headers = {
    "X-API-Key": "your-api-key",
    "X-Database-Type": "mongodb"
}
```

### TLS/HTTPS 사용

프로덕션에서는 반드시 HTTPS를 사용하세요:

```go
// Go: TLS 설정
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
}
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: tlsConfig,
    },
}
```

---

## 에러 핸들링

### 표준 에러 응답 형식

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

### HTTP 상태 코드

| 코드 | 의미 | 처리 방법 |
|------|------|----------|
| 200 | 성공 | 정상 처리 |
| 201 | 생성 성공 | 정상 처리 |
| 400 | 잘못된 요청 | 요청 데이터 확인 |
| 404 | 리소스 없음 | 존재 확인 후 재시도 |
| 409 | 충돌 (버전 불일치) | 최신 버전 가져와서 재시도 |
| 429 | Rate Limit 초과 | Exponential backoff으로 재시도 |
| 500 | 서버 에러 | 재시도 (최대 3회) |
| 503 | 서비스 불가 | Circuit breaker 열림, 재시도 대기 |

### Go 에러 핸들링 예제

```go
func (c *DatabaseServiceClient) CreateDocumentWithRetry(ctx context.Context, req *CreateDocumentRequest, maxRetries int) (*CreateDocumentResponse, error) {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        resp, err := c.CreateDocument(ctx, req)
        if err == nil {
            return resp, nil
        }

        lastErr = err

        // HTTP 상태 코드에 따라 재시도 결정
        if httpErr, ok := err.(*HTTPError); ok {
            switch httpErr.StatusCode {
            case 400, 401, 403, 404:
                // 재시도 불가능한 에러
                return nil, err
            case 429:
                // Rate limit - exponential backoff
                backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
                time.Sleep(backoff)
                continue
            case 500, 502, 503, 504:
                // 서버 에러 - 재시도
                backoff := time.Duration(i+1) * time.Second
                time.Sleep(backoff)
                continue
            }
        }

        // 기타 네트워크 에러 - 재시도
        time.Sleep(time.Duration(i+1) * time.Second)
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

type HTTPError struct {
    StatusCode int
    Body       string
}

func (e *HTTPError) Error() string {
    return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}
```

### Python 에러 핸들링 예제

```python
import time
from requests.exceptions import RequestException, HTTPError

class RetryableClient(DatabaseServiceClient):
    def create_document_with_retry(self, collection: str, data: dict, max_retries: int = 3):
        """재시도 로직이 포함된 문서 생성"""
        for i in range(max_retries):
            try:
                return self.create_document(collection, data)
            except HTTPError as e:
                status_code = e.response.status_code

                # 재시도 불가능한 에러
                if status_code in [400, 401, 403, 404]:
                    raise

                # Rate limit - exponential backoff
                if status_code == 429:
                    backoff = 2 ** i
                    time.sleep(backoff)
                    continue

                # 서버 에러 - 재시도
                if status_code >= 500:
                    backoff = i + 1
                    time.sleep(backoff)
                    continue

                raise
            except RequestException as e:
                # 네트워크 에러 - 재시도
                if i < max_retries - 1:
                    time.sleep(i + 1)
                    continue
                raise

        raise Exception(f"Max retries ({max_retries}) exceeded")
```

---

## Connection 관리

### Connection Pooling

#### Go

```go
// HTTP 클라이언트 Connection Pool 설정
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,              // 전체 최대 idle 연결
        MaxIdleConnsPerHost: 10,               // 호스트당 최대 idle 연결
        MaxConnsPerHost:     50,               // 호스트당 최대 연결
        IdleConnTimeout:     90 * time.Second, // Idle 타임아웃
        TLSHandshakeTimeout: 10 * time.Second,
        DisableKeepAlives:   false,            // Keep-Alive 활성화
    },
}
```

#### Python

```python
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

# Connection Pool 설정
session = requests.Session()
adapter = HTTPAdapter(
    pool_connections=10,    # 연결 풀 개수
    pool_maxsize=20,        # 풀당 최대 연결 수
    max_retries=Retry(
        total=3,
        backoff_factor=0.3,
        status_forcelist=[500, 502, 503, 504]
    )
)
session.mount('http://', adapter)
session.mount('https://', adapter)
```

#### Node.js

```javascript
const axios = require('axios');
const http = require('http');
const https = require('https');

// Connection Pool 설정
const httpAgent = new http.Agent({
  keepAlive: true,
  maxSockets: 50,
  maxFreeSockets: 10,
  timeout: 60000,
});

const httpsAgent = new https.Agent({
  keepAlive: true,
  maxSockets: 50,
  maxFreeSockets: 10,
  timeout: 60000,
});

const client = axios.create({
  httpAgent: httpAgent,
  httpsAgent: httpsAgent,
  timeout: 30000,
});
```

### 타임아웃 설정

```go
// Go: Context 기반 타임아웃
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.CreateDocument(ctx, req)
```

```python
# Python: 타임아웃 설정
response = session.post(url, json=data, timeout=(5, 30))  # (연결, 읽기)
```

```javascript
// Node.js: 타임아웃 설정
const response = await axios.post(url, data, { timeout: 30000 });
```

---

## 베스트 프랙티스

### 1. Circuit Breaker 패턴

서비스 장애 시 Circuit Breaker를 사용하여 연쇄 장애 방지:

```go
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    failures     int
    lastFailTime time.Time
    state        string // "closed", "open", "half-open"
    mu           sync.Mutex
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    // Circuit이 열려있으면 즉시 실패
    if cb.state == "open" {
        if time.Since(cb.lastFailTime) > cb.resetTimeout {
            cb.state = "half-open"
        } else {
            return errors.New("circuit breaker is open")
        }
    }

    err := fn()

    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()

        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        return err
    }

    // 성공 시 리셋
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

### 2. Retry with Exponential Backoff

```go
func ExponentialBackoff(ctx context.Context, maxRetries int, fn func() error) error {
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }

        if i < maxRetries-1 {
            backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
            select {
            case <-time.After(backoff):
                continue
            case <-ctx.Done():
                return ctx.Err()
            }
        }

        return err
    }
    return errors.New("max retries exceeded")
}
```

### 3. Request ID 추적

```go
// 요청마다 고유 ID 생성
requestID := uuid.New().String()
req.Header.Set("X-Request-ID", requestID)

// 로그에 Request ID 포함
log.Printf("[%s] Creating document in collection: %s", requestID, collection)
```

### 4. 낙관적 잠금 (Optimistic Locking)

버전 충돌 시 재시도:

```go
func UpdateDocumentWithOptimisticLocking(ctx context.Context, client *DatabaseServiceClient,
    collection, id string, updateFn func(data map[string]interface{}) map[string]interface{}) error {

    maxRetries := 3

    for i := 0; i < maxRetries; i++ {
        // 최신 문서 가져오기
        doc, err := client.GetDocument(ctx, collection, id)
        if err != nil {
            return err
        }

        // 업데이트할 데이터 계산
        newData := updateFn(doc.Data)

        // 업데이트 시도
        err = client.UpdateDocument(ctx, collection, id, newData, doc.Version)
        if err == nil {
            return nil
        }

        // 409 Conflict (버전 충돌)이면 재시도
        if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == 409 {
            time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
            continue
        }

        return err
    }

    return errors.New("optimistic locking failed after retries")
}
```

### 5. 배치 처리

여러 문서를 한 번에 처리:

```go
// BulkInsert 사용
documents := []map[string]interface{}{
    {"name": "User1", "age": 25},
    {"name": "User2", "age": 30},
    {"name": "User3", "age": 35},
}

result, err := client.BulkInsert(ctx, "users", documents)
```

### 6. 캐싱 전략

```go
type CachedClient struct {
    client *DatabaseServiceClient
    cache  *cache.Cache
}

func (c *CachedClient) GetDocument(ctx context.Context, collection, id string) (*Document, error) {
    // 캐시 확인
    cacheKey := fmt.Sprintf("%s:%s", collection, id)
    if cached, found := c.cache.Get(cacheKey); found {
        return cached.(*Document), nil
    }

    // DB 조회
    doc, err := c.client.GetDocument(ctx, collection, id)
    if err != nil {
        return nil, err
    }

    // 캐시 저장 (5분)
    c.cache.Set(cacheKey, doc, 5*time.Minute)

    return doc, nil
}
```

---

## 마이크로서비스 통합 패턴

### 1. Kubernetes 환경에서의 서비스 디스커버리

```yaml
# Kubernetes Service
apiVersion: v1
kind: Service
metadata:
  name: database-service
spec:
  selector:
    app: database-service
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: grpc
    port: 9090
    targetPort: 9090
```

```go
// Go 클라이언트: Kubernetes DNS 사용
client := NewClient("http://database-service.default.svc.cluster.local:8080", "mongodb")
```

### 2. Service Mesh (Istio) 통합

```yaml
# VirtualService: 트래픽 라우팅
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: database-service
spec:
  hosts:
  - database-service
  http:
  - match:
    - headers:
        x-database-type:
          exact: mongodb
    route:
    - destination:
        host: database-service
        subset: v1
      weight: 100
  - route:
    - destination:
        host: database-service
        subset: v1
```

### 3. 분산 추적 (Distributed Tracing)

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// Trace context 전파
ctx, span := tracer.Start(ctx, "create-document")
defer span.End()

// HTTP 헤더에 trace context 추가
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

resp, err := client.CreateDocument(ctx, createReq)
```

### 4. 이벤트 기반 통합 (Kafka)

Database Service의 CDC 이벤트 구독:

```go
import "github.com/confluentinc/confluent-kafka-go/kafka"

// Kafka Consumer 설정
consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
    "bootstrap.servers": "kafka:9092",
    "group.id":          "my-service",
    "auto.offset.reset": "earliest",
})

// 문서 생성 이벤트 구독
consumer.Subscribe("documents.created", nil)

for {
    msg, err := consumer.ReadMessage(-1)
    if err == nil {
        // 이벤트 처리
        var event DocumentCreatedEvent
        json.Unmarshal(msg.Value, &event)

        log.Printf("Document created: %s in %s", event.DocumentID, event.Collection)
    }
}
```

### 5. 헬스체크 및 Readiness Probe

```go
// 헬스체크 주기적 실행
func HealthCheckLoop(client *DatabaseServiceClient, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

        health, err := client.HealthCheck(ctx)
        if err != nil {
            log.Printf("Health check failed: %v", err)
        } else {
            log.Printf("Health check passed: %+v", health)
        }

        cancel()
    }
}
```

```yaml
# Kubernetes Readiness Probe
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
```

### 6. 멀티 데이터베이스 사용 예제

```go
// 시나리오: 사용자 데이터는 MongoDB, 로그는 Elasticsearch에 저장

// MongoDB 클라이언트
mongoClient := NewClient("http://database-service:8080", "mongodb")

// Elasticsearch 클라이언트
esClient := NewClient("http://database-service:8080", "elasticsearch")

// 사용자 생성 (MongoDB)
user, err := mongoClient.CreateDocument(ctx, &CreateDocumentRequest{
    Collection: "users",
    Data: map[string]interface{}{
        "name": "John Doe",
        "email": "john@example.com",
    },
})

// 로그 저장 (Elasticsearch)
_, err = esClient.CreateDocument(ctx, &CreateDocumentRequest{
    Collection: "user_logs",
    Data: map[string]interface{}{
        "user_id": user.ID,
        "action": "user_created",
        "timestamp": time.Now(),
    },
})
```

---

## 성능 최적화 팁

### 1. Connection Reuse

```go
// ❌ 나쁜 예: 매번 새 클라이언트 생성
func BadExample() {
    client := NewClient("http://database-service:8080", "mongodb")
    client.CreateDocument(ctx, req)
    // 연결 재사용 안됨
}

// ✅ 좋은 예: 클라이언트 재사용
var globalClient = NewClient("http://database-service:8080", "mongodb")

func GoodExample() {
    globalClient.CreateDocument(ctx, req)
    // 연결 재사용
}
```

### 2. 병렬 처리

```go
// 여러 문서를 병렬로 생성
func CreateDocumentsConcurrently(ctx context.Context, client *DatabaseServiceClient, docs []CreateDocumentRequest) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, doc := range docs {
        doc := doc // 클로저 변수 캡처
        g.Go(func() error {
            _, err := client.CreateDocument(ctx, &doc)
            return err
        })
    }

    return g.Wait()
}
```

### 3. 배치 크기 최적화

```go
// 큰 데이터셋을 배치로 나누어 처리
func ProcessInBatches(items []map[string]interface{}, batchSize int) {
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }

        batch := items[i:end]
        client.BulkInsert(ctx, "collection", batch)
    }
}
```

---

## 문제 해결 (Troubleshooting)

### 연결 실패

```bash
# DNS 확인
nslookup database-service.default.svc.cluster.local

# 네트워크 연결 확인
curl -v http://database-service:8080/health

# Pod 로그 확인
kubectl logs -f deployment/database-service
```

### 성능 문제

```go
// 요청 시간 측정
start := time.Now()
resp, err := client.CreateDocument(ctx, req)
duration := time.Since(start)

if duration > 1*time.Second {
    log.Printf("Slow request detected: %v", duration)
}
```

### 디버깅 모드

```go
// HTTP 요청/응답 로깅
type LoggingTransport struct {
    Transport http.RoundTripper
}

func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    // 요청 로깅
    log.Printf("Request: %s %s", req.Method, req.URL)

    resp, err := t.Transport.RoundTrip(req)

    // 응답 로깅
    if err == nil {
        log.Printf("Response: %d", resp.StatusCode)
    }

    return resp, err
}
```

---

## 추가 리소스

- **API 명세서**: [REST_API_SPECIFICATION.md](./REST_API_SPECIFICATION.md)
- **아키텍처 가이드**: [ARCHITECTURE.md](./ARCHITECTURE.md)
- **빠른 시작**: [QUICKSTART.md](./QUICKSTART.md)
- **GitHub Repository**: https://github.com/YouSangSon/database-service

---

## 지원

질문이나 이슈가 있으면 GitHub Issues에 등록해주세요:
https://github.com/YouSangSon/database-service/issues
