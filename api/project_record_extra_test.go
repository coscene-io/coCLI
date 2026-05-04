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
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProjectClientCreateProjectUsingTemplate(t *testing.T) {
	description := "from template"
	mock := &mockProjectServiceClient{
		createProjectUsingTemplateFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateProjectUsingTemplateRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
			assert.Equal(t, "robot-demo", req.Msg.Project.Slug)
			assert.Equal(t, "Robot Demo", req.Msg.Project.DisplayName)
			assert.Equal(t, enums.ProjectVisibilityEnum_PRIVATE, req.Msg.Project.Visibility)
			assert.Equal(t, description, req.Msg.Project.GetDescription())
			assert.Equal(t, "projects/template", req.Msg.ProjectTemplate)
			assert.Equal(t, []openv1alpha1service.CreateProjectUsingTemplateRequest_TemplateScope{
				openv1alpha1service.CreateProjectUsingTemplateRequest_CUSTOM_FIELDS,
				openv1alpha1service.CreateProjectUsingTemplateRequest_ACTIONS,
			}, req.Msg.TemplateScopes)
			return connect.NewResponse(&openv1alpha1resource.Project{Name: "projects/new"}), nil
		},
	}
	client := NewProjectClient(mock, nil)

	project, err := client.CreateProjectUsingTemplate(context.Background(), &CreateProjectUsingTemplateOptions{
		Slug:            "robot-demo",
		DisplayName:     "Robot Demo",
		Visibility:      enums.ProjectVisibilityEnum_PRIVATE,
		Description:     description,
		ProjectTemplate: "projects/template",
		TemplateScopes: []openv1alpha1service.CreateProjectUsingTemplateRequest_TemplateScope{
			openv1alpha1service.CreateProjectUsingTemplateRequest_CUSTOM_FIELDS,
			openv1alpha1service.CreateProjectUsingTemplateRequest_ACTIONS,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "projects/new", project.Name)
}

func TestProjectClientFileListingVariants(t *testing.T) {
	proj := &name.Project{ProjectID: "project-a"}
	file := &openv1alpha1resource.File{Filename: "data.bin"}
	mock := &mockFileServiceClient{
		listFilesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListFilesRequest]) (*connect.Response[openv1alpha1service.ListFilesResponse], error) {
			assert.Equal(t, proj.String(), req.Msg.Parent)
			switch req.Msg.Filter {
			case `filename.startsWith("logs/")`:
				assert.Equal(t, int32(constants.MaxPageSize), req.Msg.PageSize)
			case `filename.endsWith(".mcap")`:
				assert.Equal(t, int32(25), req.Msg.PageSize)
				assert.Equal(t, int32(50), req.Msg.Skip)
			default:
				assert.Empty(t, req.Msg.Filter)
				assert.Equal(t, int32(10), req.Msg.PageSize)
				assert.Equal(t, int32(20), req.Msg.Skip)
			}
			return connect.NewResponse(&openv1alpha1service.ListFilesResponse{Files: []*openv1alpha1resource.File{file}, TotalSize: 1}), nil
		},
	}
	client := NewProjectClient(nil, mock)

	all, err := client.ListAllFilesWithFilter(context.Background(), proj, `filename.startsWith("logs/")`)
	require.NoError(t, err)
	assert.Equal(t, []*openv1alpha1resource.File{file}, all)

	page, err := client.ListFilesWithPagination(context.Background(), proj, 10, 20)
	require.NoError(t, err)
	assert.Equal(t, []*openv1alpha1resource.File{file}, page)

	filteredPage, err := client.ListFilesWithPaginationAndFilter(context.Background(), proj, 25, 50, `filename.endsWith(".mcap")`)
	require.NoError(t, err)
	assert.Equal(t, []*openv1alpha1resource.File{file}, filteredPage)
}

