package main

import (
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ─── Theme tests ─────────────────────────────────────────────────────────────

func TestThemeColors(t *testing.T) {
	th := isuTheme{}
	variant := fyne.ThemeVariant(0)

	tests := []struct {
		name     string
		colorKey fyne.ThemeColorName
		expected color.Color
	}{
		{"background", theme.ColorNameBackground, palBg},
		{"foreground", theme.ColorNameForeground, palFg},
		{"primary/accent", theme.ColorNamePrimary, palAccent},
		{"focus", theme.ColorNameFocus, palAccent},
		{"button", theme.ColorNameButton, palSurface},
		{"placeholder", theme.ColorNamePlaceHolder, palMuted},
		{"input-bg", theme.ColorNameInputBackground, palInput},
		{"input-border", theme.ColorNameInputBorder, palBorder},
		{"hover", theme.ColorNameHover, palHover},
		{"scrollbar", theme.ColorNameScrollBar, color.RGBA{R: 0x2A, G: 0x2A, B: 0x2A, A: 0xFF}},
		{"separator", theme.ColorNameSeparator, palBorder},
		{"error", theme.ColorNameError, palLogErr},
		{"success", theme.ColorNameSuccess, palSuccess},
		{"warning", theme.ColorNameWarning, palLogWarn},
		{"log-ok", colorLogOK, palLogOK},
		{"log-err", colorLogErr, palLogErr},
		{"log-dim", colorLogDim, palLogDim},
		{"log-warn", colorLogWarn, palLogWarn},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := th.Color(tc.colorKey, variant)
			if got != tc.expected {
				t.Errorf("Color(%s) = %v, want %v", tc.colorKey, got, tc.expected)
			}
		})
	}
}

func TestThemeFallbackColor(t *testing.T) {
	th := isuTheme{}
	// An unrecognized color name should fall through to the default theme
	got := th.Color("unknown-color-name", 0)
	if got == nil {
		t.Error("expected non-nil color for fallback")
	}
}

func TestThemeSizeReturnsPositive(t *testing.T) {
	th := isuTheme{}
	sizeNames := []fyne.ThemeSizeName{
		theme.SizeNamePadding,
		theme.SizeNameInlineIcon,
		theme.SizeNameScrollBar,
		theme.SizeNameScrollBarSmall,
		theme.SizeNameText,
		theme.SizeNameCaptionText,
	}
	for _, name := range sizeNames {
		size := th.Size(name)
		if size <= 0 {
			t.Errorf("Size(%s) = %f, expected > 0", name, size)
		}
	}
}

func TestThemeSizeWindowsPadding(t *testing.T) {
	// This test documents that on non-Windows platforms, Size uses the default.
	// On Windows, it adds extra padding. We verify the function doesn't panic
	// and returns a reasonable value.
	th := isuTheme{}
	base := theme.DefaultTheme().Size(theme.SizeNamePadding)
	got := th.Size(theme.SizeNamePadding)

	if runtime.GOOS == "windows" {
		if got != base+1 {
			t.Errorf("expected Windows padding = %f+1, got %f", base, got)
		}
	} else {
		if got != base {
			t.Errorf("expected non-Windows padding = %f, got %f", base, got)
		}
	}
}

func TestThemeFontNotNil(t *testing.T) {
	th := isuTheme{}
	font := th.Font(fyne.TextStyle{})
	if font == nil {
		t.Error("Font() returned nil")
	}
	fontBold := th.Font(fyne.TextStyle{Bold: true})
	if fontBold == nil {
		t.Error("Font(Bold) returned nil")
	}
	fontMono := th.Font(fyne.TextStyle{Monospace: true})
	if fontMono == nil {
		t.Error("Font(Monospace) returned nil")
	}
}

func TestThemeIconNotNil(t *testing.T) {
	th := isuTheme{}
	icon := th.Icon(theme.IconNameConfirm)
	if icon == nil {
		t.Error("Icon() returned nil for confirm icon")
	}
}

// ─── Sizing helper tests ──────────────────────────────────────────────────────

func TestWindowSize(t *testing.T) {
	size := windowSize()
	if size.Width < 800 || size.Height < 500 {
		t.Errorf("windowSize() = %v, expected at least 800x500", size)
	}

	if runtime.GOOS == "windows" {
		if size.Width != 1150 || size.Height != 760 {
			t.Errorf("expected Windows windowSize 1150x760, got %v", size)
		}
	} else {
		if size.Width != 1100 || size.Height != 720 {
			t.Errorf("expected non-Windows windowSize 1100x720, got %v", size)
		}
	}
}

func TestMinWindowSize(t *testing.T) {
	min := minWindowSize()
	if min.Width != 800 || min.Height != 520 {
		t.Errorf("minWindowSize() = %v, expected 800x520", min)
	}
}

func TestTextSize(t *testing.T) {
	base := float32(14.0)
	got := textSize(base)

	if runtime.GOOS == "windows" {
		if got != base+1 {
			t.Errorf("textSize(%f) on Windows = %f, expected %f", base, got, base+1)
		}
	} else {
		if got != base {
			t.Errorf("textSize(%f) on non-Windows = %f, expected %f", base, got, base)
		}
	}
}

