import http from 'k6/http';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

/**
 * WebSocket Load Test
 *
 * Purpose: Test WebSocket connection handling
 * Duration: 5 minutes
 * Users: 10-50 concurrent connections
 *
 * Run: k6 run websocket_test.js
 */

const wsMessages = new Counter('ws_messages');
const wsErrors = new Rate('ws_errors');
const wsLatency = new Trend('ws_latency');
const wsConnectTime = new Trend('ws_connect_time');

export const options = {
  stages: [
    { duration: '1m', target: 10 },
    { duration: '3m', target: 30 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    ws_errors: ['rate<0.05'],
    ws_latency: ['p(95)<100'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'ws://localhost:8080';
const WS_URL = BASE_URL.replace('http', 'ws');
const USERNAME = __ENV.TEST_USER || 'admin';
const PASSWORD = __ENV.TEST_PASS || 'admin';

function getAuthToken() {
  const url = `${BASE_URL.replace('ws', 'http')}/api/v1/auth/login`;
  const res = http.post(url, JSON.stringify({
    username: USERNAME,
    password: PASSWORD,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  if (res.status !== 200) {
    return null;
  }

  return res.json('access_token');
}

export default function () {
  const token = getAuthToken();
  if (!token) {
    wsErrors.add(1);
    return;
  }

  const url = `${WS_URL}/ws?token=${token}`;
  const startTime = Date.now();

  const res = ws.connect(url, null, function (socket) {
    const connectTime = Date.now() - startTime;
    wsConnectTime.add(connectTime);

    socket.on('open', function () {
      console.log('WebSocket connected');

      // Subscribe to root folder
      socket.send(JSON.stringify({
        action: 'subscribe',
        folder_id: 'root',
      }));
    });

    socket.on('message', function (message) {
      wsMessages.add(1);

      try {
        const data = JSON.parse(message);
        check(data, {
          'message has type': (d) => d.type !== undefined,
        });
      } catch (e) {
        wsErrors.add(1);
      }
    });

    socket.on('close', function () {
      console.log('WebSocket closed');
    });

    socket.on('error', function (e) {
      console.error('WebSocket error:', e);
      wsErrors.add(1);
    });

    // Keep connection open for random time
    sleep(Math.random() * 30 + 10); // 10-40 seconds

    // Unsubscribe before closing
    socket.send(JSON.stringify({
      action: 'unsubscribe',
      folder_id: 'root',
    }));

    socket.close();
  });

  check(res, {
    'WebSocket connected': (r) => r && r.status === 101,
  });

  if (res.status !== 101) {
    wsErrors.add(1);
  }

  sleep(2);
}

export function setup() {
  console.log('📡 WebSocket load test starting...');
  return { wsUrl: WS_URL };
}

export function teardown() {
  console.log('✅ WebSocket test completed');
}
