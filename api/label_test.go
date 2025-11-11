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
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLabelServiceClientForTest is a mock implementation for label tests
type mockLabelServiceClientForTest struct {
	openv1alpha1connect.LabelServiceClient
	ctrl *gomock.Controller

	listLabelsFunc  func(context.Context, *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error)
	createLabelFunc func(context.Context, *connect.Request[openv1alpha1service.CreateLabelRequest]) (*connect.Response[openv1alpha1resource.Label], error)
}

func (m *mockLabelServiceClientForTest) ListLabels(ctx context.Context, req *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error) {
	if m.listLabelsFunc != nil {
		return m.listLabelsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockLabelServiceClientForTest) CreateLabel(ctx context.Context, req *connect.Request[openv1alpha1service.CreateLabelRequest]) (*connect.Response[openv1alpha1resource.Label], error) {
	if m.createLabelFunc != nil {
		return m.createLabelFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestLabelClient_GetByDisplayNameOrCreate(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectName := &name.Project{ProjectID: "test-project"}
	displayName := "test-label"

	t.Run("Label already exists", func(t *testing.T) {
		existingLabel := &openv1alpha1resource.Label{
			Name:        "projects/test-project/labels/label-123",
			DisplayName: displayName,
		}

		mockLabelService := &mockLabelServiceClientForTest{
			ctrl: ctrl,
			listLabelsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error) {
				assert.Equal(t, projectName.String(), req.Msg.Parent)
				assert.Contains(t, req.Msg.Filter, displayName)
				return connect.NewResponse(&openv1alpha1service.ListLabelsResponse{
					Labels:    []*openv1alpha1resource.Label{existingLabel},
					TotalSize: 1,
				}), nil
			},
		}

		client := NewLabelClient(mockLabelService)

		label, err := client.GetByDisplayNameOrCreate(ctx, displayName, projectName)
		require.NoError(t, err)
		assert.Equal(t, existingLabel, label)
	})

	t.Run("Label needs to be created", func(t *testing.T) {
		newLabel := &openv1alpha1resource.Label{
			Name:        "projects/test-project/labels/new-label-456",
			DisplayName: displayName,
		}

		listCallCount := 0
		mockLabelService := &mockLabelServiceClientForTest{
			ctrl: ctrl,
			listLabelsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error) {
				listCallCount++
				assert.Equal(t, projectName.String(), req.Msg.Parent)

				if listCallCount == 1 {
					// First call: label doesn't exist
					return connect.NewResponse(&openv1alpha1service.ListLabelsResponse{
						Labels: []*openv1alpha1resource.Label{},
					}), nil
				} else {
					// Second call after creation: label exists
					return connect.NewResponse(&openv1alpha1service.ListLabelsResponse{
						Labels:    []*openv1alpha1resource.Label{newLabel},
						TotalSize: 1,
					}), nil
				}
			},
			createLabelFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateLabelRequest]) (*connect.Response[openv1alpha1resource.Label], error) {
				assert.Equal(t, projectName.String(), req.Msg.Parent)
				assert.Equal(t, displayName, req.Msg.Label.DisplayName)
				return connect.NewResponse(newLabel), nil
			},
		}

		client := NewLabelClient(mockLabelService)

		label, err := client.GetByDisplayNameOrCreate(ctx, displayName, projectName)
		require.NoError(t, err)
		assert.Equal(t, newLabel, label)
		assert.Equal(t, 1, listCallCount, "Should list labels once before creation")
	})

	t.Run("Create fails but label exists on retry", func(t *testing.T) {
		existingLabel := &openv1alpha1resource.Label{
			Name:        "projects/test-project/labels/label-789",
			DisplayName: displayName,
		}

		listCallCount := 0
		mockLabelService := &mockLabelServiceClientForTest{
			ctrl: ctrl,
			listLabelsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error) {
				listCallCount++
				if listCallCount == 1 {
					// First call: label doesn't exist
					return connect.NewResponse(&openv1alpha1service.ListLabelsResponse{
						Labels: []*openv1alpha1resource.Label{},
					}), nil
				} else {
					// Second call: label was created by another process
					return connect.NewResponse(&openv1alpha1service.ListLabelsResponse{
						Labels: []*openv1alpha1resource.Label{existingLabel},
					}), nil
				}
			},
			createLabelFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateLabelRequest]) (*connect.Response[openv1alpha1resource.Label], error) {
				// Creation fails (e.g., already exists due to race condition)
				return nil, connect.NewError(connect.CodeAlreadyExists, nil)
			},
		}

		client := NewLabelClient(mockLabelService)

		_, err := client.GetByDisplayNameOrCreate(ctx, displayName, projectName)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already_exists")
		assert.Contains(t, err.Error(), "create label test-label failed")
	})

	t.Run("Both list and create fail", func(t *testing.T) {
		mockLabelService := &mockLabelServiceClientForTest{
			ctrl: ctrl,
			listLabelsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListLabelsRequest]) (*connect.Response[openv1alpha1service.ListLabelsResponse], error) {
				// First list succeeds but finds nothing
				return connect.NewResponse(&openv1alpha1service.ListLabelsResponse{
					Labels:    []*openv1alpha1resource.Label{},
					TotalSize: 0,
				}), nil
			},
			createLabelFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateLabelRequest]) (*connect.Response[openv1alpha1resource.Label], error) {
				// Create fails with non-AlreadyExists error
				return nil, connect.NewError(connect.CodeInternal, nil)
			},
		}

		client := NewLabelClient(mockLabelService)

		_, err := client.GetByDisplayNameOrCreate(ctx, displayName, projectName)
		require.Error(t, err)
		assert.Equal(t, connect.CodeOf(err), connect.CodeInternal)
	})
}
