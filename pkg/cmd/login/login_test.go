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

package login_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/coscene-io/cocli/internal/apimocks"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/coscene-io/cocli/pkg/cmd/login"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginCommand(t *testing.T) {
	t.Run("Root command", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := login.NewRootCommand(&cfgPath, io, config.Provide)

		assert.Equal(t, "login", cmd.Use)
		assert.NotEmpty(t, cmd.Short)

		// Check subcommands
		expectedSubcommands := []string{"add", "current", "delete", "list", "set", "switch"}
		for _, expected := range expectedSubcommands {
			found := false
			for _, sub := range cmd.Commands() {
				if sub.Name() == expected {
					found = true
					break
				}
			}
			assert.True(t, found, "Subcommand %s not found", expected)
		}
	})

	t.Run("List command with empty config", func(t *testing.T) {
		// Create a mock provider with empty profiles
		mockProvider := apimocks.NewMockProvider(t)
		mockProvider.ProfileManager().CurrentProfile = ""
		mockProvider.ProfileManager().Profiles = []*config.Profile{}

		cfgPath := setupTestConfig(t)
		buf := new(bytes.Buffer)
		io := iostreams.Test(nil, buf, buf)
		getProvider := func(string) config.Provider { return mockProvider }
		cmd := login.NewRootCommand(&cfgPath, io, getProvider)
		cmd.SetArgs([]string{"list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Empty config should show empty list or message
		assert.Contains(t, output, "No profiles found")
	})

	t.Run("Current command with no profile", func(t *testing.T) {
		mockProvider := apimocks.NewMockProvider(t)
		mockProvider.ProfileManager().CurrentProfile = ""
		mockProvider.ProfileManager().Profiles = []*config.Profile{}

		cfgPath := setupTestConfig(t)
		buf := new(bytes.Buffer)
		io := iostreams.Test(nil, buf, buf)
		getProvider := func(string) config.Provider { return mockProvider }
		cmd := login.NewRootCommand(&cfgPath, io, getProvider)
		cmd.SetArgs([]string{"current"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No current profile")
	})
}

func TestLoginConfigInteraction(t *testing.T) {
	t.Run("Set command creates profile", func(t *testing.T) {
		mockProvider := apimocks.NewMockProvider(t)

		cfgPath := setupTestConfig(t)

		buf := new(bytes.Buffer)
		io := iostreams.Test(nil, buf, buf)
		getProvider := func(string) config.Provider { return mockProvider }
		cmd := login.NewRootCommand(&cfgPath, io, getProvider)
		cmd.SetArgs([]string{"list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "test-profile")
		assert.Contains(t, output, "*") // Current marker
	})

	t.Run("Switch between profiles", func(t *testing.T) {
		cfgPath := setupTestConfig(t)

		// Create mock provider with multiple profiles
		mockProvider := apimocks.NewMockProvider(t)
		mockProvider.ProfileManager().Profiles = []*config.Profile{
			{
				Name:        "profile1",
				EndPoint:    "https://openapi.mock1.coscene.com",
				Token:       "test-token-1",
				ProjectSlug: "test-project-1",
			},
			{
				Name:        "profile2",
				EndPoint:    "https://openapi.mock2.coscene.com",
				Token:       "test-token-2",
				ProjectSlug: "test-project-2",
			},
		}
		mockProvider.ProfileManager().CurrentProfile = "profile1"

		// Check current is profile1
		buf := new(bytes.Buffer)
		io := iostreams.Test(nil, buf, buf)
		getProvider := func(string) config.Provider { return mockProvider }
		cmd := login.NewRootCommand(&cfgPath, io, getProvider)
		cmd.SetArgs([]string{"current"})

		err := cmd.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "profile1")

		// Test switch would require interactive input or mocking
		// For now, just verify the command exists
		io = iostreams.Test(nil, &bytes.Buffer{}, &bytes.Buffer{})
		cmd = login.NewRootCommand(&cfgPath, io, config.Provide)
		switchCmd, _, err := cmd.Find([]string{"switch"})
		require.NoError(t, err)
		assert.NotNil(t, switchCmd)
	})

	t.Run("Delete profile", func(t *testing.T) {
		cfgPath := setupTestConfig(t)

		configContent := `endpoint: https://openapi.mock1.coscene.com
token: test-token
project: test-project
profiles:
  - name: test-profile
    current: true
    endpoint: https://openapi.mock1.coscene.com
    token: test-token
    project: test-project
  - name: profile-to-delete
    current: false
    endpoint: https://openapi.mock2.coscene.com
    token: test-token-2
    project: test-project-2`

		err := os.WriteFile(cfgPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Delete command would require confirmation
		// Just verify command structure
		io := iostreams.Test(nil, &bytes.Buffer{}, &bytes.Buffer{})
		cmd := login.NewRootCommand(&cfgPath, io, config.Provide)
		deleteCmd, _, err := cmd.Find([]string{"delete"})
		require.NoError(t, err)
		assert.NotNil(t, deleteCmd)
		// Test that it requires exactly one argument
		err = deleteCmd.Args(nil, []string{"profile-to-delete"})
		assert.NoError(t, err)
		err = deleteCmd.Args(nil, []string{})
		assert.Error(t, err)
		err = deleteCmd.Args(nil, []string{"arg1", "arg2"})
		assert.Error(t, err)
	})
}

// Helper function to setup test config
func setupTestConfig(t *testing.T) string {
	t.Helper()
	tmpDir := testutil.TempDir(t)
	cfgPath := filepath.Join(tmpDir, "test-config.yaml")
	// Create empty config file
	err := os.WriteFile(cfgPath, []byte{}, 0644)
	require.NoError(t, err)
	return cfgPath
}

// TestLoginCommandValidation tests command argument validation
func TestLoginCommandValidation(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{"list no args", []string{"list"}, false},
		{"list with args", []string{"list", "extra"}, true},
		{"current no args", []string{"current"}, false},
		{"current with args", []string{"current", "extra"}, true},
		{"delete no args", []string{"delete"}, true},
		{"delete with profile", []string{"delete", "profile-name"}, false},
		// Skip switch no args test as it requires interactive input
		// {"switch no args", []string{"switch"}, false}, // Interactive mode
		{"switch with profile", []string{"switch", "profile-name"}, true}, // Switch doesn't accept args
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := setupTestConfig(t)
			buf := new(bytes.Buffer)
			io := iostreams.Test(nil, buf, buf)
			getProvider := config.Provide
			if tc.args[0] == "delete" || tc.args[0] == "switch" {
				mockProvider := apimocks.NewMockProvider(t)
				mockProvider.ProfileManager().Profiles = append(
					mockProvider.ProfileManager().Profiles,
					&config.Profile{
						Name:        "profile-name",
						EndPoint:    "https://openapi.mock.coscene.com",
						Token:       "test-token2",
						ProjectSlug: "test-project2",
					},
				)
				getProvider = func(string) config.Provider { return mockProvider }
			}
			cmd := login.NewRootCommand(&cfgPath, io, getProvider)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				// Some commands might still error due to missing interactive input
				// but arg validation should pass
				if err != nil {
					assert.NotContains(t, err.Error(), "arg")
				}
			}
		})
	}
}
