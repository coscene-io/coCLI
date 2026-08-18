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

package record

import (
	"bytes"
	"context"
	"errors"
	"testing"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	momentTestProjectID = "11111111-1111-1111-1111-111111111111"
	momentTestID        = "22222222-2222-2222-2222-222222222222"
	momentTestName      = "projects/" + momentTestProjectID + "/events/" + momentTestID
)

type fakeMomentDeleter struct {
	event      *openv1alpha1resource.Event
	getErr     error
	deleteErr  error
	gotGet     *name.Event
	gotDeleted *name.Event
	calls      []string
}

func (f *fakeMomentDeleter) GetEvent(_ context.Context, eventName *name.Event) (*openv1alpha1resource.Event, error) {
	f.calls = append(f.calls, "get")
	f.gotGet = eventName
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.event != nil {
		return f.event, nil
	}
	return &openv1alpha1resource.Event{Name: eventName.String(), DisplayName: "Collision"}, nil
}

func (f *fakeMomentDeleter) DeleteEvent(_ context.Context, eventName *name.Event) error {
	f.calls = append(f.calls, "delete")
	f.gotDeleted = eventName
	return f.deleteErr
}

func TestMomentDeleteCommandRegistrationAndFlags(t *testing.T) {
	cfgPath := t.TempDir() + "/test-config.yaml"
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &out)
	cmd := NewMomentCommand(&cfgPath, io, config.Provide)

	deleteCmd, _, err := cmd.Find([]string{"delete"})
	require.NoError(t, err)
	assert.Equal(t, "delete", deleteCmd.Name())
	assert.NotNil(t, deleteCmd.Flag("project"))
	assert.Equal(t, "p", deleteCmd.Flag("project").Shorthand)
	assert.NotNil(t, deleteCmd.Flag("force"))
	assert.Equal(t, "f", deleteCmd.Flag("force").Shorthand)
	assert.Error(t, deleteCmd.Args(deleteCmd, nil))
	assert.NoError(t, deleteCmd.Args(deleteCmd, []string{momentTestID}))
	assert.Error(t, deleteCmd.Args(deleteCmd, []string{momentTestID, "extra"}))
}

func TestMomentDeleteRuntimeErrorDoesNotPrintUsage(t *testing.T) {
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &out)
	cfgPath := t.TempDir() + "/test-config.yaml"
	cmd := NewMomentDeleteCommand(&cfgPath, io, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetContext(config.ContextWithProfileManager(context.Background(), &config.ProfileManager{
		CurrentProfile: "test",
		Profiles: []*config.Profile{{
			Name:        "test",
			EndPoint:    "http://127.0.0.1:0",
			Token:       "test-token",
			ProjectSlug: "test-project",
			ProjectName: "projects/" + momentTestProjectID,
		}},
	}))
	cmd.SetArgs([]string{momentTestName, "--force"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, out.String(), "Usage:")
	assert.NotContains(t, out.String(), "Error:")
}

func TestMomentDeleteArgumentErrorPrintsUsage(t *testing.T) {
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &out)
	cfgPath := t.TempDir() + "/test-config.yaml"
	cmd := NewMomentDeleteCommand(&cfgPath, io, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, out.String(), "Usage:")
}

func TestMomentDeleteInvalidReferenceDoesNotResolveProject(t *testing.T) {
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &out)
	cfgPath := t.TempDir() + "/test-config.yaml"
	cmd := NewMomentDeleteCommand(&cfgPath, io, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetContext(config.ContextWithProfileManager(context.Background(), &config.ProfileManager{
		CurrentProfile: "test",
		Profiles: []*config.Profile{{
			Name: "test",
		}},
	}))
	cmd.SetArgs([]string{"not-a-moment", "--project", "missing-project", "--force"})

	err := cmd.Execute()
	assert.EqualError(t, err, "invalid moment resource name or id: not-a-moment")
	assert.NotContains(t, out.String(), "Usage:")
}

func TestResolveMomentName(t *testing.T) {
	project := &name.Project{ProjectID: momentTestProjectID}

	t.Run("full resource name takes precedence", func(t *testing.T) {
		eventName, err := resolveMomentName(momentTestName, func() (*name.Project, error) {
			t.Fatal("full resource name must not resolve a project")
			return nil, nil
		})
		require.NoError(t, err)
		assert.Equal(t, momentTestName, eventName.String())
	})

	t.Run("bare UUID uses selected project", func(t *testing.T) {
		eventName, err := resolveMomentName(momentTestID, func() (*name.Project, error) {
			return project, nil
		})
		require.NoError(t, err)
		assert.Equal(t, momentTestName, eventName.String())
	})

	t.Run("invalid reference does not resolve a project", func(t *testing.T) {
		_, err := resolveMomentName("not-a-moment", func() (*name.Project, error) {
			t.Fatal("invalid reference must not resolve a project")
			return nil, nil
		})
		assert.EqualError(t, err, "invalid moment resource name or id: not-a-moment")
	})

	t.Run("project error", func(t *testing.T) {
		_, err := resolveMomentName(momentTestID, func() (*name.Project, error) {
			return nil, errors.New("project unavailable")
		})
		assert.EqualError(t, err, "project unavailable")
	})
}

