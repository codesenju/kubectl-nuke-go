# kubectl-nuke-go Windows PowerShell installer script
# This script automatically downloads and installs kubectl-nuke for Windows

param(
    [string]$InstallPath = "",
    [switch]$Force
)

# GitHub repository information
$REPO = "codesenju/kubectl-nuke-go"
$BINARY_NAME = "kubectl-nuke"

# Function to write colored output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Function to detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Function to get the latest release version
function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
        return $response.tag_name
    }
    catch {
        Write-Error "Failed to get latest version from GitHub API: $($_.Exception.Message)"
        exit 1
    }
}

# Function to determine install directory
function Get-InstallDirectory {
    param([string]$CustomPath)
    
    if ($CustomPath) {
        if (Test-Path $CustomPath) {
            return $CustomPath
        }
        else {
            Write-Error "Custom install path does not exist: $CustomPath"
            exit 1
        }
    }
    
    # Check common installation directories
    $possiblePaths = @(
        "$env:LOCALAPPDATA\Microsoft\WindowsApps",
        "$env:ProgramFiles\kubectl-nuke",
        "$env:USERPROFILE\.local\bin"
    )
    
    foreach ($path in $possiblePaths) {
        if (Test-Path $path) {
            # Check if directory is writable
            try {
                $testFile = Join-Path $path "test_write_$(Get-Random).tmp"
                New-Item -Path $testFile -ItemType File -Force | Out-Null
                Remove-Item $testFile -Force
                return $path
            }
            catch {
                continue
            }
        }
    }
    
    # Create user local bin directory
    $localBin = "$env:USERPROFILE\.local\bin"
    if (-not (Test-Path $localBin)) {
        New-Item -Path $localBin -ItemType Directory -Force | Out-Null
        Write-Warning "Created $localBin - you may need to add it to your PATH"
        Write-Warning "To add to PATH, run: `$env:PATH += ';$localBin'"
    }
    
    return $localBin
}

# Function to download and install binary
function Install-Binary {
    param(
        [string]$Architecture,
        [string]$Version,
        [string]$InstallDir
    )
    
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    
    Write-Status "Creating temporary directory: $($tempDir.FullName)"
    
    $platform = "windows-$Architecture"
    $downloadUrl = "https://github.com/$REPO/releases/download/$Version/kubectl-nuke-go-$platform.zip"
    $downloadFile = Join-Path $tempDir.FullName "kubectl-nuke-go-$platform.zip"
    
    Write-Status "Downloading kubectl-nuke $Version for $platform..."
    Write-Status "URL: $downloadUrl"
    
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $downloadFile -UseBasicParsing
    }
    catch {
        Write-Error "Failed to download kubectl-nuke: $($_.Exception.Message)"
        Remove-Item $tempDir -Recurse -Force
        exit 1
    }
    
    Write-Status "Extracting binary..."
    
    try {
        Expand-Archive -Path $downloadFile -DestinationPath $tempDir.FullName -Force
    }
    catch {
        Write-Error "Failed to extract archive: $($_.Exception.Message)"
        Remove-Item $tempDir -Recurse -Force
        exit 1
    }
    
    # Find the binary in the extracted files
    $binaryPath = Join-Path $tempDir.FullName "$BINARY_NAME.exe"
    if (-not (Test-Path $binaryPath)) {
        Write-Error "Binary not found in extracted files"
        Remove-Item $tempDir -Recurse -Force
        exit 1
    }
    
    Write-Status "Installing kubectl-nuke to $InstallDir..."
    
    # Ensure install directory exists
    if (-not (Test-Path $InstallDir)) {
        New-Item -Path $InstallDir -ItemType Directory -Force | Out-Null
    }
    
    $targetPath = Join-Path $InstallDir "$BINARY_NAME.exe"
    
    try {
        Copy-Item $binaryPath $targetPath -Force
    }
    catch {
        Write-Error "Failed to copy binary to install directory: $($_.Exception.Message)"
        Remove-Item $tempDir -Recurse -Force
        exit 1
    }
    
    # Clean up
    Remove-Item $tempDir -Recurse -Force
    
    Write-Success "kubectl-nuke installed successfully to $targetPath"
    return $targetPath
}

# Function to verify installation
function Test-Installation {
    param([string]$BinaryPath)
    
    if (Test-Path $BinaryPath) {
        Write-Success "Installation verified!"
        Write-Status "You can now use: kubectl-nuke or kubectl nuke"
        
        # Test the binary
        try {
            & $BinaryPath --help | Out-Null
            Write-Success "Binary is working correctly"
        }
        catch {
            Write-Warning "Binary installed but may not be working correctly"
        }
    }
    else {
        Write-Error "Installation verification failed"
        exit 1
    }
}

# Function to check if binary is in PATH
function Test-PathAccess {
    param([string]$InstallDir)
    
    $pathDirs = $env:PATH -split ';'
    $isInPath = $pathDirs -contains $InstallDir
    
    if (-not $isInPath) {
        Write-Warning "Install directory is not in your PATH"
        Write-Status "To add to PATH for current session, run:"
        Write-Status "`$env:PATH += ';$InstallDir'"
        Write-Status ""
        Write-Status "To add permanently, add the directory to your system PATH environment variable"
    }
}

# Main installation function
function Main {
    Write-Status "kubectl-nuke-go Windows PowerShell installer"
    Write-Status "============================================="
    
    # Detect architecture
    $architecture = Get-Architecture
    Write-Status "Detected architecture: $architecture"
    
    # Get latest version
    $version = Get-LatestVersion
    Write-Status "Latest version: $version"
    
    # Determine install directory
    $installDir = Get-InstallDirectory -CustomPath $InstallPath
    Write-Status "Install directory: $installDir"
    
    # Check if binary already exists
    $existingBinary = Join-Path $installDir "$BINARY_NAME.exe"
    
    if ((Test-Path $existingBinary) -and -not $Force) {
        Write-Warning "kubectl-nuke is already installed at $existingBinary"
        $response = Read-Host "Do you want to overwrite it? (y/N)"
        if ($response -notmatch '^[Yy]$') {
            Write-Status "Installation cancelled"
            exit 0
        }
    }
    
    # Install binary
    $binaryPath = Install-Binary -Architecture $architecture -Version $version -InstallDir $installDir
    
    # Verify installation
    Test-Installation -BinaryPath $binaryPath
    
    # Check PATH access
    Test-PathAccess -InstallDir $installDir
    
    Write-Success "Installation complete!"
    Write-Status ""
    Write-Status "Usage:"
    Write-Status "  kubectl-nuke ns <namespace>     # Direct usage"
    Write-Status "  kubectl nuke ns <namespace>     # As kubectl plugin"
    Write-Status ""
    Write-Status "For more information, visit: https://github.com/$REPO"
}

# Run main function
try {
    Main
}
catch {
    Write-Error "Installation failed: $($_.Exception.Message)"
    exit 1
}
