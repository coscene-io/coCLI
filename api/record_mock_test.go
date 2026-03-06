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

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/golang/mock/gomock"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockRecordServiceClient struct {
	openv1alpha1connect.RecordServiceClient
	ctrl *gomock.Controller

	getRecordFunc            func(context.Context, *connect.Request[openv1alpha1service.GetRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error)
	createRecordFunc         func(context.Context, *connect.Request[openv1alpha1service.CreateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error)
	copyRecordsFunc          func(context.Context, *connect.Request[openv1alpha1service.CopyRecordsRequest]) (*connect.Response[openv1alpha1service.CopyRecordsResponse], error)
	moveRecordsFunc          func(context.Context, *connect.Request[openv1alpha1service.MoveRecordsRequest]) (*connect.Response[openv1alpha1service.MoveRecordsResponse], error)
	deleteRecordFunc         func(context.Context, *connect.Request[openv1alpha1service.DeleteRecordRequest]) (*connect.Response[emptypb.Empty], error)
	updateRecordFunc         func(context.Context, *connect.Request[openv1alpha1service.UpdateRecordRequest]) (*connect.Response[openv1alpha1resource.Record], error)
	searchRecordsFunc        func(context.Context, *connect.Request[openv1alpha1service.SearchRecordsRequest]) (*connect.Response[openv1alpha1service.SearchRecordsResponse], error)
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

func (m *mockRecordServiceClient) SearchRecords(ctx context.Context, req *connect.Request[openv1alpha1service.SearchRecordsRequest]) (*connect.Response[openv1alpha1service.SearchRecordsResponse], error) {
	if m.searchRecordsFunc != nil {
		return m.searchRecordsFunc(ctx, req)
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
