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
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/fs"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/minio/minio-go/v7"
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

	t.Run("Update marks manual quit on escape keys", func(t *testing.T) {
		um := &UploadManager{}
		model, cmd := um.Update(tea.KeyMsg{Type: tea.KeyEscape})
		u := model.(*UploadManager)
		assert.True(t, u.manualQuit)
		assert.NotNil(t, cmd)
	})
}

func TestUploadParents(t *testing.T) {
	recordName, err := name.NewRecord("projects/project-a/records/record-a")
	require.NoError(t, err)
	recordParent := NewRecordParent(recordName)
	assert.Equal(t, "projects/project-a/records/record-a", recordParent.ParentString())
	assert.Equal(t, "projects/project-a/records/record-a/files/data/file.txt", recordParent.BuildResourceName("data/file.txt"))

	projectName, err := name.NewProject("projects/project-a")
	require.NoError(t, err)
	projectParent := NewProjectParent(projectName)
	assert.Equal(t, "projects/project-a", projectParent.ParentString())
	assert.Equal(t, "projects/project-a/files/data/file.txt", projectParent.BuildResourceName("data/file.txt"))

	ctx := newParentContextFrom(projectParent)
	assert.Equal(t, "projects/project-a", ctx.parentString)
	assert.Equal(t, "projects/project-a/files/other.txt", ctx.buildResourceName("other.txt"))
}

func TestUploadManager_ParseUrl(t *testing.T) {
	um := &UploadManager{}

	bucket, key, tags, err := um.parseUrl("https://oss.example.com/upload-bucket/path/to/file.txt?X-Amz-Tagging=X-COS-RECORD-ID%3Drecord-a%26X-COS-PROJECT-ID%3Dproject-a")

	require.NoError(t, err)
	assert.Equal(t, "upload-bucket", bucket)
	assert.Equal(t, "path/to/file.txt", key)
	assert.Equal(t, map[string]string{
		"X-COS-RECORD-ID":  "record-a",
		"X-COS-PROJECT-ID": "project-a",
	}, tags)
}

func TestUploadManager_CanUpload(t *testing.T) {
	um := &UploadManager{opts: &UploadManagerOpts{partSizeUint64: 512 * 1024 * 1024}}

	assert.True(t, um.canUpload(UploadInfo{Path: "a", PartId: 20}, nil))
	assert.True(t, um.canUpload(
		UploadInfo{Path: "a", PartId: 3},
		[]UploadInfo{{Path: "a", PartId: 1}, {Path: "b", PartId: 99}},
	))
	assert.False(t, um.canUpload(
		UploadInfo{Path: "a", PartId: 4},
		[]UploadInfo{{Path: "a", PartId: 1}},
	))
}

func TestUploadManager_StateHelpers(t *testing.T) {
	um := &UploadManager{
		fileInfos: make(map[string]*FileInfo),
		errs:      make(map[string]error),
		noTTY:     true,
	}

	um.addFile("a.txt")
	require.Contains(t, um.fileInfos, "a.txt")
	assert.Equal(t, []string{"a.txt"}, um.fileList)

	um.uploadWg.Add(1)
	um.addErr("a.txt", assert.AnError)
	assert.Equal(t, UploadFailed, um.fileInfos["a.txt"].Status)
	assert.ErrorIs(t, um.errs["a.txt"], assert.AnError)
}

func TestUploadManager_UpdateUploadSpeeds(t *testing.T) {
	um := &UploadManager{
		fileList: []string{"a", "b"},
		fileInfos: map[string]*FileInfo{
			"a": {Path: "a", Status: UploadInProgress, Uploaded: 400},
			"b": {Path: "b", Status: UploadCompleted, Uploaded: 400},
		},
		lastSpeedSample: map[string]struct {
			uploaded int64
			t        time.Time
		}{
			"a": {uploaded: 100, t: time.Now().Add(-time.Second)},
		},
	}

	um.updateUploadSpeeds()

	assert.Greater(t, um.fileInfos["a"].SpeedBps, float64(0))
	assert.Equal(t, float64(0), um.fileInfos["b"].SpeedBps)
}

func TestUploadProgressReaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	require.NoError(t, os.WriteFile(path, []byte("abcdef"), 0644))

	t.Run("file reader reports bytes read", func(t *testing.T) {
		file, err := os.Open(path)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		ch := make(chan IncUploadedMsg, 1)
		reader := &uploadProgressReader{
			File:       file,
			fileInfo:   &FileInfo{Path: path},
			progressCh: ch,
		}

		buf := make([]byte, 3)
		n, err := reader.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, IncUploadedMsg{Path: path, UploadedInc: 3}, <-ch)
	})

	t.Run("section reader reports bytes read", func(t *testing.T) {
		file, err := os.Open(path)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		ch := make(chan IncUploadedMsg, 1)
		reader := &uploadProgressSectionReader{
			SectionReader: io.NewSectionReader(file, 2, 2),
			fileInfo:      &FileInfo{Path: path},
			progressCh:    ch,
		}

		buf := make([]byte, 2)
		n, err := reader.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, "cd", string(buf))
		assert.Equal(t, IncUploadedMsg{Path: path, UploadedInc: 2}, <-ch)
	})
}

func TestUploadManager_FindAllUploadUrls(t *testing.T) {
	dir := t.TempDir()
	skipPath := filepath.Join(dir, "skip.txt")
	uploadPath := filepath.Join(dir, "upload.txt")
	require.NoError(t, os.WriteFile(skipPath, []byte("already there"), 0644))
	require.NoError(t, os.WriteFile(uploadPath, []byte("new data"), 0644))

	skipSha, skipSize, err := fs.CalSha256AndSize(skipPath)
	require.NoError(t, err)

	projectName, err := name.NewProject("projects/project-a")
	require.NoError(t, err)
	api := &fakeUploadFileAPI{
		existing: map[string]*openv1alpha1resource.File{
			"projects/project-a/files/remote/skip.txt": {
				Name:   "projects/project-a/files/remote/skip.txt",
				Sha256: skipSha,
				Size:   skipSize,
			},
		},
		uploadURLs: map[string]string{
			"projects/project-a/files/remote/upload.txt": "https://oss.example.com/bucket/remote/upload.txt?X-Amz-Tagging=X-COS-PROJECT-ID%3Dproject-a",
		},
	}
	um := &UploadManager{
		apiOpts:   &ApiOpts{FileInterface: api},
		fileInfos: make(map[string]*FileInfo),
		errs:      make(map[string]error),
		noTTY:     true,
	}
	um.uploadWg.Add(1)

	got := um.findAllUploadUrls(
		context.Background(),
		[]string{skipPath, uploadPath},
		newParentContextFrom(NewProjectParent(projectName)),
		dir,
		"remote",
	)

	assert.Equal(t, map[string]string{uploadPath: "https://oss.example.com/bucket/remote/upload.txt?X-Amz-Tagging=X-COS-PROJECT-ID%3Dproject-a"}, got)
	assert.Equal(t, PreviouslyUploaded, um.fileInfos[skipPath].Status)
	assert.Equal(t, WaitingForUpload, um.fileInfos[uploadPath].Status)
	assert.Equal(t, "remote/upload.txt", um.fileInfos[uploadPath].RemotePath)
	require.Len(t, api.generatedFiles, 1)
	assert.Equal(t, "projects/project-a/files/remote/upload.txt", api.generatedFiles[0].Name)
}

func TestUploadManager_ProduceUploadInfosForSingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	require.NoError(t, os.WriteFile(path, []byte("small"), 0644))

	um := &UploadManager{
		opts: &UploadManagerOpts{partSizeUint64: 1024},
		fileInfos: map[string]*FileInfo{
			path: {Path: path, Size: 5},
		},
		fileList: []string{path},
		errs:     make(map[string]error),
	}
	ch := make(chan UploadInfo, 1)

	um.produceUploadInfos(context.Background(), map[string]string{
		path: "https://oss.example.com/bucket/key.txt?X-Amz-Tagging=X-COS-RECORD-ID%3Drecord-a",
	}, ch)

	info, ok := <-ch
	require.True(t, ok)
	defer func() { _ = info.FileReader.Close() }()
	assert.Equal(t, path, info.Path)
	assert.Equal(t, "bucket", info.Bucket)
	assert.Equal(t, "key.txt", info.Key)
	assert.Equal(t, map[string]string{"X-COS-RECORD-ID": "record-a"}, info.Tags)

	_, ok = <-ch
	assert.False(t, ok)
}

