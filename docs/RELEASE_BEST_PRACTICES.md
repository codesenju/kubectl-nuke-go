# Release Best Practices

This project uses automated releases powered by [release-please](https://github.com/googleapis/release-please-action), [GoReleaser](https://goreleaser.com/), and [Semantic Versioning](https://semver.org/).

## Commit Message Guidelines

Follow the [Conventional Commits](https://www.conventionalcommits.org/) standard to help automate versioning:

- `fix:` — for bug fixes (patch release, e.g., 1.0.1 → 1.0.2)
- `feat:` — for new features (minor release, e.g., 1.0.1 → 1.1.0)
- `docs:` — for documentation changes (patch release)
- `chore:`, `refactor:`, etc. — for maintenance (patch release)
- `BREAKING CHANGE:` — in the commit body or `!` after type for breaking changes (major release, e.g., 1.0.1 → 2.0.0)

## Release Workflow

### Automated Release Process

1. **Commit with Conventional Commits**: Push commits following the conventional commit format
2. **Release Please**: Automatically creates release PRs based on commit messages
3. **Manual Tag Creation**: Create tags manually for both stable releases and prereleases
4. **Automated Build & Deploy**: GoReleaser automatically builds and publishes releases

### Prerelease Workflow

Create prereleases for testing new features before stable release:

#### Supported Prerelease Formats

- `v1.2.3-alpha.1` - Alpha releases (early development)
- `v1.2.3-beta.1` - Beta releases (feature complete, testing)
- `v1.2.3-rc.1` - Release candidates (stable, final testing)
- `v1.2.3-pre.1` - General prereleases

#### Creating a Prerelease

```bash
# Create and push a prerelease tag
git tag v1.2.3-beta.1
git push origin v1.2.3-beta.1
```

#### Prerelease Behavior

- ✅ **Builds cross-platform binaries** (Linux, macOS, Windows)
- ✅ **Creates GitHub prerelease** with assets
- ✅ **Generates release notes** automatically
- ❌ **Skips Homebrew tap** (prereleases don't update Homebrew)
- ❌ **Skips Winget** (prereleases don't update package managers)

### Stable Release Workflow

Create stable releases for production use:

#### Creating a Stable Release

```bash
# Create and push a stable release tag
git tag v1.2.3
git push origin v1.2.3
```

#### Stable Release Behavior

- ✅ **Builds cross-platform binaries** (Linux, macOS, Windows)
- ✅ **Creates GitHub release** with assets
- ✅ **Updates Homebrew tap** (if configured)
- ✅ **Updates Winget** (if configured)
- ✅ **Generates comprehensive release notes**

### Release Types Comparison

| Feature | Prerelease | Stable Release |
|---------|------------|----------------|
| Binary builds | ✅ | ✅ |
| GitHub release | ✅ (marked as prerelease) | ✅ |
| Homebrew tap | ❌ | ✅ |
| Winget package | ❌ | ✅ |
| Auto-detection | ✅ | ✅ |
| Release notes | ✅ | ✅ |

## Tips

- Use clear, descriptive commit messages.
- Group related changes in a single PR when possible.
- For documentation-only changes, use `docs:` in your commit message.
- For breaking changes, add `BREAKING CHANGE:` in the commit body or use `feat!:` or `fix!:`.

## More

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [release-please](https://github.com/googleapis/release-please-action)
