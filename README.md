# kubectl-nuke-go

[![Go Reference](https://pkg.go.dev/badge/github.com/codesenju/kubectl-nuke-go.svg)](https://pkg.go.dev/github.com/codesenju/kubectl-nuke-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/codesenju/kubectl-nuke-go)](https://goreportcard.com/report/github.com/codesenju/kubectl-nuke-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A kubectl plugin to forcefully delete Kubernetes resources, including namespaces stuck in the Terminating state. It attempts a normal delete first, and if the resource is stuck, it forcefully removes finalizers.

## Features

- Delete a namespace normally 
- Detect and force-delete namespaces stuck in Terminating
- User-friendly CLI output

## Installation

### Download the Pre-built Binary

You can download the latest release for your platform from the [Releases page](https://github.com/codesenju/kubectl-nuke-go/releases). For example, to download the Darwin (macOS) AMD64 binary:

```sh
VERSION=v0.4.1
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

Open your terminal in the project directory and run:

```sh
go build -o kubectl-nuke
```

Move the binary to a directory in your $PATH (if not already):

```sh
sudo mv kubectl-nuke /usr/local/bin/
```

## Usage

```sh
# Delete a namespace
kubectl-nuke [--kubeconfig KUBECONFIG] ns <namespace>
kubectl-nuke [--kubeconfig KUBECONFIG] namespace <namespace>
```

## Using as a kubectl Plugin

After installation, you can use this tool as a kubectl plugin:

```sh
kubectl nuke [--kubeconfig KUBECONFIG] ns <namespace>
kubectl nuke [--kubeconfig KUBECONFIG] namespace <namespace>
```

kubectl will automatically detect executables named `kubectl-<plugin>` in your PATH and allow you to invoke them as `kubectl <plugin>`.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history and upgrade notes.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Release Best Practices

See [docs/RELEASE_BEST_PRACTICES.md](docs/RELEASE_BEST_PRACTICES.md) for how to write commit messages and how releases are automated.

## License

[MIT](LICENSE)
