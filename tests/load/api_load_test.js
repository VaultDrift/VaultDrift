import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const apiErrors = new Rate('api_errors');
const loginDuration = new Trend('login_duration');
const fileListDuration = new Trend('file_list_duration');
const uploadDuration = new Trend('upload_duration');
const requestCount = new Counter('request_count');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up to 10 users
    { duration: '5m', target: 10 },   // Stay at 10 users
    { duration: '2m', target: 20 },   // Ramp up to 20 users
    { duration: '5m', target: 20 },   // Stay at 20 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
    http_req_failed: ['rate<0.01'],   // Less than 1% errors
    api_errors: ['rate<0.05'],        // Less than 5% API errors
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const USERNAME = __ENV.TEST_USER || 'admin';
const PASSWORD = __ENV.TEST_PASS || 'admin';

// Helper: Login and get token
function login() {
  const url = `${BASE_URL}/api/v1/auth/login`;
  const payload = JSON.stringify({
    username: USERNAME,
    password: PASSWORD,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const start = Date.now();
  const res = http.post(url, payload, params);
  const duration = Date.now() - start;
  loginDuration.add(duration);
  requestCount.add(1);

  const success = check(res, {
    'login status is 200': (r) => r.status === 200,
    'login has access_token': (r) => r.json('access_token') !== undefined,
  });

  apiErrors.add(!success);

  if (!success) {
    console.error(`Login failed: ${res.status} - ${res.body}`);
    return null;
  }

  return res.json('access_token');
}

// Helper: Get user profile
function getProfile(token) {
  const url = `${BASE_URL}/api/v1/user/profile`;
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  };

  const res = http.get(url, params);
  requestCount.add(1);

  const success = check(res, {
    'profile status is 200': (r) => r.status === 200,
    'profile has id': (r) => r.json('id') !== undefined,
  });

  apiErrors.add(!success);
  return success;
}

// Helper: List files
function listFiles(token) {
  const url = `${BASE_URL}/api/v1/files`;
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  };

  const start = Date.now();
  const res = http.get(url, params);
  const duration = Date.now() - start;
  fileListDuration.add(duration);
  requestCount.add(1);

  const success = check(res, {
    'files list status is 200': (r) => r.status === 200,
    'files is array': (r) => Array.isArray(r.json('files')),
  });

  apiErrors.add(!success);
  return success;
}

// Helper: Create folder
function createFolder(token, name) {
  const url = `${BASE_URL}/api/v1/files`;
  const payload = JSON.stringify({
    name: name,
    type: 'folder',
  });

  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(url, payload, params);
  requestCount.add(1);

  const success = check(res, {
    'create folder status is 201': (r) => r.status === 201,
    'folder has id': (r) => r.json('id') !== undefined,
  });

  apiErrors.add(!success);
  return success ? res.json('id') : null;
}

// Helper: Upload small file
function uploadFile(token, folderId) {
  const url = `${BASE_URL}/api/v1/upload`;
  const fileData = http.file('Hello, World! This is a test file.', 'test.txt', 'text/plain');

  const formData = {
    name: `test-${Date.now()}.txt`,
    size: '40',
    parent_id: folderId || '',
    file: fileData,
  };

  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  };

  const start = Date.now();
  const res = http.post(url, formData, params);
  const duration = Date.now() - start;
  uploadDuration.add(duration);
  requestCount.add(1);

  const success = check(res, {
    'upload status is 200 or 201': (r) => r.status === 200 || r.status === 201,
  });

  apiErrors.add(!success);
  return success;
}

// Helper: Health check
function healthCheck() {
  const url = `${BASE_URL}/api/v1/health`;
  const res = http.get(url);
  requestCount.add(1);

  const success = check(res, {
    'health status is 200': (r) => r.status === 200,
    'health is ok': (r) => r.json('status') === 'healthy',
  });

  apiErrors.add(!success);
  return success;
}

// Main test scenario
export default function () {
  group('Health Check', () => {
    healthCheck();
  });

  group('Authentication', () => {
    const token = login();
    if (!token) {
      return;
    }

    sleep(1);

    group('User Operations', () => {
      getProfile(token);
      sleep(1);
    });

    group('File Operations', () => {
      listFiles(token);
      sleep(1);

      const folderId = createFolder(token, `load-test-${Date.now()}`);
      sleep(1);

      if (folderId) {
        uploadFile(token, folderId);
      }
    });
  });

  sleep(2);
}

// Setup: Verify server is ready
export function setup() {
  console.log(`Starting load test against: ${BASE_URL}`);

  const res = http.get(`${BASE_URL}/api/v1/health`);
  if (res.status !== 200) {
    throw new Error(`Server not ready: ${res.status}`);
  }

  return { baseUrl: BASE_URL };
}

// Teardown: Print summary
export function teardown(data) {
  console.log('Load test completed');
  console.log(`Target: ${data.baseUrl}`);
}
