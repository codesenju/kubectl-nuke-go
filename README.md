# kubectl-nuke-go

<!-- [![Go Reference](https://pkg.go.dev/badge/github.com/codesenju/kubectl-nuke-go.svg)](https://pkg.go.dev/github.com/codesenju/kubectl-nuke-go) -->
[![Go Report Card](https://goreportcard.com/badge/github.com/codesenju/kubectl-nuke-go)](https://goreportcard.com/report/github.com/codesenju/kubectl-nuke-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![kuebctl-nuke-go-demo](./media/gif/demo-kubectl-nuke-go.gif)

A kubectl plugin to forcefully delete Kubernetes resources, including namespaces stuck in the Terminating state. It attempts a normal delete first, and if the resource is stuck, it forcefully removes finalizers.

## Features

- Delete a namespace normally 
- Detect and force-delete namespaces stuck in Terminating
- User-friendly CLI output

## Installation

### Quick Install Script (Recommended)

The easiest way to install kubectl-nuke on any platform:

#### Unix-like Systems (macOS/Linux)

```sh
# Download and run the install script
curl -fsSL https://raw.githubusercontent.com/codesenju/kubectl-nuke-go/main/install.sh | bash

# Or download first, then run
curl -fsSL https://raw.githubusercontent.com/codesenju/kubectl-nuke-go/main/install.sh -o install.sh
chmod +x install.sh
./install.sh

# Install to custom path
./install.sh --path /usr/local/bin

# Force overwrite existing installation
./install.sh --force
```

#### Windows (PowerShell)

```powershell
# Download and run the PowerShell install script
Invoke-Expression (Invoke-WebRequest -Uri "https://raw.githubusercontent.com/codesenju/kubectl-nuke-go/main/install.ps1" -UseBasicParsing).Content

# Or download first, then run
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/codesenju/kubectl-nuke-go/main/install.ps1" -OutFile "install.ps1"
.\install.ps1

# Install to custom path
.\install.ps1 -InstallPath "C:\Program Files\kubectl-nuke"

# Force overwrite existing installation
.\install.ps1 -Force
```

### Homebrew (macOS/Linux)

Alternative installation method for macOS and Linux:

```sh
brew tap codesenju/kubectl-nuke
brew install kubectl-nuke
```

### Download Pre-built Binary (Manual Installation)

If you prefer to download and install manually, you can get the latest release for your platform from the [Releases page](https://github.com/codesenju/kubectl-nuke-go/releases). Available platforms:

- **macOS**: `kubectl-nuke-go-darwin-amd64.tar.gz` (Intel) / `kubectl-nuke-go-darwin-arm64.tar.gz` (Apple Silicon)
- **Linux**: `kubectl-nuke-go-linux-amd64.tar.gz` (Intel/AMD) / `kubectl-nuke-go-linux-arm64.tar.gz` (ARM)
- **Windows**: `kubectl-nuke-go-windows-amd64.zip` (Intel/AMD) / `kubectl-nuke-go-windows-arm64.zip` (ARM)

#### Example for macOS (Intel):

```sh
# Get the latest version
VERSION=$(curl -s https://api.github.com/repos/codesenju/kubectl-nuke-go/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# Download the binary
curl -LO https://github.com/codesenju/kubectl-nuke-go/releases/download/$VERSION/kubectl-nuke-go-darwin-amd64.tar.gz

# Extract the binary
tar -xzf kubectl-nuke-go-darwin-amd64.tar.gz

# Move to PATH and make executable
sudo mv kubectl-nuke /usr/local/bin/kubectl-nuke
chmod +x /usr/local/bin/kubectl-nuke
```

#### Example for Windows:

```powershell
# Download the latest Windows release
$VERSION = (Invoke-RestMethod -Uri "https://api.github.com/repos/codesenju/kubectl-nuke-go/releases/latest").tag_name
Invoke-WebRequest -Uri "https://github.com/codesenju/kubectl-nuke-go/releases/download/$VERSION/kubectl-nuke-go-windows-amd64.zip" -OutFile "kubectl-nuke-go-windows-amd64.zip"

# Extract and install (you'll need to add the directory to your PATH)
Expand-Archive -Path "kubectl-nuke-go-windows-amd64.zip" -DestinationPath "C:\Program Files\kubectl-nuke"
```

### Build from Source

Open your terminal in the project directory and run:

```sh
go build -o kubectl-nuke ./cmd/kubectl-nuke
```

Move the binary to a directory in your $PATH (if not already):

```sh
sudo mv kubectl-nuke /usr/local/bin/
```

## Usage

```sh
# Delete a namespace (standalone binary)
kubectl-nuke [--kubeconfig KUBECONFIG] ns <namespace>
kubectl-nuke [--kubeconfig KUBECONFIG] namespace <namespace>

# Delete a namespace (as kubectl plugin)
kubectl nuke [--kubeconfig KUBECONFIG] ns <namespace>
kubectl nuke [--kubeconfig KUBECONFIG] namespace <namespace>
```

## Using as a kubectl Plugin

After installation, you can use this tool as a kubectl plugin. kubectl will automatically detect executables named `kubectl-<plugin>` in your PATH and allow you to invoke them as `kubectl <plugin>`:

```sh
# These commands are equivalent:
kubectl-nuke ns my-namespace
kubectl nuke ns my-namespace

# Both support all the same options:
kubectl nuke --kubeconfig /path/to/config ns my-namespace
```

## Uninstallation

### Quick Install Script Installation

If you installed using the quick install script, the binary is typically located at:
- **Unix-like systems**: `~/.local/bin/kubectl-nuke` (default) or your specified path
- **Windows**: `%USERPROFILE%\.local\bin\kubectl-nuke.exe` (default) or your specified path

To uninstall:

#### Unix-like Systems (macOS/Linux)

```sh
# Remove the binary (default location)
rm ~/.local/bin/kubectl-nuke

# If you installed to a custom path, remove from that location
# sudo rm /usr/local/bin/kubectl-nuke

# Verify removal
which kubectl-nuke
```

#### Windows (PowerShell)

```powershell
# Remove the binary (default location)
Remove-Item "$env:USERPROFILE\.local\bin\kubectl-nuke.exe"

# If you installed to a custom path, remove from that location
# Remove-Item "C:\Program Files\kubectl-nuke\kubectl-nuke.exe"

# Verify removal
Get-Command kubectl-nuke -ErrorAction SilentlyContinue
```

### Homebrew Installation

```sh
brew uninstall kubectl-nuke
brew untap codesenju/kubectl-nuke
```

### Manual Installation

If you manually downloaded and installed the binary, remove it from wherever you placed it:

```sh
# Find the binary location
which kubectl-nuke

# Remove it (example locations)
sudo rm /usr/local/bin/kubectl-nuke
# or
rm ~/bin/kubectl-nuke
```

### Build from Source

If you built from source, remove the binary from where you placed it:

```sh
sudo rm /usr/local/bin/kubectl-nuke
```

### Cleaning Up PATH (Optional)

If you added `~/.local/bin` to your PATH specifically for kubectl-nuke and want to remove it:

#### Unix-like Systems

```sh
# Edit your shell configuration file
nano ~/.bashrc  # or ~/.zshrc for zsh

# Remove or comment out the line:
# export PATH="$HOME/.local/bin:$PATH"

# Reload your shell configuration
source ~/.bashrc  # or source ~/.zshrc
```

#### Windows

Remove the PATH entry through System Properties > Environment Variables, or if you added it via PowerShell profile:

```powershell
# Edit your PowerShell profile
notepad $PROFILE

# Remove the line that adds kubectl-nuke to PATH
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history and upgrade notes.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Release Best Practices

See [docs/RELEASE_BEST_PRACTICES.md](docs/RELEASE_BEST_PRACTICES.md) for how to write commit messages and how releases are automated.

## License

[MIT](LICENSE)
