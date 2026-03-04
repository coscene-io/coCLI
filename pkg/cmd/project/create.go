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
	"strings"

	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/prompts"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCreateCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		projectSlug    string
		displayName    string
		description    string
		templateSlug   string
		scopeStr       string
		visibility     string
		regionFlag     string
		fileSystemFlag string
		forceYes       bool
		verbose        bool
		outputFormat   string
	)
	cmd := &cobra.Command{
		Use:                   "create -p <project-slug> -n <display-name> -b <visibility> [--region <region>] [--filesystem <name>] [--template <template-slug-or-name>] [--scope <scopes>]",
		Short:                 "Create a project",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := getProvider(*cfgPath).GetProfileManager()

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

			// If template is provided, validate it exists and require scopes
			var validatedTemplateName string
			if templateSlug != "" {
				if scopeStr == "" {
					log.Fatalf("scope is required when template is provided")
				}

				if strings.HasPrefix(templateSlug, "projects/") {
					validatedTemplateName = templateSlug
				} else {
					tplProject, err := pm.ProjectCli().Name(context.Background(), templateSlug)
					if err != nil {
						log.Fatalf("failed to find template project: %v", err)
					}
					validatedTemplateName = tplProject.String()
				}

				tplName, err := name.NewProject(validatedTemplateName)
				if err != nil {
					log.Fatalf("invalid template project name: %v", err)
				}
				_, err = pm.ProjectCli().Get(context.Background(), tplName)
				if err != nil {
					log.Fatalf("template project does not exist: %v", err)
				}
			}

			// Resolve file system based on --region / --filesystem flags.
			var selectedFileSystem string
			var fileSystems []*openv1alpha1resource.FileSystem
			if fileSystemFlag != "" && regionFlag == "" {
				log.Fatalf("--filesystem requires --region")
			}
			if regionFlag != "" && templateSlug == "" {
				var fsErr error
				fileSystems, fsErr = pm.FileSystemCli().ListAllFileSystems(cmd.Context())
				if fsErr != nil {
					log.Fatalf("failed to list file systems: %v", fsErr)
				}
				selectedFileSystem = resolveFileSystem(fileSystems, regionFlag, fileSystemFlag)
			}

			// Confirm unless forced
			if !forceYes {
				summary := "Create project with the following settings:\n"
				summary += "  name: " + projectSlug + "\n"
				summary += "  display_name: " + displayName + "\n"
				if templateSlug != "" {
					summary += "  template: " + validatedTemplateName + "\n"
				}
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
				if selectedFileSystem != "" {
					summary += "  file_system: " + selectedFileSystem + "\n"
				} else {
					summary += "  file_system: (server default)\n"
				}

				if !prompts.PromptYN(summary+"Proceed?", io) {
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
					FileSystem:  selectedFileSystem,
				})
			} else {
				projectRes, err = pm.ProjectCli().CreateProjectUsingTemplate(context.Background(), &api.CreateProjectUsingTemplateOptions{
					Parent:          "",
					Slug:            projectSlug,
					DisplayName:     displayName,
					ProjectTemplate: validatedTemplateName,
					TemplateScopes:  tplScopes,
					Visibility:      visEnum,
					Description:     description,
				})
			}
			if err != nil {
				log.Fatalf("failed to create project: %v", err)
			}

			// Build filesystem info for display
			if len(fileSystems) == 0 {
				if fetched, fetchErr := pm.FileSystemCli().ListAllFileSystems(cmd.Context()); fetchErr == nil {
					fileSystems = fetched
				}
			}
			fsInfo := make(map[string]*openv1alpha1resource.FileSystem)
			for _, fs := range fileSystems {
				fsInfo[fs.Name] = fs
			}

			// Print project.
			err = printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{Verbose: verbose}}).
				PrintObj(printable.NewProjectWithFileSystemInfo([]*openv1alpha1resource.Project{projectRes}, fsInfo), io.Out)
			if err != nil {
				log.Fatalf("unable to print project: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project-slug", "p", "", "The slug of the project [required].")
	cmd.Flags().StringVarP(&displayName, "display-name", "n", "", "The display name of the project [required]")
	cmd.Flags().StringVarP(&description, "description", "d", "", "The description of the project")
	cmd.Flags().StringVarP(&templateSlug, "template", "t", "", "The template to use when creating the project. Can be either 'projects/uuid' or a project slug.")
	cmd.Flags().StringVarP(&scopeStr, "scope", "s", "", "Template scopes (CUSTOM_FIELDS|ACTIONS|TRIGGERS|LAYOUTS) comma separated. Required when template is provided.")
	cmd.Flags().StringVarP(&visibility, "visibility", "b", "", "Project visibility (private|internal) [required]")
	cmd.Flags().StringVar(&regionFlag, "region", "", "Storage region (e.g. cn-hangzhou)")
	cmd.Flags().StringVar(&fileSystemFlag, "filesystem", "", "File system name within the region (requires --region)")
	cmd.Flags().BoolVarP(&forceYes, "yes", "y", false, "Skip confirmation and create without prompting")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json)")

	_ = cmd.MarkFlagRequired("project-slug")
	_ = cmd.MarkFlagRequired("visibility")
	_ = cmd.MarkFlagRequired("display-name")

	return cmd
}

// resolveFileSystem finds the matching filesystem from the list based on region and optional name.
func resolveFileSystem(fileSystems []*openv1alpha1resource.FileSystem, region, fsName string) string {
	if fsName != "" {
		for _, fs := range fileSystems {
			fsID := extractFileSystemID(fs.Name)
			if fs.Region == region && fsID == fsName {
				return fs.Name
			}
		}
		log.Fatalf("no file system %q found in region %s", fsName, region)
	}

	var regionFs, defaults []*openv1alpha1resource.FileSystem
	for _, fs := range fileSystems {
		if fs.Region == region {
			regionFs = append(regionFs, fs)
			if fs.IsDefault {
				defaults = append(defaults, fs)
			}
		}
	}
	if len(regionFs) == 0 {
		log.Fatalf("no file systems available in region %s", region)
	}
	if len(defaults) == 0 {
		log.Fatalf("no default file system in region %s, specify --filesystem", region)
	}
	if len(defaults) > 1 {
		log.Warnf("multiple default file systems in region %s, using first", region)
	}
	return defaults[0].Name
}

func extractFileSystemID(name string) string {
	idx := strings.LastIndex(name, "/fileSystems/")
	if idx >= 0 {
		return name[idx+len("/fileSystems/"):]
	}
	return name
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
