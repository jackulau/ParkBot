package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestConfigLoad verifies that loadConfig correctly reads and parses a YAML config file,
// and that defaults are filled appropriately.
func TestConfigLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `permit_keyword: COMMUTER
vehicle_keyword: HONDA
address_keyword: MAIN
email: test@example.com
one_time: true
billing:
  card_number: "4111111111111111"
  expiry: "12/25"
  cvv: "123"
  name: "Test User"
  zip: "50010"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	// Keywords should be uppercased
	if cfg.PermitKeyword != "COMMUTER" {
		t.Errorf("PermitKeyword = %q, want %q", cfg.PermitKeyword, "COMMUTER")
	}
	if cfg.VehicleKeyword != "HONDA" {
		t.Errorf("VehicleKeyword = %q, want %q", cfg.VehicleKeyword, "HONDA")
	}
	if cfg.AddressKeyword != "MAIN" {
		t.Errorf("AddressKeyword = %q, want %q", cfg.AddressKeyword, "MAIN")
	}
	if cfg.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", cfg.Email, "test@example.com")
	}
	if !cfg.OneTime {
		t.Error("OneTime should be true")
	}

	// Billing
	if cfg.Billing.CardNumber != "4111111111111111" {
		t.Errorf("CardNumber = %q, want %q", cfg.Billing.CardNumber, "4111111111111111")
	}
	if cfg.Billing.Expiry != "12/25" {
		t.Errorf("Expiry = %q, want %q", cfg.Billing.Expiry, "12/25")
	}
	if cfg.Billing.CVV != "123" {
		t.Errorf("CVV = %q, want %q", cfg.Billing.CVV, "123")
	}
	if cfg.Billing.Name != "Test User" {
		t.Errorf("Name = %q, want %q", cfg.Billing.Name, "Test User")
	}
	if cfg.Billing.Zip != "50010" {
		t.Errorf("Zip = %q, want %q", cfg.Billing.Zip, "50010")
	}

	// ChromeProfile should be filled with default
	if cfg.ChromeProfile == "" {
		t.Error("ChromeProfile should be filled with default when empty")
	}
}

// TestConfigLoadMissing verifies that loadConfig returns an error for a missing file.
func TestConfigLoadMissing(t *testing.T) {
	_, err := loadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("loadConfig should return error for missing file")
	}
}

// TestConfigValidate verifies validation rules.
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "empty permit keyword",
			cfg:     Config{Billing: Billing{CardNumber: "1234", Expiry: "12/25", CVV: "123"}},
			wantErr: true,
		},
		{
			name:    "empty card number",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{Expiry: "12/25", CVV: "123"}},
			wantErr: true,
		},
		{
			name:    "empty expiry",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{CardNumber: "1234", CVV: "123"}},
			wantErr: true,
		},
		{
			name:    "empty CVV",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{CardNumber: "1234", Expiry: "12/25"}},
			wantErr: true,
		},
		{
			name:    "valid config",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{CardNumber: "1234", Expiry: "12/25", CVV: "123"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigSaveRoundTrip verifies that saving and reloading a config preserves values.
func TestConfigSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "roundtrip.yaml")

	original := &Config{
		PermitKeyword:  "COMMUTER",
		VehicleKeyword: "CIVIC",
		AddressKeyword: "ELM",
		Email:          "user@example.com",
		OneTime:        true,
		ChromeProfile:  "/custom/chrome/profile",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "06/28",
			CVV:        "999",
			Name:       "Jane Doe",
			Zip:        "90210",
		},
	}

	if err := original.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig after Save failed: %v", err)
	}

	// ChromeProfile should be preserved (not overwritten by default since it's non-empty)
	if loaded.ChromeProfile != "/custom/chrome/profile" {
		t.Errorf("ChromeProfile = %q, want %q", loaded.ChromeProfile, "/custom/chrome/profile")
	}
	if loaded.PermitKeyword != "COMMUTER" {
		t.Errorf("PermitKeyword = %q, want %q", loaded.PermitKeyword, "COMMUTER")
	}
	if loaded.Billing.CardNumber != "4111111111111111" {
		t.Errorf("CardNumber = %q, want %q", loaded.Billing.CardNumber, "4111111111111111")
	}
}

// TestDefaultChromeProfile verifies that defaultChromeProfile returns a
// non-empty path appropriate for the current OS.
func TestDefaultChromeProfile(t *testing.T) {
	profile := defaultChromeProfile()
	if profile == "" {
		t.Fatal("defaultChromeProfile returned empty string")
	}

	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(profile, "Application Support/Google/Chrome") {
			t.Errorf("macOS profile path unexpected: %s", profile)
		}
	case "linux":
		if !strings.Contains(profile, ".config/google-chrome") {
			t.Errorf("Linux profile path unexpected: %s", profile)
		}
	case "windows":
		if !strings.Contains(profile, "Google") || !strings.Contains(profile, "Chrome") {
			t.Errorf("Windows profile path unexpected: %s", profile)
		}
	}
}

// TestKeywordNormalization verifies that keywords are uppercased during config loading.
func TestKeywordNormalization(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `permit_keyword: "  commuter  "
vehicle_keyword: "  honda Civic  "
address_keyword: "  main Street  "
billing:
  card_number: "4111111111111111"
  expiry: "12/25"
  cvv: "123"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	if cfg.PermitKeyword != "COMMUTER" {
		t.Errorf("PermitKeyword = %q, want %q", cfg.PermitKeyword, "COMMUTER")
	}
	if cfg.VehicleKeyword != "HONDA CIVIC" {
		t.Errorf("VehicleKeyword = %q, want %q", cfg.VehicleKeyword, "HONDA CIVIC")
	}
	if cfg.AddressKeyword != "MAIN STREET" {
		t.Errorf("AddressKeyword = %q, want %q", cfg.AddressKeyword, "MAIN STREET")
	}
}

// TestBuildCrossCompile verifies that the build script exists, is executable,
// and successfully builds for the current platform.
func TestBuildCrossCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping build test in short mode")
	}

	// Verify build.sh exists and is executable
	info, err := os.Stat("build.sh")
	if err != nil {
		t.Fatalf("build.sh not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatal("build.sh is not executable")
	}

	// Run the build for current OS only
	filter := runtime.GOOS
	cmd := exec.Command("./build.sh", filter)
	cmd.Env = append(os.Environ(), "OUTDIR="+t.TempDir())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build.sh %s failed: %v\noutput: %s", filter, err, output)
	}

	t.Logf("Build output:\n%s", output)
}

// TestBuildClean verifies that the clean command works.
func TestBuildClean(t *testing.T) {
	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatalf("creating dist dir: %v", err)
	}
	// Create a dummy file
	if err := os.WriteFile(filepath.Join(distDir, "dummy"), []byte("x"), 0644); err != nil {
		t.Fatalf("writing dummy file: %v", err)
	}

	cmd := exec.Command("./build.sh", "clean")
	cmd.Env = append(os.Environ(), "OUTDIR="+distDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build.sh clean failed: %v\noutput: %s", err, output)
	}

	if _, err := os.Stat(distDir); !os.IsNotExist(err) {
		t.Error("dist directory should be removed after clean")
	}
}
