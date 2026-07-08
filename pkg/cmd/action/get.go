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
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewGetCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		projectSlug  = ""
		outputFormat = "table"
	)

	cmd := &cobra.Command{
		Use:   "get <action-resource-name/id> [-p <working-project-slug>] [-o <output-format>]",
		Short: "Get an action.",
		Long: "Get an action by resource name or id.\n\n" +
			"The -o yaml / -o json output is the full protojson Action (name, author, timestamps, spec) " +
			"and is the input format consumed by `cocli action update -f`, so `get -o yaml` can be edited " +
			"and fed back into `update` to round-trip.",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			// Resolve the action name/id. The resolve-first pattern gives a clean
			// client-side not-found message; this depends on ActionId2Name wrapping
			// its underlying error with %w so the NotFound code survives (api/action.go).
			actionName, err := pm.ActionCli().ActionId2Name(cmd.Context(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				io.Printf("failed to find action: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("failed to convert action id to name: %v", err)
			}

			// Fetch the action.
			action, err := pm.ActionCli().GetByName(cmd.Context(), actionName)
			if err != nil {
				log.Fatalf("failed to get action: %v", err)
			}

			p, err := printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{Verbose: true}})
			if err != nil {
				log.Fatal(err)
			}
			if err = p.PrintObj(printable.NewSingleAction(action), io.Out); err != nil {
				log.Fatalf("failed to print action: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")

	return cmd
}
