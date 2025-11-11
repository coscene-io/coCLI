// Copyright 2025 coScene
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
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestContext returns a context with a reasonable timeout for tests.
func TestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TempDir creates a temporary directory for testing and ensures cleanup.
func TempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cocli-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

// CreateTempFile creates a temporary file with the given content.
func CreateTempFile(t *testing.T, dir, pattern string, content []byte) string {
	t.Helper()
	file, err := os.CreateTemp(dir, pattern)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	if len(content) > 0 {
		_, err = file.Write(content)
		require.NoError(t, err)
	}

	return file.Name()
}

// CopyFile copies a file from src to dst.
func CopyFile(t *testing.T, src, dst string) {
	t.Helper()
	srcFile, err := os.Open(src)
	require.NoError(t, err)
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	require.NoError(t, err)
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	require.NoError(t, err)
}

// CreateTestFileTree creates a directory structure with files for testing.
func CreateTestFileTree(t *testing.T, baseDir string, files map[string][]byte) {
	t.Helper()
	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		dir := filepath.Dir(fullPath)

		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		if content != nil {
			err = os.WriteFile(fullPath, content, 0644)
			require.NoError(t, err)
		}
	}
}

// AssertFileContent checks that a file contains the expected content.
func AssertFileContent(t *testing.T, path string, expected []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

// SkipIfShort skips the test if running with -short flag.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// RequireEnv skips the test if the given environment variable is not set.
func RequireEnv(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test: environment variable %s not set", key)
	}
	return value
}

// CaptureOutput captures stdout and stderr for testing command output.
func CaptureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	outChan := make(chan string)
	errChan := make(chan string)

	go func() {
		buf, _ := io.ReadAll(rOut)
		outChan <- string(buf)
	}()

	go func() {
		buf, _ := io.ReadAll(rErr)
		errChan <- string(buf)
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	stdout = <-outChan
	stderr = <-errChan

	return stdout, stderr
}
