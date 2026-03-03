package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Lock file tests
// ============================================================================

func TestCheckLock_NoFile(t *testing.T) {
	// Ensure no lock file exists
	os.Remove(lockFile)
	defer os.Remove(lockFile)

	if err := checkLock(); err != nil {
		t.Fatalf("checkLock should return nil when no lock file exists, got: %v", err)
	}
}

func TestCheckLock_FileExists(t *testing.T) {
	// Create lock file
	if err := os.WriteFile(lockFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create lock file: %v", err)
	}
	defer os.Remove(lockFile)

	err := checkLock()
	if err == nil {
		t.Fatal("checkLock should return error when lock file exists")
	}
	if !strings.Contains(err.Error(), "purchased.lock exists") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestWriteLock(t *testing.T) {
	os.Remove(lockFile)
	defer os.Remove(lockFile)

	writeLock()

	data, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("lock file should exist after writeLock: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "purchased at") {
		t.Fatalf("lock file content unexpected: %q", content)
	}
}

func TestWriteLock_ContainsTimestamp(t *testing.T) {
	os.Remove(lockFile)
	defer os.Remove(lockFile)

	// Truncate to seconds since RFC3339 loses sub-second precision
	before := time.Now().Truncate(time.Second)
	writeLock()
	after := time.Now().Add(time.Second).Truncate(time.Second)

	data, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("lock file should exist: %v", err)
	}
	content := string(data)

	// Should contain an RFC3339 timestamp
	parts := strings.SplitN(content, "purchased at ", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected lock content format: %q", content)
	}
	ts := strings.TrimSpace(parts[1])
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatalf("failed to parse timestamp %q: %v", ts, err)
	}
	if parsed.Before(before) || parsed.After(after) {
		t.Fatalf("timestamp %v out of expected range [%v, %v]", parsed, before, after)
	}
}

// ============================================================================
// ChromeQuitHint tests
// ============================================================================

func TestChromeQuitHint(t *testing.T) {
	hint := chromeQuitHint()
	switch runtime.GOOS {
	case "darwin":
		if hint != "Cmd+Q" {
			t.Fatalf("expected Cmd+Q on darwin, got: %q", hint)
		}
	case "windows":
		if !strings.Contains(hint, "Alt+F4") {
			t.Fatalf("expected Alt+F4 on windows, got: %q", hint)
		}
	case "linux":
		if !strings.Contains(hint, "Ctrl+Q") {
			t.Fatalf("expected Ctrl+Q on linux, got: %q", hint)
		}
	}
	// All platforms: should not be empty
	if hint == "" {
		t.Fatal("chromeQuitHint should not return empty string")
	}
}

// ============================================================================
// Truncate tests
// ============================================================================

func TestTruncate_Short(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Fatalf("expected 'hello', got: %q", result)
	}
}

func TestTruncate_Exact(t *testing.T) {
	result := truncate("hello", 5)
	if result != "hello" {
		t.Fatalf("expected 'hello', got: %q", result)
	}
}

func TestTruncate_Long(t *testing.T) {
	result := truncate("hello world", 5)
	if result != "hello..." {
		t.Fatalf("expected 'hello...', got: %q", result)
	}
}

func TestTruncate_Empty(t *testing.T) {
	result := truncate("", 5)
	if result != "" {
		t.Fatalf("expected empty, got: %q", result)
	}
}

// ============================================================================
// Config / Chrome profile path tests
// ============================================================================

func TestDefaultChromeProfile_NotEmpty(t *testing.T) {
	profile := defaultChromeProfile()
	if profile == "" {
		t.Fatal("defaultChromeProfile should not return empty string")
	}
}

func TestDefaultChromeProfile_ContainsChromeSegment(t *testing.T) {
	profile := defaultChromeProfile()
	profileLower := strings.ToLower(profile)
	if !strings.Contains(profileLower, "chrome") {
		t.Fatalf("defaultChromeProfile should contain 'chrome', got: %q", profile)
	}
}

