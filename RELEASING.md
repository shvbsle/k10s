# Release Process

This document describes how to create a new release of k10s.

## Prerequisites

1. You have write access to the repository
2. All CI checks are passing on `main`
3. You have decided on the next version number following [Semantic Versioning](https://semver.org/)

## Release Steps

### 1. Create a Release PR

Create a new branch for the release:

```bash
# Example for v0.2.0
git checkout main
git pull origin main
git checkout -b release-v0.2.0
```

### 2. Update Version Files

Update the `VERSION` file:

```bash
echo "v0.2.0" > VERSION
```

### 3. Update CHANGELOG.md

Update [CHANGELOG.md](CHANGELOG.md):

```markdown
## [Unreleased]

## [0.2.0] - 2025-10-XX

### Added
- New feature X
- New feature Y

### Changed
- Updated feature Z

### Fixed
- Bug fix A

[Unreleased]: https://github.com/shvbsle/k10s/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/shvbsle/k10s/releases/tag/v0.2.0
[0.1.0]: https://github.com/shvbsle/k10s/releases/tag/v0.1.0
```

### 4. Commit and Push

```bash
git add VERSION CHANGELOG.md
git commit -m "Release v0.2.0"
git push origin release-v0.2.0
```

### 5. Create Pull Request

Create a PR with title: `Release v0.2.0`

- Ensure CI passes
- Get PR reviewed if needed
- Merge to `main`

### 6. Create and Push Git Tag

After the PR is merged:

```bash
git checkout main
git pull origin main
git tag v0.2.0
git push origin v0.2.0
```

### 7. Automated Release

Pushing the tag triggers the GitHub Actions workflow (`.github/workflows/release.yaml`) which:

1. Runs GoReleaser
2. Builds binaries for all platforms (macOS, Linux, Windows)
3. Creates a GitHub Release with auto-generated release notes
4. Publishes to Homebrew tap (requires `homebrew-tap` repo to exist)
5. Creates checksums and SBOMs

### 8. Verify Release

Check that:

1. GitHub Release is created: https://github.com/shvbsle/k10s/releases
2. Release notes look correct
3. Binaries are attached to the release
4. Homebrew formula is updated (if tap repo exists)

## Installing the Release

### Homebrew (macOS/Linux)

Once you've created the `homebrew-tap` repository:

```bash
brew tap shvbsle/tap
brew install k10s
```

### Direct Download

Download from the releases page:
https://github.com/shvbsle/k10s/releases

### From Source

```bash
git clone https://github.com/shvbsle/k10s.git
cd k10s
git checkout v0.2.0
make build
```

## Setting Up Homebrew Tap (One-time)

To enable Homebrew distribution, create a `homebrew-tap` repository:

1. Create a new repository: https://github.com/shvbsle/homebrew-tap
2. The repository can be empty - GoReleaser will populate it automatically
3. Make sure the repository is public
4. GoReleaser will automatically create/update the formula on each release

## Version Numbers

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v1.0.0 → v2.0.0): Incompatible API changes
- **MINOR** version (v0.1.0 → v0.2.0): New features, backwards compatible
- **PATCH** version (v0.1.0 → v0.1.1): Bug fixes, backwards compatible

## Testing Releases Locally

Test the release process without publishing:

```bash
make snapshot
```

This creates a local release in `./dist/` without pushing to GitHub.

## Troubleshooting

### Release workflow fails

- Check the GitHub Actions logs
- Ensure Go version matches in `.github/workflows/release.yaml`
- Verify `.goreleaser.yaml` syntax

### Homebrew tap not updating

- Ensure `homebrew-tap` repository exists and is public
- Check GoReleaser logs for errors
- Verify repository permissions

### Version mismatch

- Ensure `VERSION` file is updated
- Ensure git tag matches the version
- Tag format must be `vX.Y.Z` (with 'v' prefix)
