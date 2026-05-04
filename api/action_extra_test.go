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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionClientGetAndList(t *testing.T) {
	actionName := &name.Action{ProjectID: "p1", ID: "a1"}
	mock := &mockActionServiceClient{
		getActionFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetActionRequest]) (*connect.Response[openv1alpha1resource.Action], error) {
			assert.Equal(t, "projects/p1/actions/a1", req.Msg.Name)
			return connect.NewResponse(&openv1alpha1resource.Action{Name: req.Msg.Name}), nil
		},
		listActionsFunc: func() func(context.Context, *connect.Request[openv1alpha1service.ListActionsRequest]) (*connect.Response[openv1alpha1service.ListActionsResponse], error) {
			calls := 0
			return func(ctx context.Context, req *connect.Request[openv1alpha1service.ListActionsRequest]) (*connect.Response[openv1alpha1service.ListActionsResponse], error) {
				assert.Equal(t, "projects/p1", req.Msg.Parent)
				assert.Equal(t, int32(constants.MaxPageSize), req.Msg.PageSize)
				if calls == 0 {
					calls++
					actions := make([]*openv1alpha1resource.Action, constants.MaxPageSize)
					for i := range actions {
						actions[i] = &openv1alpha1resource.Action{Name: "first-page"}
					}
					return connect.NewResponse(&openv1alpha1service.ListActionsResponse{Actions: actions}), nil
				}
				assert.Equal(t, int32(constants.MaxPageSize), req.Msg.Skip)
				return connect.NewResponse(&openv1alpha1service.ListActionsResponse{
					Actions: []*openv1alpha1resource.Action{{Name: "last-page"}},
				}), nil
			}
		}(),
	}
	client := NewActionClient(mock, nil)

	action, err := client.GetByName(context.Background(), actionName)
	require.NoError(t, err)
	assert.Equal(t, "projects/p1/actions/a1", action.Name)

	actions, err := client.ListAllActions(context.Background(), &ListActionsOptions{Parent: "projects/p1"})
	require.NoError(t, err)
	assert.Len(t, actions, constants.MaxPageSize+1)
	assert.Equal(t, "last-page", actions[len(actions)-1].Name)
}

func TestActionClientCreateAndListRuns(t *testing.T) {
	recordName := &name.Record{ProjectID: "p1", RecordID: "r1"}
	action := &openv1alpha1resource.Action{Name: "projects/p1/actions/a1"}
	mock := &mockActionRunServiceClient{
		createActionRunFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateActionRunRequest]) (*connect.Response[openv1alpha1resource.ActionRun], error) {
			assert.Equal(t, "projects/p1", req.Msg.Parent)
			assert.Equal(t, action, req.Msg.ActionRun.Action)
			assert.Equal(t, []string{"projects/p1/records/r1"}, req.Msg.ActionRun.Match.Records)
			return connect.NewResponse(&openv1alpha1resource.ActionRun{}), nil
		},
		listActionRunsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListActionRunsRequest]) (*connect.Response[openv1alpha1service.ListActionRunsResponse], error) {
			assert.Equal(t, "projects/p1", req.Msg.Parent)
			assert.Equal(t, `match.records==["projects/p1/records/r1"]`, req.Msg.Filter)
			return connect.NewResponse(&openv1alpha1service.ListActionRunsResponse{
				ActionRuns: []*openv1alpha1resource.ActionRun{{Name: "projects/p1/actionRuns/run1"}},
			}), nil
		},
	}
	client := NewActionClient(nil, mock)

	require.NoError(t, client.CreateActionRun(context.Background(), action, recordName))

	runs, err := client.ListAllActionRuns(context.Background(), &ListActionRunsOptions{
		Parent:      "projects/p1",
		RecordNames: []*name.Record{recordName},
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, "projects/p1/actionRuns/run1", runs[0].Name)
}

func TestActionClientActionId2Name(t *testing.T) {
	uuid := "d9b9d56b-0d43-4719-b7cc-0d7e6616bb8a"
	mock := &mockActionServiceClient{
		getActionFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetActionRequest]) (*connect.Response[openv1alpha1resource.Action], error) {
			assert.Equal(t, "projects/p1/actions/"+uuid, req.Msg.Name)
			return connect.NewResponse(&openv1alpha1resource.Action{Name: req.Msg.Name}), nil
		},
	}
	client := NewActionClient(mock, nil)
	proj := &name.Project{ProjectID: "p1"}

	byID, err := client.ActionId2Name(context.Background(), uuid, proj)
	require.NoError(t, err)
	assert.Equal(t, "projects/p1/actions/"+uuid, byID.String())

	byName, err := client.ActionId2Name(context.Background(), "projects/p2/actions/a2", proj)
	require.NoError(t, err)
	assert.Equal(t, "projects/p2/actions/a2", byName.String())

	_, err = client.ActionId2Name(context.Background(), "not-a-uuid", proj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action id or name")
}
