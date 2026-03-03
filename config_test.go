package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ─── loadConfig ──────────────────────────────────────────────────────────────

func TestLoadConfig_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
permit_keyword: commuter
vehicle_keyword: honda
address_keyword: main
email: test@example.com
one_time: true
chrome_profile: /tmp/fake-profile
billing:
  card_number: "4111111111111111"
  expiry: "12/25"
  cvv: "123"
  name: "John Doe"
  zip: "50010"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
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
	if cfg.ChromeProfile != "/tmp/fake-profile" {
		t.Errorf("ChromeProfile = %q, want %q", cfg.ChromeProfile, "/tmp/fake-profile")
	}
	if cfg.Billing.CardNumber != "4111111111111111" {
		t.Errorf("CardNumber = %q, want %q", cfg.Billing.CardNumber, "4111111111111111")
	}
	if cfg.Billing.Expiry != "12/25" {
		t.Errorf("Expiry = %q, want %q", cfg.Billing.Expiry, "12/25")
	}
	if cfg.Billing.CVV != "123" {
		t.Errorf("CVV = %q, want %q", cfg.Billing.CVV, "123")
	}
	if cfg.Billing.Name != "John Doe" {
		t.Errorf("Name = %q, want %q", cfg.Billing.Name, "John Doe")
	}
	if cfg.Billing.Zip != "50010" {
		t.Errorf("Zip = %q, want %q", cfg.Billing.Zip, "50010")
	}
}

func TestLoadConfig_DefaultChromeProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
permit_keyword: test
billing:
  card_number: "4111111111111111"
  expiry: "12/25"
  cvv: "123"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.ChromeProfile == "" {
		t.Error("ChromeProfile should be set to default when empty")
	}

	expected := defaultChromeProfile()
	if cfg.ChromeProfile != expected {
		t.Errorf("ChromeProfile = %q, want %q", cfg.ChromeProfile, expected)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("loadConfig should return error for missing file")
	}
	if !strings.Contains(err.Error(), "reading config") {
		t.Errorf("error should mention 'reading config', got: %v", err)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("::invalid yaml{{"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(path)
	if err == nil {
		t.Error("loadConfig should return error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error should mention 'parsing config', got: %v", err)
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig should succeed with empty file: %v", err)
	}
	// Empty config should still get a default Chrome profile
	if cfg.ChromeProfile == "" {
		t.Error("ChromeProfile should be set to default for empty config")
	}
}

func TestLoadConfig_KeywordNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
permit_keyword: "  commuter  "
vehicle_keyword: "  Honda Civic  "
address_keyword: "  123 Main St  "
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.PermitKeyword != "COMMUTER" {
		t.Errorf("PermitKeyword should be trimmed and uppercased, got %q", cfg.PermitKeyword)
	}
	if cfg.VehicleKeyword != "HONDA CIVIC" {
		t.Errorf("VehicleKeyword should be trimmed and uppercased, got %q", cfg.VehicleKeyword)
	}
	if cfg.AddressKeyword != "123 MAIN ST" {
		t.Errorf("AddressKeyword should be trimmed and uppercased, got %q", cfg.AddressKeyword)
	}
}

// ─── Config.validate ─────────────────────────────────────────────────────────

func TestValidate_AllRequired(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "COMMUTER",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "123",
		},
	}
	if err := cfg.validate(); err != nil {
		t.Errorf("validate should succeed with all required fields: %v", err)
	}
}

func TestValidate_MissingPermitKeyword(t *testing.T) {
	cfg := &Config{
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "123",
		},
	}
	err := cfg.validate()
	if err == nil {
		t.Error("validate should fail without permit_keyword")
	}
	if !strings.Contains(err.Error(), "permit_keyword") {
		t.Errorf("error should mention permit_keyword, got: %v", err)
	}
}

func TestValidate_MissingCardNumber(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "COMMUTER",
		Billing: Billing{
			Expiry: "12/25",
			CVV:    "123",
		},
	}
	err := cfg.validate()
	if err == nil {
		t.Error("validate should fail without card_number")
	}
	if !strings.Contains(err.Error(), "card_number") {
		t.Errorf("error should mention card_number, got: %v", err)
	}
}

func TestValidate_MissingExpiry(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "COMMUTER",
		Billing: Billing{
			CardNumber: "4111111111111111",
			CVV:        "123",
		},
	}
	err := cfg.validate()
	if err == nil {
		t.Error("validate should fail without expiry")
	}
	if !strings.Contains(err.Error(), "expiry") {
		t.Errorf("error should mention expiry, got: %v", err)
	}
}

