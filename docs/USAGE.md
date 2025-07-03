# kubectl-nuke Documentation

## Overview

`kubectl-nuke` is a kubectl plugin for forcefully deleting Kubernetes resources, including namespaces stuck in the Terminating state. It is useful for cluster administrators and DevOps engineers who need to clean up stuck resources.

## Installation

### Download the Pre-built Binary

You can download the latest release for your platform from the [Releases page](https://github.com/codesenju/kubectl-nuke-go/releases). For example, to download the Darwin (macOS) AMD64 binary:

```sh
VERSION=$(curl -s https://api.github.com/repos/codesenju/kubectl-nuke-go/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -LO https://github.com/codesenju/kubectl-nuke-go/releases/download/$VERSION/kubectl-nuke-darwin-amd64.tar.gz
```

Extract the binary:

```sh
tar -xzf kubectl-nuke-darwin-amd64.tar.gz
```

Move the binary to a directory in your `PATH` (e.g., `/usr/local/bin`):

```sh
sudo mv kubectl-nuke-darwin-amd64 /usr/local/bin/kubectl-nuke
chmod +x /usr/local/bin/kubectl-nuke
```

### Build from Source

Alternatively, you can build from source:

```sh
git clone https://github.com/codesenju/kubectl-nuke-go.git
cd kubectl-nuke-go
go build -o kubectl-nuke ./cmd/kubectl-nuke
```

## Usage

### Basic Usage

```sh
# Delete a namespace using the 'ns' subcommand
kubectl-nuke ns <namespace>

# Delete a namespace using the 'namespace' subcommand
kubectl-nuke namespace <namespace>

# Use with custom kubeconfig
kubectl-nuke --kubeconfig /path/to/config ns <namespace>
```

### As a kubectl Plugin

After installation, you can use this tool as a kubectl plugin:

```sh
kubectl nuke ns <namespace>
kubectl nuke namespace <namespace>
```

### Examples

```sh
# Delete a namespace called 'my-app'
kubectl-nuke ns my-app

# Delete a namespace with custom kubeconfig
kubectl-nuke --kubeconfig ~/.kube/staging-config ns test-namespace

# Use as kubectl plugin
kubectl nuke ns stuck-namespace
```

## How it works

1. **Check namespace state**: The tool first checks the current state of the namespace
2. **Attempt normal delete**: It tries to delete the namespace using the standard Kubernetes API
3. **Handle stuck namespaces**: If the namespace is already in Terminating state or gets stuck, it removes finalizers to force deletion
4. **Wait and verify**: The tool waits for the namespace to be fully deleted and provides status updates

## Commands

### `kubectl-nuke ns <namespace>`

Delete a namespace, including those stuck in Terminating state.

**Aliases**: `namespace`

**Options**:
- `--kubeconfig string`: Path to the kubeconfig file (default: `~/.kube/config`)

### `kubectl-nuke version`

Print the version number of kubectl-nuke.

### `kubectl-nuke help`

Show help information for any command.

## Testing

Run unit tests:

```sh
go test ./internal/kube/...
```

Run all tests:

```sh
go test ./...
```

## Project Structure

- `cmd/kubectl-nuke/main.go`: CLI entry point with Cobra command structure
- `internal/kube/`: Namespace deletion logic and tests
- `cmd/`: Reserved for future CLI subcommands
- `docs/`: Documentation

## More

- [FAQ](FAQ.md)
- [Project Structure](PROJECT_STRUCTURE.md)
- [CONTRIBUTING.md](../CONTRIBUTING.md)
- [LICENSE](../LICENSE)
