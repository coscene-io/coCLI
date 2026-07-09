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

package action

import (
	"bytes"
	"strings"
	"testing"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The delete command must register with the -p/--project and -f/--force flags.
func TestDeleteCommandFlags(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	deleteCmd, _, err := cmd.Find([]string{"delete"})
	require.NoError(t, err)
	assert.Equal(t, "delete", deleteCmd.Name())
	assert.NotEmpty(t, deleteCmd.Short)
	assert.NotNil(t, deleteCmd.Flag("project"), "flag --project not found")
	assert.NotNil(t, deleteCmd.Flag("force"), "flag --force not found")
	assert.Equal(t, "p", deleteCmd.Flag("project").Shorthand)
	assert.Equal(t, "f", deleteCmd.Flag("force").Shorthand)
}

// delete requires exactly one positional argument (the action resource-name/id).
func TestDeleteCommandRequiresExactlyOneArg(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	deleteCmd, _, err := cmd.Find([]string{"delete"})
	require.NoError(t, err)
	assert.Error(t, deleteCmd.Args(deleteCmd, []string{}), "zero args should be rejected")
	assert.NoError(t, deleteCmd.Args(deleteCmd, []string{"a1"}))
	assert.Error(t, deleteCmd.Args(deleteCmd, []string{"a1", "a2"}), "two args should be rejected")
}

// The trigger side-effect (plan F8) must be surfaced in the confirmation prompt
// wording AND the help text, since cocli cannot count the cascaded triggers.
func TestDeleteCommandSurfacesTriggerSideEffect(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	deleteCmd, _, err := cmd.Find([]string{"delete"})
	require.NoError(t, err)

	// The constant that both the prompt and help text reuse names the effect.
	assert.Contains(t, deleteTriggerNote, "triggers")
	// Help text repeats the trigger note.
	assert.Contains(t, deleteCmd.Long, deleteTriggerNote)
}

func setupActionCmdTestConfigPath(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(t.TempDir()) + "/test-config.yaml"
}
