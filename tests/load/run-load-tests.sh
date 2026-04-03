#!/bin/bash

# VaultDrift Load Test Runner
# Usage: ./run-load-tests.sh [smoke|api|stress|spike|websocket|all]

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_USER="${TEST_USER:-admin}"
TEST_PASS="${TEST_PASS:-admin}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if k6 is installed
check_k6() {
    if ! command -v k6 &> /dev/null; then
        log_error "k6 is not installed"
        echo "Please install k6: https://k6.io/docs/get-started/installation/"
        exit 1
    fi
    log_info "k6 version: $(k6 version | head -1)"
}

# Check if server is running
check_server() {
    log_info "Checking server at $BASE_URL..."
    if ! curl -sf "$BASE_URL/api/v1/health" > /dev/null 2>&1; then
        log_error "Server is not running at $BASE_URL"
        echo "Please start the server first: go run cmd/vaultdrift/main.go"
        exit 1
    fi
    log_info "Server is running"
}

# Run a specific test
run_test() {
    local test_name=$1
    local test_file="$SCRIPT_DIR/${test_name}_test.js"

    if [[ ! -f "$test_file" ]]; then
        log_error "Test file not found: $test_file"
        return 1
    fi

    log_info "Running $test_name test..."
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_USER="$TEST_USER" \
        --env TEST_PASS="$TEST_PASS" \
        "$test_file"
}

# Show usage
usage() {
    echo "Usage: $0 [smoke|api|stress|spike|websocket|all]"
    echo ""
    echo "Commands:"
    echo "  smoke     - Quick smoke test (1 min, 3 users)"
    echo "  api       - API load test (16 min, 20 users)"
    echo "  stress    - Stress test (21 min, 200 users)"
    echo "  spike     - Spike test (5 min, 10->100 users)"
    echo "  websocket - WebSocket test (5 min, 30 users)"
    echo "  all       - Run smoke + api tests"
    echo ""
    echo "Environment variables:"
    echo "  BASE_URL  - API base URL (default: http://localhost:8080)"
    echo "  TEST_USER - Test username (default: admin)"
    echo "  TEST_PASS - Test password (default: admin)"
}

# Main
main() {
    local command=${1:-help}

    case $command in
        smoke|api|stress|spike|websocket)
            check_k6
            check_server
            run_test "$command"
            ;;
        all)
            check_k6
            check_server
            run_test "smoke"
            run_test "api"
            ;;
        help|--help|-h)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

main "$@"
