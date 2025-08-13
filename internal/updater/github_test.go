package updater

import (
	"testing"
)

func TestNewUpdateChecker(t *testing.T) {
	checker := NewUpdateChecker("v1.0.0")
	if checker.currentVersion != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", checker.currentVersion)
	}
	if checker.repoOwner != "codesenju" {
		t.Errorf("Expected repo owner codesenju, got %s", checker.repoOwner)
	}
	if checker.repoName != "kubectl-nuke-go" {
		t.Errorf("Expected repo name kubectl-nuke-go, got %s", checker.repoName)
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
		hasError bool
	}{
		{"v1.0.0", "v1.0.1", true, false},
		{"v1.0.1", "v1.0.0", false, false},
		{"v1.0.0", "v1.0.0", false, false},
		{"dev", "v1.0.0", true, false},
		{"v1.0.0", "v2.0.0", true, false},
		{"v2.0.0", "v1.0.0", false, false},
	}

	for _, test := range tests {
		checker := NewUpdateChecker(test.current)
		result, err := checker.isNewerVersion(test.latest)
		
		if test.hasError && err == nil {
			t.Errorf("Expected error for current=%s, latest=%s", test.current, test.latest)
		}
		if !test.hasError && err != nil {
			t.Errorf("Unexpected error for current=%s, latest=%s: %v", test.current, test.latest, err)
		}
		if result != test.expected {
			t.Errorf("For current=%s, latest=%s: expected %v, got %v", test.current, test.latest, test.expected, result)
		}
	}
}

func TestGetAssetName(t *testing.T) {
	checker := NewUpdateChecker("v1.0.0")
	assetName := checker.getAssetName()
	
	// The asset name should contain the platform info
	if assetName == "" {
		t.Error("Asset name should not be empty")
	}
	
	// Should contain kubectl-nuke
	if len(assetName) < 10 {
		t.Errorf("Asset name seems too short: %s", assetName)
	}
}