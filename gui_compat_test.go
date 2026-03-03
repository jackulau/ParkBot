package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func TestDefaultChromeProfile_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test on non-Linux platform")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	profile := defaultChromeProfile()
	if profile == "" {
		t.Fatal("defaultChromeProfile() returned empty string")
	}

	// On a fresh system, it should at least contain the home directory
	if !filepath.IsAbs(profile) {
		t.Errorf("profile path is not absolute: %q", profile)
	}

	// Should contain "Default" as the last component
	if filepath.Base(profile) != "Default" {
		t.Errorf("expected profile to end with 'Default', got %q", filepath.Base(profile))
	}

	// The path should be under the home directory
	if len(profile) <= len(home) {
		t.Errorf("profile path %q should be longer than home %q", profile, home)
	}
}

func TestLinuxChromeProfile_XDGConfigHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test on non-Linux platform")
	}

	// Create a temp dir simulating XDG_CONFIG_HOME with a Chrome profile
	tmpDir := t.TempDir()
	chromeDir := filepath.Join(tmpDir, "google-chrome", "Default")
	if err := os.MkdirAll(chromeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Set XDG_CONFIG_HOME and restore after
	old := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", old)

	home, _ := os.UserHomeDir()
	profile := linuxChromeProfile(home)

	expected := filepath.Join(tmpDir, "google-chrome", "Default")
	if profile != expected {
		t.Errorf("expected %q, got %q", expected, profile)
	}
}

func TestLinuxChromeProfile_ChromiumFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test on non-Linux platform")
	}

	// Create a temp dir with only Chromium present
	tmpDir := t.TempDir()
	chromiumDir := filepath.Join(tmpDir, "chromium", "Default")
	if err := os.MkdirAll(chromiumDir, 0755); err != nil {
		t.Fatal(err)
	}

	old := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", old)

	home, _ := os.UserHomeDir()
	profile := linuxChromeProfile(home)

	expected := filepath.Join(tmpDir, "chromium", "Default")
	if profile != expected {
		t.Errorf("expected Chromium fallback %q, got %q", expected, profile)
	}
}

func TestLinuxChromeProfile_DefaultFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test on non-Linux platform")
	}

	// Use a temp dir with nothing installed — should return default path
	tmpDir := t.TempDir()

	old := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", old)

	home, _ := os.UserHomeDir()
	profile := linuxChromeProfile(home)

	expected := filepath.Join(tmpDir, "google-chrome", "Default")
	if profile != expected {
		t.Errorf("expected default fallback %q, got %q", expected, profile)
	}
}

func TestDefaultConfigDir_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test on non-Linux platform")
	}

	// Test with XDG_CONFIG_HOME set
	tmpDir := t.TempDir()
	old := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", old)

	dir := defaultConfigDir()
	expected := filepath.Join(tmpDir, "parkbot")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestDefaultConfigDir_AllPlatforms(t *testing.T) {
	// This test verifies that defaultConfigDir returns a non-empty absolute path
	dir := defaultConfigDir()
	if dir == "" {
		t.Fatal("defaultConfigDir() returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
}

func TestPlatformWindowSize(t *testing.T) {
	size := platformWindowSize()
	if size.Width <= 0 || size.Height <= 0 {
		t.Errorf("invalid window size: %v", size)
	}

	// Verify reasonable bounds
	if size.Width < 800 || size.Width > 2000 {
		t.Errorf("window width out of expected range: %f", size.Width)
	}
	if size.Height < 500 || size.Height > 1200 {
		t.Errorf("window height out of expected range: %f", size.Height)
	}
}

func TestIsuThemeColor(t *testing.T) {
	th := isuTheme{}

	tests := []struct {
		name     string
		color    fyne.ThemeColorName
		expected interface{} // nil means "just verify it returns something"
	}{
		{"background", theme.ColorNameBackground, palBg},
		{"foreground", theme.ColorNameForeground, palFg},
		{"primary", theme.ColorNamePrimary, palAccent},
		{"focus", theme.ColorNameFocus, palAccent},
		{"button", theme.ColorNameButton, palSurface},
		{"placeholder", theme.ColorNamePlaceHolder, palMuted},
		{"input-bg", theme.ColorNameInputBackground, palInput},
		{"input-border", theme.ColorNameInputBorder, palBorder},
		{"hover", theme.ColorNameHover, palHover},
		{"error", theme.ColorNameError, palLogErr},
		{"success", theme.ColorNameSuccess, palSuccess},
		{"warning", theme.ColorNameWarning, palLogWarn},
		{"log-ok", colorLogOK, palLogOK},
		{"log-err", colorLogErr, palLogErr},
		{"log-dim", colorLogDim, palLogDim},
		{"log-warn", colorLogWarn, palLogWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := th.Color(tt.color, theme.VariantDark)
			if c == nil {
				t.Fatal("Color returned nil")
			}
			if tt.expected != nil {
				r1, g1, b1, a1 := c.RGBA()
				r2, g2, b2, a2 := tt.expected.(interface {
					RGBA() (uint32, uint32, uint32, uint32)
				}).RGBA()
				if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
					t.Errorf("color mismatch for %s: got RGBA(%d,%d,%d,%d), want RGBA(%d,%d,%d,%d)",
						tt.name, r1, g1, b1, a1, r2, g2, b2, a2)
				}
			}
		})
	}
}

