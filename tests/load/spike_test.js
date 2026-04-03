import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

/**
 * Spike Test
 *
 * Purpose: Test system recovery from sudden traffic spikes
 * Pattern: Sudden jump to high load, then back down
 *
 * Run: k6 run spike_test.js
 */

const errorRate = new Rate('spike_errors');

export const options = {
  stages: [
    { duration: '30s', target: 10 },    // Normal load
    { duration: '10s', target: 100 },   // Spike to 100 users
    { duration: '2m', target: 100 },    // Stay at spike
    { duration: '30s', target: 10 },    // Back to normal
    { duration: '2m', target: 10 },     // Verify recovery
    { duration: '10s', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],  // Allow higher latency during spike
    http_req_failed: ['rate<0.15'],     // Allow more errors during spike
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/health`);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
  });

  errorRate.add(!success);

  sleep(Math.random()); // Random 0-1s
}

export function setup() {
  console.log('⚡ Spike test starting...');
  console.log('Pattern: 10 users -> 100 users spike -> back to 10');

  return { startTime: Date.now() };
}

export function teardown() {
  console.log('✅ Spike test completed');
}