func TestUploadManager_HandleUploadResult(t *testing.T) {
	t.Run("single upload marks file complete", func(t *testing.T) {
		um := &UploadManager{
			fileInfos: map[string]*FileInfo{"file.txt": {Path: "file.txt"}},
			noTTY:     true,
		}
		um.uploadWg.Add(1)

		require.NoError(t, um.handleUploadResult(UploadInfo{Path: "file.txt"}))
		assert.Equal(t, UploadCompleted, um.fileInfos["file.txt"].Status)
	})

	t.Run("error result is returned", func(t *testing.T) {
		um := &UploadManager{}
		err := um.handleUploadResult(UploadInfo{Err: assert.AnError})
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("multipart upload writes checkpoint before completion", func(t *testing.T) {
		oldUploaderDir := constants.DefaultUploaderDirPath
		constants.DefaultUploaderDirPath = t.TempDir()
		t.Cleanup(func() {
			constants.DefaultUploaderDirPath = oldUploaderDir
		})

		filePath := filepath.Join(t.TempDir(), "large.bin")
		require.NoError(t, os.WriteFile(filePath, []byte("large"), 0644))
		file, err := os.Open(filePath)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		db, err := NewUploadDB(filePath, "record-a", "sha", 2)
		require.NoError(t, err)
		defer func() { _ = db.Delete() }()

		um := &UploadManager{
			fileInfos: map[string]*FileInfo{filePath: {Path: filePath}},
			noTTY:     true,
		}

		result := UploadInfo{
			Path:            filePath,
			UploadId:        "upload-a",
			TotalPartsCount: 2,
			ReadSize:        2,
			Result:          minio.ObjectPart{PartNumber: 1, ETag: "etag-1"},
			FileReader:      file,
			DB:              db,
		}
		require.NoError(t, um.handleUploadResult(result))

		var checkpoint MultipartCheckpointInfo
		require.NoError(t, db.Get(mutipartUploadInfoKey, &checkpoint))
		assert.Equal(t, "upload-a", checkpoint.UploadId)
		assert.Equal(t, int64(2), checkpoint.UploadedSize)
		require.Len(t, checkpoint.Parts, 1)
		assert.Equal(t, 1, checkpoint.Parts[0].PartNumber)
	})
}

type fakeUploadFileAPI struct {
	existing       map[string]*openv1alpha1resource.File
	uploadURLs     map[string]string
	generatedFiles []*openv1alpha1resource.File
}

func (f *fakeUploadFileAPI) GetFile(ctx context.Context, fileResourceName string) (*openv1alpha1resource.File, error) {
	if file, ok := f.existing[fileResourceName]; ok {
		return file, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeUploadFileAPI) GenerateFileUploadUrls(ctx context.Context, parent string, files []*openv1alpha1resource.File) (map[string]string, error) {
	f.generatedFiles = append(f.generatedFiles, files...)
	ret := make(map[string]string, len(files))
	for _, file := range files {
		ret[file.Name] = f.uploadURLs[file.Name]
	}
	return ret, nil
}

func (f *fakeUploadFileAPI) GenerateFileDownloadUrl(ctx context.Context, fileResourceName string) (string, error) {
	return "", errors.New("not implemented")
}

func (f *fakeUploadFileAPI) DeleteFile(ctx context.Context, fileResourceName string) error {
	return errors.New("not implemented")
}

func (f *fakeUploadFileAPI) BatchDeleteFiles(ctx context.Context, parent string, names []string) error {
	return errors.New("not implemented")
}
