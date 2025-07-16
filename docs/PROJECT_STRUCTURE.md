# kubectl-nuke-go

A kubectl plugin to forcefully delete Kubernetes resources, including namespaces stuck in the Terminating state and unresponsive pods. Provides both gentle and aggressive deletion modes.

## Project Structure

- `cmd/kubectl-nuke/main.go`: Main entry point for the CLI tool with Cobra command structure
- `internal/kube/`: Core Kubernetes resource deletion logic and unit tests
  - `resources.go`: Functions for deleting namespaces, pods, and removing finalizers
  - `namespace_test.go`: Unit tests for all resource operations
- `docs/`: Additional documentation
  - `USAGE.md`: Detailed usage instructions and examples
  - `FAQ.md`: Frequently asked questions
  - `PROJECT_STRUCTURE.md`: This file
  - `RELEASE_BEST_PRACTICES.md`: Release and commit guidelines
- `README.md`: Project overview and quick start guide
- `CONTRIBUTING.md`: Contribution guidelines
- `CHANGELOG.md`: Release history and changes
- `LICENSE`: Project license (MIT)
- `go.mod` / `go.sum`: Go module dependencies

## Command Structure

The tool uses Cobra CLI framework with the following command structure:

```
kubectl-nuke
├── ns|namespace <namespace>     # Delete a namespace
│   └── --force|-f              # Aggressive deletion mode
├── pod|pods|po <pod-name>...    # Force delete pods
│   └── --namespace|-n          # Target namespace
├── version                      # Show version information
└── help                         # Show help information
```

## Key Features

- **Multiple resource types**: Supports namespaces and pods
- **Dual deletion modes**: Standard and aggressive (--force) deletion
- **kubectl plugin compatibility**: Works as `kubectl nuke`
- **Multiple aliases**: `ns`/`namespace`, `pod`/`pods`/`po`
- **Batch operations**: Delete multiple pods in one command
- **Smart finalizer removal**: Multiple strategies for stuck resources
- **Force pod deletion**: Grace period 0 for immediate termination
- **User-friendly output**: Emoji indicators and clear status messages
- **Comprehensive help**: Built-in help and examples for all commands
- **Extensive testing**: Unit tests for all core functionality

## Getting Started

See the main [README.md](../README.md) for quick start instructions or [USAGE.md](USAGE.md) for detailed usage examples.
