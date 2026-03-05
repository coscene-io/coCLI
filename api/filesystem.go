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
	"strings"

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/constants"
)

type FileSystemInterface interface {
	ListAllFileSystems(ctx context.Context) ([]*openv1alpha1resource.FileSystem, error)
}

type fileSystemClient struct {
	fileSystemServiceClient openv1alpha1connect.FileSystemServiceClient
}

func NewFileSystemClient(fileSystemServiceClient openv1alpha1connect.FileSystemServiceClient) FileSystemInterface {
	return &fileSystemClient{fileSystemServiceClient: fileSystemServiceClient}
}

func (c *fileSystemClient) ListAllFileSystems(ctx context.Context) ([]*openv1alpha1resource.FileSystem, error) {
	var (
		pageToken string
		ret       []*openv1alpha1resource.FileSystem
	)

	for {
		req := connect.NewRequest(&openv1alpha1service.ListFileSystemsRequest{
			PageSize:  int32(constants.MaxPageSize),
			PageToken: pageToken,
		})
		res, err := c.fileSystemServiceClient.ListFileSystems(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to list file systems: %w", err)
		}

		ret = append(ret, res.Msg.FileSystems...)
		if res.Msg.NextPageToken == "" || len(res.Msg.FileSystems) < constants.MaxPageSize {
			break
		}
		pageToken = res.Msg.NextPageToken
	}

	return ret, nil
}

// FormatFileSystemLabel returns a human-readable label like "cn-hangzhou - Default [default]".
func FormatFileSystemLabel(fs *openv1alpha1resource.FileSystem) string {
	region := FormatRegion(fs.Region)
	name := fs.DisplayName
	if name == "" {
		idx := strings.LastIndex(fs.Name, "/fileSystems/")
		if idx >= 0 {
			name = fs.Name[idx+len("/fileSystems/"):]
		}
	}
	label := fmt.Sprintf("%s - %s", region, name)
	if fs.IsDefault {
		label += " [default]"
	}
	return label
}
