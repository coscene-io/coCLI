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

package project

import (
	"testing"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/stretchr/testify/assert"
)

func TestExtractFileSystemID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"regions format", "regions/cn-hangzhou/fileSystems/default", "default"},
		{"empty", "", ""},
		{"no pattern", "something", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractFileSystemID(tt.input))
		})
	}
}

func TestResolveFileSystem(t *testing.T) {
	fileSystems := []*openv1alpha1resource.FileSystem{
		{
			Name:      "regions/cn-hangzhou/fileSystems/default",
			Region:    "cn-hangzhou",
			IsDefault: true,
		},
		{
			Name:      "regions/cn-hangzhou/fileSystems/custom",
			Region:    "cn-hangzhou",
			IsDefault: false,
		},
		{
			Name:      "regions/cn-shanghai/fileSystems/default",
			Region:    "cn-shanghai",
			IsDefault: true,
		},
	}

	t.Run("region only - finds default", func(t *testing.T) {
		result := resolveFileSystem(fileSystems, "cn-hangzhou", "")
		assert.Equal(t, "regions/cn-hangzhou/fileSystems/default", result)
	})

	t.Run("region + filesystem name - exact match", func(t *testing.T) {
		result := resolveFileSystem(fileSystems, "cn-hangzhou", "custom")
		assert.Equal(t, "regions/cn-hangzhou/fileSystems/custom", result)
	})

	t.Run("different region - finds correct default", func(t *testing.T) {
		result := resolveFileSystem(fileSystems, "cn-shanghai", "")
		assert.Equal(t, "regions/cn-shanghai/fileSystems/default", result)
	})
}
