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

package action

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/pkg/errors"
)

// fakeJobRunClient is a test double for api.JobRunInterface.
type fakeJobRunClient struct {
	listResult []*openv1alpha1resource.JobRun
	listErr    error
	dag        *openv1alpha1resource.JobRunDag
	dagErr     error
	// getQueue is popped one entry per GetJobRun call (last entry repeats);
	// getErr is returned instead when set.
	getQueue []*openv1alpha1resource.JobRun
	getErr   error
}

func (f *fakeJobRunClient) ListJobRuns(_ context.Context, _ *name.ActionRun) ([]*openv1alpha1resource.JobRun, error) {
	return f.listResult, f.listErr
}

func (f *fakeJobRunClient) GetJobRun(_ context.Context, _ string) (*openv1alpha1resource.JobRun, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if len(f.getQueue) == 0 {
		return &openv1alpha1resource.JobRun{}, nil
	}
	jr := f.getQueue[0]
	if len(f.getQueue) > 1 {
		f.getQueue = f.getQueue[1:]
	}
	return jr, nil
}

func (f *fakeJobRunClient) GetJobRunDag(_ context.Context, _ string) (*openv1alpha1resource.JobRunDag, error) {
	return f.dag, f.dagErr
}

func (f *fakeJobRunClient) LogJobRun(_ context.Context, _ string, _ string) (*connect.ServerStreamForClient[openv1alpha1service.LogJobRunResponse], error) {
	return nil, nil
}

func discardIO() *iostreams.IOStreams {
	return iostreams.Test(nil, &discardWriter{}, &discardWriter{})
}

type discardWriter struct{}

func (*discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestHandleLogMessage_LiveLine(t *testing.T) {
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &discardWriter{})
	err := handleLogMessage(context.Background(),
		&openv1alpha1service.LogJobRunResponse{Message: "hello"}, io)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("live line not printed: %q", out.String())
	}
}

func TestHandleLogMessage_DownloadURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("archived-line\n"))
	}))
	defer srv.Close()

	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &discardWriter{})
	err := handleLogMessage(context.Background(),
		&openv1alpha1service.LogJobRunResponse{LogDownloadUri: srv.URL}, io)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out.String(), "archived-line") {
		t.Fatalf("archived log not printed: %q", out.String())
	}
}

func TestPrintArchivedLog_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("line-one\nline-two\n"))
	}))
	defer srv.Close()

	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &discardWriter{})
	if err := printArchivedLog(context.Background(), srv.URL, io); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "line-one") || !strings.Contains(got, "line-two") {
		t.Fatalf("archived log not printed: %q", got)
	}
}

func TestPrintArchivedLog_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := printArchivedLog(context.Background(), srv.URL, discardIO())
	if err == nil || !strings.Contains(err.Error(), "unexpected status") {
		t.Fatalf("expected non-200 error, got %v", err)
	}
}

func TestPrintArchivedLog_NotFoundIsClean(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &discardWriter{})
	if err := printArchivedLog(context.Background(), srv.URL, io); err != nil {
		t.Fatalf("404 should be handled cleanly, got err: %v", err)
	}
	if !strings.Contains(out.String(), "No logs available") {
		t.Fatalf("expected 'No logs available' message, got %q", out.String())
	}
}

func TestAwaitJobRunStart_FollowPollsUntilRunning(t *testing.T) {
	cli := &fakeJobRunClient{
		getQueue: []*openv1alpha1resource.JobRun{
			{State: openv1alpha1enums.JobRunStateEnum_RUNNING},
		},
	}
	started, err := awaitJobRunStart(context.Background(), cli, "jr",
		openv1alpha1enums.JobRunStateEnum_QUEUED, true, discardIO())
	if err != nil || !started {
		t.Fatalf("expected started after poll: started=%v err=%v", started, err)
	}
}

func TestAwaitJobRunStart_FollowGetError(t *testing.T) {
	cli := &fakeJobRunClient{getErr: errors.New("boom")}
	started, err := awaitJobRunStart(context.Background(), cli, "jr",
		openv1alpha1enums.JobRunStateEnum_QUEUED, true, discardIO())
	if started || err == nil {
		t.Fatalf("expected error from GetJobRun: started=%v err=%v", started, err)
	}
}

