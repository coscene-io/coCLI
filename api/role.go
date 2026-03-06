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
	"fmt"

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
)

type ListRolesOptions struct {
	Level     string
	PageSize  int32
	PageToken string
}

type ListRolesResult struct {
	Roles         []*openv1alpha1resource.Role
	NextPageToken string
	TotalSize     int64
}

type RoleInterface interface {
	ListRoles(ctx context.Context, opts *ListRolesOptions) (*ListRolesResult, error)
}

type roleClient struct {
	roleServiceClient openv1alpha1connect.RoleServiceClient
}

func NewRoleClient(roleServiceClient openv1alpha1connect.RoleServiceClient) RoleInterface {
	return &roleClient{
		roleServiceClient: roleServiceClient,
	}
}

func (c *roleClient) ListRoles(ctx context.Context, opts *ListRolesOptions) (*ListRolesResult, error) {
	filter := ""
	if opts.Level != "" {
		filter = fmt.Sprintf("level=%q", opts.Level)
	}

	req := connect.NewRequest(&openv1alpha1service.ListRolesRequest{
		PageSize:  opts.PageSize,
		PageToken: opts.PageToken,
		Filter:    filter,
	})

	res, err := c.roleServiceClient.ListRoles(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ListRolesResult{
		Roles:         res.Msg.GetRoles(),
		NextPageToken: res.Msg.GetNextPageToken(),
		TotalSize:     res.Msg.GetTotalSize(),
	}, nil
}
