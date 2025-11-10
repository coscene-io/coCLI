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
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/spf13/cobra"
)

func NewRootCommand(cfgPath *string, io *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage coScene container registry access",
	}

	// Registry operations don't require org/project auth checks
	cmd_utils.DisableAuthCheck(cmd)

	cmd.AddCommand(NewLoginCommand(cfgPath, io))
	cmd.AddCommand(NewCreateCredentialCommand(cfgPath, io))

	return cmd
}
