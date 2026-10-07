package config

import (
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	ConfigDir  string
	DataDir    string
	DBPath     string
	SecretsDir string
	LogLevel   string
}

func DefaultConfig() (*Config, error) {
	baseConfigDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, hErr := os.UserHomeDir()
		if hErr != nil {
			return nil, errors.Join(err, hErr)
		}
		baseConfigDir = filepath.Join(homeDir, ".config")
	}

	appConfigDir := filepath.Join(baseConfigDir, "kumokura")
	dataDir := filepath.Join(appConfigDir, "data")
	secretsDir := filepath.Join(appConfigDir, "secrets")
	dbPath := filepath.Join(dataDir, "kumokura.db")

	return &Config{
		ConfigDir:  appConfigDir,
		DataDir:    dataDir,
		DBPath:     dbPath,
		SecretsDir: secretsDir,
		LogLevel:   "INFO",
	}, nil
}

func (c *Config) EnsureDirs() error {
	for _, dir := range []string{c.ConfigDir, c.DataDir, c.SecretsDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
