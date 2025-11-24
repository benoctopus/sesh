#!/usr/bin/env bash
# Claude hook: Install development tools in remote environments only

if [ "$CLAUDE_CODE_REMOTE" != "true" ]; then
  exit 0
fi

set -e

# Check if we're in a remote environment
is_remote() {
  # Check for common remote environment indicators
  [[ -n "${REMOTE_CONTAINERS}" ]] ||
    [[ -n "${CODESPACES}" ]] ||
    [[ -n "${GITPOD_WORKSPACE_ID}" ]] ||
    [[ -n "${SSH_CONNECTION}" ]] ||
    [[ -n "${SSH_CLIENT}" ]] ||
    [[ "${TERM_PROGRAM}" == "vscode" && -n "${VSCODE_IPC_HOOK_CLI}" ]]
}

# Only proceed if in remote environment
if ! is_remote; then
  echo "Local environment detected, skipping tool installation (use nix develop)"
  exit 0
fi

echo "Remote environment detected, checking for required tools..."

# Function to check if a command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Function to install via package manager
install_tools() {
  echo "Installing development tools..."

  # Detect package manager and install
  if command_exists apt-get; then
    # Debian/Ubuntu
    sudo apt-get update -qq
    sudo apt-get install -y wget curl git

    # Install go-task
    if ! command_exists task; then
      echo "Installing go-task..."
      sudo sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
    fi

    # Install golangci-lint
    if ! command_exists golangci-lint; then
      echo "Installing golangci-lint..."
      curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/bin
    fi

    # Install fzf
    if ! command_exists fzf; then
      echo "Installing fzf..."
      git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
      ~/.fzf/install --bin
      sudo ln -sf ~/.fzf/bin/fzf /usr/local/bin/fzf
    fi

    # Install tree
    if ! command_exists tree; then
      echo "Installing tree..."
      sudo apt-get install -y tree
    fi

  elif command_exists yum; then
    # RHEL/CentOS/Fedora
    sudo yum install -y wget curl git

    # Install go-task
    if ! command_exists task; then
      echo "Installing go-task..."
      sudo sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
    fi

    # Install golangci-lint
    if ! command_exists golangci-lint; then
      echo "Installing golangci-lint..."
      curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/bin
    fi

    # Install fzf
    if ! command_exists fzf; then
      echo "Installing fzf..."
      git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
      ~/.fzf/install --bin
      sudo ln -sf ~/.fzf/bin/fzf /usr/local/bin/fzf
    fi

    # Install tree
    if ! command_exists tree; then
      echo "Installing tree..."
      sudo yum install -y tree
    fi

  elif command_exists brew; then
    # macOS (Homebrew)
    echo "Installing via Homebrew..."
    brew install go-task golangci-lint fzf tree 2>/dev/null || true

  else
    echo "Warning: Unknown package manager, attempting manual installation..."

    # Install go-task
    if ! command_exists task; then
      echo "Installing go-task..."
      sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b "$HOME/.local/bin"
      export PATH="$HOME/.local/bin:$PATH"
    fi

    # Install golangci-lint
    if ! command_exists golangci-lint; then
      echo "Installing golangci-lint..."
      curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$HOME/.local/bin"
    fi

    # Install fzf
    if ! command_exists fzf; then
      echo "Installing fzf..."
      git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
      ~/.fzf/install --bin
      mkdir -p "$HOME/.local/bin"
      ln -sf ~/.fzf/bin/fzf "$HOME/.local/bin/fzf"
    fi
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
command_exists task && echo "  task: $(task --version 2>/dev/null || echo 'installed')"
command_exists golangci-lint && echo "  golangci-lint: $(golangci-lint --version 2>/dev/null | head -n1 || echo 'installed')"
command_exists fzf && echo "  fzf: $(fzf --version 2>/dev/null || echo 'installed')"
command_exists tree && echo "  tree: $(tree --version 2>/dev/null | head -n1 || echo 'installed')"
