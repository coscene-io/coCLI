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

    openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
    openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
    "connectrpc.com/connect"
)

// ContainerRegistryInterface defines APIs for container registry operations.
type ContainerRegistryInterface interface {
    // CreateBasicCredential creates a basic docker credential (username+password).
    CreateBasicCredential(ctx context.Context) (*openv1alpha1service.BasicCredential, error)
}

type containerRegistryClient struct {
    cli openv1alpha1connect.ContainerRegistryServiceClient
}

// NewContainerRegistryClient creates a new container registry client.
func NewContainerRegistryClient(cli openv1alpha1connect.ContainerRegistryServiceClient) ContainerRegistryInterface {
    return &containerRegistryClient{cli: cli}
}

func (c *containerRegistryClient) CreateBasicCredential(ctx context.Context) (*openv1alpha1service.BasicCredential, error) {
    req := connect.NewRequest(&openv1alpha1service.CreateBasicCredentialRequest{})
    resp, err := c.cli.CreateBasicCredential(ctx, req)
    if err != nil {
        return nil, err
    }
    return resp.Msg, nil
}

