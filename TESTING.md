# Testing

This document describes how to run and write tests for the sesh project.

## Running Tests

### Run All Tests

```bash
# Using go directly
go test ./...

# Using task (recommended)
task test

# With verbose output
go test -v ./...
```

### Run Tests with Coverage

```bash
# Generate coverage report
task test:coverage

# This will:
# 1. Run all tests with coverage enabled
# 2. Generate coverage.out file
# 3. Create coverage.html for viewing in browser
```

### Run Short Tests

```bash
# Skip long-running tests
task test:short
```

### Run Tests for Specific Package

```bash
# Test a specific package
go test ./internal/workspace/...
go test ./cmd/...

# Test a specific file
go test ./internal/workspace/workspace_test.go
```

## Test Coverage

Current test coverage focuses on:

### Well-Tested Packages (>50% coverage)
- **workspace** (57.8%): Path utilities, branch name sanitization, session name generation
- **config** (49.4%): Configuration loading, environment variable handling

### Moderately Tested Packages (10-20% coverage)
- **git** (15.7%): URL parsing, project name generation, branch list parsing
- **session** (13.5%): Session manager interface, environment detection
- **cmd** (9.5%): Helper functions (formatTimeAgo, truncate)
- **project** (8.5%): Project name normalization

### Areas Needing More Coverage
- **db** (0%): Database operations require integration tests
- **fuzzy** (0%): Fuzzy finder requires interactive testing
- CLI command execution requires integration testing

## Test Structure

Tests are organized following Go conventions:

```
project/
├── cmd/
│   ├── list.go
│   └── list_test.go          # Tests for list command
├── internal/
│   ├── workspace/
│   │   ├── workspace.go
│   │   └── workspace_test.go # Tests for workspace package
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   └── ...
```

## Writing Tests

### Unit Test Example

```go
func TestSanitizeBranchName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "branch with slash",
            input:    "feature/foo",
            expected: "feature-foo",
        },
        // Add more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := SanitizeBranchName(tt.input)
            if result != tt.expected {
                t.Errorf("SanitizeBranchName(%q) = %q, want %q", 
                    tt.input, result, tt.expected)
            }
        })
    }
}
```

### Table-Driven Tests

We use table-driven tests for most functions to make it easy to add new test cases:

```go
tests := []struct {
    name     string
    input    Type
    expected Type
    wantErr  bool
}{
    // Test cases here
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic here
    })
}
```

### Testing with Environment Variables

When testing functions that use environment variables, save and restore the original values:

```go
func TestGetWorkspaceDir(t *testing.T) {
    // Save original environment
    originalEnv := os.Getenv("SESH_WORKSPACE")
    defer func() {
        if originalEnv != "" {
            os.Setenv("SESH_WORKSPACE", originalEnv)
        } else {
            os.Unsetenv("SESH_WORKSPACE")
        }
    }()

    t.Run("with environment variable", func(t *testing.T) {
        os.Setenv("SESH_WORKSPACE", "/tmp/test-workspace")
        // Test logic...
    })
}
```

## Test Categories

### Unit Tests
- Test individual functions in isolation
- No external dependencies (no git, tmux, database)
- Fast and reliable
- Located in `*_test.go` files alongside source

### Integration Tests (Future)
- Test command execution end-to-end
- Require git, tmux, and database
- Slower but more comprehensive
- Would be marked with build tags or separate directory

## Continuous Integration

Tests run automatically on CI via GitHub Actions:
- On every pull request
- On every push to main
- Includes linting and formatting checks

See `.github/workflows/ci.yml` for CI configuration.

## Best Practices

1. **Test Naming**: Use descriptive test names that explain what is being tested
2. **Table-Driven**: Use table-driven tests for functions with multiple cases
3. **Error Cases**: Always test both success and error cases
4. **Edge Cases**: Test boundary conditions and edge cases
5. **Isolation**: Tests should be independent and not depend on external state
6. **Speed**: Keep unit tests fast; mark slow tests appropriately
7. **Coverage**: Aim for good coverage of critical paths

## Coverage Goals

- **Critical utility functions**: 90%+ coverage (workspace, config helpers)
- **Business logic**: 70%+ coverage (project resolution, session management)
- **CLI commands**: Integration tests for happy paths
- **Database operations**: Integration tests with test database

## Skipping Tests

For tests that require special setup or are currently unimplemented:

```go
t.Run("requires git repo", func(t *testing.T) {
    t.Skip("Requires integration test setup with git repository")
})
```

## Running Specific Tests

```bash
# Run a specific test function
go test -v -run TestSanitizeBranchName ./internal/workspace

# Run tests matching a pattern
go test -v -run "TestSanitize.*" ./...
```
