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

package project

import (
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/spf13/cobra"
)

func NewRootCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Work with coScene project.",
	}

	cmd.AddCommand(NewListCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewCreateCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewFileCommand(cfgPath, io, getProvider))
	return cmd
}
