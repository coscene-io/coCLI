// Copyright 2024 coScene
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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errStubAuth is a sentinel error for stubbed auth-failure tests.
var errStubAuth = errors.New("stub auth failure")

// fakeProvider returns a fixed ProfileManager, implementing Provider.
type fakeProvider struct {
	pm *ProfileManager
}

func (f *fakeProvider) GetProfileManager() (*ProfileManager, error) { return f.pm, nil }
func (f *fakeProvider) Persist(_ *ProfileManager) error             { return nil }

// stubAuth replaces the network auth step for the duration of a test.
func stubAuth(t *testing.T) {
	t.Helper()
	orig := ephemeralAuth
	ephemeralAuth = func(_ context.Context, _ *ProfileManager) error { return nil }
	t.Cleanup(func() { ephemeralAuth = orig })
}

func twoProfilePM() *ProfileManager {
	return &ProfileManager{
		CurrentProfile: "p1",
		Profiles: []*Profile{
			{Name: "p1", EndPoint: "https://openapi.a.coscene.cn", Token: "t1", ProjectSlug: "s1", Org: "o1", ProjectName: "projects/1"},
			{Name: "p2", EndPoint: "https://openapi.b.coscene.cn", Token: "t2", ProjectSlug: "s2", Org: "o2", ProjectName: "projects/2"},
		},
	}
}

func clearCosEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"COS_ENDPOINT", "COS_TOKEN", "COS_PROJECT", "COS_PROJECTID", "COS_ORG"} {
		t.Setenv(k, "")
	}
}

func TestResolveProfileManager_NoOverrideNoEnv_UsesConfig(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "")
	require.NoError(t, err)
	assert.False(t, ephemeral, "config path must be persistable")
	assert.Equal(t, "p1", pm.CurrentProfile)
}

func TestResolveProfileManager_OverrideSelectsNamed(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "p2")
	require.NoError(t, err)
	assert.True(t, ephemeral, "override must be ephemeral (no persist)")
	assert.Equal(t, "p2", pm.CurrentProfile)
}

func TestResolveProfileManager_OverrideNotFound_Errors(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)

	_, _, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveProfileManager_CompleteEnv_OverridesConfig(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)
	t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
	t.Setenv("COS_TOKEN", "env-token")
	t.Setenv("COS_PROJECT", "env-slug")

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "")
	require.NoError(t, err)
	assert.True(t, ephemeral)
	assert.Equal(t, EnvProfileName, pm.CurrentProfile)
	assert.Equal(t, "env-token", pm.GetCurrentProfile().Token)
}

func TestResolveProfileManager_FlagBeatsEnv(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)
	t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
	t.Setenv("COS_TOKEN", "env-token")
	t.Setenv("COS_PROJECT", "env-slug")

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "p2")
	require.NoError(t, err)
	assert.True(t, ephemeral)
	assert.Equal(t, "p2", pm.CurrentProfile, "--profile must win over complete COS_* env")
}

func TestResolveProfileManager_PartialEnv_Ignored(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)
	t.Setenv("COS_TOKEN", "only-token") // endpoint + project missing

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "")
	require.NoError(t, err)
	assert.False(t, ephemeral, "partial env must fall through to config")
	assert.Equal(t, "p1", pm.CurrentProfile)
}

func TestResolveProfileManager_CompleteEnv_EmptyConfig(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)
	t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
	t.Setenv("COS_TOKEN", "env-token")
	t.Setenv("COS_PROJECT", "env-slug")

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: &ProfileManager{}}, "")
	require.NoError(t, err)
	assert.True(t, ephemeral)
	assert.Equal(t, EnvProfileName, pm.CurrentProfile)
}

func TestResolveProfileManager_Override_EmptyConfig_Errors(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)

	_, _, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: &ProfileManager{}}, "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveProfileManager_MalformedEnvEndpoint_Errors(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)
	t.Setenv("COS_ENDPOINT", "http://not-openapi.example.com")
	t.Setenv("COS_TOKEN", "env-token")
	t.Setenv("COS_PROJECT", "env-slug")

	_, _, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "")
	require.Error(t, err, "malformed COS_ENDPOINT must be rejected, not silently accepted")
	assert.Contains(t, err.Error(), "env profile")
}