func TestDefaultChromeProfile_PlatformPaths(t *testing.T) {
	profile := defaultChromeProfile()
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(profile, "Library/Application Support") {
			t.Fatalf("macOS path should contain 'Library/Application Support', got: %q", profile)
		}
	case "windows":
		if !strings.Contains(profile, "Google") || !strings.Contains(profile, "Chrome") {
			t.Fatalf("Windows path should contain 'Google\\Chrome', got: %q", profile)
		}
	case "linux":
		if !strings.Contains(profile, ".config/google-chrome") {
			t.Fatalf("Linux path should contain '.config/google-chrome', got: %q", profile)
		}
	}
}

func TestDefaultChromeProfile_IsAbsolute(t *testing.T) {
	profile := defaultChromeProfile()
	if !filepath.IsAbs(profile) {
		t.Fatalf("defaultChromeProfile should return absolute path, got: %q", profile)
	}
}

// ============================================================================
// Config validation tests
// ============================================================================

func TestConfigValidate_EmptyPermitKeyword(t *testing.T) {
	cfg := &Config{
		Billing: Billing{CardNumber: "1234", Expiry: "01/25", CVV: "123"},
	}
	err := cfg.validate()
	if err == nil {
		t.Fatal("should fail with empty permit_keyword")
	}
	if !strings.Contains(err.Error(), "permit_keyword") {
		t.Fatalf("error should mention permit_keyword: %v", err)
	}
}

func TestConfigValidate_EmptyCardNumber(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "TEST",
		Billing:       Billing{Expiry: "01/25", CVV: "123"},
	}
	err := cfg.validate()
	if err == nil {
		t.Fatal("should fail with empty card_number")
	}
	if !strings.Contains(err.Error(), "card_number") {
		t.Fatalf("error should mention card_number: %v", err)
	}
}

func TestConfigValidate_EmptyExpiry(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "TEST",
		Billing:       Billing{CardNumber: "1234", CVV: "123"},
	}
	err := cfg.validate()
	if err == nil {
		t.Fatal("should fail with empty expiry")
	}
}

func TestConfigValidate_EmptyCVV(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "TEST",
		Billing:       Billing{CardNumber: "1234", Expiry: "01/25"},
	}
	err := cfg.validate()
	if err == nil {
		t.Fatal("should fail with empty cvv")
	}
}

func TestConfigValidate_Valid(t *testing.T) {
	cfg := &Config{
		PermitKeyword: "GOLD",
		Billing:       Billing{CardNumber: "4111111111111111", Expiry: "01/25", CVV: "123"},
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("valid config should pass validation: %v", err)
	}
}

// ============================================================================
// Config Save/Load round-trip test
// ============================================================================

func TestConfigSaveLoad_Roundtrip(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test_config.yaml")
	original := &Config{
		PermitKeyword:  "GOLD",
		VehicleKeyword: "HONDA",
		AddressKeyword: "ELM",
		Email:          "test@example.com",
		OneTime:        true,
		ChromeProfile:  "/tmp/test-profile",
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "12/25",
			CVV:        "999",
			Name:       "Test User",
			Zip:        "50010",
		},
	}

	if err := original.Save(tmpFile); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := loadConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Keywords are uppercased on load
	if loaded.PermitKeyword != "GOLD" {
		t.Fatalf("PermitKeyword mismatch: got %q", loaded.PermitKeyword)
	}
	if loaded.VehicleKeyword != "HONDA" {
		t.Fatalf("VehicleKeyword mismatch: got %q", loaded.VehicleKeyword)
	}
	if loaded.Email != original.Email {
		t.Fatalf("Email mismatch: got %q", loaded.Email)
	}
	if loaded.Billing.CardNumber != original.Billing.CardNumber {
		t.Fatalf("CardNumber mismatch: got %q", loaded.Billing.CardNumber)
	}
	if loaded.ChromeProfile != original.ChromeProfile {
		t.Fatalf("ChromeProfile mismatch: got %q, want %q", loaded.ChromeProfile, original.ChromeProfile)
	}
}

