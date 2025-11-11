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
	"fmt"
	"testing"

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

// mockFileServiceClientForFileTest is a mock implementation for file tests
type mockFileServiceClientForFileTest struct {
	openv1alpha1connect.FileServiceClient
	ctrl *gomock.Controller

	getFileFunc                 func(context.Context, *connect.Request[openv1alpha1service.GetFileRequest]) (*connect.Response[openv1alpha1resource.File], error)
	generateFileUploadUrlsFunc  func(context.Context, *connect.Request[openv1alpha1service.GenerateFileUploadUrlsRequest]) (*connect.Response[openv1alpha1service.GenerateFileUploadUrlsResponse], error)
	generateFileDownloadUrlFunc func(context.Context, *connect.Request[openv1alpha1service.GenerateFileDownloadURLRequest]) (*connect.Response[openv1alpha1service.GenerateFileDownloadURLResponse], error)
	deleteFileFunc              func(context.Context, *connect.Request[openv1alpha1service.DeleteFileRequest]) (*connect.Response[emptypb.Empty], error)
	batchDeleteFilesFunc        func(context.Context, *connect.Request[openv1alpha1service.BatchDeleteFilesRequest]) (*connect.Response[emptypb.Empty], error)
}

func (m *mockFileServiceClientForFileTest) GetFile(ctx context.Context, req *connect.Request[openv1alpha1service.GetFileRequest]) (*connect.Response[openv1alpha1resource.File], error) {
	if m.getFileFunc != nil {
		return m.getFileFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockFileServiceClientForFileTest) GenerateFileUploadUrls(ctx context.Context, req *connect.Request[openv1alpha1service.GenerateFileUploadUrlsRequest]) (*connect.Response[openv1alpha1service.GenerateFileUploadUrlsResponse], error) {
	if m.generateFileUploadUrlsFunc != nil {
		return m.generateFileUploadUrlsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockFileServiceClientForFileTest) GenerateFileDownloadURL(ctx context.Context, req *connect.Request[openv1alpha1service.GenerateFileDownloadURLRequest]) (*connect.Response[openv1alpha1service.GenerateFileDownloadURLResponse], error) {
	if m.generateFileDownloadUrlFunc != nil {
		return m.generateFileDownloadUrlFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockFileServiceClientForFileTest) DeleteFile(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteFileRequest]) (*connect.Response[emptypb.Empty], error) {
	if m.deleteFileFunc != nil {
		return m.deleteFileFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockFileServiceClientForFileTest) BatchDeleteFiles(ctx context.Context, req *connect.Request[openv1alpha1service.BatchDeleteFilesRequest]) (*connect.Response[emptypb.Empty], error) {
	if m.batchDeleteFilesFunc != nil {
		return m.batchDeleteFilesFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestFileClient_GetFile(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileName := name.File{
		ProjectID: "test-project",
		RecordID:  "test-record",
		Filename:  "test.txt",
	}
	expectedFile := testutil.NewFileBuilder().
		WithName(fileName.String()).
		WithFilename("test.txt").
		WithSize(1024).
		Build()

	mockFileService := &mockFileServiceClientForFileTest{
		ctrl: ctrl,
		getFileFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetFileRequest]) (*connect.Response[openv1alpha1resource.File], error) {
			assert.Equal(t, fileName.String(), req.Msg.Name)
			return connect.NewResponse(expectedFile), nil
		},
	}

	client := NewFileClient(mockFileService)

	file, err := client.GetFile(ctx, fileName.String())
	require.NoError(t, err)
	assert.Equal(t, expectedFile, file)
}

func TestFileClient_GenerateFileUploadUrls(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := "projects/test-project/records/test-record"
	files := []*openv1alpha1resource.File{
		testutil.NewFileBuilder().WithFilename("file1.txt").Build(),
		testutil.NewFileBuilder().WithFilename("file2.txt").Build(),
	}

	expectedUrls := map[string]string{
		files[0].Name: "https://storage.example.com/upload/file1",
		files[1].Name: "https://storage.example.com/upload/file2",
	}

	mockFileService := &mockFileServiceClientForFileTest{
		ctrl: ctrl,
		generateFileUploadUrlsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GenerateFileUploadUrlsRequest]) (*connect.Response[openv1alpha1service.GenerateFileUploadUrlsResponse], error) {
			assert.Equal(t, parent, req.Msg.Parent)
			assert.Len(t, req.Msg.Files, 2)

			return connect.NewResponse(&openv1alpha1service.GenerateFileUploadUrlsResponse{
				PreSignedUrls: expectedUrls,
			}), nil
		},
	}

	client := NewFileClient(mockFileService)

	urls, err := client.GenerateFileUploadUrls(ctx, parent, files)
	require.NoError(t, err)
	assert.Equal(t, expectedUrls, urls)
}

func TestFileClient_GenerateFileDownloadUrl(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileResourceName := "projects/test-project/records/test-record/files/test.txt"
	expectedUrl := "https://storage.example.com/download/test.txt?token=abc123"

	mockFileService := &mockFileServiceClientForFileTest{
		ctrl: ctrl,
		generateFileDownloadUrlFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GenerateFileDownloadURLRequest]) (*connect.Response[openv1alpha1service.GenerateFileDownloadURLResponse], error) {
			assert.Equal(t, fileResourceName, req.Msg.File.Name)
			return connect.NewResponse(&openv1alpha1service.GenerateFileDownloadURLResponse{
				PreSignedUrl: expectedUrl,
			}), nil
		},
	}

	client := NewFileClient(mockFileService)

	url, err := client.GenerateFileDownloadUrl(ctx, fileResourceName)
	require.NoError(t, err)
	assert.Equal(t, expectedUrl, url)
}

