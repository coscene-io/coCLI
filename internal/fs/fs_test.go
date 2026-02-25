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

package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFiles_Empty(t *testing.T) {
	assert.Empty(t, FindFiles("", false, false))
}

func TestFindFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0644))

	files := FindFiles(f, false, false)
	assert.Equal(t, []string{f}, files)
}

func TestFindFiles_NonRecursive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644))

	files := FindFiles(dir, false, false)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0], "a.txt")
}

func TestFindFiles_Recursive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644))

	files := FindFiles(dir, true, false)
	assert.Len(t, files, 2)
}

func TestFindFiles_HiddenFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("h"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("v"), 0644))

	without := FindFiles(dir, false, false)
	assert.Len(t, without, 1)

	with := FindFiles(dir, false, true)
	assert.Len(t, with, 2)
}

func TestFindFiles_SkipsCocliDir(t *testing.T) {
	dir := t.TempDir()
	cocliDir := filepath.Join(dir, ".cocli")
	require.NoError(t, os.Mkdir(cocliDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cocliDir, "state.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("d"), 0644))

	files := FindFiles(dir, true, true)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0], "data.txt")
}

func TestCalSha256AndSize(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.bin")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(f, content, 0644))

	hash, size, err := CalSha256AndSize(f)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", hash)
}

func TestCalSha256AndSize_NonExistent(t *testing.T) {
	_, _, err := CalSha256AndSize("/nonexistent/file")
	assert.Error(t, err)
}

func TestCalSha256AndSize_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "empty")
	require.NoError(t, os.WriteFile(f, []byte{}, 0644))

	hash, size, err := CalSha256AndSize(f)
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
}
