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

package upload_utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasGlobPattern(t *testing.T) {
	assert.True(t, hasGlobPattern("*.txt"))
	assert.True(t, hasGlobPattern("dir/*/file"))
	assert.True(t, hasGlobPattern("file?.log"))
	assert.True(t, hasGlobPattern("[abc].txt"))
	assert.False(t, hasGlobPattern("normal/path/file.txt"))
	assert.False(t, hasGlobPattern(""))
}

func TestGlobBaseDir(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"a/*", "a"},
		{"a/b/*.txt", "a/b"},
		{"a/**/*.txt", "a"},
		{"*.txt", "."},
		{"/tmp/data/*.csv", "/tmp/data"},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			assert.Equal(t, tt.want, globBaseDir(tt.pattern))
		})
	}
}

func TestUploadManagerOpts_Valid(t *testing.T) {
	t.Run("default part size", func(t *testing.T) {
		opt := &UploadManagerOpts{}
		require.NoError(t, opt.Valid())
		assert.Equal(t, defaultPartSize, opt.partSizeUint64)
	})

	t.Run("custom part size", func(t *testing.T) {
		opt := &UploadManagerOpts{PartSize: "64MB"}
		require.NoError(t, opt.Valid())
		assert.Equal(t, uint64(64*1000*1000), opt.partSizeUint64)
	})

	t.Run("invalid part size", func(t *testing.T) {
		opt := &UploadManagerOpts{PartSize: "not-a-size"}
		assert.Error(t, opt.Valid())
	})
}

func TestFileOpts_Valid(t *testing.T) {
	t.Run("empty paths no additional", func(t *testing.T) {
		opt := &FileOpts{}
		assert.Error(t, opt.Valid())
	})

	t.Run("empty paths with additional uploads", func(t *testing.T) {
		opt := &FileOpts{AdditionalUploads: map[string]string{"a": "b"}}
		require.NoError(t, opt.Valid())
	})

	t.Run("single file path", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(f, []byte("x"), 0644))

		opt := &FileOpts{Paths: []string{f}}
		require.NoError(t, opt.Valid())
		assert.Equal(t, dir, opt.RelDir())
		assert.Equal(t, []string{f}, opt.GetPaths())
	})

	t.Run("nonexistent path", func(t *testing.T) {
		opt := &FileOpts{Paths: []string{"/nonexistent/file"}}
		assert.Error(t, opt.Valid())
	})

	t.Run("glob pattern", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "c.log"), []byte("c"), 0644))

		opt := &FileOpts{Paths: []string{filepath.Join(dir, "*.txt")}}
		require.NoError(t, opt.Valid())
		assert.Len(t, opt.GetPaths(), 2)
		assert.Equal(t, dir, opt.RelDir())
	})

	t.Run("glob no match", func(t *testing.T) {
		dir := t.TempDir()
		opt := &FileOpts{Paths: []string{filepath.Join(dir, "*.xyz")}}
		assert.Error(t, opt.Valid())
	})

	t.Run("multiple paths shell expansion", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		f2 := filepath.Join(dir, "b.txt")
		require.NoError(t, os.WriteFile(f1, []byte("a"), 0644))
		require.NoError(t, os.WriteFile(f2, []byte("b"), 0644))

		opt := &FileOpts{Paths: []string{f1, f2}}
		require.NoError(t, opt.Valid())
		assert.Equal(t, dir, opt.RelDir())
		assert.Equal(t, []string{f1, f2}, opt.GetPaths())
	})

	t.Run("multiple paths bad entry", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		require.NoError(t, os.WriteFile(f1, []byte("a"), 0644))

		opt := &FileOpts{Paths: []string{f1, filepath.Join(dir, "nope.txt")}}
		assert.Error(t, opt.Valid())
	})
}

func TestFileOpts_GetPaths_Empty(t *testing.T) {
	opt := &FileOpts{}
	assert.Nil(t, opt.GetPaths())
}

func TestCommonDir(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{"empty", nil, "."},
		{"single file", []string{"/a/b/file.txt"}, "/a/b"},
		{"siblings", []string{"/a/b/f1", "/a/b/f2"}, "/a/b"},
		{"nested", []string{"/a/b/c/f1", "/a/b/d/f2"}, "/a/b"},
		{"divergent", []string{"/a/x/f1", "/b/y/f2"}, "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, commonDir(tt.paths))
		})
	}
}

func TestUploadManagerOpts_ShouldUseInteractiveMode(t *testing.T) {
	t.Run("NoTTY forces non-interactive", func(t *testing.T) {
		opt := &UploadManagerOpts{NoTTY: true}
		assert.False(t, opt.ShouldUseInteractiveMode())
	})

	t.Run("TTY forces interactive", func(t *testing.T) {
		opt := &UploadManagerOpts{TTY: true}
		assert.True(t, opt.ShouldUseInteractiveMode())
	})
}
