# Release Process

This document explains how dnsctl's automated release process works.

## Overview

dnsctl uses an automated CI/CD pipeline powered by:
- **GitHub Actions** for CI and release automation
- **release-please** for automated version bumps and changelog generation
- **Conventional Commits** for semantic versioning

When commits following the conventional commit format are pushed to `main`, release-please automatically creates or updates a release PR. Merging that PR triggers a new release with pre-built binaries.

## Repository Setup

Before release-please can create pull requests, you must enable the required GitHub Actions permission:

1. Go to your repository's **Settings** > **Actions** > **General**
2. Scroll to **Workflow permissions**
3. Enable **"Allow GitHub Actions to create and approve pull requests"**
4. Click **Save**

Without this setting, release-please will fail with: "GitHub Actions is not permitted to create or approve pull requests".

## Conventional Commits

All commits should follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types and Version Bumps

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat` | New feature | Minor (0.x.0) |
| `fix` | Bug fix | Patch (0.0.x) |
| `docs` | Documentation only | None |
| `style` | Formatting, no code change | None |
| `refactor` | Code change that neither fixes nor adds | None |
| `perf` | Performance improvement | Patch (0.0.x) |
| `test` | Adding/updating tests | None |
| `chore` | Maintenance tasks | None |
| `ci` | CI/CD changes | None |

### Breaking Changes

Breaking changes trigger a major version bump (x.0.0). Mark them with either:
- Add `!` after the type: `feat!: change config format`
- Add `BREAKING CHANGE:` in the commit footer

### Examples

```bash
# Feature (minor bump)
git commit -m "feat: add quad9 DNS profile"

# Bug fix (patch bump)
git commit -m "fix: resolve DNS cache flush timeout"

# Breaking change (major bump)
git commit -m "feat!: change config file format"

# With scope
git commit -m "feat(tui): add keyboard shortcut for flush"

# With body
git commit -m "fix: handle empty DNS response

The DNS client now returns an empty slice instead of nil
when no DNS servers are configured."
```

## CI Pipeline

The CI workflow (`.github/workflows/ci.yml`) runs on:
- Pushes to `main` branch
- Pushes to `feature/*` branches
- Pull requests targeting `main`

### Jobs

#### Test Job

Runs on `ubuntu-latest`:
1. Checkout code
2. Setup Go 1.24
3. Download dependencies
4. Run tests with coverage
5. Upload coverage report (retained for 7 days)

#### Build Job

Runs after tests pass. Builds binaries for all supported platforms using a matrix strategy:

| OS | Architecture |
|----|--------------|
| darwin (macOS) | amd64 |
| darwin (macOS) | arm64 |
| linux | amd64 |
| linux | arm64 |

### Artifact Retention

Build artifacts are retained based on the trigger:

| Trigger | Retention |
|---------|-----------|
| Feature branch | 1 day |
| Pull request | 7 days |
| Main branch | 90 days |

## Release Workflow

The release workflow (`.github/workflows/release.yml`) runs on pushes to `main`.

### How release-please Works

1. **Commit Analysis**: release-please analyzes commits since the last release
2. **Release PR**: Creates/updates a PR with:
   - Version bump based on conventional commits
   - Updated CHANGELOG.md
   - Version updates in relevant files
3. **Release Creation**: When the release PR is merged:
   - A new GitHub release is created
   - A git tag is created (e.g., `v1.2.0`)

### Binary Attachment

After a release is created, the `upload-assets` job:
1. Builds binaries for all platform/architecture combinations
2. Uploads them to the GitHub release using `gh release upload`

The `--clobber` flag ensures binaries can be re-uploaded if needed.

## GITHUB_TOKEN

The workflows use the automatic `GITHUB_TOKEN` provided by GitHub Actions. No manual token setup is required.

### Permissions Used

The release workflow requests:
- `contents: write` - Create releases and upload assets
- `pull-requests: write` - Create and update release PRs

## Manual Steps

### Merging Release PRs

When release-please creates a release PR:

1. Review the proposed version bump and changelog
2. Verify CI checks pass
3. Merge the PR (squash or merge commit both work)
4. The release and binaries are created automatically

> **Note:** Branch protection requires the `build` status check to pass before
> the PR can be merged. This typically takes 1-2 minutes. The merge button will
> be disabled until checks complete.

### Verifying Releases

After merging a release PR:

1. Go to the repository's **Releases** page
2. Verify the new release appears with the correct tag
3. Confirm all four binaries are attached:
   - `dnsctl-darwin-amd64`
   - `dnsctl-darwin-arm64`
   - `dnsctl-linux-amd64`
   - `dnsctl-linux-arm64`

## Local Hook Setup

A `commit-msg` git hook validates that commit messages follow Conventional Commits format before they reach the remote. Run the setup script once after cloning:

```bash
./scripts/setup-hooks.sh
```

This sets `core.hooksPath` to the `hooks/` directory in the repository. The hook rejects messages that don't match `<type>[scope][!]: <description>` and allows merge/revert commits through unchanged.

## Troubleshooting

### Release PR Not Created

**Cause**: No releasable commits since last release.

**Solution**: Ensure commits use conventional commit format with types that trigger releases (`feat`, `fix`, `perf`).

### Binaries Missing from Release

**Cause**: The `upload-assets` job may have failed.

**Solution**:
1. Check the Actions tab for the failed workflow run
2. Re-run the failed job, or
3. Manually build and upload using:
   ```bash
   GOOS=darwin GOARCH=arm64 go build -o dnsctl-darwin-arm64 ./cmd/dnsctl
   gh release upload v1.2.3 dnsctl-darwin-arm64
   ```

### Version Bump Incorrect

**Cause**: Commits don't follow conventional commit format exactly.

**Solution**:
- Ensure no typos in commit type (`feat` not `feature`)
- Breaking changes need `!` or `BREAKING CHANGE:` footer
- The scope is optional but must be in parentheses: `feat(tui):`

### Tests Failing in CI

**Cause**: Code works locally but fails in CI.

**Solution**:
- CI runs on `ubuntu-latest` - check for macOS-specific code paths
- Verify Go version matches (1.24.x)
- Check if tests depend on network or file system state

### Release PR Permission Error

**Cause**: "GitHub Actions is not permitted to create or approve pull requests"

**Solution**: Enable PR creation in repository settings. See [Repository Setup](#repository-setup) section above.

### Release PR Not Mergeable

**Cause**: Branch protection requires CI status checks to pass before merging.

**Solution**: Wait for CI to complete on the release PR (~1-2 minutes). The merge button will become enabled once the `build` check passes. Do not use `--admin` flag to bypass unless necessary.

### CI Passes but Release Doesn't Run

**Cause**: Release workflow only runs after CI workflow completes successfully.

**Solution**: Check the Actions tab to verify CI workflow completed with success status. The release workflow triggers on `workflow_run` completion of the CI workflow.
