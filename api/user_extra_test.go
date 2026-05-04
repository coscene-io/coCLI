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
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserServiceClient struct {
	openv1alpha1connect.UserServiceClient
	batchGetUsersFunc func(context.Context, *connect.Request[openv1alpha1service.BatchGetUsersRequest]) (*connect.Response[openv1alpha1service.BatchGetUsersResponse], error)
	listUsersFunc     func(context.Context, *connect.Request[openv1alpha1service.ListUsersRequest]) (*connect.Response[openv1alpha1service.ListUsersResponse], error)
	getUserFunc       func(context.Context, *connect.Request[openv1alpha1service.GetUserRequest]) (*connect.Response[openv1alpha1resource.User], error)
}

func (m *mockUserServiceClient) BatchGetUsers(ctx context.Context, req *connect.Request[openv1alpha1service.BatchGetUsersRequest]) (*connect.Response[openv1alpha1service.BatchGetUsersResponse], error) {
	if m.batchGetUsersFunc != nil {
		return m.batchGetUsersFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockUserServiceClient) ListUsers(ctx context.Context, req *connect.Request[openv1alpha1service.ListUsersRequest]) (*connect.Response[openv1alpha1service.ListUsersResponse], error) {
	if m.listUsersFunc != nil {
		return m.listUsersFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (m *mockUserServiceClient) GetUser(ctx context.Context, req *connect.Request[openv1alpha1service.GetUserRequest]) (*connect.Response[openv1alpha1resource.User], error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, req)
	}
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestUserClientBatchGetUsers(t *testing.T) {
	t.Run("empty set avoids request", func(t *testing.T) {
		client := NewUserClient(&mockUserServiceClient{})
		got, err := client.BatchGetUsers(context.Background(), mapset.NewSet[name.User]())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("maps response by resource name", func(t *testing.T) {
		nickname := "Alice"
		mock := &mockUserServiceClient{
			batchGetUsersFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.BatchGetUsersRequest]) (*connect.Response[openv1alpha1service.BatchGetUsersResponse], error) {
				assert.ElementsMatch(t, []string{"users/u1"}, req.Msg.Names)
				return connect.NewResponse(&openv1alpha1service.BatchGetUsersResponse{
					Users: []*openv1alpha1resource.User{{Name: "users/u1", Nickname: &nickname}},
				}), nil
			},
		}
		client := NewUserClient(mock)

		got, err := client.BatchGetUsers(context.Background(), mapset.NewSet(name.User{UserID: "u1"}))

		require.NoError(t, err)
		require.Contains(t, got, "users/u1")
		assert.Equal(t, "Alice", got["users/u1"].GetNickname())
	})
}

func TestUserClientListGetAndFind(t *testing.T) {
	mock := &mockUserServiceClient{
		listUsersFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListUsersRequest]) (*connect.Response[openv1alpha1service.ListUsersResponse], error) {
			assert.Equal(t, "organizations/current", req.Msg.Parent)
			assert.Equal(t, int32(20), req.Msg.PageSize)
			assert.Equal(t, "next", req.Msg.PageToken)
			assert.Equal(t, `role.code="admin"`, req.Msg.Filter)
			return connect.NewResponse(&openv1alpha1service.ListUsersResponse{
				Users:         []*openv1alpha1resource.User{{Name: "users/u1"}},
				NextPageToken: "after",
				TotalSize:     1,
			}), nil
		},
		getUserFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetUserRequest]) (*connect.Response[openv1alpha1resource.User], error) {
			assert.Equal(t, "users/u1", req.Msg.Name)
			return connect.NewResponse(&openv1alpha1resource.User{Name: "users/u1"}), nil
		},
	}
	client := NewUserClient(mock)

	list, err := client.ListUsers(context.Background(), &ListUsersOptions{
		Parent:    "organizations/current",
		PageSize:  20,
		PageToken: "next",
		RoleCode:  "admin",
	})
	require.NoError(t, err)
	assert.Equal(t, "after", list.NextPageToken)
	assert.Equal(t, int64(1), list.TotalSize)

	user, err := client.GetUser(context.Background(), "users/u1")
	require.NoError(t, err)
	assert.Equal(t, "users/u1", user.Name)
}

func TestUserClientFindUsersByNicknameKeepsExactMatches(t *testing.T) {
	alice := "Alice"
	aliceTeam := "Alice Team"
	mock := &mockUserServiceClient{
		listUsersFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListUsersRequest]) (*connect.Response[openv1alpha1service.ListUsersResponse], error) {
			assert.Equal(t, `nickname="Alice"`, req.Msg.Filter)
			return connect.NewResponse(&openv1alpha1service.ListUsersResponse{
				Users: []*openv1alpha1resource.User{
					{Name: "users/u1", Nickname: &alice},
					{Name: "users/u2", Nickname: &aliceTeam},
				},
			}), nil
		},
	}
	client := NewUserClient(mock)

	got, err := client.FindUsersByNickname(context.Background(), "Alice")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "users/u1", got[0].Name)
}
