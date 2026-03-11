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

package api

import (
	"context"
	"testing"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

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

func TestRecordClient_SearchAll(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	project := &name.Project{ProjectID: "test-project"}
	options := &SearchRecordsOptions{
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
		searchRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.SearchRecordsRequest]) (*connect.Response[openv1alpha1service.SearchRecordsResponse], error) {
			assert.Equal(t, project.String(), req.Msg.Parent)

			callCount++
			if callCount == 1 {
				return connect.NewResponse(&openv1alpha1service.SearchRecordsResponse{
					Records:   []*openv1alpha1resource.Record{record1, record2},
					TotalSize: 2,
				}), nil
			}
			return connect.NewResponse(&openv1alpha1service.SearchRecordsResponse{
				Records: []*openv1alpha1resource.Record{},
			}), nil
		},
	}

	client := NewRecordClient(mockRecordService, nil, nil, mockLabelService)

	records, err := client.SearchAll(ctx, options)
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

func TestRecordClient_Get_ErrorCodePropagation(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recordName := &name.Record{
		ProjectID: "test-project",
		RecordID:  "test-record",
	}

	codes := []connect.Code{
		connect.CodeNotFound,
		connect.CodeInvalidArgument,
		connect.CodePermissionDenied,
		connect.CodeUnauthenticated,
		connect.CodeUnavailable,
		connect.CodeInternal,
		connect.CodeResourceExhausted,
		connect.CodeAlreadyExists,
		connect.CodeFailedPrecondition,
	}

	for _, code := range codes {
		t.Run(code.String(), func(t *testing.T) {
			mockRecordService := &mockRecordServiceClient{
				ctrl: ctrl,
				getRecordFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error) {
					return nil, connect.NewError(code, nil)
				},
			}

			client := NewRecordClient(mockRecordService, nil, nil, nil)

			_, err := client.Get(ctx, recordName)
			require.Error(t, err)
			assert.Equal(t, code, connect.CodeOf(err), "error code should be preserved through API client layer")
		})
	}
}

func TestRecordClient_Copy(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	src := &name.Record{ProjectID: "p1", RecordID: "r1"}
	dst := &name.Project{ProjectID: "p2"}
	expected := testutil.NewRecordBuilder().WithName("projects/p2/records/r1-copy").Build()

	t.Run("success", func(t *testing.T) {
		mock := &mockRecordServiceClient{
			ctrl: ctrl,
			copyRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CopyRecordsRequest]) (*connect.Response[openv1alpha1service.CopyRecordsResponse], error) {
				return connect.NewResponse(&openv1alpha1service.CopyRecordsResponse{Records: []*openv1alpha1resource.Record{expected}}), nil
			},
		}
		client := NewRecordClient(mock, nil, nil, nil)
		rec, err := client.Copy(ctx, src, dst)
		require.NoError(t, err)
		assert.Equal(t, expected, rec)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockRecordServiceClient{
			ctrl: ctrl,
			copyRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CopyRecordsRequest]) (*connect.Response[openv1alpha1service.CopyRecordsResponse], error) {
				return nil, connect.NewError(connect.CodeNotFound, nil)
			},
		}
		client := NewRecordClient(mock, nil, nil, nil)
		_, err := client.Copy(ctx, src, dst)
		assert.Error(t, err)
	})
}

func TestRecordClient_Move(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	src := &name.Record{ProjectID: "p1", RecordID: "r1"}
	dst := &name.Project{ProjectID: "p2"}
	expected := testutil.NewRecordBuilder().WithName("projects/p2/records/r1").Build()

	mock := &mockRecordServiceClient{
		ctrl: ctrl,
		moveRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.MoveRecordsRequest]) (*connect.Response[openv1alpha1service.MoveRecordsResponse], error) {
			return connect.NewResponse(&openv1alpha1service.MoveRecordsResponse{Records: []*openv1alpha1resource.Record{expected}}), nil
		},
	}
	client := NewRecordClient(mock, nil, nil, nil)
	rec, err := client.Move(ctx, src, dst)
	require.NoError(t, err)
	assert.Equal(t, expected, rec)
}

