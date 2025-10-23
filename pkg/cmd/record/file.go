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
	"strings"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/samber/lo"
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
	cmd.AddCommand(NewFileCopyCommand(cfgPath))
	cmd.AddCommand(NewFileMoveCommand(cfgPath))

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
		keywords     = ""
	)

	cmd := &cobra.Command{
		Use:                   "list <record-resource-name/id> [-p <working-project-slug>] [-v] [--page-size <size>] [--page <number>] [--all] [--keywords <path>]",
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
				if keywords != "" {
					files, err = pm.RecordCli().ListAllFilesWithFilter(context.TODO(), recordName, fmt.Sprintf("path=\"%s\"", keywords))
				} else {
					files, err = pm.RecordCli().ListAllFiles(context.TODO(), recordName)
				}
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

				if keywords != "" {
					files, err = pm.RecordCli().ListFilesWithPaginationAndFilter(context.TODO(), recordName, effectivePageSize, skip, fmt.Sprintf("path=\"%s\"", keywords))
				} else {
					files, err = pm.RecordCli().ListFilesWithPagination(context.TODO(), recordName, effectivePageSize, skip)
				}
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
				if keywords != "" {
					files, err = pm.RecordCli().ListFilesWithPaginationAndFilter(context.TODO(), recordName, defaultPageSize, 0, fmt.Sprintf("path=\"%s\"", keywords))
				} else {
					files, err = pm.RecordCli().ListFilesWithPagination(context.TODO(), recordName, defaultPageSize, 0)
				}
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
	cmd.Flags().StringVar(&keywords, "keywords", "", "filter files by path (e.g., 'myfile.txt' or 'folder/file')")

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("all", "page-size")
	cmd.MarkFlagsMutuallyExclusive("all", "page")

	return cmd
}

func NewFileDeleteCommand(cfgPath *string) *cobra.Command {
	var (
		force       = false
		projectSlug = ""
		fileNames   []string
	)

	cmd := &cobra.Command{
		Use:                   "delete <record-resource-name/id> [<filename>] [-p <working-project-slug>] [--files <file1,file2,...>] [-f]",
		Short:                 "Delete file(s) or directory from a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := config.Provide(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			recordName, err := pm.RecordCli().RecordId2Name(context.TODO(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				fmt.Printf("failed to find record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get record name from %s: %v", args[0], err)
			}

			var filesToDelete []string
			if len(args) == 2 {
				filesToDelete = append(filesToDelete, args[1])
			}
			if len(fileNames) > 0 {
				filesToDelete = append(filesToDelete, fileNames...)
			}

			if len(filesToDelete) == 0 {
				log.Fatalf("must specify at least one file to delete")
			}

			// Expand directories
			var finalFilesToDelete []string
			for _, fileName := range filesToDelete {
				if strings.HasSuffix(fileName, "/") {
					allFiles, err := pm.RecordCli().ListAllFiles(context.TODO(), recordName)
					if err != nil {
						log.Fatalf("unable to list record files: %v", err)
					}
					for _, f := range allFiles {
						recordFile, err := name.NewFile(f.Name)
						if err != nil {
							continue
						}
						if strings.HasPrefix(recordFile.Filename, fileName) {
							finalFilesToDelete = append(finalFilesToDelete, recordFile.Filename)
						}
					}
				} else {
					finalFilesToDelete = append(finalFilesToDelete, fileName)
				}
			}

			if len(finalFilesToDelete) == 0 {
				fmt.Println("No files found to delete.")
				return
			}

			// Confirm deletion
			if !force {
				fmt.Printf("About to delete %d file(s) from record:\n", len(finalFilesToDelete))
				for _, f := range finalFilesToDelete {
					fmt.Printf("  - %s\n", f)
				}
				if confirmed := prompts.PromptYN("Do you want to continue?"); !confirmed {
					fmt.Println("Delete file aborted.")
					return
				}
			}

			// Build full resource names for batch delete
			resourceNames := make([]string, len(finalFilesToDelete))
			for i, fileName := range finalFilesToDelete {
				resourceNames[i] = name.File{
					ProjectID: recordName.ProjectID,
					RecordID:  recordName.RecordID,
					Filename:  fileName,
				}.String()
			}

			// Always use batch delete for consistency
			if err := pm.FileCli().BatchDeleteFiles(context.TODO(), recordName.String(), resourceNames); err != nil {
				log.Fatalf("failed to delete files: %v", err)
			}

			fmt.Printf("Successfully deleted %d file(s).\n", len(finalFilesToDelete))
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", force, "Force delete without confirmation")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringSliceVar(&fileNames, "files", []string{}, "additional files to delete (comma-separated)")

	return cmd
}

func NewFileCopyCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug    = ""
		dstProjectSlug = ""
		fileNames      []string
		force          = false
	)

	cmd := &cobra.Command{
		Use:                   "copy <source-record-resource-name/id> <destination-record-resource-name/id> [-p <working-project-slug>] [-P <dst-project-slug>] [--files <filename1,filename2,...>] [-f]",
		Short:                 "Copy files from one record to another",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			// Get working project.
			proj, err := pm.ProjectName(context.TODO(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			// Handle args and flags.
			sourceRecordName, err := pm.RecordCli().RecordId2Name(context.TODO(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				fmt.Printf("failed to find source record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get source record name from %s: %v", args[0], err)
			}

			// Determine destination project - use dst project if specified, otherwise use source project
			destProject := proj
			if dstProjectSlug != "" {
				destProject, err = pm.ProjectName(context.TODO(), dstProjectSlug)
				if err != nil {
					log.Fatalf("unable to get destination project name: %v", err)
				}
			}

			destRecordName, err := pm.RecordCli().RecordId2Name(context.TODO(), args[1], destProject)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				fmt.Printf("failed to find destination record: %s in project: %s\n", args[1], destProject)
				return
			} else if err != nil {
				log.Fatalf("unable to get destination record name from %s: %v", args[1], err)
			}

			// Determine which files to copy
			var allFiles []*openv1alpha1resource.File
			if len(fileNames) > 0 {
				allFiles = lo.Map(fileNames, func(fileName string, _ int) *openv1alpha1resource.File {
					return &openv1alpha1resource.File{
						Filename: fileName,
					}
				})
			} else {
				log.Fatalf("either --all or --files must be specified")
			}

			if len(allFiles) == 0 {
				fmt.Println("No files found to copy.")
				return
			}

			// Show confirmation
			if !force {
				fmt.Printf("About to copy %d files from %s to %s.\n", len(allFiles), sourceRecordName, destRecordName)
				for _, file := range allFiles {
					fmt.Printf("  - %s\n", file.Filename)
				}

				yn := prompts.PromptYN("Do you want to continue?")
				if !yn {
					fmt.Println("Copy operation cancelled.")
					return
				}
			}

			// Perform the copy operation (server will handle authorization)
			err = pm.RecordCli().CopyFiles(context.TODO(), sourceRecordName, destRecordName, allFiles)
			if err != nil {
				log.Fatalf("failed to copy files: %v", err)
			}

			fmt.Printf("Successfully copied %d files to %s.\n", len(allFiles), destRecordName)

			// Display destination record URL
			destRecordUrl, err := pm.GetRecordUrl(destRecordName)
			if err != nil {
				log.Errorf("unable to get destination record url: %v", err)
			} else {
				fmt.Printf("View copied files at: %s\n", destRecordUrl)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&dstProjectSlug, "dst-project", "P", "", "destination project slug (defaults to source project)")
	cmd.Flags().StringSliceVar(&fileNames, "files", []string{}, "exact filenames to copy (can specify multiple, comma-separated)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force copy without confirmation")

	return cmd
}

func NewFileMoveCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug    = ""
		dstProjectSlug = ""
		fileNames      []string
		force          = false
	)

	cmd := &cobra.Command{
		Use:                   "move <source-record-resource-name/id> <destination-record-resource-name/id> [-p <working-project-slug>] [-P <dst-project-slug>] [--files <filename1,filename2,...>] [-f]",
		Short:                 "Move files from one record to another",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			// Get working project.
			proj, err := pm.ProjectName(context.TODO(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			sourceRecordName, err := pm.RecordCli().RecordId2Name(context.TODO(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				fmt.Printf("failed to find source record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get source record name from %s: %v", args[0], err)
			}

			// Determine destination project - use dst project if specified, otherwise use source project
			destProject := proj
			if dstProjectSlug != "" {
				destProject, err = pm.ProjectName(context.TODO(), dstProjectSlug)
				if err != nil {
					log.Fatalf("unable to get destination project name: %v", err)
				}
			}

			destRecordName, err := pm.RecordCli().RecordId2Name(context.TODO(), args[1], destProject)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				fmt.Printf("failed to find destination record: %s in project: %s\n", args[1], destProject)
				return
			} else if err != nil {
				log.Fatalf("unable to get destination record name from %s: %v", args[1], err)
			}

			var allFiles []*openv1alpha1resource.File
			if len(fileNames) > 0 {
				allFiles = lo.Map(fileNames, func(fileName string, _ int) *openv1alpha1resource.File {
					return &openv1alpha1resource.File{
						Filename: fileName,
					}
				})
			} else {
				log.Fatalf("either --all or --files must be specified")
			}

			if len(allFiles) == 0 {
				fmt.Println("No files found to move.")
				return
			}

			if !force {
				fmt.Printf("About to move %d files from %s to %s.\n", len(allFiles), sourceRecordName, destRecordName)
				for _, file := range allFiles {
					fmt.Printf("  - %s\n", file.Filename)
				}

				yn := prompts.PromptYN("Do you want to continue?")
				if !yn {
					fmt.Println("Move operation cancelled.")
					return
				}
			}

			err = pm.RecordCli().MoveFiles(context.TODO(), sourceRecordName, destRecordName, allFiles)
			if err != nil {
				log.Fatalf("failed to move files: %v", err)
			}

			fmt.Printf("Successfully moved %d files to %s.\n", len(allFiles), destRecordName)

			destRecordUrl, err := pm.GetRecordUrl(destRecordName)
			if err != nil {
				log.Errorf("unable to get destination record url: %v", err)
			} else {
				fmt.Printf("View moved files at: %s\n", destRecordUrl)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&dstProjectSlug, "dst-project", "P", "", "destination project slug (defaults to source project)")
	cmd.Flags().StringSliceVar(&fileNames, "files", []string{}, "exact filenames to move (can specify multiple, comma-separated)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force move without confirmation")

	return cmd
}
