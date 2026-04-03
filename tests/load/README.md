# Load Testing Suite

This directory contains k6 load tests for the VaultDrift API.

## Prerequisites

Install k6:

```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D786D12CD588A1FD50

echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Windows (with chocolatey)
choco install k6
```

Or use Docker:
```bash
docker pull grafana/k6
```

## Quick Start

### 1. Smoke Test (Quick verification)
```bash
k6 run smoke_test.js
```

### 2. API Load Test
```bash
# With default settings
k6 run api_load_test.js

# With custom settings
k6 run --env BASE_URL=http://localhost:8080 --env TEST_USER=admin --env TEST_PASS=admin api_load_test.js
```

### 3. Stress Test
```bash
k6 run stress_test.js
```

### 4. Spike Test
```bash
k6 run spike_test.js
```

### 5. WebSocket Test
```bash
k6 run websocket_test.js
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | API base URL |
| `TEST_USER` | `admin` | Test username |
| `TEST_PASS` | `admin` | Test password |

## Test Descriptions

### smoke_test.js
- **Duration**: 1 minute
- **Users**: 3 concurrent
- **Purpose**: Quick verification that system works

### api_load_test.js
- **Duration**: ~16 minutes
- **Users**: 20 concurrent (max)
- **Purpose**: Standard load testing with realistic user patterns
- **Covers**: Auth, file operations, uploads

### stress_test.js
- **Duration**: ~21 minutes
- **Users**: 200 concurrent (max)
- **Purpose**: Find system breaking points

### spike_test.js
- **Duration**: ~5 minutes
- **Users**: 10 -> 100 -> 10
- **Purpose**: Test recovery from sudden traffic spikes

### websocket_test.js
- **Duration**: 5 minutes
- **Users**: 30 concurrent
- **Purpose**: Test WebSocket connection handling

## Running with Docker

```bash
docker run -v $(pwd):/tests -e BASE_URL=http://host.docker.internal:8080 grafana/k6 run /tests/smoke_test.js
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Load Tests

on:
  push:
    branches: [ main ]

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup k6
        uses: grafana/k6-action@v0.3.1
        with:
          filename: tests/load/smoke_test.js
        env:
          BASE_URL: http://localhost:8080
```

## Interpreting Results

### Key Metrics

| Metric | Good | Warning | Critical |
|--------|------|---------|----------|
| http_req_duration (p95) | < 200ms | 200-500ms | > 500ms |
| http_req_failed | < 1% | 1-5% | > 5% |
| iterations_per_second | > 100 | 50-100 | < 50 |

### Output Example

```
     ✓ status is 200
     ✓ response time < 500ms

     checks.....................: 100.00% ✓ 1234 ✗ 0
     data_received..............: 1.2 MB  73 kB/s
     data_sent..................: 156 kB  9.4 kB/s
     http_req_blocked...........: avg=1.23ms min=0s      med=1µs   max=45ms
     http_req_connecting........: avg=0.87ms min=0s      med=0s    max=32ms
     http_req_duration..........: avg=45.67ms min=12ms   med=38ms  max=234ms
     http_req_failed............: 0.00%   ✓ 0    ✗ 1234
     http_req_receiving.........: avg=0.12ms min=0s      med=0s    max=5ms
     http_req_sending...........: avg=0.05ms min=0s      med=0s    max=3ms
     http_req_waiting...........: avg=45.5ms  min=12ms   med=38ms  max=234ms
     http_reqs..................: 1234    74.2/s
     iteration_duration.........: avg=1.05s   min=1.01s  med=1.02s max=1.23s
     iterations.................: 617     37.1/s
     vus........................: 10      min=10 max=10
     vus_max....................: 10      min=10 max=10
```

## Troubleshooting

### "Server not ready" error
- Verify the server is running: `curl http://localhost:8080/api/v1/health`
- Check `BASE_URL` environment variable

### High error rates
- Check server logs for errors
- Verify database connection
- Check rate limiting settings

### Slow response times
- Monitor server CPU/memory
- Check database query performance
- Verify network latency

## Advanced Usage

### Custom thresholds
```javascript
export const options = {
  thresholds: {
    http_req_duration: ['p(95)<100'],  // Stricter
    http_req_failed: ['rate<0.001'],    // Stricter
  },
};
```

### Output to InfluxDB
```bash
k6 run --out influxdb=http://localhost:8086/k6 api_load_test.js
```

### Output to Prometheus
```bash
k6 run --out prometheus api_load_test.js
```
