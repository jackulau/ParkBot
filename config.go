package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type Billing struct {
	CardNumber string `yaml:"card_number" json:"card_number"`
	Expiry     string `yaml:"expiry"      json:"expiry"`
	CVV        string `yaml:"cvv"         json:"cvv"`
	Name       string `yaml:"name"        json:"name"`
	Zip        string `yaml:"zip"         json:"zip"`
}

type Config struct {
	PermitKeyword  string  `yaml:"permit_keyword"  json:"permit_keyword"`
	VehicleKeyword string  `yaml:"vehicle_keyword" json:"vehicle_keyword"`
	AddressKeyword string  `yaml:"address_keyword" json:"address_keyword"`
	Email          string  `yaml:"email"           json:"email"`
	OneTime        bool    `yaml:"one_time"        json:"one_time"`
	ChromeProfile  string  `yaml:"chrome_profile"  json:"chrome_profile"`
	Billing        Billing `yaml:"billing"         json:"billing"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.ChromeProfile == "" {
		cfg.ChromeProfile = defaultChromeProfile()
	}

	// Normalize keywords to uppercase for case-insensitive matching
	cfg.PermitKeyword = strings.ToUpper(strings.TrimSpace(cfg.PermitKeyword))
	cfg.VehicleKeyword = strings.ToUpper(strings.TrimSpace(cfg.VehicleKeyword))
	cfg.AddressKeyword = strings.ToUpper(strings.TrimSpace(cfg.AddressKeyword))

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.PermitKeyword == "" {
		return fmt.Errorf("permit_keyword must not be empty")
	}
	if c.Billing.CardNumber == "" {
		return fmt.Errorf("billing.card_number must not be empty")
	}
	if c.Billing.Expiry == "" {
		return fmt.Errorf("billing.expiry must not be empty")
	}
	if c.Billing.CVV == "" {
		return fmt.Errorf("billing.cvv must not be empty")
	}
	return nil
}

// Save writes the config to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// defaultConfigDir returns the platform-specific configuration directory for
// ParkBot. On Linux, it respects the XDG Base Directory specification by using
// XDG_CONFIG_HOME if set, falling back to ~/.config/parkbot.
func defaultConfigDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "linux":
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(home, ".config")
		}
		return filepath.Join(base, "parkbot")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "ParkBot")
		}
		return filepath.Join(home, "AppData", "Roaming", "ParkBot")
	default: // darwin
		return filepath.Join(home, "Library", "Application Support", "ParkBot")
	}
}

// defaultChromeProfile returns the default Chrome/Chromium user data directory
// for the current platform. On Linux, it checks for Chrome, Chromium, Snap, and
// Flatpak installations in order of preference, returning the first path that
// exists on disk. This handles the variety of browser packaging on Linux.
func defaultChromeProfile() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Google", "Chrome", "User Data", "Default")
		}
		return filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default")
	case "linux":
		return linuxChromeProfile(home)
	default: // darwin
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default")
	}
}

// linuxChromeProfile checks multiple well-known Chrome/Chromium profile
// locations on Linux. Returns the first path that exists, or falls back to the
// standard google-chrome location. Checked paths (in order):
//  1. ~/.config/google-chrome/Default          (standard Chrome)
//  2. ~/.config/chromium/Default               (Chromium)
//  3. ~/snap/chromium/common/chromium/Default   (Snap Chromium)
//  4. ~/.var/app/com.google.Chrome/config/google-chrome/Default (Flatpak Chrome)
func linuxChromeProfile(home string) string {
	configBase := os.Getenv("XDG_CONFIG_HOME")
	if configBase == "" {
		configBase = filepath.Join(home, ".config")
	}

	candidates := []string{
		filepath.Join(configBase, "google-chrome", "Default"),
		filepath.Join(configBase, "chromium", "Default"),
		filepath.Join(home, "snap", "chromium", "common", "chromium", "Default"),
		filepath.Join(home, ".var", "app", "com.google.Chrome", "config", "google-chrome", "Default"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Default fallback: standard Chrome path even if it doesn't exist yet
	return filepath.Join(configBase, "google-chrome", "Default")
}
