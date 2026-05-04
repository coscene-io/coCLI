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

package apimocks

import (
	"context"
	"testing"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockProfileManager(t *testing.T) {
	pm := NewMockProfileManager(t)

	assert.True(t, pm.CheckAuth())
	require.NoError(t, pm.Auth(context.Background()))
	assert.Nil(t, pm.RecordCli())
	assert.Nil(t, pm.FileCli())
	assert.Nil(t, pm.LabelCli())
	assert.Nil(t, pm.ProjectCli())
	assert.Nil(t, pm.ActionCli())
	assert.Nil(t, pm.TaskCli())
	assert.Nil(t, pm.EventCli())
	assert.Nil(t, pm.CustomFieldCli())

	project, err := pm.ProjectName(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "test-project", project.ProjectID)

	project, err = pm.ProjectName(context.Background(), "override-project")
	require.NoError(t, err)
	assert.Equal(t, "override-project", project.ProjectID)

	recordURL, err := pm.GetRecordUrl(context.Background(), &name.Record{ProjectID: "p", RecordID: "r"})
	require.NoError(t, err)
	assert.Equal(t, "https://openapi.mock.coscene.com/records/r", recordURL)

	projectURL, err := pm.GetProjectUrl(context.Background(), &name.Project{ProjectID: "p"})
	require.NoError(t, err)
	assert.Equal(t, "https://openapi.mock.coscene.com/projects/p", projectURL)
}

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider(t)

	pm, err := provider.GetProfileManager()
	require.NoError(t, err)
	assert.Equal(t, "test-profile", pm.CurrentProfile)
	assert.Equal(t, provider.ProfileManager().ProfileManager, pm)

	custom := NewMockProfileManager(t)
	custom.ProfileManager.CurrentProfile = "custom"
	provider.SetProfileManager(custom)
	pm, err = provider.GetProfileManager()
	require.NoError(t, err)
	assert.Equal(t, "custom", pm.CurrentProfile)

	require.NoError(t, provider.Persist(&config.ProfileManager{}))
}
