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
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewDeleteCommand(cfgPath *string) *cobra.Command {
	var (
		force       = false
		projectSlug = ""
		fileName    = ""
	)

	cmd := &cobra.Command{
		Use:                   "delete <record-resource-name/id> [-p <working-project-slug>] [-f] [--file <filename>]",
		Short:                 "Delete a record or a specific file from a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			// Handle args and flags.
			recordName, err := pm.RecordCli().RecordId2Name(context.TODO(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				fmt.Printf("failed to find record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get record name from %s: %v", args[0], err)
			}

			if fileName != "" {
				// Delete specific file from record
				if !force {
					if confirmed := prompts.PromptYN(fmt.Sprintf("Are you sure you want to delete the file '%s' from record?", fileName)); !confirmed {
						fmt.Println("Delete file aborted.")
						return
					}
				}

				if err = pm.RecordCli().DeleteFile(context.TODO(), recordName, fileName); err != nil {
					log.Fatalf("failed to delete file: %v", err)
				}

				fmt.Printf("File successfully deleted.\n")
			} else {
				// Delete entire record
				if !force {
					if confirmed := prompts.PromptYN("Are you sure you want to delete the record?"); !confirmed {
						fmt.Println("Delete record aborted.")
						return
					}
				}

				// Delete record.
				if err = pm.RecordCli().Delete(context.TODO(), recordName); err != nil {
					log.Fatalf("failed to delete record: %v", err)
				}

				fmt.Printf("Record successfully deleted.\n")
			}
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", force, "Force delete without confirmation")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVar(&fileName, "file", "", "filename to delete from the record (use with record list-files to see available files)")

	return cmd
}