func TestFileClient_DeleteFile(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileResourceName := "projects/test-project/records/test-record/files/test.txt"

	mockFileService := &mockFileServiceClientForFileTest{
		ctrl: ctrl,
		deleteFileFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteFileRequest]) (*connect.Response[emptypb.Empty], error) {
			assert.Equal(t, fileResourceName, req.Msg.Name)
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
	}

	client := NewFileClient(mockFileService)

	err := client.DeleteFile(ctx, fileResourceName)
	require.NoError(t, err)
}

func TestFileClient_BatchDeleteFiles(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := "projects/test-project/records/test-record"
	names := []string{
		"projects/test-project/records/test-record/files/file1.txt",
		"projects/test-project/records/test-record/files/file2.txt",
		"projects/test-project/records/test-record/files/file3.txt",
	}

	mockFileService := &mockFileServiceClientForFileTest{
		ctrl: ctrl,
		batchDeleteFilesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.BatchDeleteFilesRequest]) (*connect.Response[emptypb.Empty], error) {
			assert.Equal(t, parent, req.Msg.Parent)
			assert.Equal(t, names, req.Msg.Names)
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
	}

	client := NewFileClient(mockFileService)

	err := client.BatchDeleteFiles(ctx, parent, names)
	require.NoError(t, err)
}

func TestFileClient_ErrorHandling(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("GetFile not found", func(t *testing.T) {
		mockFileService := &mockFileServiceClientForFileTest{
			ctrl: ctrl,
			getFileFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetFileRequest]) (*connect.Response[openv1alpha1resource.File], error) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("file not found"))
			},
		}

		client := NewFileClient(mockFileService)

		_, err := client.GetFile(ctx, "non-existent-file")
		require.Error(t, err)
		assert.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
	})

	t.Run("GenerateFileUploadUrls permission denied", func(t *testing.T) {
		mockFileService := &mockFileServiceClientForFileTest{
			ctrl: ctrl,
			generateFileUploadUrlsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GenerateFileUploadUrlsRequest]) (*connect.Response[openv1alpha1service.GenerateFileUploadUrlsResponse], error) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
			},
		}

		client := NewFileClient(mockFileService)

		_, err := client.GenerateFileUploadUrls(ctx, "parent", nil)
		require.Error(t, err)
		assert.Equal(t, connect.CodeOf(err), connect.CodePermissionDenied)
	})
}
