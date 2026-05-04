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

package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTempFileHelpers(t *testing.T) {
	dir := TempDir(t)
	src := CreateTempFile(t, dir, "src-*.txt", []byte("hello"))
	dst := filepath.Join(dir, "dst.txt")

	CopyFile(t, src, dst)
	AssertFileContent(t, dst, []byte("hello"))
}

func TestCreateTestFileTree(t *testing.T) {
	dir := TempDir(t)

	CreateTestFileTree(t, dir, map[string][]byte{
		"a.txt":        []byte("a"),
		"nested/b.txt": []byte("b"),
	})

	AssertFileContent(t, filepath.Join(dir, "a.txt"), []byte("a"))
	AssertFileContent(t, filepath.Join(dir, "nested", "b.txt"), []byte("b"))
}

func TestCaptureOutput(t *testing.T) {
	stdout, stderr := CaptureOutput(t, func() {
		_, _ = fmt.Fprint(os.Stdout, "out")
		_, _ = fmt.Fprint(os.Stderr, "err")
	})

	assert.Equal(t, "out", stdout)
	assert.Equal(t, "err", stderr)
}

func TestBuilders(t *testing.T) {
	record := NewRecordBuilder().
		WithName("projects/p/records/r").
		WithTitle("title").
		WithDescription("desc").
		WithLabels([]*openv1alpha1resource.Label{CreateTestLabel("road")}).
		WithDevice("devices/d").
		Build()
	assert.Equal(t, "projects/p/records/r", record.Name)
	assert.Equal(t, "title", record.Title)
	assert.Equal(t, "devices/d", record.Device.Name)
	require.Len(t, record.Labels, 1)

	file := NewFileBuilder().
		WithName("projects/p/records/r/files/f.txt").
		WithFilename("f.txt").
		WithSize(7).
		WithSha256("sha").
		Build()
	assert.Equal(t, int64(7), file.Size)
	assert.Equal(t, "sha", file.Sha256)

	project := NewProjectBuilder().
		WithName("projects/p").
		WithDisplayName("Project").
		Build()
	assert.Equal(t, "Project", project.DisplayName)

	label := CreateTestLabel("city")
	assert.Equal(t, "city", label.DisplayName)
	assert.Equal(t, "test-project", TestProjectName.ProjectID)
	assert.Equal(t, "test-record", TestRecordName.RecordID)
	assert.Equal(t, "test.txt", TestFileName.Filename)
}
