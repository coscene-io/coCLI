// Copyright 2025 coScene
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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	openv1alpha1enum "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewMomentCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "moment",
		Short: "Manage moments in records",
	}

	cmd.AddCommand(NewMomentCreateCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewMomentListCommand(cfgPath, io, getProvider))
	cmd.AddCommand(NewMomentDownloadCommand(cfgPath, io, getProvider))

	return cmd
}

func NewMomentCreateCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		projectSlug                           = ""
		displayName                           = ""
		durationRaw                           = float64(0)
		customizedFieldsRaw                   = ""
		customizedFields    map[string]string = nil
		description                           = ""
		triggerTime                           = float64(0)
		assigner                              = ""
		assignee                              = ""
		skipCreateTask                        = false
		syncTask                              = false
		ruleName                              = ""
	)

	cmd := &cobra.Command{
		Use:                   "create <record-resource-name/id> [-p <working-project-slug>]",
		Short:                 "Create a moment in a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := getProvider(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			recordName, err := pm.RecordCli().RecordId2Name(cmd.Context(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				io.Printf("failed to find record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get record name from %s: %v", args[0], err)
			}
			if err = json.Unmarshal([]byte(customizedFieldsRaw), &customizedFields); err != nil {
				log.Fatalf("unable to unmarshal customized fields: %v", err)
			}
			duration, err := time.ParseDuration(fmt.Sprintf("%fs", durationRaw))
			if err != nil {
				log.Fatalf("unable to parse duration: %v", err)
			}

			eventToCreate := &openv1alpha1resource.Event{
				DisplayName: displayName,
				TriggerTime: timestamppb.New(time.Unix(
					int64(triggerTime),
					int64((triggerTime-float64(int64(triggerTime)))*1e9),
				)),
				Duration:         durationpb.New(duration),
				Description:      description,
				CustomizedFields: customizedFields,
				Record:           recordName.String(),
			}

			if ruleName != "" {
				eventToCreate.Rule = &openv1alpha1resource.DiagnosisRule{
					Name: ruleName,
				}
			}

			obtainEventRes, err := pm.EventCli().ObtainEvent(context.Background(), recordName.Project().String(), eventToCreate)
			if err != nil {
				log.Fatalf("failed to create moment: %v", err)
			}
			log.Infof("created moment: %s", obtainEventRes.GetEvent().GetName())

			if skipCreateTask {
				log.Infof("specified to skip creating task")
				return
			}
			if !obtainEventRes.GetIsNew() {
				log.Infof("moment already existed, skip creating task")
				return
			}

			taskDescription, _ := json.Marshal(map[string]interface{}{
				"root": map[string]interface{}{
					"children": []map[string]interface{}{
						{
							"children": []map[string]interface{}{
								{
									"mode":    "normal",
									"text":    fmt.Sprintln(description),
									"type":    "text",
									"version": 1,
								},
							},
							"indent":    0,
							"direction": "ltr",
							"format":    "",
							"type":      "paragraph",
							"version":   1,
						},
						{
							"children": []map[string]interface{}{
								{
									"sourceName": obtainEventRes.GetEvent().GetName(),
									"sourceType": "moment",
									"type":       "source",
									"version":    1,
								},
							},
							"direction": nil,
							"format":    "",
							"indent":    0,
							"type":      "paragraph",
							"version":   1,
						},
					},
					"direction": nil,
					"indent":    0,
					"format":    "",
					"type":      "root",
					"version":   1,
				},
			})

			upsertTaskRes, err := pm.TaskCli().UpsertTask(
				context.Background(),
				recordName.Project().String(),
				&openv1alpha1resource.Task{
					Title:       displayName,
					Description: string(taskDescription),
					Creator:     assigner,
					Assigner:    assigner,
					Assignee:    assignee,
					Category:    openv1alpha1enum.TaskCategoryEnum_COMMON,
					State:       openv1alpha1enum.TaskStateEnum_PROCESSING,
					Detail: &openv1alpha1resource.Task_CommonTaskDetail{CommonTaskDetail: &openv1alpha1resource.CommonTaskDetail{
						Related: &openv1alpha1resource.CommonTaskDetail_Event{
							Event: obtainEventRes.GetEvent().GetName(),
						},
					}},
				},
			)
			if err != nil {
				log.Fatalf("failed to upsert task: %v", err)
			}

			log.Infof("upserted task: %s", upsertTaskRes.Name)

			if syncTask {
				syncTaskRes, err := pm.TaskCli().SyncTask(context.Background(), upsertTaskRes.Name)
				if err != nil {
					log.Fatalf("failed to sync task: %v", err)
				}
				log.Infof("synced task: %s", syncTaskRes.Name)
			}

		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&displayName, "display-name", "n", "", "The name of the moment.")
	cmd.Flags().StringVarP(&description, "description", "d", "", "The description of the moment.")
	cmd.Flags().StringVarP(&customizedFieldsRaw, "customized-fields", "j", "{}", "The customized fields of the moment.")
	cmd.Flags().Float64VarP(&triggerTime, "trigger-time", "T", 0, "trigger time in seconds.")
	cmd.Flags().Float64VarP(&durationRaw, "duration", "D", 1, "The duration of the moment in seconds.")
	cmd.Flags().StringVarP(&assigner, "assigner", "a", "", "The assigner of task.")
	cmd.Flags().StringVarP(&assignee, "assignee", "e", "", "The assignee of task.")
	cmd.Flags().StringVarP(&ruleName, "rule-name", "R", "", "The name of the rule to create moment.")
	cmd.Flags().BoolVarP(&skipCreateTask, "skip-create-task", "s", false, "Create task or not.")
	cmd.Flags().BoolVarP(&syncTask, "sync-task", "S", false, "Sync task or not.")

	_ = cmd.MarkFlagRequired("display-name")
	_ = cmd.MarkFlagRequired("duration")
	_ = cmd.MarkFlagRequired("trigger-time")
	return cmd
}

func NewMomentListCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		verbose      = false
		outputFormat = ""
		projectSlug  = ""
	)

	cmd := &cobra.Command{
		Use:                   "list <record-resource-name/id> [-v] [-p <working-project-slug>]",
		Short:                 "List moments in a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := getProvider(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			recordName, err := pm.RecordCli().RecordId2Name(cmd.Context(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				io.Printf("failed to find record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get record name from %s: %v", args[0], err)
			}

			moments, err := pm.RecordCli().ListAllEvents(cmd.Context(), recordName)
			if err != nil {
				moments = []*openv1alpha1resource.Event{}
				log.Errorf("unable to list moments: %v", err)
			}

			if err = printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{
				Verbose: verbose,
			}}).PrintObj(printable.NewEvent(moments), io.Out); err != nil {
				log.Fatalf("unable to print moments: %v", err)
			}

		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json)")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")

	return cmd
}

func NewMomentDownloadCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		projectSlug = ""
		flat        = false
	)

	cmd := &cobra.Command{
		Use:                   "download <record-resource-name/id> <dst-dir> [-p <working-project-slug>] [--flat]",
		Short:                 "Download moments.json from a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := getProvider(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			recordName, err := pm.RecordCli().RecordId2Name(context.TODO(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				io.Printf("failed to find record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get record name from %s: %v", args[0], err)
			}

			dirPath, err := filepath.Abs(args[1])
			if err != nil {
				log.Fatalf("unable to get absolute path: %v", err)
			}
			if dirInfo, err := os.Stat(dirPath); err != nil {
				log.Fatalf("Error checking destination directory: %v", err)
			} else if !dirInfo.IsDir() {
				log.Fatalf("Destination directory is not a directory: %s", dirPath)
			}

			var dstDir string
			if flat {
				dstDir = dirPath
			} else {
				dstDir = filepath.Join(dirPath, recordName.RecordID)
			}

			io.Println("-------------------------------------------------------------")
			io.Printf("Downloading moments for record %s\n", recordName.RecordID)
			recordUrl, err := pm.GetRecordUrl(cmd.Context(), recordName)
			if err == nil {
				io.Println("View record at:", recordUrl)
			} else {
				log.Errorf("unable to get record url: %v", err)
			}
			io.Printf("Saving to %s\n", dstDir)

			moments, err := pm.RecordCli().ListAllMoments(cmd.Context(), recordName)
			if err != nil {
				log.Fatalf("unable to list moments: %v", err)
			}

			if err = cmd_utils.SaveMomentsJson(moments, dstDir); err != nil {
				log.Fatalf("unable to save moments: %v", err)
			}

			io.Printf("\nDownload completed! moments.json saved to %s\n", filepath.Join(dstDir, "moments.json"))
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().BoolVar(&flat, "flat", false, "download directly to the specified directory without creating a subdirectory named with record-id")

	return cmd
}