func TestRecordClient_DeleteFile(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rec := &name.Record{ProjectID: "p1", RecordID: "r1"}

	t.Run("success", func(t *testing.T) {
		mockFile := &mockFileServiceClient{
			ctrl: ctrl,
			deleteFileFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteFileRequest]) (*connect.Response[emptypb.Empty], error) {
				assert.Equal(t, "projects/p1/records/r1/files/data.bin", req.Msg.Name)
				return connect.NewResponse(&emptypb.Empty{}), nil
			},
		}
		client := NewRecordClient(nil, mockFile, nil, nil)
		err := client.DeleteFile(ctx, rec, "data.bin")
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mockFile := &mockFileServiceClient{
			ctrl: ctrl,
			deleteFileFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteFileRequest]) (*connect.Response[emptypb.Empty], error) {
				return nil, connect.NewError(connect.CodeNotFound, nil)
			},
		}
		client := NewRecordClient(nil, mockFile, nil, nil)
		err := client.DeleteFile(ctx, rec, "data.bin")
		assert.Error(t, err)
	})
}

func TestRecordClient_ListAllEvents(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rec := &name.Record{ProjectID: "p1", RecordID: "r1"}

	t.Run("empty", func(t *testing.T) {
		mock := &mockRecordServiceClient{
			ctrl: ctrl,
			listRecordEventsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListRecordEventsRequest]) (*connect.Response[openv1alpha1service.ListRecordEventsResponse], error) {
				return connect.NewResponse(&openv1alpha1service.ListRecordEventsResponse{}), nil
			},
		}
		client := NewRecordClient(mock, nil, nil, nil)
		events, err := client.ListAllEvents(ctx, rec)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockRecordServiceClient{
			ctrl: ctrl,
			listRecordEventsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListRecordEventsRequest]) (*connect.Response[openv1alpha1service.ListRecordEventsResponse], error) {
				return nil, connect.NewError(connect.CodeInternal, nil)
			},
		}
		client := NewRecordClient(mock, nil, nil, nil)
		_, err := client.ListAllEvents(ctx, rec)
		assert.Error(t, err)
	})
}

func TestRecordClient_GenerateRecordThumbnailUploadUrl(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rec := &name.Record{ProjectID: "p1", RecordID: "r1"}

	mock := &mockRecordServiceClient{
		ctrl: ctrl,
		generateThumbnailUrlFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GenerateRecordThumbnailUploadUrlRequest]) (*connect.Response[openv1alpha1service.GenerateRecordThumbnailUploadUrlResponse], error) {
			return connect.NewResponse(&openv1alpha1service.GenerateRecordThumbnailUploadUrlResponse{PreSignedUri: "https://example.com/upload"}), nil
		},
	}
	client := NewRecordClient(mock, nil, nil, nil)
	url, err := client.GenerateRecordThumbnailUploadUrl(ctx, rec)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/upload", url)
}

func TestRecordClient_SearchWithPageToken(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("success", func(t *testing.T) {
		expected := testutil.NewRecordBuilder().Build()
		mock := &mockRecordServiceClient{
			ctrl: ctrl,
			searchRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.SearchRecordsRequest]) (*connect.Response[openv1alpha1service.SearchRecordsResponse], error) {
				return connect.NewResponse(&openv1alpha1service.SearchRecordsResponse{
					Records:       []*openv1alpha1resource.Record{expected},
					NextPageToken: "next",
					TotalSize:     1,
				}), nil
			},
		}
		client := NewRecordClient(mock, nil, nil, nil)
		result, err := client.SearchWithPageToken(ctx, &SearchRecordsOptions{
			Project:  &name.Project{ProjectID: "p1"},
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Len(t, result.Records, 1)
		assert.Equal(t, "next", result.NextPageToken)
	})

	t.Run("empty project", func(t *testing.T) {
		client := NewRecordClient(nil, nil, nil, nil)
		_, err := client.SearchWithPageToken(ctx, &SearchRecordsOptions{
			Project: &name.Project{},
		})
		assert.Error(t, err)
	})
}

