package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	githubAPIURL = "https://api.github.com/repos/codesenju/kubectl-nuke-go/releases/latest"
	timeout      = 30 * time.Second
)

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	Body string `json:"body"`
}

// UpdateChecker handles version checking and updates
type UpdateChecker struct {
	currentVersion string
	repoOwner      string
	repoName       string
}

// NewUpdateChecker creates a new update checker
func NewUpdateChecker(currentVersion string) *UpdateChecker {
	return &UpdateChecker{
		currentVersion: currentVersion,
		repoOwner:      "codesenju",
		repoName:       "kubectl-nuke-go",
	}
}

// CheckForUpdate checks if a newer version is available
func (u *UpdateChecker) CheckForUpdate() (*Release, bool, error) {
	fmt.Printf("🔍 Checking for updates...\n")
	
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(githubAPIURL)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read response: %w", err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, false, fmt.Errorf("failed to parse release info: %w", err)
	}

	// Compare versions
	hasUpdate, err := u.isNewerVersion(release.TagName)
	if err != nil {
		return nil, false, fmt.Errorf("failed to compare versions: %w", err)
	}

	return &release, hasUpdate, nil
}

// isNewerVersion compares the current version with the latest release
func (u *UpdateChecker) isNewerVersion(latestTag string) (bool, error) {
	// Handle dev version
	if u.currentVersion == "dev" {
		return true, nil
	}

	// Clean version strings (remove 'v' prefix if present)
	currentVer := strings.TrimPrefix(u.currentVersion, "v")
	latestVer := strings.TrimPrefix(latestTag, "v")

	current, err := semver.NewVersion(currentVer)
	if err != nil {
		return false, fmt.Errorf("invalid current version %s: %w", currentVer, err)
	}

	latest, err := semver.NewVersion(latestVer)
	if err != nil {
		return false, fmt.Errorf("invalid latest version %s: %w", latestVer, err)
	}

	return latest.GreaterThan(current), nil
}

// PerformUpdate downloads and installs the latest version
func (u *UpdateChecker) PerformUpdate(release *Release, force bool) error {
	// Find the appropriate asset for current platform
	assetName := u.GetAssetName()
	var downloadURL string
	
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no suitable binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("📥 Downloading %s...\n", assetName)
	
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Create temporary directory for download and extraction
	tmpDir, err := os.MkdirTemp("", "kubectl-nuke-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Download the archive
	archivePath := filepath.Join(tmpDir, assetName)
	if err := u.downloadFile(downloadURL, archivePath); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Extract the binary from the archive
	binaryPath, err := u.extractBinary(archivePath, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	// Make it executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Test the new binary
	if !force {
		fmt.Printf("🧪 Testing new binary...\n")
		cmd := exec.Command(binaryPath, "version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("new binary failed validation: %w", err)
		}
	}

	// Replace current binary
	fmt.Printf("🔄 Installing update...\n")
	
	// Create backup
	backupPath := currentExe + ".backup"
	if err := copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace the binary
	if err := copyFile(binaryPath, currentExe); err != nil {
		// Restore backup on failure
		copyFile(backupPath, currentExe)
		os.Remove(backupPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Cleanup
	os.Remove(backupPath)

	fmt.Printf("✅ Successfully updated to version %s!\n", release.TagName)
	fmt.Printf("📝 Release notes:\n%s\n", release.Body)
	
	return nil
}

// GetAssetName returns the expected asset name for the current platform
func (u *UpdateChecker) GetAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	
	// Keep original Go arch names (don't map to x86_64)
	// The actual release assets use amd64, not x86_64
	
	// Match the actual release asset naming pattern
	switch goos {
	case "darwin", "linux":
		return fmt.Sprintf("kubectl-nuke-go-%s-%s.tar.gz", goos, goarch)
	case "windows":
		return fmt.Sprintf("kubectl-nuke-go-%s-%s.zip", goos, goarch)
	default:
		return fmt.Sprintf("kubectl-nuke-go-%s-%s.tar.gz", goos, goarch)
	}
}

// downloadFile downloads a file from URL to local path
func (u *UpdateChecker) downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	return os.Chmod(dst, sourceInfo.Mode())
}

// extractBinary extracts the kubectl-nuke binary from a tar.gz or zip archive
func (u *UpdateChecker) extractBinary(archivePath, extractDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".tar.gz") {
		return u.extractFromTarGz(archivePath, extractDir)
	} else if strings.HasSuffix(archivePath, ".zip") {
		return u.extractFromZip(archivePath, extractDir)
	}
	return "", fmt.Errorf("unsupported archive format: %s", archivePath)
}

// extractFromTarGz extracts the binary from a tar.gz archive
func (u *UpdateChecker) extractFromTarGz(archivePath, extractDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Look for the kubectl-nuke binary (without extension)
		if header.Typeflag == tar.TypeReg && (header.Name == "kubectl-nuke" || strings.HasSuffix(header.Name, "/kubectl-nuke")) {
			binaryPath := filepath.Join(extractDir, "kubectl-nuke")
			outFile, err := os.Create(binaryPath)
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tr)
			if err != nil {
				return "", err
			}

			return binaryPath, nil
		}
	}

	return "", fmt.Errorf("kubectl-nuke binary not found in archive")
}

// extractFromZip extracts the binary from a zip archive
func (u *UpdateChecker) extractFromZip(archivePath, extractDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		// Look for the kubectl-nuke binary (with or without .exe extension)
		if f.Name == "kubectl-nuke" || f.Name == "kubectl-nuke.exe" || strings.HasSuffix(f.Name, "/kubectl-nuke") || strings.HasSuffix(f.Name, "/kubectl-nuke.exe") {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			// Determine output filename (keep .exe for Windows)
			outputName := "kubectl-nuke"
			if runtime.GOOS == "windows" {
				outputName = "kubectl-nuke.exe"
			}
			
			binaryPath := filepath.Join(extractDir, outputName)
			outFile, err := os.Create(binaryPath)
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			if err != nil {
				return "", err
			}

			return binaryPath, nil
		}
	}

	return "", fmt.Errorf("kubectl-nuke binary not found in archive")
}