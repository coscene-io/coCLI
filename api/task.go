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
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/pkg/errors"
)

type TaskInterface interface {
	UpsertTask(ctx context.Context, parent string, task *openv1alpha1resource.Task) (*openv1alpha1resource.Task, error)
	SyncTask(ctx context.Context, taskName string) (*openv1alpha1resource.Task, error)
}

type taskClient struct {
	taskClient openv1alpha1connect.TaskServiceClient
}

func NewTaskClient(taskServiceClient openv1alpha1connect.TaskServiceClient) TaskInterface {
	return &taskClient{
		taskClient: taskServiceClient,
	}
}

func (c *taskClient) UpsertTask(ctx context.Context, parent string, task *openv1alpha1resource.Task) (*openv1alpha1resource.Task, error) {
	upsertTaskReq := connect.NewRequest(&openv1alpha1service.UpsertTaskRequest{
		Parent: parent,
		Task:   task,
	})
	createEventRes, err := c.taskClient.UpsertTask(ctx, upsertTaskReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to upsert task %s", task.GetTitle())
	}

	return createEventRes.Msg, nil
}

func (c *taskClient) SyncTask(ctx context.Context, taskName string) (*openv1alpha1resource.Task, error) {
	syncTaskReq := connect.NewRequest(&openv1alpha1service.SyncTaskRequest{
		Name: taskName,
	})
	syncTaskRes, err := c.taskClient.SyncTask(ctx, syncTaskReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sync task %s", taskName)
	}

	return syncTaskRes.Msg, nil
}
