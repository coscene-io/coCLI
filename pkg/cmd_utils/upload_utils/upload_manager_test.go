// Copyright 2026 coScene
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this code except in compliance with the License.
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
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{15 * time.Second, "15s"},
		{59 * time.Second, "59s"},
		{time.Minute, "1m"},
		{90 * time.Second, "1m 30s"},
		{2*time.Minute + 30*time.Second, "2m 30s"},
		{time.Hour, "1h"},
		{time.Hour + 5*time.Minute, "1h 5m"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatDuration(tt.d))
		})
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		bps  float64
		want string
	}{
		{0, "0 B/s"},
		{500, "500 B/s"},
		{1024, "1 KB/s"},
		{1536, "2 KB/s"},
		{1024 * 1024, "1.0 MB/s"},
		{12.3 * 1024 * 1024, "12.3 MB/s"},
		{1024 * 1024 * 1024, "1.0 GB/s"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSpeed(tt.bps))
		})
	}
}

func TestCalculateUploadProgress(t *testing.T) {
	um := &UploadManager{
		fileInfos: map[string]*FileInfo{
			"a": {Size: 100, Uploaded: 50},
			"b": {Size: 100, Uploaded: 100},
			"c": {Size: 0, Uploaded: 0},
			"d": {Size: 0, Uploaded: 100},
		},
	}
	assert.Equal(t, 50.0, um.calculateUploadProgress("a"))
	assert.Equal(t, 100.0, um.calculateUploadProgress("b"))
	assert.Equal(t, 100.0, um.calculateUploadProgress("c")) // size 0 -> 100%
	assert.Equal(t, 100.0, um.calculateUploadProgress("d"))
}

func TestUploadManager_View(t *testing.T) {
	t.Run("UploadInProgress shows progress bar, speed, and time spent", func(t *testing.T) {
		startTime := time.Now().Add(-125 * time.Second) // ~2m 5s ago
		um := &UploadManager{
			fileInfos: map[string]*FileInfo{
				"/tmp/file.txt": {
					Path:            "/tmp/file.txt",
					RemotePath:      "file.txt",
					Size:            1000,
					Uploaded:        500,
					Status:          UploadInProgress,
					SpeedBps:        1024 * 1024, // 1 MB/s
					UploadStartTime: startTime,
				},
			},
			fileList:    []string{"/tmp/file.txt"},
			windowWidth: 80,
			spinnerIdx:  0,
		}
		s := um.View()
		require.Contains(t, s, "Upload Status:")
		require.Contains(t, s, "file.txt")
		require.Contains(t, s, "[")
		require.Contains(t, s, "]")
		require.Contains(t, s, "%")
		require.Contains(t, s, "50.00%")
		require.Contains(t, s, "1.0 MB/s")
		// Time format varies slightly; just check we have elapsed (e.g. "2m" or "2m 5s")
		require.Regexp(t, `\d+[ms]`, s)
	})

	t.Run("other statuses render without progress bar", func(t *testing.T) {
		um := &UploadManager{
			fileInfos: map[string]*FileInfo{
				"a": {Path: "a", RemotePath: "a", Status: UploadCompleted},
				"b": {Path: "b", RemotePath: "b", Status: PreviouslyUploaded},
				"c": {Path: "c", RemotePath: "c", Status: UploadFailed},
			},
			fileList:    []string{"a", "b", "c"},
			windowWidth: 80,
		}
		s := um.View()
		require.Contains(t, s, "Upload completed")
		require.Contains(t, s, "Previously uploaded")
		require.Contains(t, s, "Upload failed")
		require.NotContains(t, s, "MB/s") // no speed for non-uploading
	})

	t.Run("Update sets window width from WindowSizeMsg", func(t *testing.T) {
		um := &UploadManager{windowWidth: 40}
		model, _ := um.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
		u := model.(*UploadManager)
		assert.Equal(t, 120, u.windowWidth)
	})
}
