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

package printable

import (
	"testing"

	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/stretchr/testify/assert"
)

func TestProject_ResolveRegion(t *testing.T) {
	fsInfo := map[string]*openv1alpha1resource.FileSystem{
		"storageClusters/abc/fileSystems/default": {
			Name:   "storageClusters/abc/fileSystems/default",
			Region: openv1alpha1enums.RegionEnum_CN_HANGZHOU,
		},
	}
	p := NewProjectWithFileSystemInfo(nil, fsInfo)

	t.Run("from project region field", func(t *testing.T) {
		proj := &openv1alpha1resource.Project{Region: openv1alpha1enums.RegionEnum_CN_SHANGHAI}
		assert.Equal(t, "cn-shanghai", p.resolveRegion(proj))
	})

	t.Run("from filesystem lookup", func(t *testing.T) {
		proj := &openv1alpha1resource.Project{FileSystem: "storageClusters/abc/fileSystems/default"}
		assert.Equal(t, "cn-hangzhou", p.resolveRegion(proj))
	})

	t.Run("empty when no match", func(t *testing.T) {
		proj := &openv1alpha1resource.Project{FileSystem: "storageClusters/xyz/fileSystems/unknown"}
		assert.Equal(t, "", p.resolveRegion(proj))
	})
}

func TestProject_ResolveFileSystem(t *testing.T) {
	fsInfo := map[string]*openv1alpha1resource.FileSystem{
		"storageClusters/abc/fileSystems/default": {
			Name:        "storageClusters/abc/fileSystems/default",
			DisplayName: "Default Bucket",
		},
	}
	p := NewProjectWithFileSystemInfo(nil, fsInfo)

	t.Run("display name from lookup", func(t *testing.T) {
		assert.Equal(t, "Default Bucket", p.resolveFileSystem("storageClusters/abc/fileSystems/default"))
	})

	t.Run("extract name from compound format", func(t *testing.T) {
		assert.Equal(t, "custom", p.resolveFileSystem("storageClusters/xyz/fileSystems/custom"))
	})

	t.Run("passthrough when no format match", func(t *testing.T) {
		assert.Equal(t, "something", p.resolveFileSystem("something"))
	})
}

func TestProject_LookupFileSystem(t *testing.T) {
	fs := &openv1alpha1resource.FileSystem{Name: "storageClusters/abc/fileSystems/default"}
	p := NewProjectWithFileSystemInfo(nil, map[string]*openv1alpha1resource.FileSystem{
		"storageClusters/abc/fileSystems/default": fs,
	})

	assert.Equal(t, fs, p.lookupFileSystem("storageClusters/abc/fileSystems/default"))
	assert.Nil(t, p.lookupFileSystem("storageClusters/xyz/fileSystems/other"))
	assert.Nil(t, p.lookupFileSystem(""))

	empty := NewProject(nil)
	assert.Nil(t, empty.lookupFileSystem("anything"))
}
