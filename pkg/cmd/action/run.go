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

package action

import (
	"fmt"

	openv1alpha1commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func NewRunCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		params      = map[string]string{}
		skipParams  = false
		force       = false
		projectSlug = ""
	)

	cmd := &cobra.Command{
		Use:                   "run <action-resource-name/id> <record-resource-name/id> [-p <working-project-slug>] [-P <key1=value1>...] [--skip-params] [-f]",
		Short:                 "Create an action run.",
		Args:                  cobra.ExactArgs(2),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			// Handle args and flags.
			actionName, err := pm.ActionCli().ActionId2Name(cmd.Context(), args[0], proj)
			if err != nil {
				log.Fatalf("failed to convert action id to name: %v", err)
			}
			recordName, err := pm.RecordCli().RecordId2Name(cmd.Context(), args[1], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				io.Printf("failed to find record: %s in project: %s\n", args[1], proj)
				return
			} else if err != nil {
				log.Fatalf("failed to convert record id to name: %v", err)
			}
			act, err := pm.ActionCli().GetByName(cmd.Context(), actionName)
			if err != nil {
				log.Fatalf("failed to get action by name %s: %v", actionName, err)
			}

			var runParams map[string]string
			if !skipParams {
				if cmd.Flags().Changed("param") {
					runParams = params
				} else {
					runParams = promptActionRunParameters(act.Spec.Parameters, prompts.PromptString)
				}
			}

			// Print final parameters
			io.Println("\nThe final parameters in the action run to be created:")
			if skipParams || len(runParams) == 0 {
				io.Println("Using default parameters configured on the server.")
			} else {
				for k, v := range runParams {
					io.Printf("%s: %s\n", k, v)
				}
			}

			// Prompt user for confirmation
			if !force {
				if !prompts.PromptYN("Confirm to run action?", io) {
					io.Println("Action run creation aborted.")
					return
				}
			}

			// Create action run
			err = pm.ActionCli().CreateActionRun(cmd.Context(), newActionRunAction(act, runParams), recordName)
			if err != nil {
				log.Fatalf("failed to create action run: %v", err)
			}

			io.Println("Action run created successfully.")
		},
	}

	cmd.Flags().StringToStringVarP(&params, "param", "P", nil, "action parameters")
	cmd.Flags().BoolVar(&skipParams, "skip-params", false, "skip parameter input and use default values")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force create action run without confirmation")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")

	_ = cmd.MarkFlagRequired("record")
	cmd.MarkFlagsMutuallyExclusive("skip-params", "param")

	return cmd
}

func promptActionRunParameters(defaults map[string]string, prompt func(string, string) string) map[string]string {
	overrides := make(map[string]string)
	for key, defaultValue := range defaults {
		value := prompt(fmt.Sprintf("Enter value for parameter %s", key), defaultValue)
		if value != defaultValue {
			overrides[key] = value
		}
	}
	return overrides
}

func newActionRunAction(action *openv1alpha1resource.Action, parameters map[string]string) *openv1alpha1resource.Action {
	runAction := proto.Clone(action).(*openv1alpha1resource.Action)
	if runAction.Spec == nil {
		runAction.Spec = &openv1alpha1commons.ActionSpec{}
	}
	runAction.Spec.Parameters = parameters
	return runAction
}
