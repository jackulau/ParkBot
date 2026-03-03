package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const appName = "ParkBot"

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

// appDataDir returns the platform-appropriate directory for ParkBot application
// data (config, lock files, etc.). It creates the directory if it does not exist.
//
//   - macOS:   ~/Library/Application Support/ParkBot
//   - Windows: %APPDATA%\ParkBot  (roaming app data)
//   - Linux:   ~/.config/ParkBot
func appDataDir() string {
	return appDataDirForOS(runtime.GOOS)
}

// appDataDirForOS is the testable core of appDataDir. It returns the
// platform-appropriate application data directory for the given OS string
// without creating it on disk.
func appDataDirForOS(goos string) string {
	home, _ := os.UserHomeDir()

	switch goos {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, appName)
		}
		return filepath.Join(home, "AppData", "Roaming", appName)
	case "linux":
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, appName)
		}
		return filepath.Join(home, ".config", appName)
	default: // darwin
		return filepath.Join(home, "Library", "Application Support", appName)
	}
}

// defaultConfigPath returns the platform-appropriate path for the config file.
// On first run (no config exists yet) the directory is created automatically.
func defaultConfigPath() string {
	dir := appDataDir()
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.yaml")
}

// defaultLockFilePath returns the platform-appropriate path for the purchase
// lock file (purchased.lock). The parent directory is created if needed.
func defaultLockFilePath() string {
	dir := appDataDir()
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "purchased.lock")
}

// defaultChromeProfile returns the default Chrome user-data profile directory
// for the current platform.
func defaultChromeProfile() string {
	return chromeProfileForOS(runtime.GOOS)
}

// chromeProfileForOS is the testable core of defaultChromeProfile.
func chromeProfileForOS(goos string) string {
	home, _ := os.UserHomeDir()

	switch goos {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Google", "Chrome", "User Data", "Default")
		}
		return filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default")
	case "linux":
		return filepath.Join(home, ".config", "google-chrome", "Default")
	default: // darwin
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default")
	}
}
