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

package login

import (
	"bytes"
	"errors"
	"testing"

	"github.com/coscene-io/cocli/internal/apimocks"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSwitch(t *testing.T) {
	t.Run("selection cancelled", func(t *testing.T) {
		mockProvider := apimocks.NewMockProvider(t)
		buf := new(bytes.Buffer)
		io := iostreams.Test(nil, buf, buf)

		err := runSwitch(
			"test-config.yaml",
			io,
			func(string) config.Provider { return mockProvider },
			func(_ []*config.Profile, _ string) (*config.Profile, error) {
				return nil, errProfileSelectionAborted
			},
			func() bool { return false },
		)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Profile switch cancelled.")
	})

	t.Run("prompt failed in headless mode", func(t *testing.T) {
		mockProvider := apimocks.NewMockProvider(t)
		io := iostreams.Test(nil, &bytes.Buffer{}, &bytes.Buffer{})

		err := runSwitch(
			"test-config.yaml",
			io,
			func(string) config.Provider { return mockProvider },
			func(_ []*config.Profile, _ string) (*config.Profile, error) {
				return nil, errors.New("prompt failed")
			},
			func() bool { return true },
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-interactive mode")
		assert.Contains(t, err.Error(), "cocli login set -n <profile-name>")
	})

	t.Run("switch success", func(t *testing.T) {
		mockProvider := apimocks.NewMockProvider(t)
		mockProvider.ProfileManager().Profiles = []*config.Profile{
			{
				Name:        "profile-1",
				EndPoint:    "https://openapi.mock1.coscene.com",
				Token:       "token-1",
				Org:         "org",
				ProjectSlug: "project-1",
				ProjectName: "project-1",
			},
			{
				Name:        "profile-2",
				EndPoint:    "https://openapi.mock2.coscene.com",
				Token:       "token-2",
				Org:         "org",
				ProjectSlug: "project-2",
				ProjectName: "project-2",
			},
		}
		mockProvider.ProfileManager().CurrentProfile = "profile-1"
		buf := new(bytes.Buffer)
		io := iostreams.Test(nil, buf, buf)

		err := runSwitch(
			"test-config.yaml",
			io,
			func(string) config.Provider { return mockProvider },
			func(profiles []*config.Profile, _ string) (*config.Profile, error) {
				return profiles[1], nil
			},
			func() bool { return false },
		)
		require.NoError(t, err)
		assert.Equal(t, "profile-2", mockProvider.ProfileManager().CurrentProfile)
		assert.Contains(t, buf.String(), "Successfully switched to profile:")
	})

	t.Run("empty profile manager", func(t *testing.T) {
		mockProvider := apimocks.NewMockProvider(t)
		mockProvider.ProfileManager().Profiles = []*config.Profile{}
		mockProvider.ProfileManager().CurrentProfile = ""
		io := iostreams.Test(nil, &bytes.Buffer{}, &bytes.Buffer{})

		err := runSwitch(
			"test-config.yaml",
			io,
			func(string) config.Provider { return mockProvider },
			func(_ []*config.Profile, _ string) (*config.Profile, error) {
				return nil, nil
			},
			func() bool { return false },
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no login profiles found")
	})
}

func TestPromptForProfileEmpty(t *testing.T) {
	_, err := promptForProfile([]*config.Profile{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no profiles found")
}