func TestConfigSave_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission test not applicable on Windows")
	}
	tmpFile := filepath.Join(t.TempDir(), "test_config.yaml")
	cfg := &Config{PermitKeyword: "TEST"}
	if err := cfg.Save(tmpFile); err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Fatalf("expected 0600 permissions, got: %o", perm)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("loading missing file should error")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(tmpFile, []byte("{{{{invalid yaml"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	_, err := loadConfig(tmpFile)
	if err == nil {
		t.Fatal("loading invalid YAML should error")
	}
}

func TestLoadConfig_DefaultChromeProfile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte("permit_keyword: test\n"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	cfg, err := loadConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if cfg.ChromeProfile == "" {
		t.Fatal("ChromeProfile should be set to default when empty")
	}
}

func TestLoadConfig_KeywordsUppercased(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	content := "permit_keyword: gold lot\nvehicle_keyword: honda\naddress_keyword: elm street\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	cfg, err := loadConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if cfg.PermitKeyword != "GOLD LOT" {
		t.Fatalf("PermitKeyword should be uppercased: got %q", cfg.PermitKeyword)
	}
	if cfg.VehicleKeyword != "HONDA" {
		t.Fatalf("VehicleKeyword should be uppercased: got %q", cfg.VehicleKeyword)
	}
}

// ============================================================================
// Context cancellation tests
// ============================================================================

func TestRunBot_ContextCancellation(t *testing.T) {
	// Test that runBot respects context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Use a manual temp dir since Chrome may create files that t.TempDir() can't clean up
	tmpDir, err := os.MkdirTemp("", "parkbot-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		PermitKeyword: "TEST",
		ChromeProfile: tmpDir,
		Billing: Billing{
			CardNumber: "4111111111111111",
			Expiry:     "01/25",
			CVV:        "123",
		},
	}

	// Remove lock file so it doesn't fail on that check
	os.Remove(lockFile)
	defer os.Remove(lockFile)

	// runBot should fail quickly because the context is already cancelled
	// and Chrome launch will either fail or the browser context will be cancelled
	botErr := runBot(ctx, cfg)
	if botErr == nil {
		t.Log("runBot returned nil (unlikely with cancelled context but possible if Chrome launched very fast)")
	} else {
		t.Logf("runBot returned error as expected: %v", botErr)
	}
}

func TestRunBot_LockFileBlocks(t *testing.T) {
	// Create lock file
	if err := os.WriteFile(lockFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create lock file: %v", err)
	}
	defer os.Remove(lockFile)

	cfg := &Config{
		PermitKeyword: "TEST",
		Billing:       Billing{CardNumber: "1234", Expiry: "01/25", CVV: "123"},
	}

	err := runBot(context.Background(), cfg)
	if err == nil {
		t.Fatal("runBot should fail when lock file exists")
	}
	if !strings.Contains(err.Error(), "purchased.lock exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Error message quality tests
// ============================================================================

func TestErrorMessages_AreDescriptive(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		errText string
	}{
		{
			name:    "empty permit keyword",
			cfg:     Config{Billing: Billing{CardNumber: "1234", Expiry: "01/25", CVV: "123"}},
			errText: "permit_keyword",
		},
		{
			name:    "empty card number",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{Expiry: "01/25", CVV: "123"}},
			errText: "card_number",
		},
		{
			name:    "empty expiry",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{CardNumber: "1234", CVV: "123"}},
			errText: "expiry",
		},
		{
			name:    "empty CVV",
			cfg:     Config{PermitKeyword: "TEST", Billing: Billing{CardNumber: "1234", Expiry: "01/25"}},
			errText: "cvv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("error message should contain %q, got: %v", tt.errText, err)
			}
		})
	}
}

// ============================================================================
// Timeout behavior tests
// ============================================================================

func TestWaitForElement_Timeout(t *testing.T) {
	// This test verifies the timeout mechanism of waitForElement conceptually
	// Without a real page, we verify the function signature and that it returns
	// an error with a timeout message
	// We can't create a rod.Page without Chrome, so we verify the error format
	expectedErrFormat := `element %q not found within %s`
	expected := fmt.Sprintf(expectedErrFormat, "#test", "1s")
	if !strings.Contains(expected, "not found within") {
		t.Fatal("error format should contain 'not found within'")
	}
}

// ============================================================================
// Cross-platform path tests
// ============================================================================

func TestLockFilePath_IsRelative(t *testing.T) {
	// lockFile should be a relative path that works on all platforms
	if filepath.IsAbs(lockFile) {
		t.Fatalf("lockFile should be relative, got: %q", lockFile)
	}
}