func TestResolveProfileManager_FlagSelectsConfigProfile_NotEnvTainted(t *testing.T) {
	stubAuth(t)
	clearCosEnv(t)
	t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
	t.Setenv("COS_TOKEN", "env-token")
	t.Setenv("COS_PROJECT", "env-slug")

	pm, _, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "p2")
	require.NoError(t, err)
	assert.Equal(t, "t2", pm.GetCurrentProfile().Token, "must select config p2, not the env profile")
}

func TestBuildEnvProfileFromOS_IgnoresCosName(t *testing.T) {
	clearCosEnv(t)
	t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
	t.Setenv("COS_TOKEN", "tok")
	t.Setenv("COS_PROJECT", "slug")
	t.Setenv("COS_NAME", "attacker-alias")

	p, err := buildEnvProfileFromOS()
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, EnvProfileName, p.Name, "COS_NAME must not rename the env sentinel profile")
}

func TestProfileCheckAuth_NilSafe(t *testing.T) {
	var p *Profile
	assert.False(t, p.CheckAuth(), "CheckAuth on nil profile must not panic")
}

func TestAuthEphemeral_AlreadyAuthed_NoNetwork(t *testing.T) {
	// A profile already carrying Org + ProjectName passes CheckAuth, so
	// authEphemeral must short-circuit and return nil without any network call.
	pm := &ProfileManager{
		CurrentProfile: "p1",
		Profiles: []*Profile{
			{Name: "p1", EndPoint: "https://openapi.a.coscene.cn", Token: "t1", ProjectSlug: "s1", Org: "o1", ProjectName: "projects/1"},
		},
	}
	require.NoError(t, authEphemeral(context.Background(), pm))
}

func TestResolveProfileManager_EphemeralAuthError_Override(t *testing.T) {
	clearCosEnv(t)
	orig := ephemeralAuth
	ephemeralAuth = func(_ context.Context, _ *ProfileManager) error {
		return errStubAuth
	}
	t.Cleanup(func() { ephemeralAuth = orig })

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "p2")
	require.Error(t, err, "auth failure on override path must propagate")
	assert.Nil(t, pm)
	assert.False(t, ephemeral)
}

func TestResolveProfileManager_EphemeralAuthError_Env(t *testing.T) {
	clearCosEnv(t)
	t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
	t.Setenv("COS_TOKEN", "env-token")
	t.Setenv("COS_PROJECT", "env-slug")
	orig := ephemeralAuth
	ephemeralAuth = func(_ context.Context, _ *ProfileManager) error {
		return errStubAuth
	}
	t.Cleanup(func() { ephemeralAuth = orig })

	pm, ephemeral, err := ResolveProfileManager(context.Background(), &fakeProvider{pm: twoProfilePM()}, "")
	require.Error(t, err, "auth failure on env path must propagate")
	assert.Nil(t, pm)
	assert.False(t, ephemeral)
}

func TestBuildEnvProfileFromOS(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		clearCosEnv(t)
		t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
		t.Setenv("COS_TOKEN", "tok")
		t.Setenv("COS_PROJECT", "slug")

		p, err := buildEnvProfileFromOS()
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, EnvProfileName, p.Name)
		assert.Equal(t, "slug", p.ProjectSlug)
	})

	t.Run("partial returns nil", func(t *testing.T) {
		clearCosEnv(t)
		t.Setenv("COS_TOKEN", "tok")

		p, err := buildEnvProfileFromOS()
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("projectid hack fills project name", func(t *testing.T) {
		clearCosEnv(t)
		t.Setenv("COS_ENDPOINT", "https://openapi.env.coscene.cn")
		t.Setenv("COS_TOKEN", "tok")
		t.Setenv("COS_PROJECT", "slug")
		t.Setenv("COS_PROJECTID", "abc123")

		p, err := buildEnvProfileFromOS()
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Contains(t, p.ProjectName, "abc123")
	})
}