func TestJobRunNotStarted(t *testing.T) {
	notStarted := []openv1alpha1enums.JobRunStateEnum_JobRunState{
		openv1alpha1enums.JobRunStateEnum_JOB_RUN_STATE_UNSPECIFIED,
		openv1alpha1enums.JobRunStateEnum_QUEUED,
		openv1alpha1enums.JobRunStateEnum_SCHEDULING,
	}
	for _, s := range notStarted {
		if !jobRunNotStarted(s) {
			t.Errorf("state %s should be not-started", s.String())
		}
	}
	started := []openv1alpha1enums.JobRunStateEnum_JobRunState{
		openv1alpha1enums.JobRunStateEnum_RUNNING,
		openv1alpha1enums.JobRunStateEnum_SUCCEEDED,
		openv1alpha1enums.JobRunStateEnum_FAILED,
		openv1alpha1enums.JobRunStateEnum_ABORTED,
	}
	for _, s := range started {
		if jobRunNotStarted(s) {
			t.Errorf("state %s should be started", s.String())
		}
	}
}

func TestAwaitJobRunStart_AlreadyStarted(t *testing.T) {
	started, err := awaitJobRunStart(context.Background(), &fakeJobRunClient{}, "jr",
		openv1alpha1enums.JobRunStateEnum_RUNNING, true, discardIO())
	if err != nil || !started {
		t.Fatalf("running job should be started immediately: started=%v err=%v", started, err)
	}
}

func TestAwaitJobRunStart_NoFollowReports(t *testing.T) {
	started, err := awaitJobRunStart(context.Background(), &fakeJobRunClient{}, "jr",
		openv1alpha1enums.JobRunStateEnum_QUEUED, false, discardIO())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if started {
		t.Fatal("queued job without follow should not be started")
	}
}