func TestIsuThemeColorFallback(t *testing.T) {
	th := isuTheme{}
	// An unknown color name should fall back to the default theme
	c := th.Color("some-unknown-color-name", theme.VariantDark)
	if c == nil {
		t.Fatal("Color returned nil for unknown color name")
	}
}

func TestIsuThemeFont(t *testing.T) {
	th := isuTheme{}

	// Verify that the theme returns fonts for all text styles
	styles := []fyne.TextStyle{
		{},
		{Bold: true},
		{Italic: true},
		{Monospace: true},
		{Bold: true, Italic: true},
	}

	for _, style := range styles {
		font := th.Font(style)
		if font == nil {
			t.Errorf("Font returned nil for style %+v", style)
		}
	}
}

func TestIsuThemeSize(t *testing.T) {
	th := isuTheme{}

	// Verify standard sizes are non-negative
	sizeNames := []fyne.ThemeSizeName{
		theme.SizeNamePadding,
		theme.SizeNameInlineIcon,
		theme.SizeNameText,
		theme.SizeNameCaptionText,
	}

	for _, name := range sizeNames {
		size := th.Size(name)
		if size < 0 {
			t.Errorf("Size(%q) returned negative value: %f", name, size)
		}
	}

	// On Linux, padding should be slightly larger than default
	if runtime.GOOS == "linux" {
		defaultSize := theme.DefaultTheme().Size(theme.SizeNamePadding)
		linuxSize := th.Size(theme.SizeNamePadding)
		if linuxSize <= defaultSize {
			t.Errorf("expected Linux padding (%f) > default padding (%f)", linuxSize, defaultSize)
		}
	}
}

func TestIsuThemeIcon(t *testing.T) {
	th := isuTheme{}

	// Verify that theme returns icons for common icon names
	iconNames := []fyne.ThemeIconName{
		theme.IconNameHome,
		theme.IconNameSettings,
		theme.IconNameError,
	}

	for _, name := range iconNames {
		icon := th.Icon(name)
		if icon == nil {
			t.Errorf("Icon returned nil for %q", name)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				PermitKeyword: "COMMUTER",
				Billing: Billing{
					CardNumber: "4111111111111111",
					Expiry:     "12/25",
					CVV:        "123",
				},
			},
			wantErr: false,
		},
		{
			name: "missing permit keyword",
			cfg: Config{
				Billing: Billing{
					CardNumber: "4111111111111111",
					Expiry:     "12/25",
					CVV:        "123",
				},
			},
			wantErr: true,
		},
		{
			name: "missing card number",
			cfg: Config{
				PermitKeyword: "COMMUTER",
				Billing: Billing{
					Expiry: "12/25",
					CVV:    "123",
				},
			},
			wantErr: true,
		},
		{
			name: "missing expiry",
			cfg: Config{
				PermitKeyword: "COMMUTER",
				Billing: Billing{
					CardNumber: "4111111111111111",
					CVV:        "123",
				},
			},
			wantErr: true,
		},
		{
			name: "missing cvv",
			cfg: Config{
				PermitKeyword: "COMMUTER",
				Billing: Billing{
					CardNumber: "4111111111111111",
					Expiry:     "12/25",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	original := &Config{
		PermitKeyword:  "COMMUTER",
		VehicleKeyword: "HONDA",
		AddressKeyword: "MAIN ST",
		Email:          "test@example.com",
		OneTime:        true,
		ChromeProfile:  "/tmp/test-profile",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "123",
			Name:       "Test User",
			Zip:        "50010",
		},
	}

	if err := original.Save(cfgPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}

	if loaded.PermitKeyword != original.PermitKeyword {
		t.Errorf("PermitKeyword: got %q, want %q", loaded.PermitKeyword, original.PermitKeyword)
	}
	if loaded.Email != original.Email {
		t.Errorf("Email: got %q, want %q", loaded.Email, original.Email)
	}
	if loaded.Billing.CardNumber != original.Billing.CardNumber {
		t.Errorf("CardNumber: got %q, want %q", loaded.Billing.CardNumber, original.Billing.CardNumber)
	}
	if loaded.OneTime != original.OneTime {
		t.Errorf("OneTime: got %v, want %v", loaded.OneTime, original.OneTime)
	}

	// Verify file permissions (should be 0600 for security)
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permissions: got %o, want %o", perm, 0600)
	}
}
