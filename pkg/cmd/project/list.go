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
	"context"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewListCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		verbose        = false
		outputFormat   = ""
		pageSize       = 0
		page           = 0
		all            = false
		keywords       []string
		includeArchive = false
	)

	cmd := &cobra.Command{
		Use:                   "list [-v] [--page-size <size>] [--page <number>] [--all] [--keywords <keyword1,keyword2>] [--include-archive]",
		Short:                 "List projects in the current organization",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			// Validate pagination flags
			if pageSize > 0 && (pageSize < 10 || pageSize > 100) {
				log.Fatalf("--page-size must be between 10 and 100")
			}
			if page < 1 {
				log.Fatalf("--page must be >= 1")
			}

			pm, _ := getProvider(*cfgPath).GetProfileManager()

			opts := &api.ListProjectsOptions{
				DisplayNames:   keywords,
				IncludeArchive: includeArchive,
			}

			var projects []*openv1alpha1resource.Project
			var err error

			if all {
				projects, err = pm.ProjectCli().ListAllUserProjects(context.Background(), opts)
				if err != nil {
					log.Fatalf("unable to list projects: %v", err)
				}
			} else if pageSize > 0 || page > 1 {
				effectivePageSize := pageSize
				if effectivePageSize <= 0 {
					effectivePageSize = constants.MaxPageSize
				}

				skip := 0
				if page > 1 {
					skip = (page - 1) * effectivePageSize
				}

				projects, err = pm.ProjectCli().ListProjectsWithPagination(context.Background(), effectivePageSize, skip, opts)
				if err != nil {
					log.Fatalf("unable to list projects: %v", err)
				}

				if pageSize <= 0 && page > 1 {
					io.Eprintf("Note: Using default page size of %d projects for page %d.\n\n", effectivePageSize, page)
				}
			} else {
				defaultPageSize := constants.MaxPageSize
				projects, err = pm.ProjectCli().ListProjectsWithPagination(context.Background(), defaultPageSize, 0, opts)
				if err != nil {
					log.Fatalf("unable to list projects: %v", err)
				}

				if len(projects) == defaultPageSize {
					io.Eprintf("Note: Showing first %d projects (default page size). Use --all to list all projects or --page-size to specify page size.\n\n", defaultPageSize)
				}
			}

			// Print listed projects.
			err = printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{
				Verbose: verbose,
			}}).PrintObj(printable.NewProject(projects), io.Out)
			if err != nil {
				log.Fatalf("unable to print projects: %v", err)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json)")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "number of projects per page (10-100)")
	cmd.Flags().IntVar(&page, "page", 1, "page number (1-based)")
	cmd.Flags().BoolVar(&all, "all", false, "list all projects (overrides default page size)")
	cmd.Flags().StringSliceVar(&keywords, "keywords", []string{}, "filter by keywords in project name (comma-separated)")
	cmd.Flags().BoolVar(&includeArchive, "include-archive", false, "include archived projects")

	cmd.MarkFlagsMutuallyExclusive("all", "page-size")
	cmd.MarkFlagsMutuallyExclusive("all", "page")

	return cmd
}
