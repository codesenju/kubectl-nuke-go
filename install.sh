#!/bin/bash

# kubectl-nuke-go installer script for Unix-like systems (macOS/Linux)
# This script automatically detects your platform and installs kubectl-nuke

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository information
REPO="codesenju/kubectl-nuke-go"
BINARY_NAME="kubectl-nuke"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to detect OS and architecture
detect_platform() {
    local os arch
    
    # Detect OS
    case "$(uname -s)" in
        Darwin*)
            os="darwin"
            ;;
        Linux*)
            os="linux"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            print_error "This script is for Unix-like systems. For Windows, use install.ps1"
            exit 1
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}-${arch}"
}

# Function to get the latest release version
get_latest_version() {
    local version
    version=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        print_error "Failed to get latest version from GitHub API"
        exit 1
    fi
    
    echo "$version"
}

# Function to determine install directory
get_install_dir() {
    local install_dir
    
    # If custom install path is provided
    if [ -n "$INSTALL_PATH" ]; then
        if [ -d "$INSTALL_PATH" ] && [ -w "$INSTALL_PATH" ]; then
            install_dir="$INSTALL_PATH"
        else
            print_error "Custom install path is not accessible: $INSTALL_PATH"
            exit 1
        fi
    # Check if user has write access to /usr/local/bin
    elif [ -w "/usr/local/bin" ] 2>/dev/null; then
        install_dir="/usr/local/bin"
    # Check if ~/.local/bin exists and is in PATH
    elif [ -d "$HOME/.local/bin" ] && echo "$PATH" | grep -q "$HOME/.local/bin"; then
        install_dir="$HOME/.local/bin"
    # Create ~/.local/bin if it doesn't exist
    elif [ ! -d "$HOME/.local/bin" ]; then
        mkdir -p "$HOME/.local/bin"
        install_dir="$HOME/.local/bin"
        print_warning "Created $HOME/.local/bin - you may need to add it to your PATH"
        print_warning "Add this to your shell profile: export PATH=\"\$HOME/.local/bin:\$PATH\""
    else
        install_dir="$HOME/.local/bin"
    fi
    
    echo "$install_dir"
}

# Function to download and install binary
install_binary() {
    local platform="$1"
    local version="$2"
    local install_dir="$3"
    local temp_dir
    
    temp_dir=$(mktemp -d)
    
    print_status "Creating temporary directory: $temp_dir"
    
    local download_url="https://github.com/${REPO}/releases/download/${version}/kubectl-nuke-go-${platform}.tar.gz"
    local download_file="${temp_dir}/kubectl-nuke-go-${platform}.tar.gz"
    
    print_status "Downloading kubectl-nuke ${version} for ${platform}..."
    print_status "URL: $download_url"
    
    if ! curl -L -o "$download_file" "$download_url"; then
        print_error "Failed to download kubectl-nuke"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    print_status "Extracting binary..."
    
    if ! tar -xzf "$download_file" -C "$temp_dir"; then
        print_error "Failed to extract archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Find the binary in the extracted files
    local binary_path="${temp_dir}/${BINARY_NAME}"
    if [ ! -f "$binary_path" ]; then
        print_error "Binary not found in extracted files"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    print_status "Installing kubectl-nuke to ${install_dir}..."
    
    # Install the binary
    local target_path="${install_dir}/${BINARY_NAME}"
    
    # Copy binary and make executable
    if ! cp "$binary_path" "$target_path"; then
        print_error "Failed to copy binary to install directory"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    chmod +x "$target_path"
    
    # Clean up
    rm -rf "$temp_dir"
    
    print_success "kubectl-nuke installed successfully to $target_path"
}

# Function to verify installation
verify_installation() {
    local install_dir="$1"
    local binary_path="${install_dir}/${BINARY_NAME}"
    
    if [ -x "$binary_path" ]; then
        print_success "Installation verified!"
        print_status "You can now use: kubectl-nuke or kubectl nuke"
        
        # Test the binary
        if "$binary_path" --help >/dev/null 2>&1; then
            print_success "Binary is working correctly"
        else
            print_warning "Binary installed but may not be working correctly"
        fi
    else
        print_error "Installation verification failed"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -p, --path PATH     Custom installation path"
    echo "  -f, --force         Force overwrite existing installation"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  INSTALL_PATH        Custom installation path (same as --path)"
    echo ""
    echo "Examples:"
    echo "  $0                           # Install to default location"
    echo "  $0 --path /usr/local/bin     # Install to specific path"
    echo "  $0 --force                   # Force overwrite existing"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -p|--path)
                INSTALL_PATH="$2"
                shift 2
                ;;
            -f|--force)
                FORCE_INSTALL=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

# Main installation function
main() {
    print_status "kubectl-nuke-go installer for Unix-like systems"
    print_status "==============================================="
    
    # Check for required commands
    for cmd in curl tar; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            print_error "$cmd is required but not installed"
            exit 1
        fi
    done
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    print_status "Detected platform: $platform"
    
    # Get latest version
    local version
    version=$(get_latest_version)
    print_status "Latest version: $version"
    
    # Determine install directory
    local install_dir
    install_dir=$(get_install_dir)
    print_status "Install directory: $install_dir"
    
    # Check if binary already exists
    local existing_binary="${install_dir}/${BINARY_NAME}"
    
    if [ -f "$existing_binary" ] && [ "$FORCE_INSTALL" != "true" ]; then
        print_warning "kubectl-nuke is already installed at $existing_binary"
        read -p "Do you want to overwrite it? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installation cancelled"
            exit 0
        fi
    fi
    
    # Install binary
    install_binary "$platform" "$version" "$install_dir"
    
    # Verify installation
    verify_installation "$install_dir"
    
    print_success "Installation complete!"
    print_status ""
    print_status "Usage:"
    print_status "  kubectl-nuke ns <namespace>     # Direct usage"
    print_status "  kubectl nuke ns <namespace>     # As kubectl plugin"
    print_status ""
    print_status "For more information, visit: https://github.com/${REPO}"
}

# Parse arguments and run main function
parse_args "$@"
main
