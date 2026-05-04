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

package cmd_utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/coscene-io/cocli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgress_WriteAccumulatesBytes(t *testing.T) {
	progress := &Progress{TotalSize: 10, PrintPrefix: "download"}

	n, err := progress.Write([]byte("abc"))

	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, int64(3), progress.BytesRead)
}

func TestDownloadFileThroughUrl(t *testing.T) {
	t.Run("downloads into nested destination", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("hello"))
		}))
		defer server.Close()

		dst := filepath.Join(t.TempDir(), "nested", "file.txt")
		require.NoError(t, DownloadFileThroughUrl(dst, server.URL, 0))
		assertFileContent(t, dst, "hello")
	})

	t.Run("non-successful HTTP status fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusForbidden)
		}))
		defer server.Close()

		dst := filepath.Join(t.TempDir(), "file.txt")
		err := DownloadFileThroughUrl(dst, server.URL, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403 Forbidden")
	})

	t.Run("retry starts from a clean file", func(t *testing.T) {
		var attempts int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&attempts, 1) == 1 {
				w.Header().Set("Content-Length", "8")
				_, _ = w.Write([]byte("bad"))
				return
			}
			_, _ = w.Write([]byte("good"))
		}))
		defer server.Close()

		dst := filepath.Join(t.TempDir(), "file.txt")
		require.NoError(t, DownloadFileThroughUrl(dst, server.URL, 1))
		assertFileContent(t, dst, "good")
		assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
	})
}

func TestSaveMomentsJson(t *testing.T) {
	dir := t.TempDir()
	moments := []*api.Moment{
		{
			Name:        "moments/one",
			Description: "interesting point",
			Attribute:   map[string]string{"kind": "brake"},
		},
	}

	require.NoError(t, SaveMomentsJson(moments, dir))

	data, err := os.ReadFile(filepath.Join(dir, "moments.json"))
	require.NoError(t, err)

	var payload struct {
		Moments []*api.Moment `json:"moments"`
	}
	require.NoError(t, json.Unmarshal(data, &payload))
	require.Len(t, payload.Moments, 1)
	assert.Equal(t, "moments/one", payload.Moments[0].Name)
	assert.Equal(t, "brake", payload.Moments[0].Attribute["kind"])
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}
