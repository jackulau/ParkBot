package main

import (
	"image/color"
	"runtime"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ─── Theme color tests ──────────────────────────────────────────────────────

func TestISUThemeReturnsCorrectColors(t *testing.T) {
	th := isuTheme{}
	v := theme.VariantDark

	tests := []struct {
		name     string
		colorID  fyne.ThemeColorName
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
		{"separator", theme.ColorNameSeparator, palBorder},
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
			got := th.Color(tt.colorID, v)
			if got != tt.expected {
				t.Errorf("Color(%q) = %v, want %v", tt.colorID, got, tt.expected)
			}
		})
	}
}

func TestISUThemeFallsBackToDefault(t *testing.T) {
	th := isuTheme{}
	unknownColor := fyne.ThemeColorName("nonexistent-color")
	got := th.Color(unknownColor, theme.VariantDark)
	expected := theme.DefaultTheme().Color(unknownColor, theme.VariantDark)
	if got != expected {
		t.Errorf("fallback color = %v, want default theme %v", got, expected)
	}
}

// ─── Theme size tests ────────────────────────────────────────────────────────

func TestISUThemeSizes(t *testing.T) {
	th := isuTheme{}

	if runtime.GOOS == "darwin" {
		tests := []struct {
			name     string
			sizeID   fyne.ThemeSizeName
			expected float32
		}{
			{"text", theme.SizeNameText, 14},
			{"caption", theme.SizeNameCaptionText, 12},
			{"subheading", theme.SizeNameSubHeadingText, 16},
			{"heading", theme.SizeNameHeadingText, 22},
			{"padding", theme.SizeNamePadding, 6},
			{"inner-padding", theme.SizeNameInnerPadding, 10},
			{"scrollbar-small", theme.SizeNameScrollBarSmall, 4},
			{"scrollbar", theme.SizeNameScrollBar, 8},
			{"input-radius", theme.SizeNameInputRadius, 6},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := th.Size(tt.sizeID)
				if got != tt.expected {
					t.Errorf("Size(%q) = %f, want %f", tt.sizeID, got, tt.expected)
				}
			})
		}
	} else {
		// On non-macOS, all sizes should fall through to default theme.
		defaultSize := theme.DefaultTheme().Size(theme.SizeNameText)
		got := th.Size(theme.SizeNameText)
		if got != defaultSize {
			t.Errorf("non-darwin Size(text) = %f, want default %f", got, defaultSize)
		}
	}
}

func TestISUThemeSizeFallback(t *testing.T) {
	th := isuTheme{}
	unknownSize := fyne.ThemeSizeName("nonexistent-size")
	got := th.Size(unknownSize)
	expected := theme.DefaultTheme().Size(unknownSize)
	if got != expected {
		t.Errorf("fallback size = %f, want default theme %f", got, expected)
	}
}

// ─── Theme font/icon delegation tests ────────────────────────────────────────

func TestISUThemeFontDelegation(t *testing.T) {
	th := isuTheme{}
	defaultFont := theme.DefaultTheme().Font(fyne.TextStyle{})
	got := th.Font(fyne.TextStyle{})
	if got != defaultFont {
		t.Error("Font() did not delegate to default theme")
	}
}

func TestISUThemeIconDelegation(t *testing.T) {
	th := isuTheme{}
	defaultIcon := theme.DefaultTheme().Icon(theme.IconNameInfo)
	got := th.Icon(theme.IconNameInfo)
	if got != defaultIcon {
		t.Error("Icon() did not delegate to default theme")
	}
}

// ─── Palette value tests ─────────────────────────────────────────────────────

func TestPaletteValuesAreOpaque(t *testing.T) {
	colors := []struct {
		name string
		c    color.RGBA
	}{
		{"palBg", palBg},
		{"palSurface", palSurface},
		{"palBorder", palBorder},
		{"palAccent", palAccent},
		{"palFg", palFg},
		{"palMuted", palMuted},
		{"palInput", palInput},
		{"palHover", palHover},
		{"palSuccess", palSuccess},
		{"palLogOK", palLogOK},
		{"palLogErr", palLogErr},
		{"palLogDim", palLogDim},
		{"palLogWarn", palLogWarn},
	}
	for _, tt := range colors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.c.A != 0xFF {
				t.Errorf("%s alpha = %d, want 255 (opaque)", tt.name, tt.c.A)
			}
		})
	}
}

func TestPaletteBackgroundIsNotPureBlack(t *testing.T) {
	// macOS dark mode convention: background should be slightly lifted, not #000.
	if palBg.R == 0 && palBg.G == 0 && palBg.B == 0 {
		t.Error("palBg is pure black (#000); macOS dark mode should use a lifted background")
	}
}

