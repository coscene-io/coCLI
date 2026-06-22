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
	"github.com/coscene-io/cocli/internal/name"
)

// listJobRunsPageSize is the page size for ListJobRuns. The JobRunService proto
// constrains page_size to [10, 100].
const listJobRunsPageSize = 100

type JobRunInterface interface {
	// ListJobRuns lists all job runs under an action run.
	ListJobRuns(ctx context.Context, actionRun *name.ActionRun) ([]*openv1alpha1resource.JobRun, error)

	// GetJobRun gets a single job run by resource name.
	GetJobRun(ctx context.Context, jobRunName string) (*openv1alpha1resource.JobRun, error)

	// GetJobRunDag gets the DAG (nodes) of a job run.
	GetJobRunDag(ctx context.Context, jobRunName string) (*openv1alpha1resource.JobRunDag, error)

	// LogJobRun opens a server stream of log lines for a job run. node is
	// optional; when empty the server streams the default node.
	LogJobRun(ctx context.Context, jobRunName string, node string) (*connect.ServerStreamForClient[openv1alpha1service.LogJobRunResponse], error)
}

type jobRunClient struct {
	jobRunServiceClient openv1alpha1connect.JobRunServiceClient
}

func NewJobRunClient(jobRunServiceClient openv1alpha1connect.JobRunServiceClient) JobRunInterface {
	return &jobRunClient{jobRunServiceClient: jobRunServiceClient}
}

func (c *jobRunClient) ListJobRuns(ctx context.Context, actionRun *name.ActionRun) ([]*openv1alpha1resource.JobRun, error) {
	var (
		pageToken string
		ret       []*openv1alpha1resource.JobRun
	)

	for {
		req := connect.NewRequest(&openv1alpha1service.ListJobRunsRequest{
			Parent:    actionRun.String(),
			PageSize:  listJobRunsPageSize,
			PageToken: pageToken,
		})
		res, err := c.jobRunServiceClient.ListJobRuns(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to list job runs: %w", err)
		}

		ret = append(ret, res.Msg.GetJobRuns()...)
		pageToken = res.Msg.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}

	return ret, nil
}

func (c *jobRunClient) GetJobRun(ctx context.Context, jobRunName string) (*openv1alpha1resource.JobRun, error) {
	req := connect.NewRequest(&openv1alpha1service.GetJobRunRequest{Name: jobRunName})
	res, err := c.jobRunServiceClient.GetJobRun(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get job run: %w", err)
	}
	return res.Msg, nil
}

func (c *jobRunClient) GetJobRunDag(ctx context.Context, jobRunName string) (*openv1alpha1resource.JobRunDag, error) {
	req := connect.NewRequest(&openv1alpha1service.GetJobRunDagRequest{Name: jobRunName})
	res, err := c.jobRunServiceClient.GetJobRunDag(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get job run dag: %w", err)
	}
	return res.Msg, nil
}

func (c *jobRunClient) LogJobRun(ctx context.Context, jobRunName string, node string) (*connect.ServerStreamForClient[openv1alpha1service.LogJobRunResponse], error) {
	req := connect.NewRequest(&openv1alpha1service.LogJobRunRequest{
		Name: jobRunName,
		Node: node,
	})
	stream, err := c.jobRunServiceClient.LogJobRun(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to open job run log stream: %w", err)
	}
	return stream, nil
}
