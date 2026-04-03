import http from 'k6/http';
import { check, sleep } from 'k6';

/**
 * Smoke Test
 *
 * Purpose: Verify system works with minimal load
 * Duration: Short (1-2 minutes)
 * Users: Very low (1-5 VUs)
 *
 * Run: k6 run smoke_test.js
 */

export const options = {
  vus: 3,              // 3 virtual users
  duration: '1m',      // 1 minute
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% under 200ms
    http_req_failed: ['rate<0.01'],   // Less than 1% errors
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Health check
  const healthRes = http.get(`${BASE_URL}/api/v1/health`);
  check(healthRes, {
    'health is 200': (r) => r.status === 200,
    'health returns ok': (r) => r.json('status') === 'healthy',
  });

  // Metrics endpoint (if enabled)
  const metricsRes = http.get(`${BASE_URL}/api/v1/metrics`);
  check(metricsRes, {
    'metrics is 200': (r) => r.status === 200,
  });

  sleep(1);
}

export function setup() {
  console.log('🔍 Smoke test starting...');
  console.log(`Target: ${BASE_URL}`);

  const res = http.get(`${BASE_URL}/api/v1/health`);
  if (res.status !== 200) {
    throw new Error(`Server not ready: ${res.status}`);
  }

  return { baseUrl: BASE_URL };
}

export function teardown() {
  console.log('✅ Smoke test completed successfully');
}
