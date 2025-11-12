import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ============================================================================
// 설정 및 메트릭
// ============================================================================

// 환경 변수
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';
const TENANT_ID = __ENV.TENANT_ID || 'default';

// 커스텀 메트릭
const errorRate = new Rate('errors');
const apiDuration = new Trend('api_duration');
const documentCreated = new Counter('documents_created');
const documentFetched = new Counter('documents_fetched');
const documentUpdated = new Counter('documents_updated');
const documentDeleted = new Counter('documents_deleted');

// 테스트 시나리오 설정
export const options = {
  stages: [
    { duration: '1m', target: 20 },   // 1분간 20 VUs로 증가
    { duration: '3m', target: 20 },   // 3분간 20 VUs 유지
    { duration: '1m', target: 50 },   // 1분간 50 VUs로 증가
    { duration: '5m', target: 50 },   // 5분간 50 VUs 유지
    { duration: '2m', target: 0 },    // 2분간 0 VUs로 감소
  ],
  thresholds: {
    // HTTP 요청 성공률 > 95%
    'http_req_failed': ['rate<0.05'],

    // 95th percentile 응답 시간 < 500ms
    'http_req_duration': ['p(95)<500'],

    // 평균 응답 시간 < 200ms
    'api_duration': ['avg<200', 'p(95)<500', 'p(99)<1000'],

    // 에러율 < 5%
    'errors': ['rate<0.05'],
  },
};

// ============================================================================
// 헬퍼 함수
// ============================================================================

// HTTP 헤더 생성
function getHeaders() {
  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': TENANT_ID,
  };

  if (API_KEY) {
    headers['X-API-Key'] = API_KEY;
  }

  return headers;
}

// 랜덤 데이터 생성
function generateRandomDocument() {
  const timestamp = Date.now();
  const randomId = Math.random().toString(36).substring(7);

  return {
    collection: 'test_collection',
    data: {
      name: `Test Document ${randomId}`,
      description: `This is a test document created at ${timestamp}`,
      status: ['active', 'inactive', 'pending'][Math.floor(Math.random() * 3)],
      priority: Math.floor(Math.random() * 10),
      tags: ['test', 'k6', 'load-testing'],
      metadata: {
        created_by: 'k6-load-test',
        test_run_id: __ENV.TEST_RUN_ID || 'default',
        timestamp: timestamp,
      },
    },
  };
}

// 응답 체크
function checkResponse(response, expectedStatus, operationName) {
  const success = check(response, {
    [`${operationName}: status is ${expectedStatus}`]: (r) => r.status === expectedStatus,
    [`${operationName}: response time < 1s`]: (r) => r.timings.duration < 1000,
  });

  errorRate.add(!success);
  apiDuration.add(response.timings.duration);

  return success;
}

// ============================================================================
// 메인 테스트 시나리오
// ============================================================================

export default function () {
  const headers = getHeaders();
  let documentId = '';

  // 1. Health Check
  {
    const response = http.get(`${BASE_URL}/health`, { headers });
    checkResponse(response, 200, 'Health Check');
  }

  sleep(0.5);

  // 2. Create Document
  {
    const payload = JSON.stringify(generateRandomDocument());
    const response = http.post(
      `${BASE_URL}/api/v1/documents`,
      payload,
      { headers }
    );

    if (checkResponse(response, 201, 'Create Document')) {
      const body = JSON.parse(response.body);
      documentId = body.id;
      documentCreated.add(1);
    }
  }

  sleep(0.3);

  // 3. Get Document by ID
  if (documentId) {
    const response = http.get(
      `${BASE_URL}/api/v1/documents/test_collection/${documentId}`,
      { headers }
    );

    if (checkResponse(response, 200, 'Get Document')) {
      documentFetched.add(1);
    }
  }

  sleep(0.3);

  // 4. Update Document
  if (documentId) {
    const updatePayload = JSON.stringify({
      data: {
        name: `Updated Document ${documentId}`,
        status: 'active',
        updated_at: Date.now(),
      },
    });

    const response = http.put(
      `${BASE_URL}/api/v1/documents/test_collection/${documentId}`,
      updatePayload,
      { headers }
    );

    if (checkResponse(response, 200, 'Update Document')) {
      documentUpdated.add(1);
    }
  }

  sleep(0.3);

  // 5. List Documents
  {
    const response = http.get(
      `${BASE_URL}/api/v1/documents/test_collection?limit=10&offset=0`,
      { headers }
    );

    checkResponse(response, 200, 'List Documents');
  }

  sleep(0.5);

  // 6. Aggregate (Complex Query)
  {
    const aggregatePipeline = JSON.stringify({
      collection: 'test_collection',
      pipeline: [
        {
          $match: {
            status: { $in: ['active', 'pending'] },
          },
        },
        {
          $group: {
            _id: '$status',
            count: { $sum: 1 },
          },
        },
      ],
    });

    const response = http.post(
      `${BASE_URL}/api/v1/documents/test_collection/aggregate`,
      aggregatePipeline,
      { headers }
    );

    checkResponse(response, 200, 'Aggregate Query');
  }

  sleep(0.3);

  // 7. Delete Document (10% 확률)
  if (documentId && Math.random() < 0.1) {
    const response = http.del(
      `${BASE_URL}/api/v1/documents/test_collection/${documentId}`,
      null,
      { headers }
    );

    if (checkResponse(response, 200, 'Delete Document')) {
      documentDeleted.add(1);
    }
  }

  sleep(1);
}

// ============================================================================
// 셋업 및 티어다운
// ============================================================================

export function setup() {
  console.log('🚀 Starting load test...');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Tenant ID: ${TENANT_ID}`);
  console.log(`API Key: ${API_KEY ? '***' + API_KEY.slice(-4) : 'Not set'}`);

  // Health check
  const response = http.get(`${BASE_URL}/health`);
  if (response.status !== 200) {
    throw new Error(`Service is not healthy: ${response.status}`);
  }

  return { startTime: Date.now() };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`✅ Load test completed in ${duration.toFixed(2)}s`);
  console.log(`📊 Documents created: ${documentCreated.value}`);
  console.log(`📊 Documents fetched: ${documentFetched.value}`);
  console.log(`📊 Documents updated: ${documentUpdated.value}`);
  console.log(`📊 Documents deleted: ${documentDeleted.value}`);
}
