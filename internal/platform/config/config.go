package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type FolderConfig struct {
	MaxConcurrency int    `json:"max_concurrency,omitempty"`
	PartSize       int64  `json:"part_size,omitempty"`
	StorageClass   string `json:"storage_class,omitempty"`
}

type Config struct {
	ConfigDir             string `json:"config_dir"`
	DataDir               string `json:"data_dir"`
	DBPath                string `json:"db_path"`
	SecretsDir            string `json:"secrets_dir"`
	LogLevel              string `json:"log_level"`
	MaxUploadConcurrency int    `json:"max_upload_concurrency"`
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

	cfg := &Config{
		ConfigDir:             appConfigDir,
		DataDir:               dataDir,
		DBPath:                dbPath,
		SecretsDir:            secretsDir,
		LogLevel:              "INFO",
		MaxUploadConcurrency: 100,
	}

	configFile := filepath.Join(appConfigDir, "config.json")
	if data, err := os.ReadFile(configFile); err == nil && len(data) > 0 {
		var diskCfg Config
		if json.Unmarshal(data, &diskCfg) == nil {
			if diskCfg.LogLevel != "" {
				cfg.LogLevel = diskCfg.LogLevel
			}
			if diskCfg.MaxUploadConcurrency > 0 {
				cfg.MaxUploadConcurrency = diskCfg.MaxUploadConcurrency
			}
		}
	}

	return cfg, nil
}

func (c *Config) Save() error {
	if err := c.EnsureDirs(); err != nil {
		return err
	}
	configFile := filepath.Join(c.ConfigDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0600)
}

func LoadFolderConfig(folderPath string) (*FolderConfig, error) {
	candidates := []string{
		filepath.Join(folderPath, ".kumokura.json"),
		filepath.Join(folderPath, "kumokura.json"),
	}
	for _, c := range candidates {
		data, err := os.ReadFile(c)
		if err == nil && len(data) > 0 {
			var fc FolderConfig
			if err := json.Unmarshal(data, &fc); err == nil {
				return &fc, nil
			}
		}
	}
	return nil, nil
}

func (c *Config) EnsureDirs() error {
	for _, dir := range []string{c.ConfigDir, c.DataDir, c.SecretsDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
