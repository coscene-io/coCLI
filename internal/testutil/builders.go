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
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/name"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RecordBuilder helps create test Record objects.
type RecordBuilder struct {
	record *openv1alpha1resource.Record
}

// NewRecordBuilder creates a new RecordBuilder with defaults.
func NewRecordBuilder() *RecordBuilder {
	return &RecordBuilder{
		record: &openv1alpha1resource.Record{
			Name:        "projects/test-project/records/test-record",
			Title:       "Test Record",
			Description: "Test record description",
			CreateTime:  timestamppb.New(time.Now()),
			UpdateTime:  timestamppb.New(time.Now()),
		},
	}
}

// WithName sets the record name.
func (b *RecordBuilder) WithName(name string) *RecordBuilder {
	b.record.Name = name
	return b
}

// WithTitle sets the record title.
func (b *RecordBuilder) WithTitle(title string) *RecordBuilder {
	b.record.Title = title
	return b
}

// WithDescription sets the record description.
func (b *RecordBuilder) WithDescription(desc string) *RecordBuilder {
	b.record.Description = desc
	return b
}

// WithLabels sets the record labels.
func (b *RecordBuilder) WithLabels(labels []*openv1alpha1resource.Label) *RecordBuilder {
	b.record.Labels = labels
	return b
}

// WithDevice sets the record device.
func (b *RecordBuilder) WithDevice(deviceName string) *RecordBuilder {
	b.record.Device = &openv1alpha1resource.Device{
		Name: deviceName,
	}
	return b
}

// Build returns the built Record.
func (b *RecordBuilder) Build() *openv1alpha1resource.Record {
	return b.record
}

// FileBuilder helps create test File objects.
type FileBuilder struct {
	file *openv1alpha1resource.File
}

// NewFileBuilder creates a new FileBuilder with defaults.
func NewFileBuilder() *FileBuilder {
	return &FileBuilder{
		file: &openv1alpha1resource.File{
			Name:       "projects/test-project/records/test-record/files/test.txt",
			Filename:   "test.txt",
			Size:       1024,
			Sha256:     "abc123",
			CreateTime: timestamppb.New(time.Now()),
			UpdateTime: timestamppb.New(time.Now()),
		},
	}
}

// WithName sets the file resource name.
func (b *FileBuilder) WithName(name string) *FileBuilder {
	b.file.Name = name
	return b
}

// WithFilename sets the filename.
func (b *FileBuilder) WithFilename(filename string) *FileBuilder {
	b.file.Filename = filename
	return b
}

// WithSize sets the file size.
func (b *FileBuilder) WithSize(size int64) *FileBuilder {
	b.file.Size = size
	return b
}

// WithSha256 sets the file SHA256.
func (b *FileBuilder) WithSha256(sha256 string) *FileBuilder {
	b.file.Sha256 = sha256
	return b
}

// Build returns the built File.
func (b *FileBuilder) Build() *openv1alpha1resource.File {
	return b.file
}

// ProjectBuilder helps create test Project objects.
type ProjectBuilder struct {
	project *openv1alpha1resource.Project
}

// NewProjectBuilder creates a new ProjectBuilder with defaults.
func NewProjectBuilder() *ProjectBuilder {
	return &ProjectBuilder{
		project: &openv1alpha1resource.Project{
			Name:        "projects/test-project",
			DisplayName: "Test Project",
			CreateTime:  timestamppb.New(time.Now()),
			UpdateTime:  timestamppb.New(time.Now()),
		},
	}
}

// WithName sets the project name.
func (b *ProjectBuilder) WithName(name string) *ProjectBuilder {
	b.project.Name = name
	return b
}

// WithDisplayName sets the project display name.
func (b *ProjectBuilder) WithDisplayName(displayName string) *ProjectBuilder {
	b.project.DisplayName = displayName
	return b
}

// Build returns the built Project.
func (b *ProjectBuilder) Build() *openv1alpha1resource.Project {
	return b.project
}

// Common test names
var (
	TestProjectName = &name.Project{
		ProjectID: "test-project",
	}

	TestRecordName = &name.Record{
		ProjectID: "test-project",
		RecordID:  "test-record",
	}

	TestFileName = &name.File{
		ProjectID: "test-project",
		RecordID:  "test-record",
		Filename:  "test.txt",
	}
)

// CreateTestLabel creates a test label with the given display name.
func CreateTestLabel(displayName string) *openv1alpha1resource.Label {
	return &openv1alpha1resource.Label{
		Name:        "labels/test-label-" + displayName,
		DisplayName: displayName,
	}
}
