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
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCopyCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug = ""
		dstProject  = ""
		force       = false
	)

	cmd := &cobra.Command{
		Use:                   "copy <record-resource-name/id> [-p <working-project-slug>] [-P <dst-project-slug>] [-f]",
		Short:                 "Copy a record to target project",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			// Get working project.
			proj, err := pm.ProjectName(context.TODO(), projectSlug)
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
			var (
				dstProjectName *name.Project
			)
			if len(dstProject) != 0 {
				dstProjectName, err = pm.ProjectName(context.TODO(), dstProject)
				if err != nil {
					log.Fatalf("failed to get destination project name: %v", err)
				}
			} else {
				if len(projectSlug) != 0 {
					dstProject = projectSlug
				} else {
					dstProject = pm.GetCurrentProfile().ProjectSlug
				}
				dstProjectName = proj
			}

			// Show operation and confirm
			if len(dstProject) != 0 {
				fmt.Printf("Will copy entire record %s to project %s\n", recordName.RecordID, dstProject)
			} else {
				fmt.Printf("Will copy entire record %s to current project %s\n", recordName.RecordID, dstProject)
			}

			if !force {
				if confirmed := prompts.PromptYN("Are you sure you want to proceed with this copy operation?"); !confirmed {
					fmt.Println("Copy operation aborted.")
					return
				}
			}

			// Copy record.
			var copiedRecordName *name.Record
			if len(dstProject) != 0 {
				copied, err := pm.RecordCli().Copy(context.TODO(), recordName, dstProjectName)
				if err != nil {
					log.Fatalf("failed to copy record: %v", err)
				}

				fmt.Printf("Record successfully copied to %s.\n", copied.Name)
				copiedRecordName, _ = name.NewRecord(copied.Name)
			}

			copiedRecordUrl, err := pm.GetRecordUrl(copiedRecordName)
			if err != nil {
				log.Errorf("unable to get record url: %v", err)
			} else {
				fmt.Println("View copied record at:", copiedRecordUrl)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&dstProject, "dst-project", "P", "", "destination project slug")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force copy without confirmation")

	return cmd
}
