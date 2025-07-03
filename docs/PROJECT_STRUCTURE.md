# kubectl-nuke-go

A kubectl plugin to forcefully delete Kubernetes resources, including namespaces stuck in the Terminating state.

## Project Structure

- `cmd/kubectl-nuke/main.go`: Main entry point for the CLI tool with Cobra command structure
- `internal/kube/`: Core namespace deletion logic and unit tests
  - `namespace.go`: Functions for deleting namespaces and removing finalizers
  - `namespace_test.go`: Unit tests for namespace operations
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
├── ns|namespace <namespace>  # Delete a namespace
├── version                   # Show version information
└── help                      # Show help information
```

## Key Features

- **Subcommand support**: Uses `ns` or `namespace` subcommands
- **kubectl plugin compatibility**: Works as `kubectl nuke`
- **Force deletion**: Removes finalizers for stuck namespaces
- **User-friendly output**: Emoji indicators and clear status messages
- **Comprehensive help**: Built-in help and examples
- **Testable**: Unit tests for core functionality

## Getting Started

See the main [README.md](../README.md) for quick start instructions or [USAGE.md](USAGE.md) for detailed usage examples.
