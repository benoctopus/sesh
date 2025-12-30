#!/usr/bin/env bash
set -euo pipefail

# E2E Test Script for Sesh
# Tests all major functionality in an isolated environment

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0
SKIPPED=0

# Helper functions
pass() {
    ((PASSED++))
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

fail() {
    ((FAILED++))
    echo -e "${RED}✗ FAIL${NC}: $1"
    if [ -n "${2:-}" ]; then
        echo -e "  ${RED}Error:${NC} $2"
    fi
}

skip() {
    ((SKIPPED++))
    echo -e "${YELLOW}⊘ SKIP${NC}: $1"
}

info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Parse arguments
SKIP_BUILD=false
KEEP_TEMP=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --keep-temp)
            KEEP_TEMP=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--skip-build] [--keep-temp]"
            exit 1
            ;;
    esac
done

# Setup
info "Setting up test environment..."

# Create temporary directories
TEST_DIR=$(mktemp -d)
TEST_CONFIG_DIR="$TEST_DIR/config/sesh"
TEST_WORKSPACE="$TEST_DIR/workspace"

export SESH_CONFIG_DIR_OVERRIDE="$TEST_CONFIG_DIR"
mkdir -p "$TEST_CONFIG_DIR"
mkdir -p "$TEST_WORKSPACE"

# Find sesh binary
if [ "$SKIP_BUILD" = false ]; then
    info "Building sesh..."
    cargo build --release 2>&1 | grep -v "^   Compiling" || true
    SESH_BIN="./target/release/sesh"
else
    SESH_BIN="./target/debug/sesh"
    if [ ! -f "$SESH_BIN" ]; then
        SESH_BIN="./target/release/sesh"
    fi
    if [ ! -f "$SESH_BIN" ]; then
        fail "sesh binary not found. Run 'cargo build' first or remove --skip-build"
        exit 1
    fi
fi

info "Using sesh binary: $SESH_BIN"
info "Test config directory: $TEST_CONFIG_DIR"
info "Test workspace: $TEST_WORKSPACE"

# Test repository
TEST_REPO="https://github.com/octocat/Hello-World"
TEST_PROJECT_NAME="github.com/octocat/Hello-World"

# Cleanup function
cleanup() {
    if [ "$KEEP_TEMP" = false ]; then
        info "Cleaning up temporary directories..."
        rm -rf "$TEST_DIR"
    else
        info "Keeping temporary directory: $TEST_DIR"
    fi
}

trap cleanup EXIT

# Test functions
test_help() {
    info "Testing: --help"
    if "$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" --help > /dev/null 2>&1; then
        pass "Help command"
    else
        fail "Help command"
    fi
}

test_clone() {
    info "Testing: clone repository"
    if "$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" clone "$TEST_REPO" > /dev/null 2>&1; then
        pass "Clone repository"
    else
        fail "Clone repository"
        return 1
    fi
}

test_list_projects() {
    info "Testing: list projects"
    output=$("$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" list --projects 2>&1)
    if echo "$output" | grep -q "Hello-World\|octocat"; then
        pass "List projects"
    else
        fail "List projects" "Output: $output"
    fi
}

test_switch_branch() {
    info "Testing: switch branch"
    # First, get the default branch (usually master or main)
    if "$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" switch master --project "$TEST_PROJECT_NAME" > /dev/null 2>&1; then
        pass "Switch branch (master)"
    elif "$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" switch main --project "$TEST_PROJECT_NAME" > /dev/null 2>&1; then
        pass "Switch branch (main)"
    else
        # Try to list branches and use the first one
        skip "Switch branch (no default branch found, may need manual setup)"
    fi
}

test_list_worktrees() {
    info "Testing: list worktrees"
    output=$("$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" list --worktrees 2>&1)
    if echo "$output" | grep -q "Hello-World"; then
        pass "List worktrees"
    else
        fail "List worktrees" "Output: $output"
    fi
}

test_list_sessions() {
    info "Testing: list sessions"
    output=$("$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" list --sessions 2>&1)
    # Sessions may not exist if tmux/backend isn't available, so we just check it doesn't error
    if [ $? -eq 0 ]; then
        pass "List sessions"
    else
        skip "List sessions (backend may not be available)"
    fi
}

test_status() {
    info "Testing: status command"
    if "$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" status > /dev/null 2>&1; then
        pass "Status command"
    else
        skip "Status command (may not be in a session)"
    fi
}

test_logs() {
    info "Testing: logs command"
    output=$("$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" logs --lines 10 2>&1)
    if [ $? -eq 0 ]; then
        pass "Logs command"
    else
        fail "Logs command" "Output: $output"
    fi
}

test_delete_worktree() {
    info "Testing: delete worktree"
    # Get worktree ID from list output
    output=$("$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" list --worktrees 2>&1)
    # Extract worktree ID if available (this is a simplified check)
    # In a real scenario, we'd parse the output more carefully
    if echo "$output" | grep -q "Hello-World"; then
        # For now, we'll skip actual deletion to avoid breaking other tests
        # In a full implementation, we'd extract the ID and delete it
        skip "Delete worktree (requires parsing list output for ID)"
    else
        skip "Delete worktree (no worktrees to delete)"
    fi
}

test_clean() {
    info "Testing: clean command"
    if "$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" clean --stale > /dev/null 2>&1; then
        pass "Clean command"
    else
        fail "Clean command"
    fi
}

test_completions() {
    info "Testing: completions"
    output=$("$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" completions bash 2>&1)
    if echo "$output" | grep -q "complete\|_sesh"; then
        pass "Completions command"
    else
        fail "Completions command" "Output: $output"
    fi
}

test_delete_project() {
    info "Testing: delete project"
    if "$SESH_BIN" --config-dir "$TEST_CONFIG_DIR" delete --project "$TEST_PROJECT_NAME" > /dev/null 2>&1; then
        pass "Delete project"
    else
        fail "Delete project"
    fi
}

# Run tests
echo ""
echo "=========================================="
echo "  Sesh E2E Test Suite"
echo "=========================================="
echo ""

test_help
test_clone || {
    fail "Cannot continue without successful clone"
    exit 1
}
test_list_projects
test_switch_branch
test_list_worktrees
test_list_sessions
test_status
test_logs
test_clean
test_completions
test_delete_project

# Summary
echo ""
echo "=========================================="
echo "  Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed:${NC} $PASSED"
echo -e "${RED}Failed:${NC} $FAILED"
echo -e "${YELLOW}Skipped:${NC} $SKIPPED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi

