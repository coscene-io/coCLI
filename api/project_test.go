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

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProjectServiceClient struct {
	openv1alpha1connect.ProjectServiceClient
	ctrl *gomock.Controller

	getProjectFunc                 func(context.Context, *connect.Request[openv1alpha1service.GetProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error)
	listProjectsFunc               func(context.Context, *connect.Request[openv1alpha1service.ListProjectsRequest]) (*connect.Response[openv1alpha1service.ListProjectsResponse], error)
	createProjectFunc              func(context.Context, *connect.Request[openv1alpha1service.CreateProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error)
	createProjectUsingTemplateFunc func(context.Context, *connect.Request[openv1alpha1service.CreateProjectUsingTemplateRequest]) (*connect.Response[openv1alpha1resource.Project], error)
}

func (m *mockProjectServiceClient) GetProject(ctx context.Context, req *connect.Request[openv1alpha1service.GetProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
	if m.getProjectFunc != nil {
		return m.getProjectFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockProjectServiceClient) ListProjects(ctx context.Context, req *connect.Request[openv1alpha1service.ListProjectsRequest]) (*connect.Response[openv1alpha1service.ListProjectsResponse], error) {
	if m.listProjectsFunc != nil {
		return m.listProjectsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockProjectServiceClient) CreateProject(ctx context.Context, req *connect.Request[openv1alpha1service.CreateProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
	if m.createProjectFunc != nil {
		return m.createProjectFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockProjectServiceClient) CreateProjectUsingTemplate(ctx context.Context, req *connect.Request[openv1alpha1service.CreateProjectUsingTemplateRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
	if m.createProjectUsingTemplateFunc != nil {
		return m.createProjectUsingTemplateFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestProjectClient_Name(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("success", func(t *testing.T) {
		mock := &mockProjectServiceClient{
			ctrl: ctrl,
			getProjectFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
				assert.Equal(t, "projects/my-slug", req.Msg.Name)
				return connect.NewResponse(&openv1alpha1resource.Project{Name: "projects/real-uuid"}), nil
			},
		}
		client := NewProjectClient(mock, nil)
		proj, err := client.Name(ctx, "my-slug")
		require.NoError(t, err)
		assert.Equal(t, "real-uuid", proj.ProjectID)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockProjectServiceClient{
			ctrl: ctrl,
			getProjectFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
				return nil, connect.NewError(connect.CodeNotFound, nil)
			},
		}
		client := NewProjectClient(mock, nil)
		_, err := client.Name(ctx, "bad-slug")
		assert.Error(t, err)
	})
}

func TestProjectClient_Get(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expected := &openv1alpha1resource.Project{Name: "projects/p1", DisplayName: "Project 1"}
	mock := &mockProjectServiceClient{
		ctrl: ctrl,
		getProjectFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
			return connect.NewResponse(expected), nil
		},
	}
	client := NewProjectClient(mock, nil)
	proj, err := client.Get(ctx, &name.Project{ProjectID: "p1"})
	require.NoError(t, err)
	assert.Equal(t, expected, proj)
}

func TestProjectClient_ListAllUserProjects(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("empty", func(t *testing.T) {
		mock := &mockProjectServiceClient{
			ctrl: ctrl,
			listProjectsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListProjectsRequest]) (*connect.Response[openv1alpha1service.ListProjectsResponse], error) {
				return connect.NewResponse(&openv1alpha1service.ListProjectsResponse{}), nil
			},
		}
		client := NewProjectClient(mock, nil)
		projects, err := client.ListAllUserProjects(ctx, &ListProjectsOptions{})
		require.NoError(t, err)
		assert.Empty(t, projects)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockProjectServiceClient{
			ctrl: ctrl,
			listProjectsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListProjectsRequest]) (*connect.Response[openv1alpha1service.ListProjectsResponse], error) {
				return nil, connect.NewError(connect.CodeInternal, nil)
			},
		}
		client := NewProjectClient(mock, nil)
		_, err := client.ListAllUserProjects(ctx, &ListProjectsOptions{})
		assert.Error(t, err)
	})
}

func TestProjectClient_CreateProject(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("success", func(t *testing.T) {
		expected := &openv1alpha1resource.Project{Name: "projects/new-proj"}
		mock := &mockProjectServiceClient{
			ctrl: ctrl,
			createProjectFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateProjectRequest]) (*connect.Response[openv1alpha1resource.Project], error) {
				assert.Equal(t, "my-slug", req.Msg.Project.Slug)
				return connect.NewResponse(expected), nil
			},
		}
		client := NewProjectClient(mock, nil)
		proj, err := client.CreateProject(ctx, &CreateProjectOptions{Slug: "my-slug", DisplayName: "My Project"})
		require.NoError(t, err)
		assert.Equal(t, expected, proj)
	})

	t.Run("nil options", func(t *testing.T) {
		client := NewProjectClient(nil, nil)
		_, err := client.CreateProject(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("empty slug", func(t *testing.T) {
		client := NewProjectClient(nil, nil)
		_, err := client.CreateProject(ctx, &CreateProjectOptions{DisplayName: "name"})
		assert.Error(t, err)
	})

	t.Run("empty display name", func(t *testing.T) {
		client := NewProjectClient(nil, nil)
		_, err := client.CreateProject(ctx, &CreateProjectOptions{Slug: "slug"})
		assert.Error(t, err)
	})
}

func TestProjectClient_ListProjectsWithPagination(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expected := []*openv1alpha1resource.Project{{Name: "projects/p1"}}
	mock := &mockProjectServiceClient{
		ctrl: ctrl,
		listProjectsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListProjectsRequest]) (*connect.Response[openv1alpha1service.ListProjectsResponse], error) {
			assert.Equal(t, int32(10), req.Msg.PageSize)
			assert.Equal(t, int32(5), req.Msg.Skip)
			return connect.NewResponse(&openv1alpha1service.ListProjectsResponse{Projects: expected}), nil
		},
	}
	client := NewProjectClient(mock, nil)
	projects, err := client.ListProjectsWithPagination(ctx, 10, 5, nil)
	require.NoError(t, err)
	assert.Equal(t, expected, projects)
}

func TestProjectClient_ListAllFiles(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	proj := &name.Project{ProjectID: "p1"}
	expected := []*openv1alpha1resource.File{{Filename: "readme.md"}}

	mockFile := &mockFileServiceClient{
		ctrl: ctrl,
		listFilesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListFilesRequest]) (*connect.Response[openv1alpha1service.ListFilesResponse], error) {
			assert.Equal(t, proj.String(), req.Msg.Parent)
			return connect.NewResponse(&openv1alpha1service.ListFilesResponse{Files: expected, TotalSize: 1}), nil
		},
	}
	client := NewProjectClient(nil, mockFile)
	files, err := client.ListAllFiles(ctx, proj)
	require.NoError(t, err)
	assert.Equal(t, expected, files)
}
