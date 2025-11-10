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
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/spf13/cobra"
)

func NewRootCommand(cfgPath *string, io *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "Work with coScene record.",
	}

	cmd.AddCommand(NewCopyCommand(cfgPath, io))
	cmd.AddCommand(NewCreateCommand(cfgPath, io))
	cmd.AddCommand(NewDeleteCommand(cfgPath, io))
	cmd.AddCommand(NewDescribeCommand(cfgPath, io))
	cmd.AddCommand(NewDownloadCommand(cfgPath, io))
	cmd.AddCommand(NewFileCommand(cfgPath, io))
	cmd.AddCommand(NewListCommand(cfgPath, io))
	cmd.AddCommand(NewMomentCommand(cfgPath, io))
	cmd.AddCommand(NewMoveCommand(cfgPath, io))
	cmd.AddCommand(NewUpdateCommand(cfgPath, io))
	cmd.AddCommand(NewUploadCommand(cfgPath, io))
	cmd.AddCommand(NewViewCommand(cfgPath, io))

	return cmd
}