func TestDeleteMomentRefusalDoesNotDelete(t *testing.T) {
	cli := &fakeMomentDeleter{}
	io, out := momentDeleteTestIO()
	confirmed := false

	err := deleteMoment(context.Background(), io, cli, mustMomentName(t), false, func(prompt string, _ *iostreams.IOStreams) bool {
		confirmed = true
		assert.Contains(t, prompt, "cannot be undone")
		return false
	})

	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.NotNil(t, cli.gotGet)
	assert.Nil(t, cli.gotDeleted)
	assert.Contains(t, out.String(), "Collision")
	assert.Contains(t, out.String(), momentTestName)
	assert.Contains(t, out.String(), "Moment deletion aborted.")
}

func TestDeleteMomentForceSkipsConfirmation(t *testing.T) {
	cli := &fakeMomentDeleter{}
	io, out := momentDeleteTestIO()

	err := deleteMoment(context.Background(), io, cli, mustMomentName(t), true, failIfMomentConfirmed(t))
	require.NoError(t, err)
	assert.NotNil(t, cli.gotGet)
	assert.NotNil(t, cli.gotDeleted)
	assert.Equal(t, []string{"get", "delete"}, cli.calls)
	assert.Contains(t, out.String(), "Moment successfully deleted")
}

func TestDeleteMomentConfirmationAccepts(t *testing.T) {
	cli := &fakeMomentDeleter{}
	io, out := momentDeleteTestIO()
	confirmed := false

	err := deleteMoment(context.Background(), io, cli, mustMomentName(t), false, func(prompt string, _ *iostreams.IOStreams) bool {
		confirmed = true
		assert.Equal(t, momentDeleteConfirmation, prompt)
		return true
	})

	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.Equal(t, []string{"get", "delete"}, cli.calls)
	assert.Equal(t, momentTestName, cli.gotDeleted.String())
	assert.Contains(t, out.String(), "Moment successfully deleted: "+momentTestName)
}

func TestDeleteMomentShowsUnnamedTarget(t *testing.T) {
	cli := &fakeMomentDeleter{event: &openv1alpha1resource.Event{Name: momentTestName}}
	io, out := momentDeleteTestIO()

	err := deleteMoment(context.Background(), io, cli, mustMomentName(t), true, failIfMomentConfirmed(t))
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Moment: <unnamed> ("+momentTestName+")")
}

func TestDeleteMomentGetErrorDoesNotDelete(t *testing.T) {
	cli := &fakeMomentDeleter{getErr: connect.NewError(connect.CodeNotFound, errors.New("missing"))}
	io, _ := momentDeleteTestIO()

	err := deleteMoment(context.Background(), io, cli, mustMomentName(t), true, failIfMomentConfirmed(t))
	require.Error(t, err)
	assert.True(t, utils.IsConnectErrorWithCode(err, connect.CodeNotFound))
	assert.Nil(t, cli.gotDeleted)
}

func TestDeleteMomentDeleteErrorPreservesConnectCode(t *testing.T) {
	cli := &fakeMomentDeleter{deleteErr: connect.NewError(connect.CodePermissionDenied, errors.New("denied"))}
	io, _ := momentDeleteTestIO()

	err := deleteMoment(context.Background(), io, cli, mustMomentName(t), true, failIfMomentConfirmed(t))
	require.Error(t, err)
	assert.True(t, utils.IsConnectErrorWithCode(err, connect.CodePermissionDenied))
}

func mustMomentName(t *testing.T) *name.Event {
	t.Helper()
	eventName, err := name.NewEvent(momentTestName)
	require.NoError(t, err)
	return eventName
}

func momentDeleteTestIO() (*iostreams.IOStreams, *bytes.Buffer) {
	var out bytes.Buffer
	return iostreams.Test(nil, &out, &out), &out
}

func failIfMomentConfirmed(t *testing.T) confirmMomentDelete {
	t.Helper()
	return func(string, *iostreams.IOStreams) bool {
		t.Fatal("confirmation must be skipped")
		return false
	}
}
