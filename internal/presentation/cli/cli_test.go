package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sekai-labs/kumokura/internal/bootstrap"
	"github.com/sekai-labs/kumokura/internal/platform/config"
	"github.com/sekai-labs/kumokura/internal/presentation/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestApp(t *testing.T) *bootstrap.AppContainer {
	tmp := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  filepath.Join(tmp, "config"),
		DataDir:    filepath.Join(tmp, "data"),
		DBPath:     filepath.Join(tmp, "test.db"),
		SecretsDir: filepath.Join(tmp, "secrets"),
		LogLevel:   "INFO",
	}
	require.NoError(t, cfg.EnsureDirs())

	ctx := context.Background()
	app, err := bootstrap.InitializeWithConfig(ctx, cfg)
	require.NoError(t, err)
	return app
}

func executeCommand(app *bootstrap.AppContainer, args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := cli.NewRootCmd(app, stdout, stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestVersionCommand(t *testing.T) {
	app := setupTestApp(t)
	defer app.Close()

	out, _, err := executeCommand(app, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "kumokura v0.1.0")

	outJSON, _, err := executeCommand(app, "version", "--json")
	require.NoError(t, err)
	var v map[string]string
	err = json.Unmarshal([]byte(outJSON), &v)
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", v["version"])
}

func TestAccountCommands(t *testing.T) {
	app := setupTestApp(t)
	defer app.Close()

	out, _, err := executeCommand(app, "account", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "ID")

	outAdd, _, err := executeCommand(app, "account", "add",
		"--name", "test-acc",
		"--type", "AWS",
		"--region", "us-east-1",
		"--access-key", "AKIAIOSFODNN7EXAMPLE",
		"--secret-key", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"--json",
	)
	require.NoError(t, err)
	var created map[string]any
	require.NoError(t, json.Unmarshal([]byte(outAdd), &created))
	assert.Equal(t, "test-acc", created["Name"])

	outList, _, err := executeCommand(app, "account", "list", "--json")
	require.NoError(t, err)
	var list []map[string]any
	require.NoError(t, json.Unmarshal([]byte(outList), &list))
	assert.NotEmpty(t, list)
}

func TestConfigCommands(t *testing.T) {
	app := setupTestApp(t)
	defer app.Close()

	out, _, err := executeCommand(app, "config", "show")
	require.NoError(t, err)
	assert.Contains(t, out, "LogLevel")

	outGet, _, err := executeCommand(app, "config", "get", "LogLevel")
	require.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(outGet), "INFO")

	_, _, err = executeCommand(app, "config", "set", "LogLevel", "DEBUG")
	require.NoError(t, err)

	outGetUpdated, _, err := executeCommand(app, "config", "get", "LogLevel", "--json")
	require.NoError(t, err)
	var m map[string]string
	require.NoError(t, json.Unmarshal([]byte(outGetUpdated), &m))
	assert.Equal(t, "DEBUG", m["LogLevel"])
}

func TestRootHelp(t *testing.T) {
	app := setupTestApp(t)
	defer app.Close()

	out, _, err := executeCommand(app, "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "kumokura [command]")
	assert.Contains(t, out, "account")
	assert.Contains(t, out, "bucket")
	assert.Contains(t, out, "object")
	assert.Contains(t, out, "sync")
	assert.Contains(t, out, "transfer")
	assert.Contains(t, out, "config")
}
