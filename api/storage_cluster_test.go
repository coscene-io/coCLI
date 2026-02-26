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
	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStorageServiceClient struct {
	openv1alpha1connect.StorageServiceClient
	ctrl *gomock.Controller

	listFileSystemsFunc func(context.Context, *connect.Request[openv1alpha1service.ListFileSystemsRequest]) (*connect.Response[openv1alpha1service.ListFileSystemsResponse], error)
}

func (m *mockStorageServiceClient) ListFileSystems(ctx context.Context, req *connect.Request[openv1alpha1service.ListFileSystemsRequest]) (*connect.Response[openv1alpha1service.ListFileSystemsResponse], error) {
	if m.listFileSystemsFunc != nil {
		return m.listFileSystemsFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestStorageClient_ListAllFileSystems(t *testing.T) {
	ctx := testutil.TestContext(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("success with region", func(t *testing.T) {
		expected := []*openv1alpha1resource.FileSystem{
			{
				Name:        "fileSystems/default",
				DisplayName: "Default",
				IsDefault:   true,
				Region:      openv1alpha1enums.RegionEnum_CN_HANGZHOU,
			},
		}
		mock := &mockStorageServiceClient{
			ctrl: ctrl,
			listFileSystemsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListFileSystemsRequest]) (*connect.Response[openv1alpha1service.ListFileSystemsResponse], error) {
				return connect.NewResponse(&openv1alpha1service.ListFileSystemsResponse{
					FileSystems: expected,
				}), nil
			},
		}
		client := NewStorageClient(mock)
		fileSystems, err := client.ListAllFileSystems(ctx)
		require.NoError(t, err)
		assert.Len(t, fileSystems, 1)
		assert.Equal(t, "fileSystems/default", fileSystems[0].Name)
		assert.Equal(t, openv1alpha1enums.RegionEnum_CN_HANGZHOU, fileSystems[0].Region)
	})

	t.Run("empty", func(t *testing.T) {
		mock := &mockStorageServiceClient{
			ctrl: ctrl,
			listFileSystemsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListFileSystemsRequest]) (*connect.Response[openv1alpha1service.ListFileSystemsResponse], error) {
				return connect.NewResponse(&openv1alpha1service.ListFileSystemsResponse{}), nil
			},
		}
		client := NewStorageClient(mock)
		fileSystems, err := client.ListAllFileSystems(ctx)
		require.NoError(t, err)
		assert.Empty(t, fileSystems)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockStorageServiceClient{
			ctrl: ctrl,
			listFileSystemsFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListFileSystemsRequest]) (*connect.Response[openv1alpha1service.ListFileSystemsResponse], error) {
				return nil, connect.NewError(connect.CodeInternal, nil)
			},
		}
		client := NewStorageClient(mock)
		_, err := client.ListAllFileSystems(ctx)
		assert.Error(t, err)
	})
}

func TestFormatFileSystemLabel(t *testing.T) {
	fs := &openv1alpha1resource.FileSystem{
		Name:        "fileSystems/default",
		DisplayName: "Default",
		IsDefault:   true,
		Region:      openv1alpha1enums.RegionEnum_CN_HANGZHOU,
	}
	assert.Equal(t, "cn-hangzhou - Default [default]", FormatFileSystemLabel(fs))

	fs2 := &openv1alpha1resource.FileSystem{
		Name:   "fileSystems/custom",
		Region: openv1alpha1enums.RegionEnum_CN_SHANGHAI,
	}
	assert.Equal(t, "cn-shanghai - custom", FormatFileSystemLabel(fs2))
}
