#!/usr/bin/env bash
# Claude hook: Install development tools in remote environments only

if [ "$CLAUDE_CODE_REMOTE" != "true" ]; then
  exit 0
fi

set -e

echo "Remote environment detected, checking for required tools..."

# Function to check if a command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Function to install via package manager
install_tools() {
  echo "Installing development tools..."

  # Debian/Ubuntu
  apt-get update -qq
  apt-get install -y wget curl git

  # Install go-task
  if ! command_exists task; then
    go install github.com/go-task/task/v3/cmd/task@latest
  fi

  # Install golangci-lint
  if ! command_exists golangci-lint; then
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2
  fi

}

# Check if tools are already installed
missing_tools=()

if ! command_exists task; then
  missing_tools+=("task")
fi

if ! command_exists golangci-lint; then
  missing_tools+=("golangci-lint")
fi

if ! command_exists fzf; then
  missing_tools+=("fzf")
fi

if ! command_exists tree; then
  missing_tools+=("tree")
fi

# Install if any tools are missing
if [ ${#missing_tools[@]} -gt 0 ]; then
  echo "Missing tools: ${missing_tools[*]}"
  install_tools
  echo "✓ Tools installed successfully"
else
  echo "✓ All required tools already installed"
fi

# Verify installations
echo ""
echo "Tool versions:"
go mod download
command_exists task && echo "  task: $(task --version 2>/dev/null || echo 'installed')"
command_exists golangci-lint && echo "  golangci-lint: $(golangci-lint --version 2>/dev/null | head -n1 || echo 'installed')"
command_exists fzf && echo "  fzf: $(fzf --version 2>/dev/null || echo 'installed')"
command_exists tree && echo "  tree: $(tree --version 2>/dev/null | head -n1 || echo 'installed')"
