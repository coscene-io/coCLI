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

package record

import (
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/spf13/cobra"
)

func NewRootCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "Work with coScene record.",
	}

	cmd.AddCommand(NewCopyCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewCreateCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewDeleteCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewDescribeCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewDownloadCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewFileCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewListCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewMomentCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewMoveCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewUpdateCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewUploadCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewViewCommand(cfgPath, io, getProvider))

	return cmd
}
