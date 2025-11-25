# Contributing to sesh

Thank you for your interest in contributing to sesh! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions. We want to maintain a welcoming and inclusive community.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- tmux (for testing session management features)
- fzf (optional, for fuzzy finding)

### Setting Up Development Environment

1. Fork the repository on GitHub
2. Clone your fork:

```bash
git clone https://github.com/YOUR_USERNAME/sesh.git
cd sesh
```

3. Install dependencies:

```bash
go mod download
```

4. Build the project:

```bash
go build -o sesh .
```

5. Run tests:

```bash
go test ./...
```

## Development Workflow

### Using Task

This project uses [Task](https://taskfile.dev/) for build automation. Common tasks:

```bash
# Run tests
task test

# Run tests with coverage
task test:coverage

# Format code
task fmt

# Run linter
task lint

# Run all checks (format, lint, test)
task check

# Build binary
task build
```

### Making Changes

1. Create a new branch for your feature or bugfix:

```bash
git checkout -b feature/your-feature-name
```

2. Make your changes, following the coding standards below
3. Add tests for new functionality
4. Run `task check` to ensure all checks pass
5. Commit your changes with clear, descriptive commit messages
6. Push to your fork and create a pull request

## Coding Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (run `task fmt`)
- Follow the guidelines in [Effective Go](https://golang.org/doc/effective_go)
- Keep functions small and focused
- Write clear, self-documenting code

### Error Handling

- Always use `github.com/rotisserie/eris` for error wrapping
- Wrap errors with context: `eris.Wrap(err, "descriptive message")`
- Use `eris.Wrapf` for formatted messages
- Never ignore errors unless explicitly documented why

Example:

```go
result, err := someFunction()
if err != nil {
    return eris.Wrap(err, "failed to execute someFunction")
}
```

### Testing

- Write tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies (filesystem, git commands, tmux)
- Aim for >70% code coverage
- Run `task test:coverage` to check coverage

Example test structure:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Documentation

- Add godoc comments to all exported functions, types, and constants
- Keep comments clear and concise
- Update README.md if adding user-facing features
- Add examples in godoc comments for complex functionality

Example:

```go
// GenerateSessionName creates a unique session name from project name and branch.
// The session name format is: {repo-name}:{branch-name}
// Special characters in branch names are sanitized to filesystem-safe equivalents.
//
// Example:
//   GenerateSessionName("github.com/user/repo", "feature/foo")
//   // Returns: "repo:feature-foo"
func GenerateSessionName(projectName, branch string) string {
    // implementation
}
```

## Project Structure

```
sesh/
├── cmd/               # CLI commands (cobra)
│   ├── root.go       # Root command
│   ├── clone.go      # Clone command
│   ├── switch.go     # Switch command
│   └── ...
├── internal/          # Internal packages
│   ├── config/       # Configuration management
│   ├── db/           # Database layer
│   ├── git/          # Git operations
│   ├── session/      # Session manager abstraction
│   ├── state/        # Filesystem state
│   ├── ui/           # UI helpers (colors, formatting)
│   └── workspace/    # Workspace management
├── main.go           # Entry point
├── go.mod            # Go dependencies
├── Taskfile.yaml     # Task definitions
└── README.md         # Documentation
```

## Adding New Features

### New Commands

1. Create a new file in `cmd/` directory
2. Define the cobra command
3. Implement the command logic
4. Add tests in `*_test.go` file
5. Update README.md with usage examples

### New Session Backends

To add support for a new session manager (e.g., zellij, screen):

1. Create new file: `internal/session/backend_name.go`
2. Implement the `SessionManager` interface
3. Register in `NewSessionManager()` factory function
4. Add detection logic in `DetectBackend()`
5. Add tests
6. Update documentation

## Pull Request Process

1. **Update documentation** - Update README.md and godoc comments
2. **Add tests** - Ensure new code has adequate test coverage
3. **Run checks** - Run `task check` and fix any issues
4. **Write clear PR description** - Explain what changes were made and why
5. **Link related issues** - Reference any related GitHub issues
6. **Keep PRs focused** - One feature or fix per PR
7. **Respond to feedback** - Address review comments promptly

### PR Title Format

Use conventional commit format:

- `feat: add support for zellij backend`
- `fix: handle missing git remote correctly`
- `docs: update installation instructions`
- `test: add tests for workspace management`
- `refactor: simplify branch name sanitization`
- `chore: update dependencies`

## Reporting Issues

### Bug Reports

Include:

- sesh version (`sesh --version`)
- Operating system and version
- Steps to reproduce
- Expected behavior
- Actual behavior
- Relevant logs or error messages

### Feature Requests

Include:

- Clear description of the feature
- Use cases and motivation
- Proposed API or command structure (if applicable)
- Any alternatives considered

## Development Tips

### Debugging

- Use `fmt.Printf` for quick debugging
- Use `eris.ToString(err, true)` for detailed error stack traces
- Test with `go run . <command>` during development

### Testing Locally

```bash
# Build and install locally
go install .

# Test in a clean environment
mkdir /tmp/test-sesh
export SESH_WORKSPACE=/tmp/test-sesh
sesh clone https://github.com/user/repo.git
```

### Common Issues

**Import cycle errors**: Ensure proper package organization, avoid circular dependencies

**Test failures**: Run `task test` to see detailed output

**Linter errors**: Run `task lint` and fix reported issues

## Release Process

(For maintainers)

1. Update version in code
2. Update CHANGELOG.md
3. Create git tag: `git tag v1.0.0`
4. Push tag: `git push origin v1.0.0`
5. GitHub Actions will build and publish release

## Questions?

If you have questions about contributing:

- Open a GitHub issue with the `question` label
- Check existing issues and discussions
- Read through the codebase and tests for examples

## License

By contributing to sesh, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to sesh! 🎉
