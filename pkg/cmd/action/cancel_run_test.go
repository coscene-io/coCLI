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
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeActionRunTerminator struct {
	got *name.ActionRun
	err error
}

func (f *fakeActionRunTerminator) TerminateActionRun(_ context.Context, actionRun *name.ActionRun) error {
	f.got = actionRun
	return f.err
}

func TestCancelRunCommandRegistrationAndFlags(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &out)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	cancelCmd, _, err := cmd.Find([]string{"cancel-run"})
	require.NoError(t, err)
	assert.Equal(t, "cancel-run", cancelCmd.Name())
	assert.NotNil(t, cancelCmd.Flag("project"))
	assert.Equal(t, "p", cancelCmd.Flag("project").Shorthand)
	assert.NotNil(t, cancelCmd.Flag("force"))
	assert.Equal(t, "f", cancelCmd.Flag("force").Shorthand)
	assert.Error(t, cancelCmd.Args(cancelCmd, nil))
	assert.NoError(t, cancelCmd.Args(cancelCmd, []string{"run"}))
	assert.Error(t, cancelCmd.Args(cancelCmd, []string{"run", "extra"}))
}

func TestCancelActionRunResolvesFullName(t *testing.T) {
	cli := &fakeActionRunTerminator{}
	io, out := cancelRunTestIO()
	proj := &name.Project{ProjectID: "11111111-1111-1111-1111-111111111111"}

	err := cancelActionRun(context.Background(), io, cli, "projects/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/actionRuns/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", proj, true, failIfConfirmed(t))
	require.NoError(t, err)
	require.NotNil(t, cli.got)
	assert.Equal(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", cli.got.ProjectID)
	assert.Equal(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", cli.got.ID)
	assert.Contains(t, out.String(), "Action run cancellation requested successfully.")
}

func TestCancelActionRunResolvesBareUUIDWithProject(t *testing.T) {
	cli := &fakeActionRunTerminator{}
	io, _ := cancelRunTestIO()
	proj := &name.Project{ProjectID: "11111111-1111-1111-1111-111111111111"}

	err := cancelActionRun(context.Background(), io, cli, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", proj, true, failIfConfirmed(t))
	require.NoError(t, err)
	require.NotNil(t, cli.got)
	assert.Equal(t, proj.ProjectID, cli.got.ProjectID)
}

func TestCancelActionRunRefusalDoesNotCallAPI(t *testing.T) {
	cli := &fakeActionRunTerminator{}
	io, out := cancelRunTestIO()
	confirmed := false

	err := cancelActionRun(context.Background(), io, cli, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", &name.Project{ProjectID: "p1"}, false, func(prompt string, _ *iostreams.IOStreams) bool {
		confirmed = true
		assert.Contains(t, prompt, "cannot be undone")
		return false
	})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.Nil(t, cli.got)
	assert.Contains(t, out.String(), "Action run cancellation aborted.")
}

func TestCancelActionRunForceSkipsConfirmation(t *testing.T) {
	cli := &fakeActionRunTerminator{}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.NoError(t, err)
	assert.NotNil(t, cli.got)
}

func TestCancelActionRunRejectsInvalidReference(t *testing.T) {
	cli := &fakeActionRunTerminator{}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, "invalid", &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.Error(t, err)
	assert.Nil(t, cli.got)
}

func TestCancelActionRunPreservesConnectErrorCode(t *testing.T) {
	cli := &fakeActionRunTerminator{err: connect.NewError(connect.CodeInvalidArgument, errors.New("already finished"))}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.Error(t, err)
	assert.True(t, utils.IsConnectErrorWithCode(err, connect.CodeInvalidArgument))
}

func cancelRunTestIO() (*iostreams.IOStreams, *bytes.Buffer) {
	var out bytes.Buffer
	return iostreams.Test(nil, &out, &out), &out
}

func failIfConfirmed(t *testing.T) confirmCancelRun {
	t.Helper()
	return func(string, *iostreams.IOStreams) bool {
		t.Fatal("confirmation must be skipped")
		return false
	}
}
