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

package project_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/coscene-io/cocli/pkg/cmd/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig(t *testing.T) string {
	t.Helper()
	tmpDir := testutil.TempDir(t)
	return filepath.Join(tmpDir, "test-config.yaml")
}

func TestProjectCommand(t *testing.T) {
	t.Run("Root command structure", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := project.NewRootCommand(&cfgPath, io, config.Provide)

		assert.Equal(t, "project", cmd.Use)
		assert.NotEmpty(t, cmd.Short)

		expectedSubcommands := []string{"list", "create", "file"}

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
		cmd := project.NewRootCommand(&cfgPath, io, config.Provide)

		listCmd, _, err := cmd.Find([]string{"list"})
		require.NoError(t, err)

		for _, flag := range []string{"all", "keywords", "page", "page-size", "output", "include-archive"} {
			assert.NotNil(t, listCmd.Flag(flag), "Flag --%s not found", flag)
		}
	})

	t.Run("Create command flags", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := project.NewRootCommand(&cfgPath, io, config.Provide)

		createCmd, _, err := cmd.Find([]string{"create"})
		require.NoError(t, err)

		for _, flag := range []string{"project-slug", "display-name", "output", "region", "filesystem"} {
			assert.NotNil(t, createCmd.Flag(flag), "Flag --%s not found", flag)
		}
	})

	t.Run("File command flags", func(t *testing.T) {
		cfgPath := setupTestConfig(t)
		var buf bytes.Buffer
		io := iostreams.Test(nil, &buf, &buf)
		cmd := project.NewRootCommand(&cfgPath, io, config.Provide)

		fileCmd, _, err := cmd.Find([]string{"file"})
		require.NoError(t, err)

		expectedSubcmds := []string{"list", "upload", "download"}
		for _, expected := range expectedSubcmds {
			found := false
			for _, sub := range fileCmd.Commands() {
				if sub.Name() == expected {
					found = true
					break
				}
			}
			assert.True(t, found, "File subcommand %s not found", expected)
		}
	})
}
