#!/usr/bin/env bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
  echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Usage information
usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Automate version bumping and release tagging for sesh.

OPTIONS:
    -t, --type TYPE     Version bump type: patch (default), minor, or major
    -h, --help          Show this help message

EXAMPLES:
    $0                  # Bump patch version (0.1.6 -> 0.1.7)
    $0 -t minor         # Bump minor version (0.1.6 -> 0.2.0)
    $0 -t major         # Bump major version (0.1.6 -> 1.0.0)

REQUIREMENTS:
    - Clean git working directory
    - Claude Code CLI installed and authenticated
    - jq for JSON parsing

EOF
  exit 0
}

# Parse arguments
BUMP_TYPE="patch"
while [[ $# -gt 0 ]]; do
  case $1 in
  -t | --type)
    BUMP_TYPE="$2"
    shift 2
    ;;
  -h | --help)
    usage
    ;;
  *)
    log_error "Unknown option: $1"
    usage
    ;;
  esac
done

# Validate bump type
if [[ ! "$BUMP_TYPE" =~ ^(patch|minor|major)$ ]]; then
  log_error "Invalid bump type: $BUMP_TYPE. Must be patch, minor, or major."
  exit 1
fi

# Check for required tools
for cmd in jq git claude; do
  if ! command -v "$cmd" &>/dev/null; then
    log_error "Required command '$cmd' not found. Please install it first."
    exit 1
  fi
done

# Ensure we're in the git repository root
if ! git rev-parse --git-dir >/dev/null 2>&1; then
  log_error "Not in a git repository"
  exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

# Check for uncommitted changes
if [[ -n $(git status --porcelain) ]]; then
  log_error "Working directory is not clean. Please commit or stash your changes first."
  git status --short
  exit 1
fi

FLAKE_FILE="flake.nix"

# Extract current version from flake.nix
CURRENT_VERSION=$(grep -oP '^\s*version = "\K[^"]+' "$FLAKE_FILE" || true)

if [[ -z "$CURRENT_VERSION" ]]; then
  log_error "Could not find version in $FLAKE_FILE"
  exit 1
fi

log_info "Current version: $CURRENT_VERSION"

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<<"$CURRENT_VERSION"

# Increment version based on bump type
case "$BUMP_TYPE" in
major)
  MAJOR=$((MAJOR + 1))
  MINOR=0
  PATCH=0
  ;;
minor)
  MINOR=$((MINOR + 1))
  PATCH=0
  ;;
patch)
  PATCH=$((PATCH + 1))
  ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"
NEW_TAG="v$NEW_VERSION"

log_info "New version: $NEW_VERSION (tag: $NEW_TAG)"

# Check if tag already exists
if git rev-parse "$NEW_TAG" >/dev/null 2>&1; then
  log_error "Tag $NEW_TAG already exists"
  exit 1
fi

# Get the last version tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [[ -z "$LAST_TAG" ]]; then
  log_warn "No previous tags found. Will summarize all commits."
  REVISION_RANGE=""
else
  log_info "Last tag: $LAST_TAG"
  REVISION_RANGE="$LAST_TAG..HEAD"
fi

# Generate changelog/release summary using Claude Code
log_info "Generating release summary with Claude Code..."

CLAUDE_PROMPT="Review the git changes since ${LAST_TAG:-the beginning} and generate a concise release summary for version $NEW_VERSION.

Instructions:
1. Run: git log ${REVISION_RANGE} --oneline --no-decorate
2. Run: git diff ${REVISION_RANGE} --stat
3. Analyze the changes and create a brief release summary (2-4 sentences)
4. Focus on user-facing changes and improvements
5. Use conventional commit style
6. Keep it concise and professional
7. Do NOT include any markdown formatting, headers, or extra text
8. Output ONLY the summary text that will be used as the commit and tag message

Example format:
feat: add new workspace management features

This release adds support for multiple workspace backends, improves session switching performance, and fixes several bugs related to git worktree tracking. Enhanced CLI output with tree view for better visualization of project hierarchies."

# Run Claude Code headlessly and capture output
CLAUDE_OUTPUT=$(mktemp)
trap 'rm -f "$CLAUDE_OUTPUT"' EXIT

export CLAUDE_CODE_OAUTH_TOKEN="op://sesh/Claude code auth token/password"
if ! op run -- claude -p "$CLAUDE_PROMPT" --output-format text >"$CLAUDE_OUTPUT" 2>&1; then
  log_error "Failed to generate release summary with Claude Code"
  cat "$CLAUDE_OUTPUT" >&2
  exit 1
fi

# Extract the release message from Claude's output
# Claude may add some preamble, so we try to get the actual commit message
RELEASE_MESSAGE=$(cat "$CLAUDE_OUTPUT")

# Clean up the message (remove any leading/trailing whitespace)
RELEASE_MESSAGE=$(echo "$RELEASE_MESSAGE" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')

if [[ -z "$RELEASE_MESSAGE" ]]; then
  log_error "Claude Code did not generate a release message"
  exit 1
fi

log_info "Generated release summary:"
echo "---"
echo "$RELEASE_MESSAGE"
echo "---"

# Ask for confirmation
read -p "Proceed with version bump and release? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  log_warn "Release cancelled by user"
  exit 0
fi

# Update version in flake.nix
log_info "Updating version in $FLAKE_FILE..."
sed -i.bak "s/version = \"$CURRENT_VERSION\"/version = \"$NEW_VERSION\"/" "$FLAKE_FILE"
rm -f "$FLAKE_FILE.bak"

# Verify the change
UPDATED_VERSION=$(grep -oP '^\s*version = "\K[^"]+' "$FLAKE_FILE")
if [[ "$UPDATED_VERSION" != "$NEW_VERSION" ]]; then
  log_error "Failed to update version in $FLAKE_FILE"
  git checkout "$FLAKE_FILE"
  exit 1
fi

# Stage the change
git add "$FLAKE_FILE"

# Create commit with Claude's message
log_info "Creating commit..."
COMMIT_MESSAGE="chore(release): bump version to $NEW_VERSION

$RELEASE_MESSAGE"

if ! git commit -m "$COMMIT_MESSAGE"; then
  log_error "Failed to create commit"
  git reset HEAD "$FLAKE_FILE"
  git checkout "$FLAKE_FILE"
  exit 1
fi

# Create annotated tag with Claude's message
log_info "Creating tag $NEW_TAG..."
if ! git tag -a "$NEW_TAG" -m "$RELEASE_MESSAGE"; then
  log_error "Failed to create tag"
  git reset --hard HEAD~1
  exit 1
fi

log_info "${GREEN}Success!${NC} Version bumped to $NEW_VERSION and tagged as $NEW_TAG"
log_info "To push changes, run:"
echo "  git push origin main && git push origin $NEW_TAG"
