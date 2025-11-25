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
	"os"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
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
		pageToken      = ""
		all            = false
		labels         []string
		titles         []string
	)

	cmd := &cobra.Command{
		Use:                   "list [-v] [-p <working-project-slug>] [--include-archive] [--page-size <size>] [--page-token <token>] [--all] [--labels <label1,label2>] [--keywords <keyword1,keyword2>]",
		Short:                 "List records in a project",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			// Validate pagination flags
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
			var nextPageToken string

			searchOptions := &api.SearchRecordsOptions{
				Project:        proj,
				IncludeArchive: includeArchive,
				Labels:         labels,
				Titles:         titles,
				OrderBy:        "",
			}

			if all {
				records, err = pm.RecordCli().SearchAll(context.TODO(), searchOptions)
				if err != nil {
					log.Fatalf("unable to search records: %v", err)
				}
			} else if page > 1 {
				fmt.Fprintf(os.Stderr, "Warning: --page is deprecated due to backend changes. Use --page-token for pagination.\n")
				fmt.Fprintf(os.Stderr, "Note: Fetching pages 1-%d sequentially (this may be slow)...\n\n", page)

				effectivePageSize := pageSize
				if effectivePageSize <= 0 {
					effectivePageSize = constants.MaxPageSize
				}
				searchOptions.PageSize = int32(effectivePageSize)

				currentPageToken := ""
				for i := 1; i <= page; i++ {
					searchOptions.PageToken = currentPageToken
					result, err := pm.RecordCli().SearchWithPageToken(context.TODO(), searchOptions)
					if err != nil {
						log.Fatalf("unable to search records: %v", err)
					}

					isEmpty := len(result.Records) == 0
					isLastPage := isEmpty || len(result.Records) < effectivePageSize || result.NextPageToken == ""

					if i == page {
						if isEmpty {
							log.Fatalf("page %d does not exist (only %d pages available)", page, i-1)
						}
						records = result.Records
						nextPageToken = result.NextPageToken
						break
					}

					if isLastPage {
						availablePages := i
						if isEmpty {
							availablePages = i - 1
						}
						log.Fatalf("page %d does not exist (only %d pages available)", page, availablePages)
					}

					currentPageToken = result.NextPageToken
				}
			} else if pageToken != "" || pageSize > 0 {
				effectivePageSize := pageSize
				if effectivePageSize <= 0 {
					effectivePageSize = constants.MaxPageSize
				}

				searchOptions.PageSize = int32(effectivePageSize)
				searchOptions.PageToken = pageToken

				result, err := pm.RecordCli().SearchWithPageToken(context.TODO(), searchOptions)
				if err != nil {
					log.Fatalf("unable to search records: %v", err)
				}

				records = result.Records
				nextPageToken = result.NextPageToken

				if pageToken != "" && len(records) == 0 {
					fmt.Fprintf(os.Stderr, "No more records. You've reached the end of results.\n")
				}
			} else {
				defaultPageSize := constants.MaxPageSize
				searchOptions.PageSize = int32(defaultPageSize)

				result, err := pm.RecordCli().SearchWithPageToken(context.TODO(), searchOptions)
				if err != nil {
					log.Fatalf("unable to search records: %v", err)
				}

				records = result.Records
				nextPageToken = result.NextPageToken
			}

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

			effectivePageSize := pageSize
			if effectivePageSize <= 0 {
				effectivePageSize = constants.MaxPageSize
			}

			hasMorePages := nextPageToken != "" && len(records) >= effectivePageSize

			if !all && hasMorePages {
				fmt.Fprintf(os.Stderr, "\n")
				fmt.Fprintf(os.Stderr, "Next page available. To continue, add: --page-token \"%s\"\n", nextPageToken)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().BoolVar(&includeArchive, "include-archive", false, "include archived records")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "number of records per page (10-100)")
	cmd.Flags().IntVar(&page, "page", 1, "[DEPRECATED] page number (use --page-token instead)")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "page token for pagination (get from previous response)")
	cmd.Flags().BoolVar(&all, "all", false, "list all records (overrides pagination)")
	cmd.Flags().StringSliceVar(&labels, "labels", []string{}, "filter by labels (comma-separated)")
	cmd.Flags().StringSliceVar(&titles, "keywords", []string{}, "filter by keywords in titles (comma-separated)")

	cmd.MarkFlagsMutuallyExclusive("all", "page-size")
	cmd.MarkFlagsMutuallyExclusive("all", "page")
	cmd.MarkFlagsMutuallyExclusive("all", "page-token")
	cmd.MarkFlagsMutuallyExclusive("page", "page-token")

	return cmd
}
