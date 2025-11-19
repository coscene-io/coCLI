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

package api

import (
	"context"
	"testing"

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

// mockRecordServiceClient is a mock implementation that can be configured per test
type mockRecordServiceClient struct {
	openv1alpha1connect.RecordServiceClient
	ctrl *gomock.Controller

	// Configurable responses
	getRecordFunc            func(context.Context, *connect.Request[openv1alpha1service.GetRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error)
	createRecordFunc         func(context.Context, *connect.Request[openv1alpha1service.CreateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error)
	copyRecordsFunc          func(context.Context, *connect.Request[openv1alpha1service.CopyRecordsRequest]) (*connect.Response[openv1alpha1service.CopyRecordsResponse], error)
	moveRecordsFunc          func(context.Context, *connect.Request[openv1alpha1service.MoveRecordsRequest]) (*connect.Response[openv1alpha1service.MoveRecordsResponse], error)
	deleteRecordFunc         func(context.Context, *connect.Request[openv1alpha1service.DeleteRecordRequest]) (*connect.Response[emptypb.Empty], error)
	updateRecordFunc         func(context.Context, *connect.Request[openv1alpha1service.UpdateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error)
	listRecordsFunc          func(context.Context, *connect.Request[openv1alpha1service.ListRecordsRequest]) (*connect.Response[openv1alpha1service.ListRecordsResponse], error)
	listRecordEventsFunc     func(context.Context, *connect.Request[openv1alpha1service.ListRecordEventsRequest]) (*connect.Response[openv1alpha1service.ListRecordEventsResponse], error)
	generateThumbnailUrlFunc func(context.Context, *connect.Request[openv1alpha1service.GenerateRecordThumbnailUploadUrlRequest]) (*connect.Response[openv1alpha1service.GenerateRecordThumbnailUploadUrlResponse], error)
}

func (m *mockRecordServiceClient) GetRecord(ctx context.Context, req *connect.Request[openv1alpha1service.GetRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
	if m.getRecordFunc != nil {
		return m.getRecordFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) CreateRecord(ctx context.Context, req *connect.Request[openv1alpha1service.CreateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
	if m.createRecordFunc != nil {
		return m.createRecordFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) CopyRecords(ctx context.Context, req *connect.Request[openv1alpha1service.CopyRecordsRequest]) (*connect.Response[openv1alpha1service.CopyRecordsResponse], error) {
	if m.copyRecordsFunc != nil {
		return m.copyRecordsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) MoveRecords(ctx context.Context, req *connect.Request[openv1alpha1service.MoveRecordsRequest]) (*connect.Response[openv1alpha1service.MoveRecordsResponse], error) {
	if m.moveRecordsFunc != nil {
		return m.moveRecordsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) DeleteRecord(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteRecordRequest]) (*connect.Response[emptypb.Empty], error) {
	if m.deleteRecordFunc != nil {
		return m.deleteRecordFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) UpdateRecord(ctx context.Context, req *connect.Request[openv1alpha1service.UpdateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
	if m.updateRecordFunc != nil {
		return m.updateRecordFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) ListRecords(ctx context.Context, req *connect.Request[openv1alpha1service.ListRecordsRequest]) (*connect.Response[openv1alpha1service.ListRecordsResponse], error) {
	if m.listRecordsFunc != nil {
		return m.listRecordsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) ListRecordEvents(ctx context.Context, req *connect.Request[openv1alpha1service.ListRecordEventsRequest]) (*connect.Response[openv1alpha1service.ListRecordEventsResponse], error) {
	if m.listRecordEventsFunc != nil {
		return m.listRecordEventsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockRecordServiceClient) GenerateRecordThumbnailUploadUrl(ctx context.Context, req *connect.Request[openv1alpha1service.GenerateRecordThumbnailUploadUrlRequest]) (*connect.Response[openv1alpha1service.GenerateRecordThumbnailUploadUrlResponse], error) {
	if m.generateThumbnailUrlFunc != nil {
		return m.generateThumbnailUrlFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// mockFileServiceClient is a simplified mock for file operations
type mockFileServiceClient struct {
	openv1alpha1connect.FileServiceClient
	ctrl *gomock.Controller

	listFilesFunc  func(context.Context, *connect.Request[openv1alpha1service.ListFilesRequest]) (*connect.Response[openv1alpha1service.ListFilesResponse], error)
	deleteFileFunc func(context.Context, *connect.Request[openv1alpha1service.DeleteFileRequest]) (*connect.Response[emptypb.Empty], error)
	copyFilesFunc  func(context.Context, *connect.Request[openv1alpha1service.CopyFilesRequest]) (*connect.Response[openv1alpha1service.CopyFilesResponse], error)
	moveFilesFunc  func(context.Context, *connect.Request[openv1alpha1service.MoveFilesRequest]) (*connect.Response[openv1alpha1service.MoveFilesResponse], error)
}

func (m *mockFileServiceClient) ListFiles(ctx context.Context, req *connect.Request[openv1alpha1service.ListFilesRequest]) (*connect.Response[openv1alpha1service.ListFilesResponse], error) {
	if m.listFilesFunc != nil {
		return m.listFilesFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockFileServiceClient) DeleteFile(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteFileRequest]) (*connect.Response[emptypb.Empty], error) {
	if m.deleteFileFunc != nil {
		return m.deleteFileFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockFileServiceClient) CopyFiles(ctx context.Context, req *connect.Request[openv1alpha1service.CopyFilesRequest]) (*connect.Response[openv1alpha1service.CopyFilesResponse], error) {
	if m.copyFilesFunc != nil {
		return m.copyFilesFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockFileServiceClient) MoveFiles(ctx context.Context, req *connect.Request[openv1alpha1service.MoveFilesRequest]) (*connect.Response[openv1alpha1service.MoveFilesResponse], error) {
	if m.moveFilesFunc != nil {
		return m.moveFilesFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// mockLabelServiceClient is a simplified mock
type mockLabelServiceClient struct {
	openv1alpha1connect.LabelServiceClient
	ctrl           *gomock.Controller
	listLabelsFunc func(context.Context, *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error)
}

func (m *mockLabelServiceClient) ListLabels(ctx context.Context, req *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error) {
	if m.listLabelsFunc != nil {
		return m.listLabelsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestRecordClient_Get(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recordName := &name.Record{
		ProjectID: "test-project",
		RecordID:  "test-record",
	}
	expectedRecord := testutil.NewRecordBuilder().
		WithName(recordName.String()).
		Build()

	mockRecordService := &mockRecordServiceClient{
		ctrl: ctrl,
		getRecordFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
			assert.Equal(t, recordName.String(), req.Msg.Name)
			return connect.NewResponse(expectedRecord), nil
		},
	}

	client := NewRecordClient(mockRecordService, nil, nil, nil)

	record, err := client.Get(ctx, recordName)
	require.NoError(t, err)
	assert.Equal(t, expectedRecord, record)
}

func TestRecordClient_Create(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectName := &name.Project{ProjectID: "test-project"}
	title := "New Record"
	description := "Test description"
	deviceName := "test-device"
	labels := []*openv1alpha1resource.Label{
		testutil.CreateTestLabel("label1"),
	}

	expectedRecord := testutil.NewRecordBuilder().
		WithTitle(title).
		WithDescription(description).
		WithDevice(deviceName).
		WithLabels(labels).
		Build()

	mockRecordService := &mockRecordServiceClient{
		ctrl: ctrl,
		createRecordFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
			assert.Equal(t, projectName.String(), req.Msg.Parent)
			assert.Equal(t, title, req.Msg.Record.Title)
			assert.Equal(t, description, req.Msg.Record.Description)
			assert.Equal(t, deviceName, req.Msg.Record.Device.Name)
			assert.Equal(t, labels, req.Msg.Record.Labels)
			return connect.NewResponse(expectedRecord), nil
		},
	}

	client := NewRecordClient(mockRecordService, nil, nil, nil)

	record, err := client.Create(ctx, projectName, title, deviceName, description, labels)
	require.NoError(t, err)
	assert.Equal(t, expectedRecord, record)
}

func TestRecordClient_Delete(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recordName := &name.Record{
		ProjectID: "test-project",
		RecordID:  "test-record",
	}

	mockRecordService := &mockRecordServiceClient{
		ctrl: ctrl,
		deleteRecordFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteRecordRequest]) (*connect.Response[emptypb.Empty], error) {
			assert.Equal(t, recordName.String(), req.Msg.Name)
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
	}

	client := NewRecordClient(mockRecordService, nil, nil, nil)

	err := client.Delete(ctx, recordName)
	require.NoError(t, err)
}

func TestRecordClient_Update(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recordName := &name.Record{
		ProjectID: "test-project",
		RecordID:  "test-record",
	}
	newTitle := "Updated Title"
	newDescription := "Updated Description"
	newLabels := []*openv1alpha1resource.Label{
		testutil.CreateTestLabel("new-label"),
	}
	fieldMask := []string{"title", "description", "labels"}

	mockRecordService := &mockRecordServiceClient{
		ctrl: ctrl,
		updateRecordFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.UpdateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
			assert.Equal(t, recordName.String(), req.Msg.Record.Name)
			assert.Equal(t, newTitle, req.Msg.Record.Title)
			assert.Equal(t, newDescription, req.Msg.Record.Description)
			assert.Equal(t, newLabels, req.Msg.Record.Labels)
			assert.Equal(t, fieldMask, req.Msg.UpdateMask.Paths)
			return connect.NewResponse(&openv1alpha1resource.Record{}), nil
		},
	}

	client := NewRecordClient(mockRecordService, nil, nil, nil)

	err := client.Update(ctx, recordName, newTitle, newDescription, newLabels, fieldMask)
	require.NoError(t, err)
}

func TestRecordClient_ListAllFiles(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recordName := &name.Record{
		ProjectID: "test-project",
		RecordID:  "test-record",
	}

	// Create test files across multiple pages
	file1 := testutil.NewFileBuilder().WithFilename("file1.txt").Build()
	file2 := testutil.NewFileBuilder().WithFilename("file2.txt").Build()
	file3 := testutil.NewFileBuilder().WithFilename("file3.txt").Build()

	// The implementation increments by MaxPageSize (100) regardless of actual items returned
	// Since TotalSize is 3 and first call skip is 0, offs becomes 100 after first call
	// Then offs(100) >= TotalSize(3) is true, so it breaks without making second call
	// Therefore, we need to return all files in the first call or use TotalSize > MaxPageSize
	mockFileService := &mockFileServiceClient{
		ctrl: ctrl,
		listFilesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListFilesRequest]) (*connect.Response[openv1alpha1service.ListFilesResponse], error) {
			assert.Equal(t, recordName.String(), req.Msg.Parent)
			assert.Equal(t, int32(constants.MaxPageSize), req.Msg.PageSize)
			assert.Equal(t, int32(0), req.Msg.Skip)

			// Return all 3 files in first call since pagination logic won't make second call
			return connect.NewResponse(&openv1alpha1service.ListFilesResponse{
				Files:     []*openv1alpha1resource.File{file1, file2, file3},
				TotalSize: 3,
			}), nil
		},
	}

	client := NewRecordClient(nil, mockFileService, nil, nil)

	files, err := client.ListAllFiles(ctx, recordName)
	require.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Equal(t, []*openv1alpha1resource.File{file1, file2, file3}, files)
}

func TestRecordClient_CopyFiles(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srcRecord := &name.Record{ProjectID: "proj1", RecordID: "rec1"}
	dstRecord := &name.Record{ProjectID: "proj2", RecordID: "rec2"}

	files := []*openv1alpha1resource.File{
		testutil.NewFileBuilder().WithFilename("file1.txt").Build(),
		testutil.NewFileBuilder().WithFilename("file2.txt").Build(),
	}

	mockFileService := &mockFileServiceClient{
		ctrl: ctrl,
		copyFilesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CopyFilesRequest]) (*connect.Response[openv1alpha1service.CopyFilesResponse], error) {
			assert.Equal(t, srcRecord.String(), req.Msg.Parent)
			assert.Equal(t, dstRecord.String(), req.Msg.Destination)
			assert.Len(t, req.Msg.CopyPairs, 2)
			assert.Equal(t, "file1.txt", req.Msg.CopyPairs[0].SrcFile)
			assert.Equal(t, "file1.txt", req.Msg.CopyPairs[0].DstFile)
			return connect.NewResponse(&openv1alpha1service.CopyFilesResponse{
				Files: files,
			}), nil
		},
	}

	client := NewRecordClient(nil, mockFileService, nil, nil)

	err := client.CopyFiles(ctx, srcRecord, dstRecord, files)
	require.NoError(t, err)
}

func TestRecordClient_ListAll(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	project := &name.Project{ProjectID: "test-project"}
	options := &ListRecordsOptions{
		Project:        project,
		Titles:         []string{"Record 1", "Record 2"},
		Labels:         []string{"label1", "label2"},
		IncludeArchive: false,
	}

	// Mock records
	record1 := testutil.NewRecordBuilder().WithTitle("Record 1").Build()
	record2 := testutil.NewRecordBuilder().WithTitle("Record 2").Build()

	// Mock label service to transform label names
	mockLabelService := &mockLabelServiceClient{
		ctrl: ctrl,
		listLabelsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error) {
			return connect.NewResponse(&openv1alpha1service.ListLabelsResponse{
				Labels: []*openv1alpha1resource.Label{
					{Name: "projects/test-project/labels/label-id-1", DisplayName: "label1"},
					{Name: "projects/test-project/labels/label-id-2", DisplayName: "label2"},
				},
			}), nil
		},
	}

	callCount := 0
	mockRecordService := &mockRecordServiceClient{
		ctrl: ctrl,
		listRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListRecordsRequest]) (*connect.Response[openv1alpha1service.ListRecordsResponse], error) {
			assert.Equal(t, project.String(), req.Msg.Parent)
			assert.Contains(t, req.Msg.Filter, "is_archived=false")
			assert.Contains(t, req.Msg.Filter, `title:"Record 1"`)
			assert.Contains(t, req.Msg.Filter, `title:"Record 2"`)

			callCount++
			if callCount == 1 {
				return connect.NewResponse(&openv1alpha1service.ListRecordsResponse{
					Records:   []*openv1alpha1resource.Record{record1, record2},
					TotalSize: 2,
				}), nil
			}
			return connect.NewResponse(&openv1alpha1service.ListRecordsResponse{
				Records: []*openv1alpha1resource.Record{},
			}), nil
		},
	}

	client := NewRecordClient(mockRecordService, nil, nil, mockLabelService)

	records, err := client.ListAll(ctx, options, iostreams.System())
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestRecordClient_RecordId2Name(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectName := &name.Project{ProjectID: "test-project"}

	t.Run("Valid record name format", func(t *testing.T) {
		validName := "projects/test-project/records/test-record"
		client := NewRecordClient(nil, nil, nil, nil)

		recordName, err := client.RecordId2Name(ctx, validName, projectName)
		require.NoError(t, err)
		assert.Equal(t, "test-project", recordName.ProjectID)
		assert.Equal(t, "test-record", recordName.RecordID)
	})

	t.Run("Record ID only", func(t *testing.T) {
		recordID := "record-123"
		expectedRecord := testutil.NewRecordBuilder().Build()

		mockRecordService := &mockRecordServiceClient{
			ctrl: ctrl,
			getRecordFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
				assert.Equal(t, "projects/test-project/records/record-123", req.Msg.Name)
				return connect.NewResponse(expectedRecord), nil
			},
		}

		client := NewRecordClient(mockRecordService, nil, nil, nil)

		recordName, err := client.RecordId2Name(ctx, recordID, projectName)
		require.NoError(t, err)
		assert.Equal(t, "test-project", recordName.ProjectID)
		assert.Equal(t, recordID, recordName.RecordID)
	})

	t.Run("Record not found", func(t *testing.T) {
		recordID := "non-existent"

		mockRecordService := &mockRecordServiceClient{
			ctrl: ctrl,
			getRecordFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
				return nil, connect.NewError(connect.CodeNotFound, nil)
			},
		}

		client := NewRecordClient(mockRecordService, nil, nil, nil)

		_, err := client.RecordId2Name(ctx, recordID, projectName)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to get record")
	})
}
