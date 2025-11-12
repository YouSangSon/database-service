#!/bin/bash

# ============================================================================
# Database Service Load Testing Script
# ============================================================================
# Based on: https://github.com/YouSangSon/test_tools
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
TEST_TYPE="${1:-load}"  # load, stress, spike, soak
BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-}"
TENANT_ID="${TENANT_ID:-default}"
OUTPUT_DIR="./test/load/results"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if k6 is installed
check_k6() {
    if ! command -v k6 &> /dev/null; then
        print_error "k6 is not installed"
        print_info "Install k6:"
        print_info "  macOS: brew install k6"
        print_info "  Linux: sudo gpg -k && sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69 && echo \"deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main\" | sudo tee /etc/apt/sources.list.d/k6.list && sudo apt-get update && sudo apt-get install k6"
        print_info "  Windows: choco install k6"
        print_info "  Or visit: https://k6.io/docs/getting-started/installation/"
        exit 1
    fi
}

# Function to check service health
check_service() {
    print_info "Checking service health at $BASE_URL..."

    if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
        print_info "✅ Service is healthy"
        return 0
    else
        print_error "❌ Service is not accessible at $BASE_URL"
        print_info "Make sure the service is running:"
        print_info "  docker-compose up -d"
        exit 1
    fi
}

# Function to run load test
run_load_test() {
    local test_file="$1"
    local test_name="$2"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local result_file="$OUTPUT_DIR/${test_name}_${timestamp}"

    print_info "🚀 Running $test_name test..."
    print_info "Test file: $test_file"
    print_info "Base URL: $BASE_URL"
    print_info "Tenant ID: $TENANT_ID"
    print_info "Results will be saved to: $result_file"
    print_info ""

    # Run k6 test
    k6 run \
        --out json="${result_file}.json" \
        --summary-export="${result_file}_summary.json" \
        -e BASE_URL="$BASE_URL" \
        -e API_KEY="$API_KEY" \
        -e TENANT_ID="$TENANT_ID" \
        -e TEST_RUN_ID="${timestamp}" \
        "$test_file"

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        print_info "✅ Test completed successfully"
        print_info "Results: ${result_file}.json"
        print_info "Summary: ${result_file}_summary.json"
    else
        print_error "❌ Test failed with exit code $exit_code"
        return $exit_code
    fi

    # Generate HTML report if k6 reporter is available
    if command -v k6-reporter &> /dev/null; then
        print_info "Generating HTML report..."
        k6-reporter "${result_file}.json"
    fi

    return $exit_code
}

# Main script
main() {
    print_info "=== Database Service Load Testing ==="
    print_info ""

    # Check prerequisites
    check_k6
    check_service

    # Select test type
    case "$TEST_TYPE" in
        load)
            print_info "Running LOAD test..."
            run_load_test "./test/load/database-service-load-test.js" "load"
            ;;
        stress)
            print_info "Running STRESS test..."
            run_load_test "./test/load/scenarios/stress-test.js" "stress"
            ;;
        spike)
            print_info "Running SPIKE test..."
            if [ -f "./test/load/scenarios/spike-test.js" ]; then
                run_load_test "./test/load/scenarios/spike-test.js" "spike"
            else
                print_warning "Spike test not implemented yet"
            fi
            ;;
        soak)
            print_info "Running SOAK test..."
            if [ -f "./test/load/scenarios/soak-test.js" ]; then
                run_load_test "./test/load/scenarios/soak-test.js" "soak"
            else
                print_warning "Soak test not implemented yet"
            fi
            ;;
        all)
            print_info "Running ALL tests..."
            run_load_test "./test/load/database-service-load-test.js" "load"
            run_load_test "./test/load/scenarios/stress-test.js" "stress"
            ;;
        *)
            print_error "Unknown test type: $TEST_TYPE"
            print_info "Usage: $0 [load|stress|spike|soak|all]"
            print_info ""
            print_info "Environment variables:"
            print_info "  BASE_URL    - Service URL (default: http://localhost:8080)"
            print_info "  API_KEY     - API key for authentication (optional)"
            print_info "  TENANT_ID   - Tenant ID (default: default)"
            print_info ""
            print_info "Examples:"
            print_info "  $0 load"
            print_info "  BASE_URL=http://staging:8080 $0 stress"
            print_info "  API_KEY=sk_test123 TENANT_ID=demo $0 load"
            exit 1
            ;;
    esac

    print_info ""
    print_info "=== Test completed ==="
}

# Run main function
main
