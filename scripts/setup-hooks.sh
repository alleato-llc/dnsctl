#!/bin/sh
#
# setup-hooks.sh — configure git to use the project's hooks/ directory
#
# Usage: ./scripts/setup-hooks.sh

set -e

# Verify we're inside a git repository
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo >&2 "Error: not inside a git repository."
    exit 1
fi

# Resolve the repository root
repo_root=$(git rev-parse --show-toplevel)

# Verify the hooks directory exists
if [ ! -f "$repo_root/hooks/commit-msg" ]; then
    echo >&2 "Error: hooks/commit-msg not found in repository root."
    exit 1
fi

# Check for Conventional Commits indicators
found_indicator=""
for pattern in \
    "$repo_root/release-please-config.json" \
    "$repo_root/.commitlintrc"* \
    "$repo_root/commitlint.config."* \
    "$repo_root/cliff.toml"; do
    # shellcheck disable=SC2254
    for f in $pattern; do
        if [ -f "$f" ]; then
            found_indicator=$(basename "$f")
            break 2
        fi
    done
done

if [ -z "$found_indicator" ]; then
    echo "Warning: no Conventional Commits indicator found (release-please, commitlint, git-cliff)."
    echo "Installing hooks anyway since you ran this script explicitly."
    echo ""
fi

# Set core.hooksPath
git config core.hooksPath hooks
echo "Git hooks configured successfully."
echo ""
echo "  core.hooksPath = hooks"
if [ -n "$found_indicator" ]; then
    echo "  Detected: $found_indicator"
fi
echo ""
echo "Commit messages will now be validated against Conventional Commits format."
