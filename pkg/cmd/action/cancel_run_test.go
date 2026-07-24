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

	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cancelRunTestID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

type fakeActionRunCanceler struct {
	runs       []*openv1alpha1resource.ActionRun
	listErr    error
	listCalled bool
	got        *name.ActionRun
	err        error
}

func (f *fakeActionRunCanceler) ListAllActionRuns(_ context.Context, opts *api.ListActionRunsOptions) ([]*openv1alpha1resource.ActionRun, error) {
	f.listCalled = true
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.runs != nil {
		return f.runs, nil
	}
	return []*openv1alpha1resource.ActionRun{{
		Name:  opts.Parent + "/actionRuns/" + cancelRunTestID,
		State: openv1alpha1enums.ActionRunStateEnum_RUNNING,
	}}, nil
}

func (f *fakeActionRunCanceler) TerminateActionRun(_ context.Context, actionRun *name.ActionRun) error {
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

func TestCancelRunRuntimeErrorDoesNotPrintUsage(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &out)
	cmd := NewCancelRunCommand(&cfgPath, io, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetContext(config.ContextWithProfileManager(context.Background(), &config.ProfileManager{
		CurrentProfile: "test",
		Profiles: []*config.Profile{{
			Name: "test",
		}},
	}))
	cmd.SetArgs([]string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "--force"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, out.String(), "Usage:")
	assert.NotContains(t, out.String(), "Error:")
}

func TestCancelRunArgumentErrorPrintsUsage(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &out)
	cmd := NewCancelRunCommand(&cfgPath, io, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, out.String(), "Usage:")
}

func TestCancelActionRunResolvesFullName(t *testing.T) {
	cli := &fakeActionRunCanceler{}
	io, out := cancelRunTestIO()
	proj := &name.Project{ProjectID: "11111111-1111-1111-1111-111111111111"}

	err := cancelActionRun(context.Background(), io, cli, "projects/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/actionRuns/"+cancelRunTestID, proj, true, failIfConfirmed(t))
	require.NoError(t, err)
	require.NotNil(t, cli.got)
	assert.Equal(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", cli.got.ProjectID)
	assert.Equal(t, cancelRunTestID, cli.got.ID)
	assert.Contains(t, out.String(), "Action run cancellation requested successfully.")
}

func TestCancelActionRunResolvesBareUUIDWithProject(t *testing.T) {
	cli := &fakeActionRunCanceler{}
	io, _ := cancelRunTestIO()
	proj := &name.Project{ProjectID: "11111111-1111-1111-1111-111111111111"}

	err := cancelActionRun(context.Background(), io, cli, cancelRunTestID, proj, true, failIfConfirmed(t))
	require.NoError(t, err)
	require.NotNil(t, cli.got)
	assert.Equal(t, proj.ProjectID, cli.got.ProjectID)
}

func TestCancelActionRunRefusalDoesNotCallAPI(t *testing.T) {
	cli := &fakeActionRunCanceler{}
	io, out := cancelRunTestIO()
	confirmed := false

	err := cancelActionRun(context.Background(), io, cli, cancelRunTestID, &name.Project{ProjectID: "p1"}, false, func(prompt string, _ *iostreams.IOStreams) bool {
		confirmed = true
		assert.Contains(t, prompt, "cannot be undone")
		return false
	})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, cli.listCalled)
	assert.Nil(t, cli.got)
	assert.Contains(t, out.String(), "Action run cancellation aborted.")
}

func TestCancelActionRunForceSkipsConfirmation(t *testing.T) {
	cli := &fakeActionRunCanceler{}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, cancelRunTestID, &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.NoError(t, err)
	assert.True(t, cli.listCalled)
	assert.NotNil(t, cli.got)
}

func TestCancelActionRunRejectsInvalidReference(t *testing.T) {
	cli := &fakeActionRunCanceler{}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, "invalid", &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.Error(t, err)
	assert.False(t, cli.listCalled)
	assert.Nil(t, cli.got)
}

func TestCancelActionRunPreservesConnectErrorCode(t *testing.T) {
	cli := &fakeActionRunCanceler{err: connect.NewError(connect.CodeInvalidArgument, errors.New("already finished"))}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, cancelRunTestID, &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.Error(t, err)
	assert.True(t, utils.IsConnectErrorWithCode(err, connect.CodeInvalidArgument))
}

func TestCancelActionRunFinishedDoesNotRequestCancellation(t *testing.T) {
	tests := []struct {
		name  string
		state openv1alpha1enums.ActionRunStateEnum_ActionRunState
	}{
		{name: "succeeded", state: openv1alpha1enums.ActionRunStateEnum_SUCCEEDED},
		{name: "failed", state: openv1alpha1enums.ActionRunStateEnum_FAILED},
		{name: "aborted", state: openv1alpha1enums.ActionRunStateEnum_ABORTED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &fakeActionRunCanceler{runs: []*openv1alpha1resource.ActionRun{{
				Name:  "projects/p1/actionRuns/" + cancelRunTestID,
				State: tt.state,
			}}}
			io, out := cancelRunTestIO()

			err := cancelActionRun(context.Background(), io, cli, cancelRunTestID, &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
			require.NoError(t, err)
			assert.Nil(t, cli.got)
			assert.Contains(t, out.String(), "Action run has already finished with state "+tt.state.String())
			assert.Contains(t, out.String(), "No cancellation request was sent.")
		})
	}
}

func TestCancelActionRunNotFoundDoesNotRequestCancellation(t *testing.T) {
	cli := &fakeActionRunCanceler{runs: []*openv1alpha1resource.ActionRun{}}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, cancelRunTestID, &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.EqualError(t, err, "action run not found: projects/p1/actionRuns/"+cancelRunTestID)
	assert.Nil(t, cli.got)
}

func TestCancelActionRunStateCheckPreservesConnectErrorCode(t *testing.T) {
	cli := &fakeActionRunCanceler{listErr: connect.NewError(connect.CodePermissionDenied, errors.New("denied"))}
	io, _ := cancelRunTestIO()

	err := cancelActionRun(context.Background(), io, cli, cancelRunTestID, &name.Project{ProjectID: "p1"}, true, failIfConfirmed(t))
	require.Error(t, err)
	assert.True(t, utils.IsConnectErrorWithCode(err, connect.CodePermissionDenied))
	assert.Nil(t, cli.got)
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
