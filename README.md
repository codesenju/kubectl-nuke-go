# kubectl-nuke-go

<!-- [![Go Reference](https://pkg.go.dev/badge/github.com/codesenju/kubectl-nuke-go.svg)](https://pkg.go.dev/github.com/codesenju/kubectl-nuke-go) -->
[![Go Report Card](https://goreportcard.com/badge/github.com/codesenju/kubectl-nuke-go)](https://goreportcard.com/report/github.com/codesenju/kubectl-nuke-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![kuebctl-nuke-go-demo](./media/gif/demo-kubectl-nuke-go.gif)

A kubectl plugin to forcefully delete Kubernetes resources, including namespaces stuck in the Terminating state and unresponsive pods. It provides both gentle and aggressive deletion modes to handle stuck resources.

## Features

- **Namespace Deletion**: Delete namespaces with automatic finalizer removal for stuck resources
- **Force Mode**: Aggressively delete all resources in a namespace before deletion (`--force` flag)
- **Diagnostic Mode**: Analyze namespace issues without making changes (`--diagnose-only` flag)
- **Pod Force Deletion**: Force delete individual pods with grace period 0
- **Multiple Resource Support**: Handles pods, services, deployments, configmaps, secrets, and more
- **Smart Finalizer Removal**: Multiple strategies for removing stubborn finalizers
- **ArgoCD Integration**: Detects and handles ArgoCD-managed resources properly
- **User-friendly CLI**: Clear status messages with emoji indicators
- **kubectl Plugin Compatible**: Works as both standalone binary and kubectl plugin

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

### Namespace Deletion

```sh
# Standard namespace deletion (standalone binary)
kubectl-nuke ns <namespace>
kubectl-nuke namespace <namespace>

# Standard namespace deletion (as kubectl plugin)
kubectl nuke ns <namespace>
kubectl nuke namespace <namespace>

# Force mode - aggressively delete all resources first
kubectl-nuke ns <namespace> --force
kubectl-nuke ns <namespace> -f

# Diagnostic mode - analyze issues without making changes
kubectl-nuke ns <namespace> --diagnose-only

# With custom kubeconfig
kubectl-nuke --kubeconfig /path/to/config ns <namespace>
kubectl nuke --kubeconfig /path/to/config ns <namespace> --force
```

### Pod Force Deletion

```sh
# Force delete a single pod (grace period 0)
kubectl-nuke pod <pod-name> -n <namespace>

# Force delete multiple pods
kubectl-nuke pods <pod1> <pod2> <pod3> -n <namespace>

# Using the 'po' alias (like kubectl)
kubectl-nuke po <pod-name> -n <namespace>

# As kubectl plugin
kubectl nuke pod <pod-name> -n <namespace>
kubectl nuke pods <pod1> <pod2> -n <namespace>
```

### Command Examples

```sh
# Delete a stuck namespace normally
kubectl-nuke ns my-stuck-namespace

# Aggressively delete a namespace and all its contents
kubectl-nuke ns my-namespace --force

# Analyze namespace issues without making changes
kubectl-nuke ns my-namespace --diagnose-only

# Force delete unresponsive pods
kubectl-nuke pods nginx-123 redis-456 -n production

# Clean up test environment completely
kubectl-nuke ns test-env -f
```

## Using as a kubectl Plugin

After installation, you can use this tool as a kubectl plugin. kubectl will automatically detect executables named `kubectl-<plugin>` in your PATH and allow you to invoke them as `kubectl <plugin>`:

```sh
# These commands are equivalent:
kubectl-nuke ns my-namespace
kubectl nuke ns my-namespace

# Force mode works the same way:
kubectl-nuke ns my-namespace --force
kubectl nuke ns my-namespace -f

# Pod deletion also works as a plugin:
kubectl-nuke pod stuck-pod -n my-namespace
kubectl nuke pods pod1 pod2 -n my-namespace

# All support custom kubeconfig:
kubectl nuke --kubeconfig /path/to/config ns my-namespace --force
```

### Enhanced Diagnostics with ArgoCD Support

```bash
# Comprehensive diagnostics including ArgoCD analysis
kubectl-nuke ns my-namespace --diagnose-only
```

When ArgoCD applications are detected, you'll see detailed information:
- ArgoCD applications managing the namespace
- Application sync and health status
- Specific cleanup recommendations
- Proper deletion sequence

## ArgoCD Integration

kubectl-nuke now provides enhanced support for namespaces containing ArgoCD-managed resources:

- **Automatic Detection**: Identifies ArgoCD Applications targeting the namespace
- **Smart Cleanup**: Deletes ArgoCD Applications first to prevent reconciliation conflicts  
- **Enhanced Diagnostics**: Shows detailed ArgoCD application status and recommendations
- **Finalizer Handling**: Properly removes ArgoCD finalizers from stuck applications

For detailed information about ArgoCD integration, see [docs/ARGOCD_INTEGRATION.md](docs/ARGOCD_INTEGRATION.md).

## Command Reference

| Command | Description | Example |
|---------|-------------|---------|
| `ns\|namespace <name>` | Delete a namespace (standard mode) | `kubectl-nuke ns my-namespace` |
| `ns\|namespace <name> -f` | Aggressively delete namespace and all contents | `kubectl-nuke ns my-namespace --force` |
| `ns\|namespace <name> --diagnose-only` | Analyze namespace issues without deletion | `kubectl-nuke ns my-namespace --diagnose-only` |
| `pod\|pods\|po <name>...` | Force delete pods with grace period 0 | `kubectl-nuke pods pod1 pod2 -n my-ns` |
| `version` | Show version information | `kubectl-nuke version` |
| `help` | Show help for any command | `kubectl-nuke help ns` |

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

## Releases

This project uses automated releases with both stable releases and prereleases:

### Stable Releases
- **Production-ready** versions (e.g., `v1.2.3`)
- **Full distribution** via GitHub releases, Homebrew, and package managers
- **Comprehensive testing** and documentation

### Prereleases  
- **Testing versions** for new features (e.g., `v1.2.3-beta.1`, `v1.2.3-rc.1`)
- **GitHub releases only** (no package manager updates)
- **Early access** to new functionality

### Installation from Prereleases
```bash
# Download prerelease binaries from GitHub releases
VERSION="v1.2.3-beta.1"  # Replace with desired prerelease version
curl -LO https://github.com/codesenju/kubectl-nuke-go/releases/download/$VERSION/kubectl-nuke-go-linux-amd64.tar.gz
```

See [docs/RELEASE_BEST_PRACTICES.md](docs/RELEASE_BEST_PRACTICES.md) for detailed release workflow and commit message guidelines.

## License

[MIT](LICENSE)
