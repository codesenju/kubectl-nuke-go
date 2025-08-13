package updater

import (
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
	assetName := u.getAssetName()
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

	// Create temporary file for download
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "kubectl-nuke-new")
	
	// Download the new binary
	if err := u.downloadFile(downloadURL, tmpFile); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Make it executable
	if err := os.Chmod(tmpFile, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Test the new binary
	if !force {
		fmt.Printf("🧪 Testing new binary...\n")
		cmd := exec.Command(tmpFile, "version")
		if err := cmd.Run(); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("new binary failed validation: %w", err)
		}
	}

	// Replace current binary
	fmt.Printf("🔄 Installing update...\n")
	
	// Create backup
	backupPath := currentExe + ".backup"
	if err := copyFile(currentExe, backupPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace the binary
	if err := copyFile(tmpFile, currentExe); err != nil {
		// Restore backup on failure
		copyFile(backupPath, currentExe)
		os.Remove(tmpFile)
		os.Remove(backupPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Cleanup
	os.Remove(tmpFile)
	os.Remove(backupPath)

	fmt.Printf("✅ Successfully updated to version %s!\n", release.TagName)
	fmt.Printf("📝 Release notes:\n%s\n", release.Body)
	
	return nil
}

// getAssetName returns the expected asset name for the current platform
func (u *UpdateChecker) getAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	
	// Map Go arch names to common naming conventions
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "arm64"
	}
	
	// Common binary naming patterns
	switch goos {
	case "darwin":
		return fmt.Sprintf("kubectl-nuke-darwin-%s", goarch)
	case "linux":
		return fmt.Sprintf("kubectl-nuke-linux-%s", goarch)
	case "windows":
		return fmt.Sprintf("kubectl-nuke-windows-%s.exe", goarch)
	default:
		return fmt.Sprintf("kubectl-nuke-%s-%s", goos, goarch)
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