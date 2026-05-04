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

package registry

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryRootCommand(t *testing.T) {
	cfgPath := filepath.Join(testutil.TempDir(t), "config.yaml")
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)

	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	assert.Equal(t, "registry", cmd.Use)
	assert.False(t, cmd_utils.IsAuthCheckEnabled(cmd))
	for _, name := range []string{"login", "create-credential"} {
		sub, _, err := cmd.Find([]string{name})
		require.NoError(t, err)
		assert.Equal(t, name, sub.Name())
		assert.NotEmpty(t, sub.Short)
	}
}

func TestCreateCredentialCommandShape(t *testing.T) {
	cfgPath := filepath.Join(testutil.TempDir(t), "config.yaml")
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)

	cmd := NewCreateCredentialCommand(&cfgPath, io, config.Provide)

	assert.Equal(t, "create-credential", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NoError(t, cmd.Args(cmd, nil))
	assert.Error(t, cmd.Args(cmd, []string{"extra"}))
	require.NotNil(t, cmd.Flag("output"))
	assert.Equal(t, "o", cmd.Flag("output").Shorthand)
}