func TestValidate_MissingCVV(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "COMMUTER",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
		},
	}
	err := cfg.validate()
	if err == nil {
		t.Error("validate should fail without cvv")
	}
	if !strings.Contains(err.Error(), "cvv") {
		t.Errorf("error should mention cvv, got: %v", err)
	}
}

func TestValidate_OptionalFields(t *testing.T) {
	// Vehicle, address, email, name, zip are optional
	cfg := &Config{
		PermitKeyword: "COMMUTER",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "123",
		},
	}
	if err := cfg.validate(); err != nil {
		t.Errorf("validate should succeed with only required fields: %v", err)
	}
}

// ─── Config.Save & round-trip ────────────────────────────────────────────────

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &Config{
		PermitKeyword:  "COMMUTER",
		VehicleKeyword: "HONDA",
		AddressKeyword: "MAIN",
		Email:          "test@example.com",
		OneTime:        true,
		ChromeProfile:  "/tmp/test-profile",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "123",
			Name:       "John Doe",
			Zip:        "50010",
		},
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Verify file exists and has restricted permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("saved file not found: %v", err)
	}
	if runtime.GOOS != "windows" {
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("file permissions = %o, want 0600", perm)
		}
	}

	// Reload and verify
	loaded, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig on saved file returned error: %v", err)
	}

	if loaded.PermitKeyword != original.PermitKeyword {
		t.Errorf("PermitKeyword = %q, want %q", loaded.PermitKeyword, original.PermitKeyword)
	}
	if loaded.VehicleKeyword != original.VehicleKeyword {
		t.Errorf("VehicleKeyword = %q, want %q", loaded.VehicleKeyword, original.VehicleKeyword)
	}
	if loaded.Billing.CardNumber != original.Billing.CardNumber {
		t.Errorf("CardNumber = %q, want %q", loaded.Billing.CardNumber, original.Billing.CardNumber)
	}
	if loaded.OneTime != original.OneTime {
		t.Errorf("OneTime = %v, want %v", loaded.OneTime, original.OneTime)
	}
}

func TestSave_InvalidPath(t *testing.T) {
	cfg := &Config{PermitKeyword: "TEST"}
	err := cfg.Save("/nonexistent/directory/config.yaml")
	if err == nil {
		t.Error("Save should fail for nonexistent directory")
	}
}

// ─── defaultChromeProfile ────────────────────────────────────────────────────

func TestDefaultChromeProfile(t *testing.T) {
	profile := defaultChromeProfile()
	if profile == "" {
		t.Error("defaultChromeProfile should not return empty string")
	}

	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(profile, "Application Support/Google/Chrome") {
			t.Errorf("macOS profile path unexpected: %q", profile)
		}
	case "linux":
		if !strings.Contains(profile, ".config/google-chrome") {
			t.Errorf("Linux profile path unexpected: %q", profile)
		}
	case "windows":
		if !strings.Contains(profile, "Google") || !strings.Contains(profile, "Chrome") {
			t.Errorf("Windows profile path unexpected: %q", profile)
		}
	}

	if !strings.HasSuffix(profile, "Default") {
		t.Errorf("profile should end with 'Default', got %q", profile)
	}
}

// ─── Lock file ───────────────────────────────────────────────────────────────

func TestCheckLock_NoFile(t *testing.T) {
	// Ensure lock file doesn't exist
	os.Remove("purchased.lock")
	defer os.Remove("purchased.lock")

	err := checkLock()
	if err != nil {
		t.Errorf("checkLock should succeed when no lock file: %v", err)
	}
}

func TestCheckLock_FileExists(t *testing.T) {
	if err := os.WriteFile("purchased.lock", []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("purchased.lock")

	err := checkLock()
	if err == nil {
		t.Error("checkLock should fail when lock file exists")
	}
	if !strings.Contains(err.Error(), "purchased.lock") {
		t.Errorf("error should mention lock file, got: %v", err)
	}
}

func TestWriteLock(t *testing.T) {
	os.Remove("purchased.lock")
	defer os.Remove("purchased.lock")

	writeLock()

	data, err := os.ReadFile("purchased.lock")
	if err != nil {
		t.Fatalf("lock file should exist after writeLock: %v", err)
	}
	if !strings.Contains(string(data), "purchased at") {
		t.Errorf("lock file content unexpected: %q", string(data))
	}
}

// ─── truncate helper ─────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		n        int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 0, "..."},
	}

	for _, tc := range tests {
		result := truncate(tc.input, tc.n)
		if result != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.n, result, tc.expected)
		}
	}
}
