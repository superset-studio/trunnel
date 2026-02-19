package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CLIConfig holds client-side session state persisted between CLI invocations.
type CLIConfig struct {
	ServerURL    string `json:"serverUrl"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ActiveOrgID  string `json:"activeOrgId"`
	UserEmail    string `json:"userEmail"`
	UserName     string `json:"userName"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kapstan", "config.json")
}

// LoadConfig reads the CLI config from ~/.kapstan/config.json.
// Returns an empty config if the file does not exist.
func LoadConfig() (*CLIConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &CLIConfig{}, nil
		}
		return nil, err
	}

	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig writes the CLI config to ~/.kapstan/config.json with 0600 permissions.
func SaveConfig(cfg *CLIConfig) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

// ClearConfig removes the config file.
func ClearConfig() error {
	err := os.Remove(configPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
