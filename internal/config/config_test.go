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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		profile *Profile
		wantErr string
	}{
		{
			name:    "valid",
			profile: &Profile{Name: "dev", EndPoint: "https://openapi.dev.coscene.cn", Token: "tok", ProjectSlug: "proj"},
		},
		{
			name:    "empty name",
			profile: &Profile{Name: "", EndPoint: "https://openapi.dev.coscene.cn", Token: "tok", ProjectSlug: "proj"},
			wantErr: "profile name cannot be empty",
		},
		{
			name:    "bad endpoint",
			profile: &Profile{Name: "dev", EndPoint: "https://api.coscene.cn", Token: "tok", ProjectSlug: "proj"},
			wantErr: "endpoint should start with https://openapi.",
		},
		{
			name:    "empty token",
			profile: &Profile{Name: "dev", EndPoint: "https://openapi.dev.coscene.cn", Token: "", ProjectSlug: "proj"},
			wantErr: "token cannot be empty",
		},
		{
			name:    "empty project",
			profile: &Profile{Name: "dev", EndPoint: "https://openapi.dev.coscene.cn", Token: "tok", ProjectSlug: ""},
			wantErr: "project cannot be empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProfileManager_Validate(t *testing.T) {
	validProfile := &Profile{Name: "dev", EndPoint: "https://openapi.dev.coscene.cn", Token: "tok", ProjectSlug: "proj"}

	t.Run("empty manager is valid", func(t *testing.T) {
		pm := &ProfileManager{}
		assert.NoError(t, pm.Validate())
	})

	t.Run("valid single profile", func(t *testing.T) {
		pm := &ProfileManager{
			CurrentProfile: "dev",
			Profiles:       []*Profile{validProfile},
		}
		assert.NoError(t, pm.Validate())
	})

	t.Run("duplicate profile names", func(t *testing.T) {
		pm := &ProfileManager{
			CurrentProfile: "dev",
			Profiles:       []*Profile{validProfile, validProfile},
		}
		err := pm.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("current profile not found", func(t *testing.T) {
		pm := &ProfileManager{
			CurrentProfile: "missing",
			Profiles:       []*Profile{validProfile},
		}
		err := pm.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("current profile set but no profiles", func(t *testing.T) {
		pm := &ProfileManager{
			CurrentProfile: "dev",
		}
		err := pm.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid profile content", func(t *testing.T) {
		bad := &Profile{Name: "bad", EndPoint: "http://wrong", Token: "tok", ProjectSlug: "proj"}
		pm := &ProfileManager{
			CurrentProfile: "bad",
			Profiles:       []*Profile{bad},
		}
		err := pm.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}
