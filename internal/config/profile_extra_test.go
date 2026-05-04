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
	"os"
	"path/filepath"
	"testing"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileDisplayAndURLs(t *testing.T) {
	profile := &Profile{
		Name:        "dev",
		EndPoint:    "https://openapi.coscene.cn",
		Org:         "org-a",
		ProjectSlug: "project-a",
		ProjectName: "projects/project-a",
	}

	assert.Equal(t, "dev (*)", profile.StringWithOpts(true, false))
	verbose := profile.StringWithOpts(false, true)
	assert.Contains(t, verbose, "Profile Name:")
	assert.Contains(t, verbose, "Endpoint:")
	assert.True(t, profile.CheckAuth())
	assert.Equal(t, "https://coscene.cn", profile.GetBaseUrl())

	profile.EndPoint = "https://openapi.api.coscene.dev"
	assert.Equal(t, "https://home.coscene.dev", profile.GetBaseUrl())
}

func TestProfileAuthAndURLsWithInjectedClients(t *testing.T) {
	profile := &Profile{
		Name:        "dev",
		EndPoint:    "https://openapi.coscene.cn",
		Token:       "token",
		ProjectSlug: "project-a",
	}
	profile.cliOnce.Do(func() {})
	profile.orgcli = fakeOrgClient{slug: "org-a"}
	profile.projcli = fakeProjectClient{project: &openv1alpha1resource.Project{
		Name: "projects/project-a",
		Slug: "project-a",
	}}

	require.NoError(t, profile.Auth(context.Background()))
	assert.Equal(t, "org-a", profile.Org)
	assert.Equal(t, "projects/project-a", profile.ProjectName)

	recordURL, err := profile.GetRecordUrl(context.Background(), &name.Record{ProjectID: "project-a", RecordID: "record-a"})
	require.NoError(t, err)
	assert.Equal(t, "https://coscene.cn/org-a/project-a/records/record-a", recordURL)

	projectURL, err := profile.GetProjectUrl(context.Background(), &name.Project{ProjectID: "project-a"})
	require.NoError(t, err)
	assert.Equal(t, "https://coscene.cn/org-a/project-a", projectURL)

	var nilProfile *Profile
	assert.NoError(t, nilProfile.Auth(context.Background()))
}

func TestProfileClientAccessorsInitializeClients(t *testing.T) {
	profile := &Profile{
		Name:        "dev",
		EndPoint:    "https://openapi.coscene.cn",
		Token:       "token",
		ProjectSlug: "project-a",
	}

	assert.NotNil(t, profile.OrgCli())
	assert.NotNil(t, profile.ProjectCli())
	assert.NotNil(t, profile.RecordCli())
	assert.NotNil(t, profile.LabelCli())
	assert.NotNil(t, profile.UserCli())
	assert.NotNil(t, profile.FileCli())
	assert.NotNil(t, profile.ActionCli())
	assert.NotNil(t, profile.SecurityTokenCli())
	assert.NotNil(t, profile.EventCli())
	assert.NotNil(t, profile.TaskCli())
	assert.NotNil(t, profile.ContainerRegistryCli())
	assert.NotNil(t, profile.FileSystemCli())
	assert.NotNil(t, profile.RoleCli())
	assert.NotNil(t, profile.CustomFieldCli())
}

func TestGlobalConfigGetProfileManagerFromYaml(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
current-profile: dev
profiles:
  - name: dev
    endpoint: https://openapi.dev.coscene.cn
    token: token-a
    project: project-a
`), 0644))

	pm, err := Provide(cfgPath).GetProfileManager()

	require.NoError(t, err)
	assert.Equal(t, "dev", pm.CurrentProfile)
	require.Len(t, pm.Profiles, 1)
	assert.Equal(t, "project-a", pm.Profiles[0].ProjectSlug)
}

func TestGlobalConfigGetProfileManagerFromEnv(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`{}`), 0644))
	t.Setenv("COS_ENDPOINT", "https://openapi.dev.coscene.cn")
	t.Setenv("COS_TOKEN", "token-a")
	t.Setenv("COS_PROJECT", "project-a")
	t.Setenv("COS_PROJECTID", "project-id-a")

	pm, err := Provide(cfgPath).GetProfileManager()

	require.NoError(t, err)
	assert.Equal(t, "ENV_LOADED_PROFILE", pm.CurrentProfile)
	require.Len(t, pm.Profiles, 1)
	assert.Equal(t, "projects/project-id-a", pm.Profiles[0].ProjectName)
}

func TestGlobalConfigPersist(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
current-profile: old
profiles:
  - name: old
    endpoint: https://openapi.dev.coscene.cn
    token: old-token
    project: old-project
`), 0644))

	pm := &ProfileManager{
		CurrentProfile: "new",
		Profiles: []*Profile{
			{Name: "new", EndPoint: "https://openapi.dev.coscene.cn", Token: "new-token", ProjectSlug: "new-project"},
		},
	}
	require.NoError(t, Provide(cfgPath).Persist(pm))

	reloaded, err := Provide(cfgPath).GetProfileManager()
	require.NoError(t, err)
	assert.Equal(t, "new", reloaded.CurrentProfile)
	require.Len(t, reloaded.Profiles, 1)
	assert.Equal(t, "new-project", reloaded.Profiles[0].ProjectSlug)
}

type fakeOrgClient struct {
	slug string
}

func (f fakeOrgClient) Slug(ctx context.Context, org *name.Organization) (string, error) {
	return f.slug, nil
}

type fakeProjectClient struct {
	api.ProjectInterface
	project *openv1alpha1resource.Project
}

func (f fakeProjectClient) Name(ctx context.Context, projectSlug string) (*name.Project, error) {
	return name.NewProject(f.project.Name)
}

func (f fakeProjectClient) Get(ctx context.Context, projectName *name.Project) (*openv1alpha1resource.Project, error) {
	return f.project, nil
}
