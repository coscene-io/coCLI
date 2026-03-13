// Copyright 2024 coScene
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

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/samber/lo"
)

type ListUsersOptions struct {
	Parent    string
	PageSize  int32
	PageToken string
	RoleCode  string
	Filter    string
}

type ListUsersResult struct {
	Users         []*openv1alpha1resource.User
	NextPageToken string
	TotalSize     int64
}

type UserInterface interface {
	BatchGetUsers(ctx context.Context, userNameList mapset.Set[name.User]) (map[string]*openv1alpha1resource.User, error)
	ListUsers(ctx context.Context, opts *ListUsersOptions) (*ListUsersResult, error)
	GetUser(ctx context.Context, userName string) (*openv1alpha1resource.User, error)
	FindUsersByNickname(ctx context.Context, nickname string) ([]*openv1alpha1resource.User, error)
}

type userClient struct {
	userServiceClient openv1alpha1connect.UserServiceClient
}

func NewUserClient(userServiceClient openv1alpha1connect.UserServiceClient) UserInterface {
	return &userClient{
		userServiceClient: userServiceClient,
	}
}

func (c *userClient) BatchGetUsers(ctx context.Context, userNameSet mapset.Set[name.User]) (map[string]*openv1alpha1resource.User, error) {
	userNameList := userNameSet.ToSlice()
	if len(userNameList) == 0 {
		return map[string]*openv1alpha1resource.User{}, nil
	}
	req := connect.NewRequest(&openv1alpha1service.BatchGetUsersRequest{
		Names: lo.Map(userNameList, func(u name.User, _ int) string {
			return u.String()
		}),
	})
	res, err := c.userServiceClient.BatchGetUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	return lo.Associate(res.Msg.Users, func(u *openv1alpha1resource.User) (string, *openv1alpha1resource.User) {
		return u.Name, u
	}), nil
}

func (c *userClient) ListUsers(ctx context.Context, opts *ListUsersOptions) (*ListUsersResult, error) {
	filter := opts.Filter
	if filter == "" && opts.RoleCode != "" {
		filter = fmt.Sprintf("role.code=%q", opts.RoleCode)
	}

	req := connect.NewRequest(&openv1alpha1service.ListUsersRequest{
		Parent:    opts.Parent,
		PageSize:  opts.PageSize,
		PageToken: opts.PageToken,
		Filter:    filter,
	})

	res, err := c.userServiceClient.ListUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ListUsersResult{
		Users:         res.Msg.GetUsers(),
		NextPageToken: res.Msg.GetNextPageToken(),
		TotalSize:     res.Msg.GetTotalSize(),
	}, nil
}

func (c *userClient) GetUser(ctx context.Context, userName string) (*openv1alpha1resource.User, error) {
	req := connect.NewRequest(&openv1alpha1service.GetUserRequest{
		Name: userName,
	})

	res, err := c.userServiceClient.GetUser(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Msg, nil
}

func (c *userClient) FindUsersByNickname(ctx context.Context, nickname string) ([]*openv1alpha1resource.User, error) {
	result, err := c.ListUsers(ctx, &ListUsersOptions{
		Filter:   fmt.Sprintf("nickname=%q", nickname),
		PageSize: 100,
	})
	if err != nil {
		return nil, err
	}

	// Server does substring matching (LIKE %nickname%), so filter locally for exact match.
	var exact []*openv1alpha1resource.User
	for _, u := range result.Users {
		if u.GetNickname() == nickname {
			exact = append(exact, u)
		}
	}
	return exact, nil
}
