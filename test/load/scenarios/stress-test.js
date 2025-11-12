import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Stress Test: 시스템 한계점을 찾기 위한 테스트
// 점진적으로 부하를 증가시켜 시스템이 언제 실패하는지 확인

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '2m', target: 50 },    // 50 VUs까지 증가
    { duration: '5m', target: 50 },    // 50 VUs 유지
    { duration: '2m', target: 100 },   // 100 VUs까지 증가
    { duration: '5m', target: 100 },   // 100 VUs 유지
    { duration: '2m', target: 200 },   // 200 VUs까지 증가
    { duration: '5m', target: 200 },   // 200 VUs 유지
    { duration: '10m', target: 0 },    // 점진적 감소
  ],
  thresholds: {
    'http_req_failed': ['rate<0.1'],   // 10% 실패율까지 허용
    'http_req_duration': ['p(95)<2000'], // 2초 이하
    'errors': ['rate<0.1'],
  },
};

export default function () {
  const payload = JSON.stringify({
    collection: 'stress_test',
    data: {
      timestamp: Date.now(),
      test_type: 'stress',
      data: Array(100).fill('x').join(''), // 약간 큰 데이터
    },
  });

  const response = http.post(`${BASE_URL}/api/v1/documents`, payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': 'default',
    },
  });

  const success = check(response, {
    'status is 201': (r) => r.status === 201,
    'response time < 2s': (r) => r.timings.duration < 2000,
  });

  errorRate.add(!success);
  sleep(0.5);
}
