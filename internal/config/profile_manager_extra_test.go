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

package config

import (
	"context"
	"testing"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileManagerAccessors(t *testing.T) {
	pm := &ProfileManager{
		CurrentProfile: "dev",
		Profiles: []*Profile{
			{
				Name:        "dev",
				EndPoint:    "https://openapi.coscene.cn",
				Token:       "token",
				Org:         "org-a",
				ProjectSlug: "project-a",
				ProjectName: "projects/project-a",
			},
		},
	}

	assert.False(t, pm.IsEmpty())
	assert.True(t, pm.CheckAuth())
	assert.Equal(t, "https://coscene.cn", pm.GetBaseUrl())
	assert.Equal(t, pm.Profiles[0], pm.GetCurrentProfile())
	assert.Equal(t, pm.Profiles, pm.GetProfiles())

	project, err := pm.ProjectName(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "project-a", project.ProjectID)
}

func TestProfileManagerProjectNameRejectsInvalidCurrentProject(t *testing.T) {
	pm := &ProfileManager{
		CurrentProfile: "dev",
		Profiles: []*Profile{
			{Name: "dev", EndPoint: "https://openapi.coscene.cn", Token: "token", ProjectSlug: "project-a", ProjectName: "bad"},
		},
	}

	_, err := pm.ProjectName(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new project name")
}

func TestProfileManagerDeleteProfile(t *testing.T) {
	pm := &ProfileManager{
		CurrentProfile: "dev",
		Profiles: []*Profile{
			{Name: "dev", EndPoint: "https://openapi.coscene.cn", Token: "token", ProjectSlug: "dev-project"},
			{Name: "prod", EndPoint: "https://openapi.coscene.cn", Token: "token", ProjectSlug: "prod-project"},
		},
	}

	require.NoError(t, pm.DeleteProfile("dev"))
	assert.Equal(t, "prod", pm.CurrentProfile)
	require.Len(t, pm.Profiles, 1)
	assert.Equal(t, "prod", pm.Profiles[0].Name)

	err := pm.DeleteProfile("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile missing not found")
}

func TestProfileManagerMutationsWithInjectedClients(t *testing.T) {
	t.Run("AddProfile authenticates first profile", func(t *testing.T) {
		pm := &ProfileManager{}
		require.NoError(t, pm.AddProfile(testProfileWithClients("dev", "project-a")))
		assert.Equal(t, "dev", pm.CurrentProfile)
		require.Len(t, pm.Profiles, 1)
		assert.Equal(t, "org-a", pm.Profiles[0].Org)
		assert.Equal(t, "projects/project-a", pm.Profiles[0].ProjectName)
	})

	t.Run("SetProfile updates current profile and refreshes auth fields", func(t *testing.T) {
		pm := &ProfileManager{
			CurrentProfile: "dev",
			Profiles:       []*Profile{testProfileWithClients("dev", "old-project")},
		}
		require.NoError(t, pm.SetProfile(&Profile{
			Name:        "dev",
			EndPoint:    "https://openapi.coscene.cn",
			Token:       "new-token",
			ProjectSlug: "project-a",
		}))
		assert.Equal(t, "dev", pm.CurrentProfile)
		assert.Equal(t, "new-token", pm.Profiles[0].Token)
		assert.Equal(t, "projects/old-project", pm.Profiles[0].ProjectName)
	})

	t.Run("SwitchProfile moves current profile", func(t *testing.T) {
		pm := &ProfileManager{
			CurrentProfile: "dev",
			Profiles: []*Profile{
				testProfileWithClients("dev", "project-a"),
				testProfileWithClients("prod", "project-b"),
			},
		}
		require.NoError(t, pm.SwitchProfile("prod"))
		assert.Equal(t, "prod", pm.CurrentProfile)
		assert.Equal(t, "projects/project-b", pm.GetCurrentProfile().ProjectName)

		err := pm.SwitchProfile("missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid profile name")
	})
}

func testProfileWithClients(profileName string, projectID string) *Profile {
	profile := &Profile{
		Name:        profileName,
		EndPoint:    "https://openapi.coscene.cn",
		Token:       "token",
		ProjectSlug: projectID,
	}
	profile.cliOnce.Do(func() {})
	profile.orgcli = fakeOrgClient{slug: "org-a"}
	profile.projcli = fakeProjectClient{project: &openv1alpha1resource.Project{
		Name: "projects/" + projectID,
		Slug: projectID,
	}}
	return profile
}
