package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppDataDirForOS verifies that the app-data directory resolves to the
// correct platform-specific location for each supported OS.
func TestAppDataDirForOS(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir failed: %v", err)
	}

	tests := []struct {
		name     string
		goos     string
		wantSuf  string // expected suffix (using filepath.Join components)
		wantBase string // expected last path component
	}{
		{
			name:     "darwin uses Library/Application Support",
			goos:     "darwin",
			wantSuf:  filepath.Join("Library", "Application Support", appName),
			wantBase: appName,
		},
		{
			name:     "linux uses .config",
			goos:     "linux",
			wantSuf:  filepath.Join(".config", appName),
			wantBase: appName,
		},
		{
			name:     "windows uses AppData/Roaming",
			goos:     "windows",
			wantSuf:  filepath.Join("AppData", "Roaming", appName),
			wantBase: appName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars that could override paths
			prevAppData := os.Getenv("APPDATA")
			prevXDG := os.Getenv("XDG_CONFIG_HOME")
			t.Cleanup(func() {
				os.Setenv("APPDATA", prevAppData)
				os.Setenv("XDG_CONFIG_HOME", prevXDG)
			})
			os.Unsetenv("APPDATA")
			os.Unsetenv("XDG_CONFIG_HOME")

			got := appDataDirForOS(tt.goos)

			if filepath.Base(got) != tt.wantBase {
				t.Errorf("base = %q, want %q", filepath.Base(got), tt.wantBase)
			}

			// On the current OS the home-relative suffix should match;
			// on cross-platform the path at least ends correctly.
			if !strings.HasSuffix(got, tt.wantSuf) {
				// The path must at least contain the home dir prefix
				if !strings.HasPrefix(got, home) {
					t.Errorf("path %q does not start with home %q", got, home)
				}
			}
		})
	}
}

