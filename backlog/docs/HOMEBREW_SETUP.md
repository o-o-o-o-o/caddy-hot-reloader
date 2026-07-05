# Homebrew Tap Setup Guide

This guide walks you through creating and maintaining a Homebrew tap for `caddy-hot-reloader`.

## Overview

A Homebrew tap is a GitHub repository that contains Homebrew formulas. Users can install your software with:

```bash
brew tap o-o-o-o-o/caddy-hot-reloader https://github.com/o-o-o-o-o/caddy-hot-reloader
brew install caddy-hot-reloader
```

## Initial Setup

### 1. Use This Repository as Tap Source

This repository already contains:

- `Formula/caddy-hot-reloader.rb`
- `.github/workflows/update-formula.yml`

No separate `homebrew-*` repository is required.

### 2. Clone and Setup

```bash
git clone https://github.com/o-o-o-o-o/caddy-hot-reloader
cd caddy-hot-reloader

# Ensure Formula and workflow are committed
git add Formula/caddy-hot-reloader.rb .github/workflows/update-formula.yml
git commit -m "Add Homebrew formula and update workflow"
git push origin main

```

### 3. Create First Release

In your **main plugin repository** (not the tap):

```bash
cd /path/to/caddy-hot-reloader

# Tag a release
git tag -a v1.0.0 -m "Initial release"
git push origin v1.0.0
```

This will trigger a GitHub Release.

### 4. Update Formula with Release SHA256

GitHub Actions will automatically create a PR to update the formula, but for the first release you need the SHA256:

```bash
# Download the release tarball
curl -sL https://github.com/o-o-o-o-o/caddy-hot-reloader/archive/refs/tags/v1.0.0.tar.gz -o release.tar.gz

# Calculate SHA256
shasum -a 256 release.tar.gz

# Update Formula/caddy-hot-reloader.rb:
# - Set `url` to the GitHub release tarball URL
# - Set `sha256` to the calculated hash
```

Edit `Formula/caddy-hot-reloader.rb`:

```ruby
url "https://github.com/o-o-o-o-o/caddy-hot-reloader/archive/refs/tags/v1.0.0.tar.gz"
sha256 "abc123...your-actual-sha256"
```

Commit and push:

```bash
git add Formula/caddy-hot-reloader.rb
git commit -m "Update formula to v1.0.0"
git push origin main
```

### 5. Test Installation

```bash
# Install from your tap
brew tap o-o-o-o-o/caddy-hot-reloader https://github.com/o-o-o-o-o/caddy-hot-reloader
brew install caddy-hot-reloader

# Verify it works
caddy version

# Test the module
caddy list-modules | grep hot_reloader
```

## Maintenance

### Releasing New Versions

1. **Tag a new release** in the main plugin repo:

   ```bash
   cd /path/to/caddy-hot-reloader
   git tag -a v1.0.1 -m "Bug fixes and improvements"
   git push origin v1.0.1
   ```

2. **GitHub Actions automatically**:
   - Detects the new release
   - Downloads the tarball
   - Calculates SHA256
   - Creates a PR to update the formula

3. **Review and merge** the auto-generated PR

4. **Users update** with:
   ```bash
   brew update
   brew upgrade caddy-hot-reloader
   ```

### Dependency Updates

**Dependabot** automatically monitors dependencies:

- **Go modules** (Caddy, fsnotify, websocket, etc.) - checked every Monday
- **GitHub Actions** - checked every Monday

When updates are available:

1. **Dependabot creates PR** with dependency changes and changelog
2. **Review and test locally**:

   ```bash
   # Fetch the PR branch
   git fetch origin pull/PR_NUMBER/head:deps-update
   git checkout deps-update

   # Test the build
   ./build.sh
   ./caddy version

   # Test hot-reload functionality
   ./caddy run --config ./Caddyfile
   ```

3. **Merge if tests pass**
4. **Tag a new release** to publish to Homebrew users:
   ```bash
   git tag -a v1.0.2 -m "Update Caddy to v2.x.x"
   git push origin v1.0.2
   ```

### Manual Formula Update (if automation fails)

```bash
cd /path/to/caddy-hot-reloader

# Download new release and calculate SHA
VERSION="v1.0.1"
curl -sL "https://github.com/o-o-o-o-o/caddy-hot-reloader/archive/refs/tags/${VERSION}.tar.gz" -o release.tar.gz
SHA256=$(shasum -a 256 release.tar.gz | awk '{print $1}')

# Update formula
sed -i '' "s|url \".*\"|url \"https://github.com/o-o-o-o-o/caddy-hot-reloader/archive/refs/tags/${VERSION}.tar.gz\"|" Formula/caddy-hot-reloader.rb
sed -i '' "s|sha256 \".*\"|sha256 \"${SHA256}\"|" Formula/caddy-hot-reloader.rb

# Commit
git add Formula/caddy-hot-reloader.rb
git commit -m "Update formula to ${VERSION}"
git push origin main

# Clean up
rm release.tar.gz
```

### Testing Formula Locally

Before pushing changes:

```bash
# Uninstall current version
brew uninstall caddy-hot-reloader

# Install from local formula
brew install --build-from-source ./Formula/caddy-hot-reloader.rb

# Run tests
brew test caddy-hot-reloader

# Check audit
brew audit --strict --online caddy-hot-reloader
```

## Troubleshooting

### Formula fails to install

1. Check build logs:

   ```bash
   brew install --verbose --debug caddy-hot-reloader
   ```

2. Verify Go is available (formula builds xcaddy via `go run`):

   ```bash
   brew install go
   ```

3. Test formula syntax:
   ```bash
   brew audit --strict Formula/caddy-hot-reloader.rb
   ```

### GitHub Actions not triggering

1. Ensure workflows are enabled in tap repo settings
2. Check that `GITHUB_TOKEN` has required permissions
3. Verify webhook delivery in Settings → Webhooks

### Users report outdated version

```bash
# Force formula refresh
brew update --force
brew upgrade caddy-hot-reloader
```

## Best Practices

1. **Semantic Versioning**: Use `v1.0.0`, `v1.0.1`, etc.
2. **Test Before Release**: Always test locally before tagging
3. **Changelog**: Maintain a CHANGELOG.md in the main repo
4. **Breaking Changes**: Clearly document in release notes
5. **Dependencies**: Keep `depends_on` minimal and up-to-date

## Distribution Options

### Public Tap (Recommended for Open Source)

- Anyone can `brew tap` and install
- Listed in `brew search`
- Can be submitted to homebrew/core eventually

### Private Tap (for Internal Use)

- Use private GitHub repo
- Requires authentication: `brew tap o-o-o-o-o/private`
- Users need GitHub access

## Resources

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Homebrew Taps](https://docs.brew.sh/Taps)
- [GitHub Actions for Homebrew](https://github.com/Homebrew/actions)

## Support

For issues with:

- **The plugin itself**: Open issue in [caddy-hot-reloader](https://github.com/o-o-o-o-o/caddy-hot-reloader)
- **Formula/installation**: Open issue in this tap repository
