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

package role

import (
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewListCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		level        = ""
		verbose      = false
		outputFormat = ""
		pageSize     = 0
		pageToken    = ""
	)

	cmd := &cobra.Command{
		Use:                   "list [--level <organization|project>] [--page-size <size>] [--page-token <token>]",
		Short:                 "List available roles",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if level != "" && level != "organization" && level != "project" {
				log.Fatalf("--level must be \"organization\" or \"project\"")
			}
			if pageSize > 0 && (pageSize < 10 || pageSize > 100) {
				log.Fatalf("--page-size must be between 10 and 100")
			}

			pm, _ := getProvider(*cfgPath).GetProfileManager()

			effectivePageSize := int32(pageSize)
			if effectivePageSize <= 0 {
				effectivePageSize = int32(constants.MaxPageSize)
			}

			result, err := pm.RoleCli().ListRoles(cmd.Context(), &api.ListRolesOptions{
				Level:     level,
				PageSize:  effectivePageSize,
				PageToken: pageToken,
			})
			if err != nil {
				log.Fatalf("unable to list roles: %v", err)
			}

			p, err := printer.Printer(outputFormat, &printer.Options{TableOpts: roleTableOpts(verbose, outputFormat)})
			if err != nil {
				log.Fatal(err)
			}
			if err = p.PrintObj(printable.NewRole(result.Roles, result.NextPageToken), io.Out); err != nil {
				log.Fatalf("unable to print roles: %v", err)
			}

			hasMorePages := result.NextPageToken != "" && len(result.Roles) >= int(effectivePageSize)
			if hasMorePages {
				io.Eprintf("\n")
				io.Eprintf("Next page available. To continue, add: --page-token \"%s\"\n", result.NextPageToken)
			}
		},
	}

	cmd.Flags().StringVar(&level, "level", "", "filter by role level (organization|project)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|wide|json|yaml)")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "number of roles per page (10-100)")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "page token for pagination (get from previous response)")

	return cmd
}
