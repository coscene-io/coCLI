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

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/pkg/errors"
)

// stubJobRunServiceClient is a minimal connect client stub. ListJobRuns returns
// queued pages in order; the other methods return preset values.
type stubJobRunServiceClient struct {
	listPages [][]*openv1alpha1resource.JobRun
	listToken []string // next_page_token per page
	listCall  int

	getJobRun *openv1alpha1resource.JobRun
	dag       *openv1alpha1resource.JobRunDag
	logErr    error
}

func (s *stubJobRunServiceClient) ListJobRuns(_ context.Context, _ *connect.Request[openv1alpha1service.ListJobRunsRequest]) (*connect.Response[openv1alpha1service.ListJobRunsResponse], error) {
	i := s.listCall
	s.listCall++
	return connect.NewResponse(&openv1alpha1service.ListJobRunsResponse{
		JobRuns:       s.listPages[i],
		NextPageToken: s.listToken[i],
	}), nil
}

func (s *stubJobRunServiceClient) GetJobRun(_ context.Context, _ *connect.Request[openv1alpha1service.GetJobRunRequest]) (*connect.Response[openv1alpha1resource.JobRun], error) {
	return connect.NewResponse(s.getJobRun), nil
}

func (s *stubJobRunServiceClient) GetJobRunDag(_ context.Context, _ *connect.Request[openv1alpha1service.GetJobRunDagRequest]) (*connect.Response[openv1alpha1resource.JobRunDag], error) {
	return connect.NewResponse(s.dag), nil
}

func (s *stubJobRunServiceClient) LogJobRun(_ context.Context, _ *connect.Request[openv1alpha1service.LogJobRunRequest]) (*connect.ServerStreamForClient[openv1alpha1service.LogJobRunResponse], error) {
	return nil, s.logErr
}

func TestJobRunClient_ListJobRunsPaginates(t *testing.T) {
	stub := &stubJobRunServiceClient{
		listPages: [][]*openv1alpha1resource.JobRun{
			{{Name: "jr1"}, {Name: "jr2"}},
			{{Name: "jr3"}},
		},
		listToken: []string{"tok", ""},
	}
	cli := NewJobRunClient(stub)

	runs, err := cli.ListJobRuns(context.Background(), &name.ActionRun{ProjectID: "p", ID: "ar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("got %d runs across pages, want 3", len(runs))
	}
	if stub.listCall != 2 {
		t.Fatalf("expected 2 list calls (pagination), got %d", stub.listCall)
	}
}

func TestJobRunClient_GetJobRun(t *testing.T) {
	cli := NewJobRunClient(&stubJobRunServiceClient{getJobRun: &openv1alpha1resource.JobRun{Name: "jr"}})
	got, err := cli.GetJobRun(context.Background(), "jr")
	if err != nil || got.GetName() != "jr" {
		t.Fatalf("got %v, err %v", got, err)
	}
}

func TestJobRunClient_GetJobRunDag(t *testing.T) {
	cli := NewJobRunClient(&stubJobRunServiceClient{dag: &openv1alpha1resource.JobRunDag{
		Nodes: map[string]*openv1alpha1resource.JobRunNode{"a": {Name: "a"}},
	}})
	got, err := cli.GetJobRunDag(context.Background(), "jr")
	if err != nil || len(got.GetNodes()) != 1 {
		t.Fatalf("got %v, err %v", got, err)
	}
}

func TestJobRunClient_LogJobRunError(t *testing.T) {
	cli := NewJobRunClient(&stubJobRunServiceClient{logErr: errors.New("dial failed")})
	if _, err := cli.LogJobRun(context.Background(), "jr", ""); err == nil {
		t.Fatal("expected error to be propagated")
	}
}
