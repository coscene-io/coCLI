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

package project

import (
	"context"
	"fmt"
	"os"
	"strings"

	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/prompts"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCreateCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug  string
		displayName  string
		description  string
		templateSlug string
		scopeStr     string
		visibility   string
		forceYes     bool
		verbose      bool
		outputFormat string
	)
	cmd := &cobra.Command{
		Use:                   "create -p <project-slug> -n <display-name> -b <visibility> [--template <template-slug>] [--description <description>]",
		Short:                 "Create a project.",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			if projectSlug == "" {
				log.Fatalf("project name cannot be empty")
			}

			var (
				projectRes *openv1alpha1resource.Project
				err        error
			)

			// Visibility is required
			if visibility != "private" && visibility != "internal" {
				log.Fatalf("visibility must be one of: private, internal")
			}
			var visEnum openv1alpha1enums.ProjectVisibilityEnum_ProjectVisibility
			if visibility == "private" {
				visEnum = openv1alpha1enums.ProjectVisibilityEnum_PRIVATE
			} else {
				visEnum = openv1alpha1enums.ProjectVisibilityEnum_INTERNAL
			}

			// Confirm unless forced
			if !forceYes {
				// Build a short summary for confirmation
				summary := "Create project with the following settings:\n"
				summary += "  name: " + projectSlug + "\n"
				summary += "  display_name: " + displayName + "\n"
				if templateSlug != "" {
					summary += "  template: " + templateSlug + "\n"
				}
				// scopes shown only for template. if provided without template, show note
				if scopeStr != "" {
					if templateSlug == "" {
						summary += "  scopes: " + scopeStr + " (ignored without template)\n"
					} else {
						summary += "  scopes: " + scopeStr + "\n"
					}
				}
				if description != "" {
					summary += "  description: " + description + "\n"
				}
				summary += "  visibility: " + visibility + "\n"

				if !prompts.PromptYN(summary + "Proceed?") {
					log.Fatalf("aborted by user")
				}
			}

			// Prepare template scopes if template is provided
			var tplScopes []openv1alpha1service.CreateProjectUsingTemplateRequest_TemplateScope
			if templateSlug == "" {
				if scopeStr != "" {
					log.Warnf("scope is ignored when template is not provided")
				}
			} else if scopeStr != "" {
				// Parse comma-separated scopes
				parsed, parseErr := parseTemplateScopes(scopeStr)
				if parseErr != nil {
					log.Fatalf("invalid scope: %v", parseErr)
				}
				tplScopes = parsed
			}

			// Create project either directly or using a template.
			if templateSlug == "" {
				projectRes, err = pm.ProjectCli().CreateProject(context.Background(), &api.CreateProjectOptions{
					Slug:        projectSlug,
					DisplayName: displayName,
					Visibility:  visEnum,
					Description: description,
				})
			} else {
				projectRes, err = pm.ProjectCli().CreateProjectUsingTemplate(context.Background(), &api.CreateProjectUsingTemplateOptions{
					Parent:          "",
					Slug:            projectSlug,
					DisplayName:     displayName,
					ProjectTemplate: templateSlug,
					TemplateScopes:  tplScopes,
					Visibility:      visEnum,
					Description:     description,
				})
			}
			if err != nil {
				log.Fatalf("failed to create project: %v", err)
			}

			// Print project.
			err = printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{Verbose: verbose}}).
				PrintObj(printable.NewProject([]*openv1alpha1resource.Project{projectRes}), os.Stdout)
			if err != nil {
				log.Fatalf("unable to print project: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project-slug", "p", "", "The slug of the project [required].")
	cmd.Flags().StringVarP(&displayName, "display-name", "n", "", "The display name of the project [required]")
	cmd.Flags().StringVarP(&description, "description", "d", "", "The description of the project")
	cmd.Flags().StringVarP(&templateSlug, "template", "t", "", "The template to use when creating the project.")
	cmd.Flags().StringVarP(&scopeStr, "scope", "s", "", "Template scopes (unused; reserved).")
	cmd.Flags().StringVarP(&visibility, "visibility", "b", "", "Project visibility (private|internal) [required]")
	cmd.Flags().BoolVarP(&forceYes, "yes", "y", false, "Skip confirmation and create without prompting")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json)")

	// Required flags
	_ = cmd.MarkFlagRequired("project-slug")
	_ = cmd.MarkFlagRequired("visibility")
	_ = cmd.MarkFlagRequired("display-name")

	return cmd
}

// parseTemplateScopes parses a comma-separated scopes string into the enum slice.
// Allowed values: CUSTOM_FIELDS, ACTIONS, TRIGGERS, LAYOUTS
func parseTemplateScopes(scopes string) ([]openv1alpha1service.CreateProjectUsingTemplateRequest_TemplateScope, error) {
	if scopes == "" {
		return nil, nil
	}
	items := strings.Split(scopes, ",")
	ret := make([]openv1alpha1service.CreateProjectUsingTemplateRequest_TemplateScope, 0, len(items))
	for _, raw := range items {
		v := strings.TrimSpace(strings.ToUpper(raw))
		switch v {
		case "CUSTOM_FIELDS":
			ret = append(ret, openv1alpha1service.CreateProjectUsingTemplateRequest_CUSTOM_FIELDS)
		case "ACTIONS":
			ret = append(ret, openv1alpha1service.CreateProjectUsingTemplateRequest_ACTIONS)
		case "TRIGGERS":
			ret = append(ret, openv1alpha1service.CreateProjectUsingTemplateRequest_TRIGGERS)
		case "LAYOUTS":
			ret = append(ret, openv1alpha1service.CreateProjectUsingTemplateRequest_LAYOUTS)
		default:
			return nil, fmt.Errorf("unsupported scope: %s", raw)
		}
	}
	return ret, nil
}
