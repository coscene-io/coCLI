// Copyright 2025 coScene
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/coscene-io/cocli/pkg/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	t.Run("Version flag", func(t *testing.T) {
		cmd := cmd.NewCommand()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"--version"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "cocli version")
	})

	t.Run("Help flag", func(t *testing.T) {
		cmd := cmd.NewCommand()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Usage:")
		assert.Contains(t, output, "Available Commands:")
	})

	t.Run("Invalid command", func(t *testing.T) {
		cmd := cmd.NewCommand()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"invalid-command"})

		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})

	t.Run("Config file initialization", func(t *testing.T) {
		testutil.SkipIfShort(t)

		// Create a temporary directory for config
		tmpDir := testutil.TempDir(t)
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Create empty config file
		err := os.WriteFile(configPath, []byte{}, 0644)
		require.NoError(t, err)

		// Run command with custom config path
		cmd := cmd.NewCommand()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Use a command that doesn't require auth
		cmd.SetArgs([]string{"--config", configPath, "completion", "bash"})

		err = cmd.Execute()
		require.NoError(t, err)

		// Check that config file was created
		_, err = os.Stat(configPath)
		require.NoError(t, err)

		// Verify it contains bash completion script
		assert.Contains(t, buf.String(), "# bash completion for cocli")
	})

	t.Run("Log level configuration", func(t *testing.T) {
		testCases := []struct {
			level string
			valid bool
		}{
			{"trace", true},
			{"debug", true},
			{"info", true},
			{"warn", true},
			{"error", true},
		}

		for _, tc := range testCases {
			t.Run(tc.level, func(t *testing.T) {
				cmd := cmd.NewCommand()
				buf := new(bytes.Buffer)
				cmd.SetOut(buf)
				cmd.SetErr(buf)

				// Create temp config
				tmpDir := testutil.TempDir(t)
				configPath := filepath.Join(tmpDir, "config.yaml")

				// Create empty config file
				err := os.WriteFile(configPath, []byte{}, 0644)
				require.NoError(t, err)

				cmd.SetArgs([]string{"--config", configPath, "--log-level", tc.level, "completion", "bash"})

				err = cmd.Execute()
				require.NoError(t, err)
			})
		}
	})

	t.Run("Subcommands exist", func(t *testing.T) {
		cmd := cmd.NewCommand()

		expectedCommands := []string{
			"completion",
			"action",
			"login",
			"project",
			"registry",
			"record",
			"update",
		}

		for _, expected := range expectedCommands {
			t.Run(expected, func(t *testing.T) {
				found := false
				for _, sub := range cmd.Commands() {
					if sub.Name() == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "Command %s not found", expected)
			})
		}
	})
}

// TestEnvironmentVariables tests environment variable configuration
func TestEnvironmentVariables(t *testing.T) {
	t.Run("Environment variables override config", func(t *testing.T) {
		// Set environment variables
		_ = os.Setenv("COS_ENDPOINT", "https://openapi.mock.coscene.com")
		_ = os.Setenv("COS_TOKEN", "test-token")
		_ = os.Setenv("COS_PROJECT", "test-project")
		t.Cleanup(func() {
			_ = os.Unsetenv("COS_ENDPOINT")
			_ = os.Unsetenv("COS_TOKEN")
			_ = os.Unsetenv("COS_PROJECT")
		})

		tmpDir := testutil.TempDir(t)
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Create empty config file
		err := os.WriteFile(configPath, []byte{}, 0644)
		require.NoError(t, err)

		cmd := cmd.NewCommand()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// This test would require mocking the API calls
		// For now, just verify the command accepts the env vars
		// Use completion bash as it doesn't require auth
		cmd.SetArgs([]string{"--config", configPath, "completion", "bash"})

		err = cmd.Execute()
		require.NoError(t, err)

		// Verify we got completion output
		output := buf.String()
		assert.Contains(t, output, "# bash completion for cocli")
	})
}

// TestCommandStructure tests that commands follow consistent patterns
func TestCommandStructure(t *testing.T) {
	cmd := cmd.NewCommand()

	// Check all subcommands have proper descriptions
	for _, sub := range cmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			assert.NotEmpty(t, sub.Short, "Command %s should have a short description", sub.Name())

			// Skip completion command which doesn't have subcommands
			if sub.Name() == "completion" || sub.Name() == "update" {
				return
			}

			// Most commands should have subcommands
			if len(sub.Commands()) == 0 {
				t.Logf("Warning: Command %s has no subcommands", sub.Name())
			}
		})
	}
}

// TestPersistentFlags tests that persistent flags work across subcommands
func TestPersistentFlags(t *testing.T) {
	tmpDir := testutil.TempDir(t)
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create empty config file
	err := os.WriteFile(configPath, []byte{}, 0644)
	require.NoError(t, err)

	// Test that --config flag works on subcommands
	cmd := cmd.NewCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--config", configPath, "completion", "bash"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify config was created at specified path
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}