func TestAwaitJobRunStart_FollowCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before the first wait
	started, err := awaitJobRunStart(ctx, &fakeJobRunClient{}, "jr",
		openv1alpha1enums.JobRunStateEnum_QUEUED, true, discardIO())
	if started {
		t.Fatal("expected not started on canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestFollowLogs_CleanEnd(t *testing.T) {
	calls := 0
	streamFn := func(_ context.Context, _ string, _ *iostreams.IOStreams) error { calls++; return nil }

	if err := followLogs(context.Background(), []string{"n"}, false, discardIO(), streamFn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("streamFn calls = %d, want 1", calls)
	}
}

func TestFollowLogs_CanceledExitsClean(t *testing.T) {
	streamFn := func(_ context.Context, _ string, _ *iostreams.IOStreams) error { return context.Canceled }

	err := followLogs(context.Background(), []string{"n"}, true, discardIO(), streamFn)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestFollowLogs_NodeNotFoundWithoutFollowSkipsNode(t *testing.T) {
	calls := 0
	var errOut bytes.Buffer
	io := iostreams.Test(nil, &discardWriter{}, &errOut)
	streamFn := func(_ context.Context, _ string, _ *iostreams.IOStreams) error {
		calls++
		return connect.NewError(connect.CodeInvalidArgument, errors.New("pod node not found"))
	}

	if err := followLogs(context.Background(), []string{"pending"}, false, io, streamFn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("streamFn calls = %d, want 1", calls)
	}
	if !strings.Contains(errOut.String(), `node "pending"`) {
		t.Fatalf("expected node-specific stderr, got %q", errOut.String())
	}
}

func TestFollowLogs_NonRetriableReturns(t *testing.T) {
	wantErr := connect.NewError(connect.CodeUnauthenticated, errors.New("nope"))
	streamFn := func(_ context.Context, _ string, _ *iostreams.IOStreams) error { return wantErr }

	err := followLogs(context.Background(), []string{"n"}, true, discardIO(), streamFn)
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestFollowLogs_RetriableReconnectStopsOnCanceledBackoff(t *testing.T) {
	// Cancelled context makes the backoff sleep return immediately, exercising
	// the reconnect branch without a real delay.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	streamFn := func(_ context.Context, _ string, _ *iostreams.IOStreams) error {
		return connect.NewError(connect.CodeUnavailable, errors.New("flaky"))
	}

	err := followLogs(ctx, []string{"n"}, true, discardIO(), streamFn)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled from backoff, got %v", err)
	}
}

func TestResolveLogNodes(t *testing.T) {
	t.Run("explicit node validates against dag", func(t *testing.T) {
		cli := &fakeJobRunClient{dag: &openv1alpha1resource.JobRunDag{
			Nodes: map[string]*openv1alpha1resource.JobRunNode{"echo": {Name: "echo"}},
		}}
		got, err := resolveLogNodes(context.Background(), cli, "jr", "echo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Join(got, ",") != "echo" {
			t.Fatalf("got %v, want [echo]", got)
		}
	})

	t.Run("explicit unknown node errors before streaming", func(t *testing.T) {
		cli := &fakeJobRunClient{dag: &openv1alpha1resource.JobRunDag{
			Nodes: map[string]*openv1alpha1resource.JobRunNode{"echo": {Name: "echo"}},
		}}
		_, err := resolveLogNodes(context.Background(), cli, "jr", "missing")
		if err == nil || !strings.Contains(err.Error(), `job run node "missing" not found`) ||
			!strings.Contains(err.Error(), "Available: [echo]") {
			t.Fatalf("expected invalid node error with available nodes, got %v", err)
		}
	})

	t.Run("dependency order", func(t *testing.T) {
		cli := &fakeJobRunClient{dag: &openv1alpha1resource.JobRunDag{
			Nodes: map[string]*openv1alpha1resource.JobRunNode{
				"c": {Name: "c", DependentNodes: []string{"b"}},
				"a": {Name: "a"},
				"b": {Name: "b", DependentNodes: []string{"a"}},
			},
		}}
		got, err := resolveLogNodes(context.Background(), cli, "jr", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Join(got, ",") != "a,b,c" {
			t.Fatalf("got %v, want [a b c]", got)
		}
	})

	t.Run("parallel layer sorted by name", func(t *testing.T) {
		cli := &fakeJobRunClient{dag: &openv1alpha1resource.JobRunDag{
			Nodes: map[string]*openv1alpha1resource.JobRunNode{
				"z": {Name: "z"},
				"a": {Name: "a"},
				"m": {Name: "m", DependentNodes: []string{"a", "z"}},
			},
		}}
		got, err := resolveLogNodes(context.Background(), cli, "jr", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Join(got, ",") != "a,z,m" {
			t.Fatalf("got %v, want [a z m]", got)
		}
	})

	t.Run("missing dependencies ignored", func(t *testing.T) {
		cli := &fakeJobRunClient{dag: &openv1alpha1resource.JobRunDag{
			Nodes: map[string]*openv1alpha1resource.JobRunNode{
				"b": {Name: "b", DependentNodes: []string{"missing"}},
				"a": {Name: "a"},
			},
		}}
		got, err := resolveLogNodes(context.Background(), cli, "jr", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Join(got, ",") != "a,b" {
			t.Fatalf("got %v, want [a b]", got)
		}
	})

	t.Run("cycle errors", func(t *testing.T) {
		cli := &fakeJobRunClient{dag: &openv1alpha1resource.JobRunDag{
			Nodes: map[string]*openv1alpha1resource.JobRunNode{
				"a": {Name: "a", DependentNodes: []string{"b"}},
				"b": {Name: "b", DependentNodes: []string{"a"}},
			},
		}}
		_, err := resolveLogNodes(context.Background(), cli, "jr", "")
		if err == nil || !strings.Contains(err.Error(), "dependency cycle") {
			t.Fatalf("expected cycle error, got %v", err)
		}
	})

	t.Run("dag error propagates", func(t *testing.T) {
		cli := &fakeJobRunClient{dagErr: errors.New("boom")}
		if _, err := resolveLogNodes(context.Background(), cli, "jr", ""); err == nil {
			t.Fatal("expected dag error")
		}
	})
}

func TestNodePrefixOutput(t *testing.T) {
	t.Run("live line", func(t *testing.T) {
		var out bytes.Buffer
		base := iostreams.Test(nil, &out, &discardWriter{})
		io := withNodePrefix(base, "b", len("build"))
		err := handleLogMessage(context.Background(),
			&openv1alpha1service.LogJobRunResponse{Message: "hello"}, io)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got, want := out.String(), "b      hello\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("archived lines", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("line-one\nline-two\n"))
		}))
		defer srv.Close()

		var out bytes.Buffer
		base := iostreams.Test(nil, &out, &discardWriter{})
		io := withNodePrefix(base, "archive", len("archive"))
		err := handleLogMessage(context.Background(),
			&openv1alpha1service.LogJobRunResponse{LogDownloadUri: srv.URL}, io)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got, want := out.String(), "archive  line-one\narchive  line-two\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("no logs line", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		var out bytes.Buffer
		base := iostreams.Test(nil, &out, &discardWriter{})
		io := withNodePrefix(base, "node", len("node"))
		if err := printArchivedLog(context.Background(), srv.URL, io); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.HasPrefix(out.String(), "node  No logs available") {
			t.Fatalf("expected prefixed no-log line, got %q", out.String())
		}
	})
}

func TestListJobRunsWithWait(t *testing.T) {
	actionRun := &name.ActionRun{ProjectID: "p", ID: "ar"}

	t.Run("returns runs immediately", func(t *testing.T) {
		cli := &fakeJobRunClient{listResult: []*openv1alpha1resource.JobRun{{Name: "jr1"}}}
		got, err := listJobRunsWithWait(context.Background(), cli, actionRun, false, discardIO())
		if err != nil || len(got) != 1 {
			t.Fatalf("got %d runs, err %v", len(got), err)
		}
	})

	t.Run("empty without follow returns empty", func(t *testing.T) {
		cli := &fakeJobRunClient{listResult: nil}
		got, err := listJobRunsWithWait(context.Background(), cli, actionRun, false, discardIO())
		if err != nil || len(got) != 0 {
			t.Fatalf("got %d runs, err %v", len(got), err)
		}
	})

	t.Run("empty with follow polls then gives up on canceled ctx", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cli := &fakeJobRunClient{listResult: nil}
		_, err := listJobRunsWithWait(ctx, cli, actionRun, true, discardIO())
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	})
}

