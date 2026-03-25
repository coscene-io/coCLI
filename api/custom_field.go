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

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
)

type CustomFieldInterface interface {
	GetRecordCustomFieldSchema(ctx context.Context, project *name.Project) (*commons.CustomFieldSchema, error)
}

type customFieldClient struct {
	customFieldServiceClient openv1alpha1connect.CustomFieldServiceClient
}

func NewCustomFieldClient(client openv1alpha1connect.CustomFieldServiceClient) CustomFieldInterface {
	return &customFieldClient{customFieldServiceClient: client}
}

func (c *customFieldClient) GetRecordCustomFieldSchema(ctx context.Context, project *name.Project) (*commons.CustomFieldSchema, error) {
	req := connect.NewRequest(&openv1alpha1service.GetRecordCustomFieldSchemaRequest{
		Project: project.String(),
	})
	res, err := c.customFieldServiceClient.GetRecordCustomFieldSchema(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}
