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

package record

import (
	"context"
	"fmt"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/spf13/cobra"
)

const momentDeleteConfirmation = "Delete this moment? This cannot be undone."

type momentDeleter interface {
	GetEvent(context.Context, *name.Event) (*openv1alpha1resource.Event, error)
	DeleteEvent(context.Context, *name.Event) error
}

type confirmMomentDelete func(string, *iostreams.IOStreams) bool

func NewMomentDeleteCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		force       bool
		projectSlug string
	)

	cmd := &cobra.Command{
		Use:                   "delete <moment-resource-name/id> [-p <working-project-slug>] [-f]",
		Short:                 "Delete a moment from a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			eventName, err := resolveMomentName(args[0], func() (*name.Project, error) {
				proj, projectErr := pm.ProjectName(cmd.Context(), projectSlug)
				if projectErr != nil {
					return nil, fmt.Errorf("unable to get project name: %w", projectErr)
				}
				return proj, nil
			})
			if err != nil {
				return err
			}

			return deleteMoment(cmd.Context(), io, pm.EventCli(), eventName, force, prompts.PromptYN)
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "delete without confirmation")

	return cmd
}

func resolveMomentName(momentRef string, resolveProject func() (*name.Project, error)) (*name.Event, error) {
	if eventName, err := name.NewEvent(momentRef); err == nil {
		return eventName, nil
	}
	if !name.IsUUID(momentRef) {
		return nil, fmt.Errorf("invalid moment resource name or id: %s", momentRef)
	}

	proj, err := resolveProject()
	if err != nil {
		return nil, err
	}
	return &name.Event{ProjectID: proj.ProjectID, ID: momentRef}, nil
}

func deleteMoment(ctx context.Context, io *iostreams.IOStreams, cli momentDeleter, eventName *name.Event, force bool, confirm confirmMomentDelete) error {
	event, err := cli.GetEvent(ctx, eventName)
	if err != nil {
		return fmt.Errorf("failed to get moment %s: %w", eventName, err)
	}

	displayName := event.GetDisplayName()
	if displayName == "" {
		displayName = "<unnamed>"
	}
	io.Printf("Moment: %s (%s)\n", displayName, eventName)

	if !force && !confirm(momentDeleteConfirmation, io) {
		io.Println("Moment deletion aborted.")
		return nil
	}

	if err = cli.DeleteEvent(ctx, eventName); err != nil {
		return fmt.Errorf("failed to delete moment %s: %w", eventName, err)
	}

	io.Printf("Moment successfully deleted: %s\n", eventName)
	return nil
}
