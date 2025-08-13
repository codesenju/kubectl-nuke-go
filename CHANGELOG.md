# Changelog

## [Unreleased]
### Added
- Enhanced storage provider resource handling for Longhorn, Rook-Ceph, and OpenEBS
- Specialized handlers for storage provider custom resources
- More aggressive finalizer removal strategies using JSON patch
- Comprehensive custom resource finalizer removal
- Improved diagnostics for storage provider issues

### Changed
- Refactored custom resource deletion to be more thorough
- Enhanced namespace deletion workflow with better error handling
- Improved resource discovery and processing

### Fixed
- Fixed issue with stuck namespaces containing Longhorn resources
- Improved handling of custom resources with finalizers
- Better error reporting for resource deletion failures

## [1.0.0] - 2025-07-01
### Added
- Initial release of kubectl-nuke
- Support for namespace deletion
- Support for pod force deletion
- Basic finalizer removal