func TestResolveActionRun(t *testing.T) {
	proj := &name.Project{ProjectID: "11111111-1111-1111-1111-111111111111"}

	t.Run("full resource name", func(t *testing.T) {
		got, err := resolveActionRun("projects/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/actionRuns/bbbb", proj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ProjectID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || got.ID != "bbbb" {
			t.Fatalf("unexpected parse: %+v", got)
		}
	})

	t.Run("bare uuid uses project", func(t *testing.T) {
		got, err := resolveActionRun("22222222-2222-2222-2222-222222222222", proj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ProjectID != proj.ProjectID || got.ID != "22222222-2222-2222-2222-222222222222" {
			t.Fatalf("unexpected: %+v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := resolveActionRun("not-a-name", proj); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestIsRetriable(t *testing.T) {
	retriable := []connect.Code{
		connect.CodeUnknown, connect.CodeInternal, connect.CodeUnavailable,
		connect.CodeAborted, connect.CodeResourceExhausted,
	}
	for _, code := range retriable {
		if !isRetriable(connect.NewError(code, errors.New("x"))) {
			t.Fatalf("code %v should be retriable", code)
		}
	}
	notRetriable := []connect.Code{connect.CodeNotFound, connect.CodeUnauthenticated, connect.CodeInvalidArgument}
	for _, code := range notRetriable {
		if isRetriable(connect.NewError(code, errors.New("x"))) {
			t.Fatalf("code %v should not be retriable", code)
		}
	}
	if isRetriable(errors.New("plain")) {
		t.Fatal("non-connect error should not be retriable")
	}
}

func TestNextDelay(t *testing.T) {
	if got := nextDelay(2 * time.Second); got != 4*time.Second {
		t.Fatalf("got %v, want 4s", got)
	}
	if got := nextDelay(reconnectMaxDelay); got != reconnectMaxDelay {
		t.Fatalf("delay should cap at %v, got %v", reconnectMaxDelay, got)
	}
}

func TestIsNodeNotFound(t *testing.T) {
	if !isNodeNotFound(connect.NewError(connect.CodeInvalidArgument, errors.New("x"))) {
		t.Fatal("InvalidArgument should be treated as node-not-found")
	}
	if isNodeNotFound(connect.NewError(connect.CodeUnavailable, errors.New("x"))) {
		t.Fatal("Unavailable should not be node-not-found")
	}
}
