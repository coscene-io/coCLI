package upload_utils

import (
	"strings"
	"testing"
)

func TestUploadManagerViewMissingFileInfoDoesNotPanic(t *testing.T) {
	um := &UploadManager{
		fileList:    []string{"/tmp/missing"},
		fileInfos:   map[string]*FileInfo{},
		windowWidth: 80,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked: %v", r)
		}
	}()

	view := um.View()
	if !strings.Contains(view, "Waiting for upload") {
		t.Fatalf("unexpected view output: %q", view)
	}
}

func TestUploadManagerCalculateUploadProgressMissingEntry(t *testing.T) {
	um := &UploadManager{
		fileInfos: make(map[string]*FileInfo),
	}

	if progress := um.calculateUploadProgress("/tmp/absent"); progress != 0 {
		t.Fatalf("expected progress 0, got %v", progress)
	}
}