func TestRecordClient_MoveFiles(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	src := &name.Record{ProjectID: "p1", RecordID: "r1"}
	dst := &name.Record{ProjectID: "p1", RecordID: "r2"}
	files := []*openv1alpha1resource.File{
		{Filename: "a.bin"},
		{Filename: "b.bin"},
	}

	mockFile := &mockFileServiceClient{
		ctrl: ctrl,
		moveFilesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.MoveFilesRequest]) (*connect.Response[openv1alpha1service.MoveFilesResponse], error) {
			assert.Equal(t, src.String(), req.Msg.Parent)
			assert.Equal(t, dst.String(), req.Msg.Destination)
			return connect.NewResponse(&openv1alpha1service.MoveFilesResponse{}), nil
		},
	}
	client := NewRecordClient(nil, mockFile, nil, nil)
	err := client.MoveFiles(ctx, src, dst, files)
	assert.NoError(t, err)
}

func TestRecordClient_SearchWithAdvancedFilter(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	project := &name.Project{ProjectID: "test-project"}

	t.Run("search JSON uses advanced_filter", func(t *testing.T) {
		searchJSON := `{"and":[{"==":[{"var":"isArchived"},false]},{">":[{"var":"create_time"},"2024-01-01T00:00:00Z"]}]}`
		mock := &mockRecordServiceClient{
			ctrl: ctrl,
			searchRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.SearchRecordsRequest]) (*connect.Response[openv1alpha1service.SearchRecordsResponse], error) {
				f := req.Msg.GetQueryFilter()
				af, ok := f.(*openv1alpha1service.SearchRecordsRequest_AdvancedFilter)
				require.True(t, ok, "expected AdvancedFilter, got %T", f)
				assert.NotNil(t, af.AdvancedFilter)
				assert.Contains(t, af.AdvancedFilter.Fields, "and")
				return connect.NewResponse(&openv1alpha1service.SearchRecordsResponse{}), nil
			},
		}
		client := NewRecordClient(mock, nil, nil, nil)
		_, err := client.SearchWithPageToken(ctx, &SearchRecordsOptions{
			Project:  project,
			Search:   searchJSON,
			PageSize: 10,
		})
		require.NoError(t, err)
	})

	t.Run("invalid search JSON returns error", func(t *testing.T) {
		client := NewRecordClient(nil, nil, nil, nil)
		_, err := client.SearchWithPageToken(ctx, &SearchRecordsOptions{
			Project:  project,
			Search:   `{not valid json`,
			PageSize: 10,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid search JSON")
	})

	t.Run("no search uses AIP-160 filter", func(t *testing.T) {
		mock := &mockRecordServiceClient{
			ctrl: ctrl,
			searchRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.SearchRecordsRequest]) (*connect.Response[openv1alpha1service.SearchRecordsResponse], error) {
				f := req.Msg.GetQueryFilter()
				filterReq, ok := f.(*openv1alpha1service.SearchRecordsRequest_Filter)
				require.True(t, ok, "expected Filter, got %T", f)
				assert.Equal(t, "isArchived = false", filterReq.Filter)
				return connect.NewResponse(&openv1alpha1service.SearchRecordsResponse{}), nil
			},
		}
		client := NewRecordClient(mock, nil, nil, nil)
		_, err := client.SearchWithPageToken(ctx, &SearchRecordsOptions{
			Project:  project,
			PageSize: 10,
		})
		require.NoError(t, err)
	})
}

func TestRecordClient_Search_ErrorCodePropagation(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	codes := []connect.Code{
		connect.CodeNotFound,
		connect.CodePermissionDenied,
		connect.CodeInternal,
		connect.CodeUnavailable,
	}

	for _, code := range codes {
		t.Run(code.String(), func(t *testing.T) {
			mockRecordService := &mockRecordServiceClient{
				ctrl: ctrl,
				searchRecordsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.SearchRecordsRequest]) (*connect.Response[openv1alpha1service.SearchRecordsResponse], error) {
					return nil, connect.NewError(code, nil)
				},
			}

			client := NewRecordClient(mockRecordService, nil, nil, nil)

			_, err := client.SearchAll(ctx, &SearchRecordsOptions{
				Project: &name.Project{ProjectID: "test-project"},
			})
			require.Error(t, err)
			assert.Equal(t, code, connect.CodeOf(err), "error code should be preserved through API client layer")
		})
	}
}
