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
	"os"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewListCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug    = ""
		verbose        = false
		includeArchive = false
		outputFormat   = ""
		pageSize       = 0
		page           = 0
		labels         []string
		titles         []string
	)

	cmd := &cobra.Command{
		Use:                   "list [-v] [-p <working-project-slug>] [--include-archive] [--page-size <size>] [--page <number>] [--labels <label1,label2>] [--keywords <keyword1,keyword2>]",
		Short:                 "List records in the project.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			// Validate pagination flags
			if page > 1 && pageSize <= 0 {
				log.Fatalf("--page requires --page-size to be specified")
			}
			if pageSize > 0 && (pageSize < 10 || pageSize > 100) {
				log.Fatalf("--page-size must be between 10 and 100")
			}

			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			var records []*openv1alpha1resource.Record

			// Prepare list options with filters
			listOptions := &api.ListRecordsOptions{
				Project:        proj,
				IncludeArchive: includeArchive,
				Labels:         labels,
				Titles:         titles,
			}

			// Use pagination if page size is specified
			if pageSize > 0 {
				// Calculate skip based on page number (page is 1-based)
				skip := 0
				if page > 1 {
					skip = (page - 1) * pageSize
				}

				records, err = pm.RecordCli().ListWithPagination(context.TODO(), listOptions, pageSize, skip)
				if err != nil {
					log.Fatalf("unable to list records: %v", err)
				}
			} else {
				// List all records (existing behavior)
				records, err = pm.RecordCli().ListAll(context.TODO(), listOptions)
				if err != nil {
					log.Fatalf("unable to list records: %v", err)
				}
			}

			// Print listed records.
			var omitFields []string
			if !includeArchive {
				omitFields = append(omitFields, "ARCHIVED")
			}
			err = printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{
				Verbose:    verbose,
				OmitFields: omitFields,
			}}).PrintObj(printable.NewRecord(records), os.Stdout)
			if err != nil {
				log.Fatalf("unable to print records: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().BoolVar(&includeArchive, "include-archive", false, "include archived records")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json)")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "number of records per page (10-100, 0 for all records)")
	cmd.Flags().IntVar(&page, "page", 1, "page number (1-based, requires --page-size)")
	cmd.Flags().StringSliceVar(&labels, "labels", []string{}, "filter by labels (comma-separated)")
	cmd.Flags().StringSliceVar(&titles, "keywords", []string{}, "filter by keywords in titles (comma-separated)")

	return cmd
}
