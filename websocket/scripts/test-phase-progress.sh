#!/bin/bash

# =============================================================================
# Phase-Based Progress Integration Test Scripts
# =============================================================================
# Purpose: Manual testing for phase-based progress messages via Redis
# Usage: ./scripts/test-phase-progress.sh [test-name]
# =============================================================================

set -e

# Configuration
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"

# Project/User IDs for testing
TEST_PROJECT_ID="test_project_001"
TEST_USER_ID="user_123"
CHANNEL="project:${TEST_PROJECT_ID}:${TEST_USER_ID}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Redis command helper
redis_cmd() {
    if [ -n "$REDIS_PASSWORD" ]; then
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" "$@"
    else
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" "$@"
    fi
}

# Print helper
print_header() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# =============================================================================
# Test Cases
# =============================================================================

test_phase_progress_initializing() {
    print_header "Test: Phase Progress - INITIALIZING"
    
    local message='{
        "type": "project_progress",
        "payload": {
            "project_id": "'"$TEST_PROJECT_ID"'",
            "status": "INITIALIZING",
            "overall_progress_percent": 0
        }
    }'
    
    print_info "Publishing to channel: $CHANNEL"
    echo "$message" | jq .
    redis_cmd PUBLISH "$CHANNEL" "$message"
    print_success "Published INITIALIZING message"
}

test_phase_progress_crawling() {
    print_header "Test: Phase Progress - Crawl Phase"
    
    local message='{
        "type": "project_progress",
        "payload": {
            "project_id": "'"$TEST_PROJECT_ID"'",
            "status": "PROCESSING",
            "crawl": {
                "total": 100,
                "done": 45,
                "errors": 2,
                "progress_percent": 47.0
            },
            "overall_progress_percent": 18.8
        }
    }'
    
    print_info "Publishing to channel: $CHANNEL"
    echo "$message" | jq .
    redis_cmd PUBLISH "$CHANNEL" "$message"
    print_success "Published Crawl Phase progress message"
}

test_phase_progress_analyzing() {
    print_header "Test: Phase Progress - Both Phases"
    
    local message='{
        "type": "project_progress",
        "payload": {
            "project_id": "'"$TEST_PROJECT_ID"'",
            "status": "PROCESSING",
            "crawl": {
                "total": 100,
                "done": 98,
                "errors": 2,
                "progress_percent": 100.0
            },
            "analyze": {
                "total": 98,
                "done": 45,
                "errors": 1,
                "progress_percent": 46.9
            },
            "overall_progress_percent": 68.1
        }
    }'
    
    print_info "Publishing to channel: $CHANNEL"
    echo "$message" | jq .
    redis_cmd PUBLISH "$CHANNEL" "$message"
    print_success "Published Both Phases progress message"
}

test_phase_progress_completed() {
    print_header "Test: Phase Progress - COMPLETED"
    
    local message='{
        "type": "project_completed",
        "payload": {
            "project_id": "'"$TEST_PROJECT_ID"'",
            "status": "DONE",
            "crawl": {
                "total": 100,
                "done": 98,
                "errors": 2,
                "progress_percent": 100.0
            },
            "analyze": {
                "total": 98,
                "done": 95,
                "errors": 3,
                "progress_percent": 100.0
            },
            "overall_progress_percent": 100.0
        }
    }'
    
    print_info "Publishing to channel: $CHANNEL"
    echo "$message" | jq .
    redis_cmd PUBLISH "$CHANNEL" "$message"
    print_success "Published COMPLETED message"
}

test_phase_progress_failed() {
    print_header "Test: Phase Progress - FAILED"
    
    local message='{
        "type": "project_completed",
        "payload": {
            "project_id": "'"$TEST_PROJECT_ID"'",
            "status": "FAILED",
            "crawl": {
                "total": 100,
                "done": 30,
                "errors": 70,
                "progress_percent": 30.0
            },
            "overall_progress_percent": 12.0
        }
    }'
    
    print_info "Publishing to channel: $CHANNEL"
    echo "$message" | jq .
    redis_cmd PUBLISH "$CHANNEL" "$message"
    print_success "Published FAILED message"
}

test_legacy_format() {
    print_header "Test: Legacy Format (Backward Compatibility)"
    
    local message='{
        "status": "PROCESSING",
        "progress": {
            "current": 50,
            "total": 100,
            "percentage": 50.0,
            "eta": 10.5,
            "errors": []
        }
    }'
    
    print_info "Publishing LEGACY format to channel: $CHANNEL"
    echo "$message" | jq .
    redis_cmd PUBLISH "$CHANNEL" "$message"
    print_success "Published Legacy format message"
}

test_simulate_full_progress() {
    print_header "Test: Simulate Full Progress Flow"
    
    print_info "Step 1: INITIALIZING"
    test_phase_progress_initializing
    sleep 1
    
    print_info "Step 2: Crawl Phase 50%"
    test_phase_progress_crawling
    sleep 1
    
    print_info "Step 3: Both Phases"
    test_phase_progress_analyzing
    sleep 1
    
    print_info "Step 4: COMPLETED"
    test_phase_progress_completed
    
    print_success "Full progress flow simulation complete!"
}

# =============================================================================
# Main
# =============================================================================

show_help() {
    echo "Phase-Based Progress Integration Test Scripts"
    echo ""
    echo "Usage: $0 [test-name]"
    echo ""
    echo "Tests:"
    echo "  init          Test INITIALIZING status"
    echo "  crawl         Test Crawl phase progress"
    echo "  analyze       Test Both phases progress"
    echo "  completed     Test COMPLETED status"
    echo "  failed        Test FAILED status"
    echo "  legacy        Test Legacy format (backward compatibility)"
    echo "  full          Simulate full progress flow"
    echo "  all           Run all individual tests"
    echo ""
    echo "Environment Variables:"
    echo "  REDIS_HOST     Redis host (default: localhost)"
    echo "  REDIS_PORT     Redis port (default: 6379)"
    echo "  REDIS_PASSWORD Redis password (optional)"
}

case "${1:-all}" in
    init)
        test_phase_progress_initializing
        ;;
    crawl)
        test_phase_progress_crawling
        ;;
    analyze)
        test_phase_progress_analyzing
        ;;
    completed)
        test_phase_progress_completed
        ;;
    failed)
        test_phase_progress_failed
        ;;
    legacy)
        test_legacy_format
        ;;
    full)
        test_simulate_full_progress
        ;;
    all)
        test_phase_progress_initializing
        echo ""
        test_phase_progress_crawling
        echo ""
        test_phase_progress_analyzing
        echo ""
        test_phase_progress_completed
        echo ""
        test_phase_progress_failed
        echo ""
        test_legacy_format
        print_header "All tests completed!"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Unknown test: $1"
        show_help
        exit 1
        ;;
esac
