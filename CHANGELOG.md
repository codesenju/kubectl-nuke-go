# Changelog

## [Unreleased]
### Added
- **Intelligent CRD Discovery**: Automatically discover and analyze CRDs causing namespace termination issues
- **Smart CRD Cleanup**: Intelligently clean up problematic CRDs based on namespace conditions
- **Enhanced --force Mode**: Now includes automatic CRD discovery and cleanup for comprehensive namespace deletion
- **--dry-run Flag**: Added as an alias for --diagnose-only with enhanced functionality
- **Force Dry-Run Mode**: `--force --dry-run` shows detailed debug output of what aggressive cleanup would do
- **Namespace Condition Analysis**: Parse namespace conditions to identify specific finalizer and resource issues
- **Comprehensive Diagnostics**: Enhanced diagnostics with CRD-aware recommendations
- Enhanced storage provider resource handling for Longhorn, Rook-Ceph, and OpenEBS
- Specialized handlers for storage provider custom resources
- More aggressive finalizer removal strategies using JSON patch
- Comprehensive custom resource finalizer removal
- Improved diagnostics for storage provider issues

### Enhanced
- **Namespace Deletion**: Now includes automatic CRD discovery in both standard and force modes
- **Diagnostics**: Enhanced with detailed CRD analysis and specific cleanup recommendations
- **User Experience**: Clearer output with step-by-step breakdown of issues and solutions
- **Command Interface**: More intuitive flag usage with --dry-run alias

### Changed
- **--force Flag**: Now includes automatic CRD discovery and cleanup (backward compatible)
- **Standard Mode**: Intelligently cleans up CRDs only when they're causing termination issues
- **Documentation**: Updated help text, examples, and README to reflect new capabilities
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
