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

type mockActionServiceClient struct {
	openv1alpha1connect.ActionServiceClient
	ctrl *gomock.Controller

	getActionFunc   func(context.Context, *connect.Request[openv1alpha1service.GetActionRequest]) (*connect.Response[openv1alpha1resource.Action], error)
	listActionsFunc func(context.Context, *connect.Request[openv1alpha1service.ListActionsRequest]) (*connect.Response[openv1alpha1service.ListActionsResponse], error)
}

func (m *mockActionServiceClient) GetAction(ctx context.Context, req *connect.Request[openv1alpha1service.GetActionRequest]) (*connect.Response[openv1alpha1resource.Action], error) {
	if m.getActionFunc != nil {
		return m.getActionFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockActionServiceClient) ListActions(ctx context.Context, req *connect.Request[openv1alpha1service.ListActionsRequest]) (*connect.Response[openv1alpha1service.ListActionsResponse], error) {
	if m.listActionsFunc != nil {
		return m.listActionsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

type mockActionRunServiceClient struct {
	openv1alpha1connect.ActionRunServiceClient
	ctrl *gomock.Controller

	createActionRunFunc func(context.Context, *connect.Request[openv1alpha1service.CreateActionRunRequest]) (*connect.Response[openv1alpha1resource.ActionRun], error)
	listActionRunsFunc  func(context.Context, *connect.Request[openv1alpha1service.ListActionRunsRequest]) (*connect.Response[openv1alpha1service.ListActionRunsResponse], error)
}

func (m *mockActionRunServiceClient) CreateActionRun(ctx context.Context, req *connect.Request[openv1alpha1service.CreateActionRunRequest]) (*connect.Response[openv1alpha1resource.ActionRun], error) {
	if m.createActionRunFunc != nil {
		return m.createActionRunFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockActionRunServiceClient) ListActionRuns(ctx context.Context, req *connect.Request[openv1alpha1service.ListActionRunsRequest]) (*connect.Response[openv1alpha1service.ListActionRunsResponse], error) {
	if m.listActionRunsFunc != nil {
		return m.listActionRunsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestActionClient_GetByName(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expected := &openv1alpha1resource.Action{Name: "projects/p1/actions/a1"}
	mock := &mockActionServiceClient{
		ctrl: ctrl,
		getActionFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetActionRequest]) (*connect.Response[openv1alpha1resource.Action], error) {
			return connect.NewResponse(expected), nil
		},
	}
	client := NewActionClient(mock, nil)
	action, err := client.GetByName(ctx, &name.Action{ProjectID: "p1", ID: "a1"})
	require.NoError(t, err)
	assert.Equal(t, expected, action)
}

func TestActionClient_ListAllActions(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("empty", func(t *testing.T) {
		mock := &mockActionServiceClient{
			ctrl: ctrl,
			listActionsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListActionsRequest]) (*connect.Response[openv1alpha1service.ListActionsResponse], error) {
				return connect.NewResponse(&openv1alpha1service.ListActionsResponse{}), nil
			},
		}
		client := NewActionClient(mock, nil)
		actions, err := client.ListAllActions(ctx, &ListActionsOptions{Parent: "projects/p1"})
		require.NoError(t, err)
		assert.Empty(t, actions)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockActionServiceClient{
			ctrl: ctrl,
			listActionsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListActionsRequest]) (*connect.Response[openv1alpha1service.ListActionsResponse], error) {
				return nil, connect.NewError(connect.CodeInternal, nil)
			},
		}
		client := NewActionClient(mock, nil)
		_, err := client.ListAllActions(ctx, &ListActionsOptions{Parent: "projects/p1"})
		assert.Error(t, err)
	})
}

func TestActionClient_CreateActionRun(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("success", func(t *testing.T) {
		mockRun := &mockActionRunServiceClient{
			ctrl: ctrl,
			createActionRunFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateActionRunRequest]) (*connect.Response[openv1alpha1resource.ActionRun], error) {
				assert.Equal(t, "projects/p1", req.Msg.Parent)
				return connect.NewResponse(&openv1alpha1resource.ActionRun{}), nil
			},
		}
		client := NewActionClient(nil, mockRun)
		err := client.CreateActionRun(ctx, &openv1alpha1resource.Action{Name: "projects/p1/actions/a1"}, &name.Record{ProjectID: "p1", RecordID: "r1"})
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mockRun := &mockActionRunServiceClient{
			ctrl: ctrl,
			createActionRunFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateActionRunRequest]) (*connect.Response[openv1alpha1resource.ActionRun], error) {
				return nil, connect.NewError(connect.CodePermissionDenied, nil)
			},
		}
		client := NewActionClient(nil, mockRun)
		err := client.CreateActionRun(ctx, &openv1alpha1resource.Action{}, &name.Record{ProjectID: "p1", RecordID: "r1"})
		assert.Error(t, err)
	})
}

func TestActionClient_ListAllActionRuns(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRun := &mockActionRunServiceClient{
		ctrl: ctrl,
		listActionRunsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListActionRunsRequest]) (*connect.Response[openv1alpha1service.ListActionRunsResponse], error) {
			return connect.NewResponse(&openv1alpha1service.ListActionRunsResponse{}), nil
		},
	}
	client := NewActionClient(nil, mockRun)
	runs, err := client.ListAllActionRuns(ctx, &ListActionRunsOptions{Parent: "projects/p1"})
	require.NoError(t, err)
	assert.Empty(t, runs)
}

func TestActionClient_ActionId2Name(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	proj := &name.Project{ProjectID: "p1"}

	t.Run("valid action name", func(t *testing.T) {
		client := NewActionClient(nil, nil)
		an, err := client.ActionId2Name(ctx, "projects/p1/actions/a1", proj)
		require.NoError(t, err)
		assert.Equal(t, "p1", an.ProjectID)
		assert.Equal(t, "a1", an.ID)
	})

	t.Run("valid wftmpl name", func(t *testing.T) {
		client := NewActionClient(nil, nil)
		an, err := client.ActionId2Name(ctx, "wftmpls/tmpl-1", proj)
		require.NoError(t, err)
		assert.Equal(t, "tmpl-1", an.ID)
		assert.True(t, an.IsWftmpl())
	})

	t.Run("invalid non-uuid string", func(t *testing.T) {
		client := NewActionClient(nil, nil)
		_, err := client.ActionId2Name(ctx, "not-a-uuid-or-name", proj)
		assert.Error(t, err)
	})

	t.Run("uuid resolved as project action", func(t *testing.T) {
		mock := &mockActionServiceClient{
			ctrl: ctrl,
			getActionFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetActionRequest]) (*connect.Response[openv1alpha1resource.Action], error) {
				return connect.NewResponse(&openv1alpha1resource.Action{Name: "projects/p1/actions/d9b9d56b-0d43-4719-b7cc-0d7e6616bb8a"}), nil
			},
		}
		client := NewActionClient(mock, nil)
		an, err := client.ActionId2Name(ctx, "d9b9d56b-0d43-4719-b7cc-0d7e6616bb8a", proj)
		require.NoError(t, err)
		assert.Equal(t, "p1", an.ProjectID)
	})
}
