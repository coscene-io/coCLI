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

package login

import (
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/spf13/cobra"
)

func NewListCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		verbose = false
	)
	cmd := &cobra.Command{
		Use:                   "list [-v]",
		Short:                 "List all login profiles.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getProvider(*cfgPath)
			pm, _ := cfg.GetProfileManager()

			profiles := pm.GetProfiles()
			if len(profiles) == 0 {
				io.Println("No profiles found.")
				return
			}

			io.Printf("%d profiles found as the following.\n", len(profiles))
			io.Println("current profile is marked with *.")
			for _, profile := range profiles {
				if profile.Name == pm.GetCurrentProfile().Name {
					io.Println(profile.StringWithOpts(true, verbose))
				} else {
					io.Println(profile.StringWithOpts(false, verbose))
				}
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show more details of the profiles.")

	return cmd
}
