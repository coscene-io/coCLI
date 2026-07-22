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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

const (
	reconnectMaxAttempts = 5
	reconnectBaseDelay   = 2 * time.Second
	reconnectMaxDelay    = 30 * time.Second
)

func NewLogsCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		projectSlug = ""
		jobIndex    = 1
		node        = ""
		follow      = false
	)

	cmd := &cobra.Command{
		Use:   "logs <action-run-name/id> [-j <job-index>] [--node <node>] [-f] [-p <working-project-slug>]",
		Short: "Stream logs of an action run's job run",
		Long: `Stream the logs of an action run's job run.

The action run is given as a full resource name
(projects/<project>/actionRuns/<uuid>) or a bare UUID.

While the job run is running, its pod logs are streamed live. Once the job
run has finished and its pod has been cleaned up, the archived log is
downloaded and printed instead — so the command works the same regardless
of whether the run is in progress or already completed.

  -j  select which job run to read when an action run has several. The index
      is 1-based (default 1 = the first).
  --node  only read this DAG node. When omitted, all DAG nodes are printed
      in dependency order.
  -f  follow: keep the stream open and reconnect on transient errors. If
      the job run has not started yet, wait for it to start. Without -f, a
      not-yet-started run is reported and the command exits.`,
		Example: `  # Print logs for a finished or running action run (first job run)
  cocli action logs projects/my-project/actionRuns/<uuid> -p my-project

  # By bare UUID, following a running job and waiting if it hasn't started
  cocli action logs <uuid> -p my-project -f

  # A specific job run and DAG node; output still includes the node prefix
  cocli action logs <uuid> -p my-project -j 1 --node encode`,
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				exitf(io, "unable to get project name: %v", err)
				return
			}

			actionRun, err := resolveActionRun(args[0], proj)
			if err != nil {
				exitf(io, "%v", err)
				return
			}

			jobRun, err := selectJobRun(cmd.Context(), pm.JobRunCli(), actionRun, jobIndex, follow, io)
			if err != nil {
				exitf(io, "%v", err)
				return
			}
			if jobRun == nil {
				return // already reported (no job runs / pending without --follow)
			}

			cli := pm.JobRunCli()
			jobRunName := jobRun.GetName()

			// A job run that hasn't started yet (queued/scheduling) has no
			// pod to stream and no archived log to download. Wait for it to
			// start when following; otherwise report and exit cleanly rather
			// than appearing to "finish" with no output.
			started, err := awaitJobRunStart(cmd.Context(), cli, jobRunName, jobRun.GetState(), follow, io)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				exitf(io, "%v", err)
				return
			}
			if !started {
				return
			}

			nodes, err := resolveLogNodes(cmd.Context(), cli, jobRunName, node)
			if err != nil {
				exitf(io, "%v", err)
				return
			}
			if len(nodes) == 0 {
				io.Println("No nodes found for this job run.")
				return
			}

			streamFn := func(ctx context.Context, n string, nodeIO *iostreams.IOStreams) error {
				return streamOnce(ctx, cli, jobRunName, n, nodeIO)
			}
			if err = followLogs(cmd.Context(), nodes, follow, io, streamFn); err != nil {
				if errors.Is(err, context.Canceled) {
					return // clean Ctrl-C exit
				}
				exitf(io, "%v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "working project slug")
	cmd.Flags().IntVarP(&jobIndex, "job", "j", 1, "1-based index of the job run to stream (default 1 = first)")
	cmd.Flags().StringVar(&node, "node", "", "DAG node to stream (default: all nodes in dependency order)")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow the log stream (reconnect on transient errors)")

	return cmd
}

// selectJobRun lists the action run's job runs and picks the requested index.
// The index is 1-based (1 = first job run). Returns (nil, nil) when there is
// nothing to stream and the situation has already been reported to the user
// (e.g. no job runs without --follow).
func selectJobRun(ctx context.Context, cli api.JobRunInterface, actionRun *name.ActionRun, jobIndex int, follow bool, io *iostreams.IOStreams) (*openv1alpha1resource.JobRun, error) {
	jobRuns, err := listJobRunsWithWait(ctx, cli, actionRun, follow, io)
	if err != nil {
		return nil, err
	}
	if len(jobRuns) == 0 {
		io.Println("No job runs found")
		return nil, nil
	}
	if jobIndex < 1 || jobIndex > len(jobRuns) {
		return nil, fmt.Errorf("job index %d out of range (valid 1..%d)", jobIndex, len(jobRuns))
	}
	return jobRuns[jobIndex-1], nil
}

// listJobRunsWithWait lists job runs, optionally polling while empty when
// --follow is set (the action run may not have scheduled any job runs yet).
func listJobRunsWithWait(ctx context.Context, cli api.JobRunInterface, actionRun *name.ActionRun, follow bool, io *iostreams.IOStreams) ([]*openv1alpha1resource.JobRun, error) {
	delay := reconnectBaseDelay
	for attempt := 0; ; attempt++ {
		jobRuns, err := cli.ListJobRuns(ctx, actionRun)
		if err != nil {
			return nil, err
		}
		if len(jobRuns) > 0 || !follow || attempt >= reconnectMaxAttempts {
			return jobRuns, nil
		}
		io.Eprintln(fmt.Sprintf("Waiting for job runs... (attempt %d/%d)", attempt+1, reconnectMaxAttempts))
		if err = sleepCtx(ctx, delay); err != nil {
			return nil, err
		}
		delay = nextDelay(delay)
	}
}

// jobRunNotStarted reports whether a job run has not begun executing yet
// (queued/scheduling/unspecified) — no pod exists to stream and no log has
// been archived.
func jobRunNotStarted(state openv1alpha1enums.JobRunStateEnum_JobRunState) bool {
	switch state {
	case openv1alpha1enums.JobRunStateEnum_JOB_RUN_STATE_UNSPECIFIED,
		openv1alpha1enums.JobRunStateEnum_QUEUED,
		openv1alpha1enums.JobRunStateEnum_SCHEDULING:
		return true
	default:
		return false
	}
}

// awaitJobRunStart blocks until a not-yet-started job run begins (running or
// terminal) when follow is set, polling its state. Without follow it reports
// the state and returns started=false so the caller exits cleanly instead of
// treating an empty stream as a finished job. Returns started=true immediately
// when the run has already started.
func awaitJobRunStart(
	ctx context.Context,
	cli api.JobRunInterface,
	jobRunName string,
	state openv1alpha1enums.JobRunStateEnum_JobRunState,
	follow bool,
	io *iostreams.IOStreams,
) (bool, error) {
	if !jobRunNotStarted(state) {
		return true, nil
	}
	if !follow {
		io.Println(fmt.Sprintf(
			"Job run has not started yet (state: %s). Pass -f to wait for logs.",
			state.String(),
		))
		return false, nil
	}

	delay := reconnectBaseDelay
	for {
		io.Eprintln(fmt.Sprintf("Waiting for job run to start (state: %s)...", state.String()))
		if err := sleepCtx(ctx, delay); err != nil {
			return false, err
		}
		jobRun, err := cli.GetJobRun(ctx, jobRunName)
		if err != nil {
			return false, err
		}
		state = jobRun.GetState()
		if !jobRunNotStarted(state) {
			return true, nil
		}
		delay = nextDelay(delay)
	}
}

// followLogs streams each selected node in order. Every stdout log line is
// prefixed with the node name, aligned to the widest selected node.
func followLogs(
	ctx context.Context,
	nodes []string,
	follow bool,
	io *iostreams.IOStreams,
	streamFn func(ctx context.Context, node string, nodeIO *iostreams.IOStreams) error,
) error {
	width := maxNodeNameWidth(nodes)
	for _, node := range nodes {
		nodeIO := withNodePrefix(io, node, width)
		if err := followNodeLogs(ctx, node, follow, io, nodeIO, streamFn); err != nil {
			return err
		}
	}
	return nil
}

func followNodeLogs(
	ctx context.Context,
	node string,
	follow bool,
	io *iostreams.IOStreams,
	nodeIO *iostreams.IOStreams,
	streamFn func(ctx context.Context, node string, nodeIO *iostreams.IOStreams) error,
) error {
	delay := reconnectBaseDelay

	for attempt := 0; ; attempt++ {
		err := streamFn(ctx, node, nodeIO)
		switch {
		case err == nil:
			return nil // stream ended cleanly (job finished)
		case errors.Is(err, context.Canceled):
			return context.Canceled
		case isNodeNotFound(err):
			if !follow {
				io.Eprintln(fmt.Sprintf("No logs available for node %q yet; pass -f to wait.", node))
				return nil
			}
			if attempt < reconnectMaxAttempts {
				io.Eprintln(fmt.Sprintf("Waiting for logs from node %q... (attempt %d/%d)", node, attempt+1, reconnectMaxAttempts))
				if sleepErr := sleepCtx(ctx, delay); sleepErr != nil {
					return sleepErr
				}
				delay = nextDelay(delay)
				continue
			}
			return err
		case follow && isRetriable(err) && attempt < reconnectMaxAttempts:
			io.Eprintln(fmt.Sprintf("Stream for node %q interrupted, reconnecting... (attempt %d/%d): %v", node, attempt+1, reconnectMaxAttempts, err))
			if sleepErr := sleepCtx(ctx, delay); sleepErr != nil {
				return sleepErr
			}
			delay = nextDelay(delay)
			continue
		default:
			return err
		}
	}
}

// streamOnce opens a single log stream and relays lines until it ends or errors.
// A running job streams live log lines (message). A finished job whose pod has
// been garbage collected has no live logs; the server then sends a single
// response carrying a presigned URL for the archived log, which we download and
// print so `action logs` works transparently regardless of job state.
func streamOnce(ctx context.Context, cli api.JobRunInterface, jobRunName, node string, io *iostreams.IOStreams) error {
	stream, err := cli.LogJobRun(ctx, jobRunName, node)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	for stream.Receive() {
		if err = handleLogMessage(ctx, stream.Msg(), io); err != nil {
			return err
		}
	}
	return stream.Err()
}

// handleLogMessage renders one LogJobRun response: a live log line (message),
// or — for a finished job run — a presigned URL (log_download_uri) whose
// archived log is downloaded and printed.
func handleLogMessage(ctx context.Context, msg *openv1alpha1service.LogJobRunResponse, io *iostreams.IOStreams) error {
	if downloadURL := msg.GetLogDownloadUri(); downloadURL != "" {
		return printArchivedLog(ctx, downloadURL, io)
	}
	io.Println(msg.GetMessage())
	return nil
}

// printArchivedLog downloads the archived job-run log from the presigned URL the
// server returned and writes it to stdout, line by line.
func printArchivedLog(ctx context.Context, downloadURL string, io *iostreams.IOStreams) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("build archived log request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download archived log: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		// The job run finished but no log was archived (e.g. the pod produced
		// none, or archival wasn't enabled when it ran). Report cleanly rather
		// than surfacing a raw 404.
		io.Println("No logs available for this job run.")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download archived log: unexpected status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Job logs can carry long lines (stack traces, serialized payloads); raise
	// the token cap well above bufio's 64KiB default.
	const maxLogLineBytes = 4 * 1024 * 1024
	scanner.Buffer(make([]byte, 0, 64*1024), maxLogLineBytes)
	for scanner.Scan() {
		io.Println(scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return fmt.Errorf("read archived log: %w", err)
	}
	return nil
}

// resolveLogNodes resolves the nodes that should be streamed. Explicit --node
// selection keeps the command scoped to that node; the default is the whole DAG
// in dependency order.
func resolveLogNodes(ctx context.Context, cli api.JobRunInterface, jobRunName string, requestedNode string) ([]string, error) {
	dag, err := cli.GetJobRunDag(ctx, jobRunName)
	if err != nil {
		return nil, err
	}
	nodes := dag.GetNodes()
	if requestedNode != "" {
		if _, ok := nodes[requestedNode]; ok {
			return []string{requestedNode}, nil
		}
		return nil, fmt.Errorf("job run node %q not found. Available: %v", requestedNode, nodeNames(nodes))
	}
	return sortDagNodes(nodes)
}

func nodeNames(nodes map[string]*openv1alpha1resource.JobRunNode) []string {
	names := make([]string, 0, len(nodes))
	for nodeName := range nodes {
		names = append(names, nodeName)
	}
	sort.Strings(names)
	return names
}

func sortDagNodes(nodes map[string]*openv1alpha1resource.JobRunNode) ([]string, error) {
	nodeNames := make([]string, 0, len(nodes))
	indegree := make(map[string]int, len(nodes))
	dependents := make(map[string][]string, len(nodes))

	for nodeName := range nodes {
		nodeNames = append(nodeNames, nodeName)
		indegree[nodeName] = 0
	}
	sort.Strings(nodeNames)

	for _, nodeName := range nodeNames {
		for _, dep := range nodes[nodeName].GetDependentNodes() {
			if _, ok := nodes[dep]; !ok {
				continue
			}
			indegree[nodeName]++
			dependents[dep] = append(dependents[dep], nodeName)
		}
	}
	for dep := range dependents {
		sort.Strings(dependents[dep])
	}

	queue := make([]string, 0, len(nodes))
	for _, nodeName := range nodeNames {
		if indegree[nodeName] == 0 {
			queue = append(queue, nodeName)
		}
	}

	ordered := make([]string, 0, len(nodes))
	for len(queue) > 0 {
		nodeName := queue[0]
		queue = queue[1:]
		ordered = append(ordered, nodeName)

		for _, dependent := range dependents[nodeName] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
		sort.Strings(queue)
	}

	if len(ordered) != len(nodes) {
		return nil, fmt.Errorf("job run DAG contains a dependency cycle")
	}
	return ordered, nil
}

func maxNodeNameWidth(nodes []string) int {
	width := 0
	for _, node := range nodes {
		width = max(width, runewidth.StringWidth(node))
	}
	return width
}

func withNodePrefix(io *iostreams.IOStreams, node string, width int) *iostreams.IOStreams {
	return iostreams.Test(io.In, &nodePrefixWriter{
		out:         io.Out,
		prefix:      padNodeName(node, width) + "  ",
		atLineStart: true,
	}, io.ErrOut)
}

func padNodeName(node string, width int) string {
	padding := width - runewidth.StringWidth(node)
	if padding <= 0 {
		return node
	}
	return node + strings.Repeat(" ", padding)
}

type nodePrefixWriter struct {
	out         interface{ Write([]byte) (int, error) }
	prefix      string
	atLineStart bool
}

func (w *nodePrefixWriter) Write(p []byte) (int, error) {
	originalLen := len(p)
	for len(p) > 0 {
		if w.atLineStart {
			if _, err := fmt.Fprint(w.out, w.prefix); err != nil {
				return 0, err
			}
			w.atLineStart = false
		}

		newline := bytes.IndexByte(p, '\n')
		if newline == -1 {
			if _, err := w.out.Write(p); err != nil {
				return 0, err
			}
			return originalLen, nil
		}

		if _, err := w.out.Write(p[:newline+1]); err != nil {
			return 0, err
		}
		w.atLineStart = true
		p = p[newline+1:]
	}

	return originalLen, nil
}

func isNodeNotFound(err error) bool {
	var connErr *connect.Error
	if errors.As(err, &connErr) {
		// matrix returns the pod-resolution failure for an unmatched template.
		return connErr.Code() == connect.CodeInvalidArgument || connErr.Code() == connect.CodeNotFound
	}
	return false
}

// isRetriable mirrors the unary retry interceptor's allow-list.
func isRetriable(err error) bool {
	var connErr *connect.Error
	if !errors.As(err, &connErr) {
		return false
	}
	switch connErr.Code() {
	case connect.CodeUnknown, connect.CodeInternal, connect.CodeUnavailable,
		connect.CodeAborted, connect.CodeResourceExhausted:
		return true
	default:
		return false
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func nextDelay(d time.Duration) time.Duration {
	next := d * 2
	if next > reconnectMaxDelay {
		return reconnectMaxDelay
	}
	return next
}

// exitf prints an error message to stderr and terminates with a non-zero exit
// code. The command uses cobra's Run (not RunE), so returning would exit 0 and
// hide failures from scripts/CI; exiting here mirrors the sibling commands'
// log.Fatalf behavior while preserving the iostreams-formatted message. Clean
// Ctrl-C (context.Canceled) paths return before reaching exitf and still exit 0.
func exitf(io *iostreams.IOStreams, format string, a ...interface{}) {
	io.Eprintln(fmt.Sprintf(format, a...))
	os.Exit(1)
}

// printActionNotFound prints the clean client-side not-found message shared by
// the get/delete/update commands so their wording stays identical. It is the
// message emitted when ActionId2Name or a follow-up GetByName returns a connect
// NotFound for a deleted/absent action, in place of the raw `not_found` error.
func printActionNotFound(io *iostreams.IOStreams, actionRef string, proj *name.Project) {
	io.Printf("failed to find action: %s in project: %s\n", actionRef, proj)
}
