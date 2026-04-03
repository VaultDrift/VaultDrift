import http from 'k6/http';
import { check, sleep, fail } from 'k6';
import { Rate } from 'k6/metrics';

/**
 * Stress Test
 *
 * Purpose: Find breaking point of the system
 * Duration: Until failure or timeout
 * Users: Ramp to high load (100-1000 VUs)
 *
 * Run: k6 run stress_test.js
 */

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '2m', target: 50 },    // Ramp to 50 users
    { duration: '5m', target: 50 },    // Stay at 50
    { duration: '2m', target: 100 },   // Ramp to 100
    { duration: '5m', target: 100 },   // Stay at 100
    { duration: '2m', target: 200 },   // Ramp to 200
    { duration: '5m', target: 200 },   // Stay at 200
    { duration: '2m', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% under 1s
    http_req_failed: ['rate<0.1'],     // Less than 10% errors
    errors: ['rate<0.1'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/health`);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  errorRate.add(!success);

  if (!success) {
    console.error(`Failed: status=${res.status}, time=${res.timings.duration}ms`);
  }

  sleep(Math.random() * 2); // Random sleep 0-2s
}

export function setup() {
  console.log('🔥 Stress test starting...');
  console.log(`Target: ${BASE_URL}`);
  console.log('This test will ramp to 200 users to find breaking points');

  const res = http.get(`${BASE_URL}/api/v1/health`);
  if (res.status !== 200) {
    throw new Error(`Server not ready: ${res.status}`);
  }

  return { baseUrl: BASE_URL, startTime: Date.now() };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`✅ Stress test completed in ${duration}s`);
}