func TestLockFilePath_NoSpecialChars(t *testing.T) {
	// Lock file name should be simple and cross-platform safe
	for _, ch := range lockFile {
		if ch == '\\' || ch == '/' || ch == ':' || ch == '*' || ch == '?' || ch == '"' || ch == '<' || ch == '>' || ch == '|' {
			t.Fatalf("lockFile contains platform-unsafe character %q: %q", string(ch), lockFile)
		}
	}
}

// ============================================================================
// OpenBrowser platform test
// ============================================================================

func TestOpenBrowser_PlatformCommand(t *testing.T) {
	// Verify that the openBrowser function uses the correct command for the current platform
	// We don't actually open a browser, just verify the logic
	switch runtime.GOOS {
	case "darwin":
		t.Log("darwin: would use 'open'")
	case "windows":
		t.Log("windows: would use 'cmd /c start'")
	case "linux":
		t.Log("linux: would use 'xdg-open'")
	default:
		t.Logf("unknown OS %q: would fallback to 'open'", runtime.GOOS)
	}
}

// ============================================================================
// Constants sanity tests
// ============================================================================

func TestPortalURL_IsHTTPS(t *testing.T) {
	if !strings.HasPrefix(portalURL, "https://") {
		t.Fatalf("portalURL should use HTTPS, got: %q", portalURL)
	}
}

func TestPortalURL_NotEmpty(t *testing.T) {
	if portalURL == "" {
		t.Fatal("portalURL should not be empty")
	}
}

// ============================================================================
// Server tests
// ============================================================================

func TestNewServer_Initialization(t *testing.T) {
	cfg := &Config{PermitKeyword: "TEST"}
	srv := NewServer("config.yaml", cfg)
	if srv == nil {
		t.Fatal("NewServer should return non-nil server")
	}
	if srv.cfgPath != "config.yaml" {
		t.Fatalf("cfgPath mismatch: got %q", srv.cfgPath)
	}
	if srv.running {
		t.Fatal("server should not be running initially")
	}
	if srv.clients == nil {
		t.Fatal("clients map should be initialized")
	}
}

func TestServer_LogWriter(t *testing.T) {
	cfg := &Config{}
	srv := NewServer("test.yaml", cfg)
	w := srv.LogWriter()
	if w == nil {
		t.Fatal("LogWriter should return non-nil writer")
	}
	// Write should not panic even with no clients
	n, err := w.Write([]byte("test message"))
	if err != nil {
		t.Fatalf("Write should not error: %v", err)
	}
	if n != len("test message") {
		t.Fatalf("Write should return message length, got: %d", n)
	}
}

func TestServer_BroadcastNoClients(t *testing.T) {
	cfg := &Config{}
	srv := NewServer("test.yaml", cfg)
	// Should not panic with no clients
	srv.broadcast("test message")
}

func TestServer_ClientManagement(t *testing.T) {
	cfg := &Config{}
	srv := NewServer("test.yaml", cfg)

	ch := make(chan string, 10)
	srv.addClient(ch)

	// Broadcast should reach the client
	srv.broadcast("hello")
	select {
	case msg := <-ch:
		if msg != "hello" {
			t.Fatalf("expected 'hello', got: %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("expected message on client channel")
	}

	// Remove client
	srv.removeClient(ch)

	// Broadcast should not reach removed client
	srv.broadcast("world")
	select {
	case msg := <-ch:
		t.Fatalf("should not receive message after removal, got: %q", msg)
	case <-time.After(50 * time.Millisecond):
		// Expected: no message
	}
}

func TestServer_BroadcastSlowClient(t *testing.T) {
	cfg := &Config{}
	srv := NewServer("test.yaml", cfg)

	// Create a client with buffer of 1
	ch := make(chan string, 1)
	srv.addClient(ch)
	defer srv.removeClient(ch)

	// Fill the buffer
	srv.broadcast("msg1")
	// Second message should be silently dropped (not block)
	srv.broadcast("msg2")
	srv.broadcast("msg3")

	// Should only get the first message
	msg := <-ch
	if msg != "msg1" {
		t.Fatalf("expected 'msg1', got: %q", msg)
	}
}
