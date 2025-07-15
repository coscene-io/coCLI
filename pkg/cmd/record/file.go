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
	"fmt"
	"os"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewFileCommand(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Manage files in records",
	}

	cmd.AddCommand(NewFileListCommand(cfgPath))
	cmd.AddCommand(NewFileDeleteCommand(cfgPath))

	return cmd
}

func NewFileListCommand(cfgPath *string) *cobra.Command {
	var (
		verbose      = false
		outputFormat = ""
		projectSlug  = ""
		pageSize     = 0
		page         = 0
		all          = false
	)

	cmd := &cobra.Command{
		Use:                   "list <record-resource-name/id> [-p <working-project-slug>] [-v] [--page-size <size>] [--page <number>] [--all]",
		Short:                 "List files in the record",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Validate pagination flags
			if pageSize > 0 && (pageSize < 10 || pageSize > 100) {
				log.Fatalf("--page-size must be between 10 and 100")
			}
			if page < 1 {
				log.Fatalf("--page must be >= 1")
			}

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

			var files []*openv1alpha1resource.File

			if all {
				files, err = pm.RecordCli().ListAllFiles(context.TODO(), recordName)
				if err != nil {
					log.Fatalf("unable to list files: %v", err)
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

				files, err = pm.RecordCli().ListFilesWithPagination(context.TODO(), recordName, effectivePageSize, skip)
				if err != nil {
					log.Fatalf("unable to list files: %v", err)
				}

				// Show note when using default page size with --page
				if pageSize <= 0 && page > 1 {
					fmt.Fprintf(os.Stderr, "Note: Using default page size of %d files for page %d.\n\n", effectivePageSize, page)
				}
			} else {
				// Default behavior: use MaxPageSize and show note
				defaultPageSize := constants.MaxPageSize
				files, err = pm.RecordCli().ListFilesWithPagination(context.TODO(), recordName, defaultPageSize, 0)
				if err != nil {
					log.Fatalf("unable to list files: %v", err)
				}

				// Show note about default behavior
				if len(files) == defaultPageSize {
					fmt.Fprintf(os.Stderr, "Note: Showing first %d files (default page size). Use --all to list all files or --page-size to specify page size.\n\n", defaultPageSize)
				}
			}

			// Print listed files.
			err = printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{
				Verbose: verbose,
			}}).PrintObj(printable.NewFile(files), os.Stdout)
			if err != nil {
				log.Fatalf("unable to print files: %v", err)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json)")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "number of files per page (10-100)")
	cmd.Flags().IntVar(&page, "page", 1, "page number (1-based, requires --page-size)")
	cmd.Flags().BoolVar(&all, "all", false, "list all files (overrides default page size)")

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("all", "page-size")
	cmd.MarkFlagsMutuallyExclusive("all", "page")

	return cmd
}

func NewFileDeleteCommand(cfgPath *string) *cobra.Command {
	var (
		force       = false
		projectSlug = ""
	)

	cmd := &cobra.Command{
		Use:                   "delete <record-resource-name/id> <filename> [-p <working-project-slug>] [-f]",
		Short:                 "Delete a specific file from a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
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

			fileName := args[1]

			// Confirm deletion.
			if !force {
				if confirmed := prompts.PromptYN(fmt.Sprintf("Are you sure you want to delete the file '%s' from record?", fileName)); !confirmed {
					fmt.Println("Delete file aborted.")
					return
				}
			}

			if err = pm.RecordCli().DeleteFile(context.TODO(), recordName, fileName); err != nil {
				log.Fatalf("failed to delete file: %v", err)
			}

			fmt.Printf("File '%s' successfully deleted.\n", fileName)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", force, "Force delete without confirmation")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")

	return cmd
}
