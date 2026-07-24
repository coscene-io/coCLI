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
	"fmt"

	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/spf13/cobra"
)

const cancelRunConfirmation = "Request cancellation for this action run? This cannot be undone."

type actionRunCanceler interface {
	ListAllActionRuns(context.Context, *api.ListActionRunsOptions) ([]*openv1alpha1resource.ActionRun, error)
	TerminateActionRun(context.Context, *name.ActionRun) error
}

type confirmCancelRun func(string, *iostreams.IOStreams) bool

func NewCancelRunCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		force       bool
		projectSlug string
	)

	cmd := &cobra.Command{
		Use:                   "cancel-run <action-run-name/id> [-p <working-project-slug>] [-f]",
		Short:                 "Request cancellation of an action run.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				return fmt.Errorf("unable to get project name: %w", err)
			}

			return cancelActionRun(cmd.Context(), io, pm.ActionCli(), args[0], proj, force, prompts.PromptYN)
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "request cancellation without confirmation")

	return cmd
}

func cancelActionRun(ctx context.Context, io *iostreams.IOStreams, cli actionRunCanceler, actionRunRef string, proj *name.Project, force bool, confirm confirmCancelRun) error {
	actionRun, err := resolveActionRun(actionRunRef, proj)
	if err != nil {
		return err
	}

	if !force && !confirm(cancelRunConfirmation, io) {
		io.Println("Action run cancellation aborted.")
		return nil
	}

	runs, err := cli.ListAllActionRuns(ctx, &api.ListActionRunsOptions{
		Parent: actionRun.Project().String(),
	})
	if err != nil {
		return fmt.Errorf("failed to check action run state: %w", err)
	}

	run := findActionRun(runs, actionRun.String())
	if run == nil {
		return fmt.Errorf("action run not found: %s", actionRun)
	}
	if isFinishedActionRun(run.GetState()) {
		io.Printf("Action run has already finished with state %s. No cancellation request was sent.\n", run.GetState())
		return nil
	}

	if err = cli.TerminateActionRun(ctx, actionRun); err != nil {
		return fmt.Errorf("failed to request action run cancellation: %w", err)
	}

	io.Println("Action run cancellation requested successfully.")
	return nil
}

func findActionRun(runs []*openv1alpha1resource.ActionRun, target string) *openv1alpha1resource.ActionRun {
	for _, run := range runs {
		if run.GetName() == target {
			return run
		}
	}
	return nil
}

func isFinishedActionRun(state openv1alpha1enums.ActionRunStateEnum_ActionRunState) bool {
	switch state {
	case openv1alpha1enums.ActionRunStateEnum_SUCCEEDED,
		openv1alpha1enums.ActionRunStateEnum_FAILED,
		openv1alpha1enums.ActionRunStateEnum_ABORTED:
		return true
	default:
		return false
	}
}
