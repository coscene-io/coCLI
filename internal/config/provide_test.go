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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleConfig = `current-profile: p1
profiles:
  - name: p1
    endpoint: https://openapi.a.coscene.cn
    token: t1
    project: s1
    org: o1
    project-name: projects/1
  - name: p2
    endpoint: https://openapi.b.coscene.cn
    token: t2
    project: s2
    org: o2
    project-name: projects/2
`

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	return path
}

func TestProvide_GetProfileManager_LoadsConfig(t *testing.T) {
	clearCosEnv(t)
	path := writeTempConfig(t, sampleConfig)

	pm, err := Provide(path).GetProfileManager()
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, "p1", pm.CurrentProfile)
	assert.Len(t, pm.Profiles, 2)
	assert.Equal(t, "t1", pm.GetCurrentProfile().Token)
}

func TestProvide_GetProfileManager_EmptyConfig(t *testing.T) {
	clearCosEnv(t)
	path := writeTempConfig(t, "")

	pm, err := Provide(path).GetProfileManager()
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.True(t, pm.IsEmpty())
}

func TestProvide_Persist_RoundTrip(t *testing.T) {
	clearCosEnv(t)
	path := writeTempConfig(t, sampleConfig)
	cfg := Provide(path)

	pm, err := cfg.GetProfileManager()
	require.NoError(t, err)

	// Switch current-profile and persist.
	pm.CurrentProfile = "p2"
	require.NoError(t, cfg.Persist(pm))

	// Reload from disk through a fresh provider and confirm the change stuck.
	reloaded, err := Provide(path).GetProfileManager()
	require.NoError(t, err)
	assert.Equal(t, "p2", reloaded.CurrentProfile)
	assert.Len(t, reloaded.Profiles, 2)
}
