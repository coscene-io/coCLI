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

//go:build integration

package integration

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveRecordNotFound(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	projectName, err := name.NewProject(pm.GetCurrentProfile().ProjectName)
	require.NoError(t, err, "current profile has invalid project-name")

	fakeRecord := &name.Record{
		ProjectID: projectName.ProjectID,
		RecordID:  "00000000-0000-0000-0000-000000000000",
	}

	_, err = pm.RecordCli().Get(ctx, fakeRecord)
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err),
		"querying non-existent record should return NOT_FOUND, got: %v", err)
}

func TestLiveRecordInvalidArgument(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	badRecord := &name.Record{
		ProjectID: "not-a-valid-project-id",
		RecordID:  "not-a-valid-record-id",
	}

	_, err := pm.RecordCli().Get(ctx, badRecord)
	require.Error(t, err)
	code := connect.CodeOf(err)
	t.Logf("server returned code: %v", code)
	assert.True(t, code == connect.CodeInvalidArgument || code == connect.CodeNotFound,
		"querying with invalid name format should return INVALID_ARGUMENT or NOT_FOUND, got: %v (%v)", code, err)
}

func TestLiveProjectNotFound(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	fakeProject := &name.Project{
		ProjectID: "00000000-0000-0000-0000-000000000000",
	}

	_, err := pm.ProjectCli().Get(ctx, fakeProject)
	require.Error(t, err)
	code := connect.CodeOf(err)
	assert.True(t, code == connect.CodeNotFound || code == connect.CodePermissionDenied,
		"querying non-existent project should return NOT_FOUND or PERMISSION_DENIED (security), got: %v (%v)", code, err)
}
