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
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// deleteTriggerNote is the one-line side-effect warning surfaced in both the
// confirmation prompt and the help text. Delete is a soft delete on the backend
// that also cascades to soft-delete every trigger bound to the action (matrix
// repo.Delete, one transaction). cocli cannot count the affected triggers (the
// RPC returns Empty), so the warning is carried by wording, not a pre-delete
// query — see plan D4/U3/F8.
const deleteTriggerNote = "This also disables any triggers bound to it."

func NewDeleteCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		force       = false
		projectSlug = ""
	)

	cmd := &cobra.Command{
		Use:   "delete <action-resource-name/id> [-p <working-project-slug>] [-f]",
		Short: "Delete an action.",
		Long: "Delete an action by resource name or id.\n\n" +
			"Delete is a soft delete. " + deleteTriggerNote + " The deleted action\n" +
			"no longer appears in `cocli action list`. Historical runs are unaffected\n" +
			"(they are snapshotted at run time).",
		Example: "  # Delete an action, prompting for confirmation (also disables its triggers):\n" +
			"  cocli action delete my-action -p my-project\n\n" +
			"  # Delete without the confirmation prompt:\n" +
			"  cocli action delete my-action -p my-project -f",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			// Resolve the action name/id first (client-side). The resolve-first
			// pattern gives a clean not-found message even though the backend
			// delete is idempotent (an unknown id succeeds); it depends on
			// ActionId2Name wrapping its error with %w so the NotFound code
			// survives (api/action.go, plan D7/F2).
			actionName, err := pm.ActionCli().ActionId2Name(cmd.Context(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				io.Printf("failed to find action: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("failed to convert action id to name: %v", err)
			}

			// Confirm deletion unless forced. The prompt states the trigger
			// side effect so the user is not surprised (plan F8).
			if !force {
				if confirmed := prompts.PromptYN("Delete this action? "+deleteTriggerNote, io); !confirmed {
					io.Println("Delete action aborted.")
					return
				}
			}

			// Delete the action.
			if err = pm.ActionCli().DeleteAction(cmd.Context(), actionName); err != nil {
				log.Fatalf("failed to delete action: %v", err)
			}

			io.Printf("Action successfully deleted.\n")
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", force, "Force delete without confirmation")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")

	return cmd
}
