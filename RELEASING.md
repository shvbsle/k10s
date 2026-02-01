# Release Process

## Prerequisites

1. **Create Homebrew Tap Repository**
   - Go to https://github.com/new
   - Repository name: `homebrew-tap`
   - Make it public
   - No need to initialize with README (GoReleaser will manage it)

2. **Ensure GitHub Token has permissions**
   - The `GITHUB_TOKEN` in GitHub Actions already has the necessary permissions

## Cutting a Release

1. **Ensure all changes are committed and pushed**
   ```bash
   git status  # Should be clean
   git push
   ```

2. **Create and push a release tag**
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. **GitHub Actions will automatically:**
   - Build binaries for all platforms (linux, darwin, windows × amd64, arm64)
   - Create a GitHub release with changelog
   - Upload `.rpm` and `.deb` packages
   - Update your Homebrew tap with the formula

4. **Monitor the release**
   - Go to https://github.com/shvbsle/k10s/actions
   - Watch the release workflow complete

## Installation (After Release)

### macOS (Homebrew)
```bash
brew tap shvbsle/tap
brew install k10s
```

### Linux (Amazon Linux / RHEL / CentOS)
```bash
# Download the RPM from GitHub releases
wget https://github.com/shvbsle/k10s/releases/download/v0.1.0/k10s_0.1.0_linux_amd64.rpm

# Install
sudo yum install ./k10s_0.1.0_linux_amd64.rpm
# or
sudo rpm -i k10s_0.1.0_linux_amd64.rpm
```

### Linux (Debian / Ubuntu)
```bash
# Download the DEB from GitHub releases
wget https://github.com/shvbsle/k10s/releases/download/v0.1.0/k10s_0.1.0_linux_amd64.deb

# Install
sudo dpkg -i k10s_0.1.0_linux_amd64.deb
```

### Direct Binary Download
Download from https://github.com/shvbsle/k10s/releases

## Version Numbering

We follow semantic versioning (semver):
- `v0.x.y` - Pre-1.0 releases
- `v1.0.0` - First stable release
- Patch: `v1.0.1` - Bug fixes only
- Minor: `v1.1.0` - New features, backwards compatible
- Major: `v2.0.0` - Breaking changes

## Testing a Release Locally

Before pushing a tag, test the release process:

```bash
goreleaser release --snapshot --clean --skip=publish
```

This creates a local build in `./dist/` without publishing anything.