func TestTextSizeZero(t *testing.T) {
	// Ensure textSize doesn't produce negative sizes
	got := textSize(0)
	if got < 0 {
		t.Errorf("textSize(0) = %f, expected >= 0", got)
	}
}

// ─── Config path tests ───────────────────────────────────────────────────────

func TestDefaultChromeProfileNotEmpty(t *testing.T) {
	profile := defaultChromeProfile()
	if profile == "" {
		t.Error("defaultChromeProfile() returned empty string")
	}
}

func TestDefaultChromeProfileContainsChrome(t *testing.T) {
	profile := defaultChromeProfile()
	if !strings.Contains(strings.ToLower(profile), "chrome") {
		t.Errorf("defaultChromeProfile() = %q, expected to contain 'chrome'", profile)
	}
}

func TestDefaultChromeProfilePlatformSpecific(t *testing.T) {
	profile := defaultChromeProfile()
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(profile, "Library") {
			t.Errorf("on macOS, expected path with Library, got %q", profile)
		}
	case "linux":
		if !strings.Contains(profile, ".config") {
			t.Errorf("on Linux, expected path with .config, got %q", profile)
		}
	case "windows":
		// Should use LOCALAPPDATA or AppData
		if !strings.Contains(profile, "AppData") && !strings.Contains(profile, "Local") {
			t.Errorf("on Windows, expected path with AppData, got %q", profile)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo/bar/baz", filepath.FromSlash("foo/bar/baz")},
		{"C:/Users/test/data", filepath.FromSlash("C:/Users/test/data")},
		{"", ""},
		{"/absolute/path", filepath.FromSlash("/absolute/path")},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizePath(tc.input)
			if got != tc.expected {
				t.Errorf("NormalizePath(%q) = %q, expected %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if !strings.HasSuffix(path, "config.yaml") {
		t.Errorf("DefaultConfigPath() = %q, expected to end with config.yaml", path)
	}
}

// ─── Config validation tests ──────────────────────────────────────────────────

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
			name:    "missing permit keyword",
			cfg:     Config{Billing: Billing{CardNumber: "4111", Expiry: "12/25", CVV: "123"}},
			wantErr: true,
		},
		{
			name:    "missing card number",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{Expiry: "12/25", CVV: "123"}},
			wantErr: true,
		},
		{
			name:    "missing expiry",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{CardNumber: "4111", CVV: "123"}},
			wantErr: true,
		},
		{
			name:    "missing CVV",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{CardNumber: "4111", Expiry: "12/25"}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("validate() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_config.yaml")

	original := &Config{
		PermitKeyword:  "COMMUTER",
		VehicleKeyword: "HONDA",
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

	if err := original.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	loaded, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}

	// Keywords get uppercased by loadConfig
	if loaded.PermitKeyword != "COMMUTER" {
		t.Errorf("PermitKeyword = %q, want COMMUTER", loaded.PermitKeyword)
	}
	if loaded.Billing.CardNumber != "4111111111111111" {
		t.Errorf("CardNumber = %q, want 4111111111111111", loaded.Billing.CardNumber)
	}
	if loaded.Email != "test@example.com" {
		t.Errorf("Email = %q, want test@example.com", loaded.Email)
	}
	if !loaded.OneTime {
		t.Error("OneTime should be true")
	}
}

// ─── Palette sanity tests ─────────────────────────────────────────────────────

func TestPaletteColorsHaveFullAlpha(t *testing.T) {
	colors := map[string]color.RGBA{
		"palBg":      palBg,
		"palSurface": palSurface,
		"palBorder":  palBorder,
		"palAccent":  palAccent,
		"palFg":      palFg,
		"palMuted":   palMuted,
		"palInput":   palInput,
		"palHover":   palHover,
		"palSuccess": palSuccess,
		"palLogOK":   palLogOK,
		"palLogErr":  palLogErr,
		"palLogWarn": palLogWarn,
	}
	for name, c := range colors {
		if c.A != 0xFF {
			t.Errorf("%s has alpha %d, expected 255 (fully opaque)", name, c.A)
		}
	}
}

func TestDarkThemeContrast(t *testing.T) {
	// Verify foreground is significantly lighter than background
	// for readability on the dark theme
	fgLum := float64(palFg.R) + float64(palFg.G) + float64(palFg.B)
	bgLum := float64(palBg.R) + float64(palBg.G) + float64(palBg.B)

	if fgLum <= bgLum {
		t.Errorf("foreground (%v) should be lighter than background (%v)", palFg, palBg)
	}

	contrastRatio := fgLum / (bgLum + 1) // +1 to avoid division by zero
	if contrastRatio < 3.0 {
		t.Errorf("contrast ratio between fg and bg is %f, expected >= 3.0 for readability", contrastRatio)
	}
}

func TestLogColorsDistinct(t *testing.T) {
	// Log colors should be visually distinct from each other
	logColors := map[string]color.RGBA{
		"OK":   palLogOK,
		"Err":  palLogErr,
		"Warn": palLogWarn,
		"Dim":  palLogDim,
	}

	names := make([]string, 0, len(logColors))
	for name := range logColors {
		names = append(names, name)
	}

	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			c1 := logColors[names[i]]
			c2 := logColors[names[j]]
			if c1 == c2 {
				t.Errorf("log colors %s and %s are identical: %v", names[i], names[j], c1)
			}
		}
	}
}
