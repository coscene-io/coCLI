// Copyright 2024 coScene
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

package registry

import (
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCreateCredentialCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:                   "create-credential",
		Short:                 "Generate a temporary docker credential",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			pm, err := getProvider(*cfgPath).GetProfileManager()
			if err != nil {
				log.Fatalf("failed to load profile manager: %v", err)
			}

			cred, err := pm.ContainerRegistryCli().CreateBasicCredential(cmd.Context())
			if err != nil {
				log.Fatalf("failed to create basic credential: %v", err)
			}

			if outputFormat == "" {
				io.Printf("username: %s\n", cred.GetUsername())
				io.Printf("password: %s\n", cred.GetPassword())
				return
			}

			p := printer.Printer(outputFormat, &printer.Options{
				TableOpts: &table.PrintOpts{},
			})
			if err := p.PrintObj(printable.NewRegistryCredential(cred.GetUsername(), cred.GetPassword()), io.Out); err != nil {
				log.Fatalf("failed to print credential: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json|yaml)")

	return cmd
}