func TestPaletteSurfaceIsLighterThanBackground(t *testing.T) {
	bgLum := int(palBg.R) + int(palBg.G) + int(palBg.B)
	surfLum := int(palSurface.R) + int(palSurface.G) + int(palSurface.B)
	if surfLum <= bgLum {
		t.Errorf("surface luminance (%d) should be > background luminance (%d)", surfLum, bgLum)
	}
}

func TestPaletteInputIsDistinctFromBackground(t *testing.T) {
	if palInput == palBg {
		t.Error("palInput should differ from palBg for visual distinction")
	}
}

func TestPaletteHoverIsLighterThanSurface(t *testing.T) {
	surfLum := int(palSurface.R) + int(palSurface.G) + int(palSurface.B)
	hoverLum := int(palHover.R) + int(palHover.G) + int(palHover.B)
	if hoverLum <= surfLum {
		t.Errorf("hover luminance (%d) should be > surface luminance (%d)", hoverLum, surfLum)
	}
}

// ─── readFormConfig tests ─────────────────────────────────────────────────────

func TestReadFormConfigNormalizesKeywords(t *testing.T) {
	g := &GUI{
		permitE:  makeEntry("  commuter  "),
		vehicleE: makeEntry("  abc123  "),
		addressE: makeEntry("  helser  "),
		emailE:   makeEntry("test@example.com"),
		cardE:    makeEntry("4111 1111 1111 1111"),
		expiryE:  makeEntry("12/25"),
		cvvE:     makeEntry("123"),
		nameE:    makeEntry("John Doe"),
		zipE:     makeEntry("50011"),
		oneTimeC: makeCheck(true),
		profileE: makeEntry("/some/path"),
	}

	cfg := g.readFormConfig()

	if cfg.PermitKeyword != "COMMUTER" {
		t.Errorf("PermitKeyword = %q, want COMMUTER", cfg.PermitKeyword)
	}
	if cfg.VehicleKeyword != "ABC123" {
		t.Errorf("VehicleKeyword = %q, want ABC123", cfg.VehicleKeyword)
	}
	if cfg.AddressKeyword != "HELSER" {
		t.Errorf("AddressKeyword = %q, want HELSER", cfg.AddressKeyword)
	}
	if cfg.Billing.CardNumber != "4111111111111111" {
		t.Errorf("CardNumber = %q, want spaces removed", cfg.Billing.CardNumber)
	}
	if cfg.Email != "test@example.com" {
		t.Errorf("Email = %q, want test@example.com", cfg.Email)
	}
	if !cfg.OneTime {
		t.Error("OneTime should be true")
	}
	if cfg.ChromeProfile != "/some/path" {
		t.Errorf("ChromeProfile = %q, want /some/path", cfg.ChromeProfile)
	}
}

func TestReadFormConfigHandlesEmptyFields(t *testing.T) {
	g := &GUI{
		permitE:  makeEntry(""),
		vehicleE: makeEntry(""),
		addressE: makeEntry(""),
		emailE:   makeEntry(""),
		cardE:    makeEntry(""),
		expiryE:  makeEntry(""),
		cvvE:     makeEntry(""),
		nameE:    makeEntry(""),
		zipE:     makeEntry(""),
		oneTimeC: makeCheck(false),
		profileE: makeEntry(""),
	}

	cfg := g.readFormConfig()

	if cfg.PermitKeyword != "" {
		t.Errorf("PermitKeyword = %q, want empty", cfg.PermitKeyword)
	}
	if cfg.OneTime {
		t.Error("OneTime should be false")
	}
}

// ─── Log color classification tests ──────────────────────────────────────────

func TestClassifyLogColor(t *testing.T) {
	tests := []struct {
		msg      string
		expected fyne.ThemeColorName
	}{
		{"[debug] some debug info", colorLogDim},
		{"Bot error: something failed", colorLogErr},
		{"could not connect", colorLogErr},
		{"panic: runtime error", colorLogErr},
		{"WARNING: maximum permits", colorLogWarn},
		{"Config saved to config.yaml", colorLogOK},
		{"Purchase confirmed!", colorLogOK},
		{"Lock file written: purchased.lock", colorLogOK},
		{"Bot finished successfully.", colorLogOK},
		{"Just a normal message", fyne.ThemeColorName(theme.ColorNameForeground)},
		{"Launching Chrome with profile", fyne.ThemeColorName(theme.ColorNameForeground)},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := classifyLogColor(tt.msg)
			if got != tt.expected {
				t.Errorf("classifyLogColor(%q) = %q, want %q", tt.msg, got, tt.expected)
			}
		})
	}
}

// ─── ISU theme interface compliance ──────────────────────────────────────────

func TestISUThemeImplementsFyneTheme(t *testing.T) {
	// Compile-time check that isuTheme implements fyne.Theme.
	var _ fyne.Theme = isuTheme{}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func makeEntry(text string) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(text)
	return e
}

func makeCheck(checked bool) *widget.Check {
	c := widget.NewCheck("", nil)
	c.SetChecked(checked)
	return c
}