func TestRecordClientFileListingVariants(t *testing.T) {
	recordName := &name.Record{ProjectID: "project-a", RecordID: "record-a"}
	file := &openv1alpha1resource.File{Filename: "data.bin"}
	mock := &mockFileServiceClient{
		listFilesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListFilesRequest]) (*connect.Response[openv1alpha1service.ListFilesResponse], error) {
			assert.Equal(t, recordName.String(), req.Msg.Parent)
			switch req.Msg.Filter {
			case `filename.startsWith("logs/")`:
				assert.Equal(t, int32(constants.MaxPageSize), req.Msg.PageSize)
			case `filename.endsWith(".mcap")`:
				assert.Equal(t, int32(25), req.Msg.PageSize)
				assert.Equal(t, int32(50), req.Msg.Skip)
			default:
				assert.Empty(t, req.Msg.Filter)
				assert.Equal(t, int32(10), req.Msg.PageSize)
				assert.Equal(t, int32(20), req.Msg.Skip)
			}
			return connect.NewResponse(&openv1alpha1service.ListFilesResponse{Files: []*openv1alpha1resource.File{file}, TotalSize: 1}), nil
		},
	}
	client := NewRecordClient(nil, mock, nil, nil)

	all, err := client.ListAllFilesWithFilter(context.Background(), recordName, `filename.startsWith("logs/")`)
	require.NoError(t, err)
	assert.Equal(t, []*openv1alpha1resource.File{file}, all)

	page, err := client.ListFilesWithPagination(context.Background(), recordName, 10, 20)
	require.NoError(t, err)
	assert.Equal(t, []*openv1alpha1resource.File{file}, page)

	filteredPage, err := client.ListFilesWithPaginationAndFilter(context.Background(), recordName, 25, 50, `filename.endsWith(".mcap")`)
	require.NoError(t, err)
	assert.Equal(t, []*openv1alpha1resource.File{file}, filteredPage)
}

func TestRecordClientListAllMoments(t *testing.T) {
	triggerTime := time.Date(2026, 5, 4, 1, 2, 3, 0, time.UTC)
	userNickname := "Alice"
	event := &openv1alpha1resource.Event{
		DisplayName:      "Hard brake",
		Description:      "Brake event",
		TriggerTime:      timestamppb.New(triggerTime),
		Duration:         durationpb.New(1500 * time.Millisecond),
		CustomizedFields: map[string]string{"source": "auto"},
		CustomFieldValues: []*commons.CustomFieldValue{
			{
				Property: &commons.Property{Name: "note", Type: &commons.Property_Text{Text: &commons.TextType{}}},
				Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "sharp"}},
			},
			{
				Property: &commons.Property{Name: "score", Type: &commons.Property_Number{Number: &commons.NumberType{}}},
				Value:    &commons.CustomFieldValue_Number{Number: &commons.NumberValue{Value: 7}},
			},
			{
				Property: &commons.Property{Name: "kind", Type: &commons.Property_Enums{Enums: &commons.EnumType{Values: map[string]string{"brake": "Brake"}}}},
				Value:    &commons.CustomFieldValue_Enums{Enums: &commons.EnumValue{Id: "brake"}},
			},
			{
				Property: &commons.Property{Name: "when", Type: &commons.Property_Time{Time: &commons.TimeType{}}},
				Value:    &commons.CustomFieldValue_Time{Time: &commons.TimeValue{Value: timestamppb.New(triggerTime)}},
			},
			{
				Property: &commons.Property{Name: "owner", Type: &commons.Property_User{User: &commons.UserType{}}},
				Value:    &commons.CustomFieldValue_User{User: &commons.UserValue{Ids: []string{"u1"}}},
			},
		},
	}
	eventCalls := 0
	recordSvc := &mockRecordServiceClient{
		listRecordEventsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListRecordEventsRequest]) (*connect.Response[openv1alpha1service.ListRecordEventsResponse], error) {
			if eventCalls > 0 {
				return connect.NewResponse(&openv1alpha1service.ListRecordEventsResponse{}), nil
			}
			eventCalls++
			return connect.NewResponse(&openv1alpha1service.ListRecordEventsResponse{Events: []*openv1alpha1resource.Event{event}}), nil
		},
	}
	userSvc := &mockUserServiceClient{
		batchGetUsersFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.BatchGetUsersRequest]) (*connect.Response[openv1alpha1service.BatchGetUsersResponse], error) {
			assert.Equal(t, []string{"users/u1"}, req.Msg.Names)
			return connect.NewResponse(&openv1alpha1service.BatchGetUsersResponse{
				Users: []*openv1alpha1resource.User{{Name: "users/u1", Nickname: &userNickname}},
			}), nil
		},
	}
	client := NewRecordClient(recordSvc, nil, userSvc, nil)

	moments, err := client.ListAllMoments(context.Background(), &name.Record{ProjectID: "project-a", RecordID: "record-a"})

	require.NoError(t, err)
	require.Len(t, moments, 1)
	assert.Equal(t, "Hard brake", moments[0].Name)
	assert.Equal(t, "Brake event", moments[0].Description)
	assert.Equal(t, "1.500000000s", moments[0].Duration)
	assert.Equal(t, map[string]string{"source": "auto"}, moments[0].Attribute)
	assert.Contains(t, moments[0].CustomFieldValues, map[string]any{"note": "sharp"})
	assert.Contains(t, moments[0].CustomFieldValues, map[string]any{"score": float64(7)})
	assert.Contains(t, moments[0].CustomFieldValues, map[string]any{"kind": "Brake"})
	assert.Contains(t, moments[0].CustomFieldValues, map[string]any{"owner": []string{"Alice"}})
}
