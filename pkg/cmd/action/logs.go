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
	"context"
	"errors"
	"fmt"
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
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
		jobIndex    = 0
		node        = ""
		follow      = false
	)

	cmd := &cobra.Command{
		Use:                   "logs <action-run-name/id> [-j <job-index>] [--node <node>] [-f] [-p <working-project-slug>]",
		Short:                 "Stream logs of an action run's job run",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := getProvider(*cfgPath).GetProfileManager()
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

			jobRun, err := selectJobRun(cmd.Context(), pm, actionRun, jobIndex, follow, io)
			if err != nil {
				exitf(io, "%v", err)
				return
			}
			if jobRun == nil {
				return // already reported (no job runs / pending without --follow)
			}

			cli := pm.JobRunCli()
			jobRunName := jobRun.GetName()
			streamFn := func(ctx context.Context, n string) error {
				return streamOnce(ctx, cli, jobRunName, n, io)
			}
			dagFn := func(ctx context.Context) (string, error) {
				return resolveDefaultNode(ctx, cli, jobRunName)
			}
			if err = followLogs(cmd.Context(), node, follow, io, streamFn, dagFn); err != nil {
				if errors.Is(err, context.Canceled) {
					return // clean Ctrl-C exit
				}
				exitf(io, "%v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "working project slug")
	cmd.Flags().IntVarP(&jobIndex, "job", "j", 0, "index of the job run to stream (default 0 = first)")
	cmd.Flags().StringVar(&node, "node", "", "DAG node to stream (default: the only/first node)")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow the log stream (reconnect on transient errors)")

	return cmd
}

// resolveActionRun accepts a full action-run resource name or a bare UUID.
func resolveActionRun(arg string, proj *name.Project) (*name.ActionRun, error) {
	if actionRun, err := name.NewActionRun(arg); err == nil {
		return actionRun, nil
	}
	if name.IsUUID(arg) {
		return &name.ActionRun{ProjectID: proj.ProjectID, ID: arg}, nil
	}
	return nil, fmt.Errorf("invalid action run name or id: %s", arg)
}

// selectJobRun lists the action run's job runs and picks the requested index.
// Returns (nil, nil) when there is nothing to stream and the situation has
// already been reported to the user (e.g. no job runs without --follow).
func selectJobRun(ctx context.Context, pm *config.ProfileManager, actionRun *name.ActionRun, jobIndex int, follow bool, io *iostreams.IOStreams) (*openv1alpha1resource.JobRun, error) {
	jobRuns, err := listJobRunsWithWait(ctx, pm.JobRunCli(), actionRun, follow, io)
	if err != nil {
		return nil, err
	}
	if len(jobRuns) == 0 {
		io.Println("No job runs found")
		return nil, nil
	}
	if jobIndex < 0 || jobIndex >= len(jobRuns) {
		return nil, fmt.Errorf("job index %d out of range (%d job runs)", jobIndex, len(jobRuns))
	}
	return jobRuns[jobIndex], nil
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

// followLogs drives the log stream state machine: it resolves the DAG node when
// an empty node is rejected (multi-node workflows) and reconnects on transient
// errors when follow is set. streamFn opens and relays a single stream for the
// given node; dagFn resolves the default node. Both are injected for testability.
func followLogs(
	ctx context.Context,
	node string,
	follow bool,
	io *iostreams.IOStreams,
	streamFn func(ctx context.Context, node string) error,
	dagFn func(ctx context.Context) (string, error),
) error {
	resolvedNode := node
	delay := reconnectBaseDelay

	for attempt := 0; ; attempt++ {
		err := streamFn(ctx, resolvedNode)
		switch {
		case err == nil:
			return nil // stream ended cleanly (job finished)
		case errors.Is(err, context.Canceled):
			return context.Canceled
		case isNodeNotFound(err) && resolvedNode == "":
			// Multi-node (Steps) workflow: empty node is rejected. Resolve via DAG.
			n, resolveErr := dagFn(ctx)
			if resolveErr != nil {
				return resolveErr
			}
			resolvedNode = n
			continue
		case follow && isRetriable(err) && attempt < reconnectMaxAttempts:
			io.Eprintln(fmt.Sprintf("Stream interrupted, reconnecting... (attempt %d/%d): %v", attempt+1, reconnectMaxAttempts, err))
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
func streamOnce(ctx context.Context, cli api.JobRunInterface, jobRunName, node string, io *iostreams.IOStreams) error {
	stream, err := cli.LogJobRun(ctx, jobRunName, node)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	for stream.Receive() {
		io.Println(stream.Msg().GetMessage())
	}
	return stream.Err()
}

// resolveDefaultNode picks a node name when the job run is a multi-node DAG.
func resolveDefaultNode(ctx context.Context, cli api.JobRunInterface, jobRunName string) (string, error) {
	dag, err := cli.GetJobRunDag(ctx, jobRunName)
	if err != nil {
		return "", err
	}
	nodes := dag.GetNodes()
	if len(nodes) == 1 {
		for nodeName := range nodes {
			return nodeName, nil
		}
	}
	nodeNames := make([]string, 0, len(nodes))
	for nodeName := range nodes {
		nodeNames = append(nodeNames, nodeName)
	}
	return "", fmt.Errorf("job run has multiple nodes; specify one with --node. Available: %v", nodeNames)
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

func exitf(io *iostreams.IOStreams, format string, a ...interface{}) {
	io.Eprintln(fmt.Sprintf(format, a...))
}
