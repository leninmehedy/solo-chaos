#!/bin/bash
# Environment Variable Validation Test Script
# Run this script from the chaos/ directory to test the validation system
#
# Security Note: This script has been refactored to avoid using 'eval' with user-provided
# command strings. Instead, it uses the 'env' command to safely set environment variables
# and execute the validation script, preventing arbitrary code execution.

set -e

echo "🧪 Testing Environment Variable Validation System"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run validation script with environment variables and expect it to fail
test_should_fail() {
    local env_vars="$1"
    local desc="$2"
    local args="${3:-}"
    echo -e "${YELLOW}Testing:${NC} $desc"
    
    # Use env command to safely set environment variables
    if env $env_vars ./validate-env.sh $args >/dev/null 2>&1; then
        echo -e "${RED}❌ FAIL:${NC} Command should have failed but passed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    else
        echo -e "${GREEN}✅ PASS:${NC} Command failed as expected"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    fi
}

# Function to run validation script with environment variables and expect it to pass
test_should_pass() {
    local env_vars="$1"
    local desc="$2"
    local args="${3:-}"
    echo -e "${YELLOW}Testing:${NC} $desc"
    
    # Use env command to safely set environment variables
    if env $env_vars ./validate-env.sh $args >/dev/null 2>&1; then
        echo -e "${GREEN}✅ PASS:${NC} Command passed as expected"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}❌ FAIL:${NC} Command should have passed but failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo ""
echo "🔍 Testing REGION Validation"
echo "----------------------------"
test_should_pass "REGION=us" "Valid REGION 'us'"
test_should_pass "REGION=eu" "Valid REGION 'eu'"
test_should_pass "REGION=ap" "Valid REGION 'ap'"
test_should_fail "REGION=invalid" "Invalid REGION 'invalid'"
test_should_fail "REGION=asia" "Invalid REGION 'asia'"
test_should_fail "REGION=us;echo malicious" "Injection attempt in REGION"

echo ""
echo "🔍 Testing NAMESPACE Validation"
echo "-------------------------------"
test_should_pass "NAMESPACE=solo" "Valid NAMESPACE 'solo'"
test_should_pass "NAMESPACE=test-ns" "Valid NAMESPACE 'test-ns'"
test_should_pass "NAMESPACE=my-namespace-123" "Valid NAMESPACE 'my-namespace-123'"
test_should_fail "NAMESPACE=INVALID" "Invalid NAMESPACE 'INVALID' (uppercase)"
test_should_fail "NAMESPACE=-invalid" "Invalid NAMESPACE '-invalid' (starts with dash)"
test_should_fail "NAMESPACE=invalid-" "Invalid NAMESPACE 'invalid-' (ends with dash)"
test_should_fail "NAMESPACE=ns;kubectl delete all" "Injection attempt in NAMESPACE"

echo ""
echo "🔍 Testing NODE_NAMES Validation"
echo "--------------------------------"
test_should_pass "NODE_NAMES=node1,node2,node3" "Valid NODE_NAMES 'node1,node2,node3'"
test_should_pass "NODE_NAMES=consensus-node-1,consensus-node-2" "Valid NODE_NAMES with hyphens"
test_should_pass "NODE_NAMES=single-node" "Valid single NODE_NAME"
test_should_fail "NODE_NAMES=-invalid,node2" "Invalid NODE_NAMES '-invalid,node2'"
test_should_fail "NODE_NAMES=node1,INVALID" "Invalid NODE_NAMES with uppercase"
test_should_fail "NODE_NAMES=node1;rm -rf /" "Injection attempt in NODE_NAMES"

echo ""
echo "🔍 Testing UUID Validation"
echo "--------------------------"
test_should_pass "UUID=abc123def456" "Valid UUID 'abc123def456'"
test_should_pass "UUID=test-uuid-123" "Valid UUID with hyphens"
test_should_fail "UUID=uuid;echo malicious" "Injection attempt in UUID"

# Function to test missing required variables using unset
test_missing_required() {
    local var_name="$1"
    local flag="$2"
    local desc="$3"
    echo -e "${YELLOW}Testing:${NC} $desc"
    
    # Start with a clean environment and unset the specific variable
    if env -u "$var_name" ./validate-env.sh "$flag" >/dev/null 2>&1; then
        echo -e "${RED}❌ FAIL:${NC} Command should have failed but passed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    else
        echo -e "${GREEN}✅ PASS:${NC} Command failed as expected"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    fi
}

echo ""
echo "🔍 Testing Required Variable Validation"
echo "---------------------------------------"
test_missing_required "REGION" "--require-region" "Missing required REGION"
test_missing_required "NAMESPACE" "--require-namespace" "Missing required NAMESPACE"
test_missing_required "NODE_NAMES" "--require-node-names" "Missing required NODE_NAMES"

echo ""
echo "🔍 Testing Combined Validations"
echo "-------------------------------"
test_should_pass "REGION=us NAMESPACE=solo NODE_NAMES=node1,node2 UUID=test123" "All valid variables"
test_should_fail "REGION=invalid NAMESPACE=solo NODE_NAMES=node1,node2" "Mixed valid/invalid (bad REGION)"
test_should_fail "REGION=us NAMESPACE=INVALID NODE_NAMES=node1,node2" "Mixed valid/invalid (bad NAMESPACE)"

echo ""
echo "=================================================="
echo "🧪 Test Results Summary"
echo "=================================================="
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! Environment variable validation is working correctly.${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed. Please check the validation implementation.${NC}"
    exit 1
fi
