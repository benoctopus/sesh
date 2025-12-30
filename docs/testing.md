# Testing Guide

This document describes how to test sesh, including running the end-to-end (E2E) test suite.

## E2E Test Script

The E2E test script (`scripts/e2e-test.sh`) exercises all major sesh functionality in an isolated environment. It's designed to be run by users to quickly verify that sesh is working correctly.

### Running the E2E Tests

```bash
# Build and run all tests
./scripts/e2e-test.sh

# Skip build step (use existing binary)
./scripts/e2e-test.sh --skip-build

# Keep temporary directories for inspection
./scripts/e2e-test.sh --keep-temp
```

### What the Tests Cover

The E2E test suite exercises the following functionality:

1. **Help Command** - Verifies `sesh --help` works
2. **Clone Repository** - Clones a test repository from GitHub
3. **List Projects** - Verifies projects are tracked correctly
4. **Switch Branch** - Tests worktree creation and branch switching
5. **List Worktrees** - Verifies worktrees are tracked
6. **List Sessions** - Checks session listing (may skip if backend unavailable)
7. **Status Command** - Tests status reporting
8. **Logs Command** - Verifies log file access
9. **Clean Command** - Tests cleanup functionality
10. **Completions** - Verifies shell completion generation
11. **Delete Project** - Tests project deletion

### Test Environment

The test script creates an isolated environment:

- **Temporary Config Directory**: All sesh configuration and database files are created in a temporary directory
- **Temporary Workspace**: Projects are cloned to a temporary workspace directory
- **Automatic Cleanup**: All temporary files are removed after tests complete (unless `--keep-temp` is used)

The isolation is achieved using the `--config-dir` flag, which allows overriding the default config directory location.

### Test Repository

The tests use `https://github.com/octocat/Hello-World` as a test repository. This is a small, stable repository maintained by GitHub that's unlikely to change.

### Expected Output

The test script provides color-coded output:

- ✓ **Green (PASS)**: Test passed
- ✗ **Red (FAIL)**: Test failed
- ⊘ **Yellow (SKIP)**: Test skipped (usually due to missing dependencies)

At the end, a summary is displayed showing the total number of passed, failed, and skipped tests.

### Troubleshooting

**Tests fail with "binary not found"**
- Run `cargo build` or `cargo build --release` first
- Or use `--skip-build` if you've already built the binary

**Clone test fails**
- Check your internet connection
- Verify GitHub is accessible
- The test repository may be temporarily unavailable

**Session-related tests are skipped**
- This is normal if tmux or other session backends aren't available
- These tests are optional and don't affect the overall test result

**Tests fail with permission errors**
- Ensure the script is executable: `chmod +x scripts/e2e-test.sh`
- Check that you have write permissions in the temporary directory

## Unit Tests

Unit tests are located alongside the source code in test modules. Run them with:

```bash
cargo test
```

## Integration Tests

Integration tests for database operations and manager logic can be found in the test directories. These tests use mock backends where appropriate to avoid requiring external dependencies.

## Manual Testing

For manual testing of specific features:

1. **Isolated Testing**: Use the `--config-dir` flag to test in an isolated environment:
   ```bash
   mkdir -p /tmp/sesh-test
   sesh --config-dir /tmp/sesh-test clone https://github.com/user/repo
   ```

2. **Verbose Logging**: Enable verbose output to see detailed logs:
   ```bash
   sesh --verbose list
   ```

3. **Check Logs**: View the log file for debugging:
   ```bash
   sesh logs --lines 50
   ```

## Continuous Integration

The E2E test script is designed to be run in CI environments. It:
- Exits with code 0 on success, 1 on failure
- Provides machine-readable output
- Can skip interactive features (like fuzzy finders) when not available
- Cleans up after itself automatically

