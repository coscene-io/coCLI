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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFileThroughUrlRetriesHTTPStatusAndReplacesBody(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "download.txt")
	err := downloadFileThroughUrl(dst, server.URL, 1, time.Millisecond, time.Millisecond)

	require.NoError(t, err)
	require.Equal(t, int32(2), attempts.Load())
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(got))
}

func TestDownloadFileThroughUrlTruncatesPartialFileBeforeRetry(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Content-Length", "12")
			_, _ = w.Write([]byte("partial"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Errorf("response writer does not support hijacking")
				return
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Errorf("hijack response: %v", err)
				return
			}
			_ = conn.Close()
			return
		}
		_, _ = w.Write([]byte("complete"))
	}))
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "download.txt")
	err := downloadFileThroughUrl(dst, server.URL, 1, time.Millisecond, time.Millisecond)

	require.NoError(t, err)
	require.Equal(t, int32(2), attempts.Load())
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "complete", string(got))
}
