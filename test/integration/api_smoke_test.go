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

	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveListProjects(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	projects, err := pm.ProjectCli().ListAllUserProjects(ctx, &api.ListProjectsOptions{})
	require.NoError(t, err, "ListAllUserProjects should succeed for authenticated user")
	assert.NotEmpty(t, projects, "authenticated user should have at least one project")
}

func TestLiveGetCurrentProject(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	projectName, err := name.NewProject(pm.GetCurrentProfile().ProjectName)
	require.NoError(t, err, "current profile has invalid project-name")

	proj, err := pm.ProjectCli().Get(ctx, projectName)
	require.NoError(t, err, "GetProject should succeed for current profile's project")
	assert.NotEmpty(t, proj.Name, "project should have a name")
}

func TestLiveListRecordsInCurrentProject(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	projectName, err := name.NewProject(pm.GetCurrentProfile().ProjectName)
	require.NoError(t, err)

	records, err := pm.RecordCli().SearchAll(ctx, &api.SearchRecordsOptions{
		Project: projectName,
	})
	require.NoError(t, err, "SearchAll should succeed for current project")
	t.Logf("found %d records in current project", len(records))
}

func TestLiveListActions(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	projectName, err := name.NewProject(pm.GetCurrentProfile().ProjectName)
	require.NoError(t, err)

	actions, err := pm.ActionCli().ListAllActions(ctx, &api.ListActionsOptions{
		Parent: projectName.String(),
	})
	require.NoError(t, err, "ListAllActions should succeed for current project")
	t.Logf("found %d actions in current project", len(actions))
}

func TestLiveListFileSystems(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	fileSystems, err := pm.FileSystemCli().ListAllFileSystems(ctx)
	require.NoError(t, err, "ListAllFileSystems should succeed for authenticated user")
	t.Logf("found %d file systems", len(fileSystems))
}