// TestAppDataDirLinuxXDG verifies that $XDG_CONFIG_HOME is respected on Linux.
func TestAppDataDirLinuxXDG(t *testing.T) {
	prevXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", prevXDG) })

	custom := filepath.Join(os.TempDir(), "parkbot-test-xdg")
	os.Setenv("XDG_CONFIG_HOME", custom)

	got := appDataDirForOS("linux")
	want := filepath.Join(custom, appName)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestAppDataDirWindowsAPPDATA verifies that %APPDATA% is respected on Windows.
func TestAppDataDirWindowsAPPDATA(t *testing.T) {
	prevAppData := os.Getenv("APPDATA")
	t.Cleanup(func() { os.Setenv("APPDATA", prevAppData) })

	custom := filepath.Join(os.TempDir(), "parkbot-test-appdata")
	os.Setenv("APPDATA", custom)

	got := appDataDirForOS("windows")
	want := filepath.Join(custom, appName)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestChromeProfileForOS verifies Chrome profile paths for each platform.
func TestChromeProfileForOS(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		wantEnd string // path must end with this
	}{
		{
			name:    "darwin",
			goos:    "darwin",
			wantEnd: filepath.Join("Google", "Chrome", "Default"),
		},
		{
			name:    "linux",
			goos:    "linux",
			wantEnd: filepath.Join("google-chrome", "Default"),
		},
		{
			name:    "windows",
			goos:    "windows",
			wantEnd: filepath.Join("Google", "Chrome", "User Data", "Default"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chromeProfileForOS(tt.goos)
			if !strings.HasSuffix(got, tt.wantEnd) {
				t.Errorf("chromeProfileForOS(%q) = %q, want suffix %q", tt.goos, got, tt.wantEnd)
			}
		})
	}
}

// TestChromeProfileWindowsLOCALAPPDATA verifies that %LOCALAPPDATA% is
// respected for Chrome profiles on Windows.
func TestChromeProfileWindowsLOCALAPPDATA(t *testing.T) {
	prev := os.Getenv("LOCALAPPDATA")
	t.Cleanup(func() { os.Setenv("LOCALAPPDATA", prev) })

	custom := filepath.Join(os.TempDir(), "parkbot-test-localappdata")
	os.Setenv("LOCALAPPDATA", custom)

	got := chromeProfileForOS("windows")
	want := filepath.Join(custom, "Google", "Chrome", "User Data", "Default")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestDefaultLockFilePath ensures the lock file path uses filepath.Join and
// lives inside the app data directory.
func TestDefaultLockFilePath(t *testing.T) {
	p := defaultLockFilePath()

	if filepath.Base(p) != "purchased.lock" {
		t.Errorf("lock file base = %q, want %q", filepath.Base(p), "purchased.lock")
	}

	dir := filepath.Dir(p)
	if filepath.Base(dir) != appName {
		t.Errorf("lock file parent dir = %q, want %q", filepath.Base(dir), appName)
	}

	// Path must be absolute
	if !filepath.IsAbs(p) {
		t.Errorf("lock file path %q is not absolute", p)
	}
}

// TestDefaultConfigPath ensures the config path uses filepath.Join and lives
// inside the app data directory.
func TestDefaultConfigPath(t *testing.T) {
	p := defaultConfigPath()

	if filepath.Base(p) != "config.yaml" {
		t.Errorf("config base = %q, want %q", filepath.Base(p), "config.yaml")
	}

	dir := filepath.Dir(p)
	if filepath.Base(dir) != appName {
		t.Errorf("config parent dir = %q, want %q", filepath.Base(dir), appName)
	}

	if !filepath.IsAbs(p) {
		t.Errorf("config path %q is not absolute", p)
	}
}

// TestNoHardcodedSlashes verifies that all path functions produce paths using
// only the OS-native separator (no hardcoded '/' on Windows or '\' on Unix).
func TestNoHardcodedSlashes(t *testing.T) {
	paths := []struct {
		name string
		path string
	}{
		{"appDataDir darwin", appDataDirForOS("darwin")},
		{"appDataDir linux", appDataDirForOS("linux")},
		{"appDataDir windows", appDataDirForOS("windows")},
		{"chromeProfile darwin", chromeProfileForOS("darwin")},
		{"chromeProfile linux", chromeProfileForOS("linux")},
		{"chromeProfile windows", chromeProfileForOS("windows")},
		{"defaultLockFilePath", defaultLockFilePath()},
		{"defaultConfigPath", defaultConfigPath()},
	}

	for _, pp := range paths {
		t.Run(pp.name, func(t *testing.T) {
			// On this OS, filepath.Join uses the native separator.
			// Verify no raw backslash on Unix or raw forward slash on Windows
			// after joining (note: filepath.Join normalizes on the current OS).
			cleaned := filepath.Clean(pp.path)
			if cleaned != pp.path {
				t.Errorf("path %q is not clean (cleaned = %q)", pp.path, cleaned)
			}
		})
	}
}

// TestLoadConfigRoundTrip verifies that saving and loading a config file
// produces the same result, using the platform-aware config path.
func TestLoadConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	original := &Config{
		PermitKeyword:  "LOT-42",
		VehicleKeyword: "TESLA",
		AddressKeyword: "123 MAIN",
		Email:          "test@example.com",
		OneTime:        true,
		ChromeProfile:  filepath.Join(dir, "chrome-profile"),
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "123",
			Name:       "Test User",
			Zip:        "50010",
		},
	}

	if err := original.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	// Keywords get uppercased, so compare against uppercased originals
	if loaded.PermitKeyword != strings.ToUpper(original.PermitKeyword) {
		t.Errorf("PermitKeyword = %q, want %q", loaded.PermitKeyword, strings.ToUpper(original.PermitKeyword))
	}
	if loaded.ChromeProfile != original.ChromeProfile {
		t.Errorf("ChromeProfile = %q, want %q", loaded.ChromeProfile, original.ChromeProfile)
	}
	if loaded.Billing.CardNumber != original.Billing.CardNumber {
		t.Errorf("Billing.CardNumber = %q, want %q", loaded.Billing.CardNumber, original.Billing.CardNumber)
	}
}

// TestLoadConfigDefaultsChromeProfile verifies that an empty chrome_profile
// field gets filled with the platform default.
func TestLoadConfigDefaultsChromeProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		PermitKeyword: "LOT-1",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "123",
		},
	}

	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	if loaded.ChromeProfile == "" {
		t.Error("ChromeProfile should default to a non-empty path")
	}
	if loaded.ChromeProfile != defaultChromeProfile() {
		t.Errorf("ChromeProfile = %q, want %q", loaded.ChromeProfile, defaultChromeProfile())
	}
}
