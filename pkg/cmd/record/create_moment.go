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
	"encoding/json"
	"fmt"
	"time"

	openv1alpha1enum "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewCreateMomentCmd(cfgPath *string) *cobra.Command {
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
	)

	cmd := &cobra.Command{
		Use:                   "create-moment <record-resource-name/id>",
		Short:                 "Create moment in the record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()
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
					Tags: map[string]string{"recordName": recordName.String()},
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

	cmd.Flags().BoolVarP(&skipCreateTask, "skip-create-task", "s", false, "Create task or not.")
	cmd.Flags().BoolVarP(&syncTask, "sync-task", "S", false, "Sync task or not.")
	return cmd
}
