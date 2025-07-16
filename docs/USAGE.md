# kubectl-nuke Documentation

## Overview

`kubectl-nuke` is a kubectl plugin for forcefully deleting Kubernetes resources, including namespaces stuck in the Terminating state and unresponsive pods. It provides both gentle and aggressive deletion modes, making it essential for cluster administrators and DevOps engineers who need to clean up stuck resources.

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

### Namespace Deletion

#### Standard Mode
```sh
# Delete a namespace using the 'ns' subcommand
kubectl-nuke ns <namespace>

# Delete a namespace using the 'namespace' subcommand
kubectl-nuke namespace <namespace>

# Use with custom kubeconfig
kubectl-nuke --kubeconfig /path/to/config ns <namespace>
```

#### Force Mode (Aggressive Deletion)
```sh
# Aggressively delete a namespace and all its contents
kubectl-nuke ns <namespace> --force
kubectl-nuke ns <namespace> -f

# Force mode with custom kubeconfig
kubectl-nuke --kubeconfig /path/to/config ns <namespace> --force
```

### Pod Force Deletion

```sh
# Force delete a single pod with grace period 0
kubectl-nuke pod <pod-name> -n <namespace>

# Force delete multiple pods
kubectl-nuke pods <pod1> <pod2> <pod3> -n <namespace>

# Using the 'po' alias (like kubectl)
kubectl-nuke po <pod-name> -n <namespace>

# Force delete pods in default namespace
kubectl-nuke pod <pod-name>
```

### As a kubectl Plugin

After installation, you can use this tool as a kubectl plugin:

```sh
# Namespace operations
kubectl nuke ns <namespace>
kubectl nuke namespace <namespace> --force

# Pod operations
kubectl nuke pod <pod-name> -n <namespace>
kubectl nuke pods <pod1> <pod2> -n <namespace>
```

### Examples

```sh
# Delete a namespace called 'my-app' (standard mode)
kubectl-nuke ns my-app

# Aggressively delete a namespace and all resources
kubectl-nuke ns test-environment --force

# Force delete stuck pods
kubectl-nuke pods nginx-123 redis-456 -n production

# Delete a namespace with custom kubeconfig
kubectl-nuke --kubeconfig ~/.kube/staging-config ns test-namespace

# Use as kubectl plugin for force deletion
kubectl nuke ns stuck-namespace -f

# Clean up multiple stuck pods
kubectl nuke po pod1 pod2 pod3 -n my-namespace
```

## How it works

### Namespace Deletion (Standard Mode)
1. **Check namespace state**: The tool first checks the current state of the namespace
2. **Attempt normal delete**: It tries to delete the namespace using the standard Kubernetes API
3. **Handle stuck namespaces**: If the namespace is already in Terminating state or gets stuck, it removes finalizers to force deletion
4. **Wait and verify**: The tool waits for the namespace to be fully deleted and provides status updates

### Namespace Deletion (Force Mode)
1. **Aggressive resource cleanup**: Force deletes all pods with grace period 0
2. **Delete common resources**: Removes services, deployments, replicasets, configmaps, secrets
3. **Multiple finalizer strategies**: Uses standard removal, aggressive patching, and direct spec modification
4. **Extended monitoring**: Waits up to 30 seconds for complete deletion with progress updates

### Pod Force Deletion
1. **Validation**: Checks if specified pods exist in the target namespace
2. **Immediate termination**: Deletes pods with grace period 0 (no graceful shutdown)
3. **Batch processing**: Handles multiple pods efficiently with detailed status reporting
4. **Error handling**: Continues processing remaining pods even if some fail

## Commands

### `kubectl-nuke ns <namespace>`

Delete a namespace, including those stuck in Terminating state.

**Aliases**: `namespace`

**Options**:
- `--force, -f`: Aggressively delete all resources in the namespace first (DESTRUCTIVE)
- `--kubeconfig string`: Path to the kubeconfig file (default: `~/.kube/config`)

**Examples**:
```sh
# Standard namespace deletion
kubectl-nuke ns my-namespace

# Force mode - aggressive deletion
kubectl-nuke ns my-namespace --force
kubectl-nuke ns my-namespace -f
```

### `kubectl-nuke pod <pod-name> [pod-name2] [pod-name3]...`

Force delete one or more pods with grace period 0 (immediate termination).

**Aliases**: `pods`, `po`

**Options**:
- `--namespace, -n string`: Namespace of the pods (default: "default")
- `--kubeconfig string`: Path to the kubeconfig file (default: `~/.kube/config`)

**Examples**:
```sh
# Force delete a single pod
kubectl-nuke pod my-pod -n my-namespace

# Force delete multiple pods
kubectl-nuke pods pod1 pod2 pod3 -n my-namespace

# Using the 'po' alias
kubectl-nuke po stuck-pod -n production
```

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
