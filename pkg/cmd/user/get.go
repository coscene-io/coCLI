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

package user

import (
	"fmt"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewGetCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		verbose      = false
		outputFormat = ""
	)

	cmd := &cobra.Command{
		Use:                   "get [<user-resource-name/id>] [-o <output-format>]",
		Short:                 "Get user details",
		Long:                  "Get details of a specific user. If no argument is given, gets the current authenticated user.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := getProvider(*cfgPath).GetProfileManager()

			userName := "users/current"
			if len(args) == 1 {
				arg := args[0]
				if _, err := name.NewUser(arg); err != nil {
					userName = fmt.Sprintf("users/%s", arg)
				} else {
					userName = arg
				}
			}

			user, err := pm.UserCli().GetUser(cmd.Context(), userName)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				if userName == "users/current" {
					io.Printf("unable to get current user, please specify a user name or id\n")
				} else {
					io.Printf("user not found: %s\n", userName)
				}
				return
			} else if err != nil {
				log.Fatalf("unable to get user: %v", err)
			}

			p, err := printer.Printer(outputFormat, &printer.Options{TableOpts: userTableOpts(verbose, outputFormat)})
			if err != nil {
				log.Fatal(err)
			}
			if err = p.PrintObj(printable.NewSingleUser(user), io.Out); err != nil {
				log.Fatalf("unable to print user: %v", err)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|wide|json|yaml)")

	return cmd
}
