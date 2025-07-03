# Release Best Practices

This project uses automated releases powered by [release-please](https://github.com/googleapis/release-please-action) and [Semantic Versioning](https://semver.org/).

## Commit Message Guidelines

Follow the [Conventional Commits](https://www.conventionalcommits.org/) standard to help automate versioning:

- `fix:` — for bug fixes (patch release, e.g., 1.0.1 → 1.0.2)
- `feat:` — for new features (minor release, e.g., 1.0.1 → 1.1.0)
- `docs:` — for documentation changes (patch release)
- `chore:`, `refactor:`, etc. — for maintenance (patch release)
- `BREAKING CHANGE:` — in the commit body or `!` after type for breaking changes (major release, e.g., 1.0.1 → 2.0.0)

## Release Workflow

### Pre-releases (RC)

- Pre-releases are automated using [release-please](https://github.com/googleapis/release-please-action) and are configured in `.release-please-config.json`.
- When changes are merged to `main`, release-please will open a PR for a pre-release (e.g., `v1.2.3-rc.1`).
- Merging the PR will create a pre-release tag and GitHub pre-release.
- Pre-releases use the `rc` (release candidate) label by default (e.g., `v1.2.3-rc.1`). You can change this in `.release-please-config.json`.
- The `CHANGELOG.md` is updated automatically with each pre-release.

### General Availability (GA) Releases

- When you are ready for a stable release, manually push a tag (e.g., `v1.0.0`).
- This triggers the `Release` workflow, which builds and uploads binaries as assets to the GitHub Release.
- The workflow will automatically extract the relevant section from `CHANGELOG.md` and use it as the GitHub Release notes, so your release notes always match your changelog.
- GA releases are not created by release-please, but by manually tagging and pushing.

## Tips

- Use clear, descriptive commit messages.
- Group related changes in a single PR when possible.
- For documentation-only changes, use `docs:` in your commit message.
- For breaking changes, add `BREAKING CHANGE:` in the commit body or use `feat!:` or `fix!:`.

## More

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [release-please](https://github.com/googleapis/release-please-action)
