// Copyright 2026 coScene
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

package user_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/coscene-io/cocli/pkg/cmd/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig(t *testing.T) string {
	t.Helper()
	tmpDir := testutil.TempDir(t)
	return filepath.Join(tmpDir, "test-config.yaml")
}

func TestUserCommand(t *testing.T) {
	t.Run("Root command structure", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := user.NewRootCommand(&cfgPath, io, config.Provide)

		assert.Equal(t, "user", cmd.Use)
		assert.NotEmpty(t, cmd.Short)

		expectedSubcommands := []string{"list", "get"}

		for _, expected := range expectedSubcommands {
			found := false
			for _, sub := range cmd.Commands() {
				if sub.Name() == expected {
					found = true
					assert.NotEmpty(t, sub.Short, "Command %s should have a short description", sub.Name())
					break
				}
			}
			assert.True(t, found, "Subcommand %s not found", expected)
		}
	})

	t.Run("List command flags", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := user.NewRootCommand(&cfgPath, io, config.Provide)

		listCmd, _, err := cmd.Find([]string{"list"})
		require.NoError(t, err)

		for _, flag := range []string{"project", "role-code", "verbose", "output", "page-size", "page-token"} {
			assert.NotNil(t, listCmd.Flag(flag), "Flag --%s not found", flag)
		}
	})

	t.Run("Get command flags", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := user.NewRootCommand(&cfgPath, io, config.Provide)

		getCmd, _, err := cmd.Find([]string{"get"})
		require.NoError(t, err)

		for _, flag := range []string{"verbose", "output"} {
			assert.NotNil(t, getCmd.Flag(flag), "Flag --%s not found", flag)
		}
	})

	t.Run("Get command accepts zero or one arg", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := user.NewRootCommand(&cfgPath, io, config.Provide)

		getCmd, _, err := cmd.Find([]string{"get"})
		require.NoError(t, err)

		assert.NoError(t, getCmd.Args(getCmd, []string{}))
		assert.NoError(t, getCmd.Args(getCmd, []string{"users/some-id"}))
		assert.Error(t, getCmd.Args(getCmd, []string{"a", "b"}))
	})

	t.Run("List command output flag default", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := user.NewRootCommand(&cfgPath, io, config.Provide)

		listCmd, _, err := cmd.Find([]string{"list"})
		require.NoError(t, err)

		outputFlag := listCmd.Flag("output")
		require.NotNil(t, outputFlag)
		assert.Equal(t, "", outputFlag.DefValue)
	})

	t.Run("Get command output flag default", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := user.NewRootCommand(&cfgPath, io, config.Provide)

		getCmd, _, err := cmd.Find([]string{"get"})
		require.NoError(t, err)

		outputFlag := getCmd.Flag("output")
		require.NotNil(t, outputFlag)
		assert.Equal(t, "table", outputFlag.DefValue)
	})
}
