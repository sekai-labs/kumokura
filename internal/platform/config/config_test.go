package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sekai-labs/kumokura/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg, err := config.DefaultConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.NotEmpty(t, cfg.ConfigDir)
	assert.NotEmpty(t, cfg.DataDir)
	assert.NotEmpty(t, cfg.DBPath)
	assert.NotEmpty(t, cfg.SecretsDir)
	assert.Equal(t, "INFO", cfg.LogLevel)
	assert.Equal(t, filepath.Join(cfg.DataDir, "kumokura.db"), cfg.DBPath)
}

func TestEnsureDirs(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  filepath.Join(tmp, "config"),
		DataDir:    filepath.Join(tmp, "data"),
		SecretsDir: filepath.Join(tmp, "secrets"),
		DBPath:     filepath.Join(tmp, "data", "test.db"),
	}

	err := cfg.EnsureDirs()
	require.NoError(t, err)

	for _, dir := range []string{cfg.ConfigDir, cfg.DataDir, cfg.SecretsDir} {
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	}
}

func TestConfigSaveAndFolderConfig(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		ConfigDir:             filepath.Join(tmp, "config"),
		DataDir:               filepath.Join(tmp, "data"),
		SecretsDir:            filepath.Join(tmp, "secrets"),
		DBPath:                filepath.Join(tmp, "data", "test.db"),
		LogLevel:              "DEBUG",
		MaxUploadConcurrency: 100,
	}
	err := cfg.Save()
	require.NoError(t, err)

	folderDir := filepath.Join(tmp, "myfolder")
	require.NoError(t, os.MkdirAll(folderDir, 0755))
	folderCfgFile := filepath.Join(folderDir, ".kumokura.json")
	require.NoError(t, os.WriteFile(folderCfgFile, []byte(`{"max_concurrency": 80}`), 0644))

	fc, err := config.LoadFolderConfig(folderDir)
	require.NoError(t, err)
	require.NotNil(t, fc)
	assert.Equal(t, 80, fc.MaxConcurrency)
}
