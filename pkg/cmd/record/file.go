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
	"path/filepath"
	"strings"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/fs"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
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
	cmd.AddCommand(NewFileDownloadCommand(cfgPath))
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
		dir          = ""
	)

	cmd := &cobra.Command{
		Use:                   "list <record-resource-name/id> [-p <working-project-slug>] [-v] [--page-size <size>] [--page <number>] [--all] [--dir <path>]",
		Short:                 "List files and directories in a record",
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

			// Build filter
			var filterStr string
			if dir != "" {
				// Normalize: ensure no trailing slash for filter consistency
				normalizedDir := strings.TrimSuffix(dir, "/")
				filterStr = fmt.Sprintf("dir=\"%s\"", normalizedDir)
			}

			if all {
				if filterStr != "" {
					files, err = pm.RecordCli().ListAllFilesWithFilter(context.TODO(), recordName, filterStr)
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

				if filterStr != "" {
					files, err = pm.RecordCli().ListFilesWithPaginationAndFilter(context.TODO(), recordName, effectivePageSize, skip, filterStr)
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
				if filterStr != "" {
					files, err = pm.RecordCli().ListFilesWithPaginationAndFilter(context.TODO(), recordName, defaultPageSize, 0, filterStr)
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

			// Strip directory prefix from display if --dir was specified
			if dir != "" {
				normalizedPrefix := strings.TrimSuffix(dir, "/") + "/"
				for _, f := range files {
					f.Filename = strings.TrimPrefix(f.Filename, normalizedPrefix)
				}
			}

			// Print listed files and directories.
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
	cmd.Flags().StringVarP(&dir, "dir", "d", "", "filter by directory path")

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("all", "page-size")
	cmd.MarkFlagsMutuallyExclusive("all", "page")

	return cmd
}

func NewFileDownloadCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug = ""
		maxRetries  = 0
		dir         = ""
		fileNames   []string
		flat        = false
	)

	cmd := &cobra.Command{
		Use:                   "download <record-resource-name/id> <dst-dir> [-p <working-project-slug>] [--dir <path>] [--files <file1,file2,...>] [--flat]",
		Short:                 "Download files or directory from a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
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

			dirPath, err := filepath.Abs(args[1])
			if err != nil {
				log.Fatalf("unable to get absolute path: %v", err)
			}
			if dirInfo, err := os.Stat(dirPath); err != nil {
				log.Fatalf("Error checking destination directory: %v", err)
			} else if !dirInfo.IsDir() {
				log.Fatalf("Destination directory is not a directory: %s", dirPath)
			}

			// List files based on filters
			var files []*openv1alpha1resource.File
			if dir != "" {
				// Download specific directory recursively
				normalizedDir := strings.TrimSuffix(dir, "/")
				files, err = pm.RecordCli().ListAllFilesWithFilter(context.TODO(), recordName, fmt.Sprintf("dir=\"%s\" AND recursive=\"true\"", normalizedDir))
				if err != nil {
					log.Fatalf("unable to list record files: %v", err)
				}
			} else if len(fileNames) > 0 {
				// Download specific files
				for _, fileName := range fileNames {
					resourceName := name.File{ProjectID: recordName.ProjectID, RecordID: recordName.RecordID, Filename: fileName}.String()
					fileInfo, err := pm.FileCli().GetFile(context.TODO(), resourceName)
					if err != nil {
						log.Warnf("unable to get file %s: %v, skipping", fileName, err)
						continue
					}
					files = append(files, fileInfo)
				}
			} else {
				// Download all files
				files, err = pm.RecordCli().ListAllFiles(context.TODO(), recordName)
				if err != nil {
					log.Fatalf("unable to list files: %v", err)
				}
			}

			if len(files) == 0 {
				fmt.Println("No files found to download.")
				return
			}

			// Filter out directory markers before downloading
			var filesToDownload []*openv1alpha1resource.File
			for _, f := range files {
				fileName, err := name.NewFile(f.Name)
				if err != nil {
					log.Warnf("unable to parse file name %s: %v, skipping", f.Name, err)
					continue
				}
				if !strings.HasSuffix(fileName.Filename, "/") {
					filesToDownload = append(filesToDownload, f)
				}
			}

			if len(filesToDownload) == 0 {
				fmt.Println("No files to download (only directories found).")
				return
			}

			var dstDir string
			if flat {
				dstDir = dirPath
			} else {
				dstDir = filepath.Join(dirPath, recordName.RecordID)
			}
			fmt.Println("-------------------------------------------------------------")
			fmt.Printf("Downloading record files from %s\n", recordName.RecordID)
			recordUrl, err := pm.GetRecordUrl(recordName)
			if err == nil {
				fmt.Println("View record at:", recordUrl)
			} else {
				log.Errorf("unable to get record url: %v", err)
			}
			fmt.Printf("Saving to %s\n", dstDir)

			totalFiles := len(filesToDownload)
			successCount := 0
			for fIdx, f := range filesToDownload {
				fileName, err := name.NewFile(f.Name)
				if err != nil {
					log.Errorf("unable to parse file name %s: %v", f.Name, err)
					continue
				}

				localPath := filepath.Join(dstDir, fileName.Filename)
				fmt.Printf("\nDownloading #%d file: %s\n", fIdx+1, fileName.Filename)

				if !strings.HasPrefix(localPath, dstDir+string(os.PathSeparator)) {
					log.Errorf("illegal file name: %s", fileName.Filename)
					continue
				}

				// Check if local file exists with same checksum and size
				if _, err := os.Stat(localPath); err == nil {
					checksum, size, err := fs.CalSha256AndSize(localPath)
					if err != nil {
						log.Errorf("unable to calculate checksum and size: %v", err)
						continue
					}
					if checksum == f.Sha256 && size == f.Size {
						successCount++
						fmt.Printf("File %s already exists, skipping.\n", fileName.Filename)
						continue
					}
				}

				// Get download file pre-signed URL
				downloadUrl, err := pm.FileCli().GenerateFileDownloadUrl(context.TODO(), f.Name)
				if err != nil {
					log.Errorf("unable to get download URL for file %s: %v", fileName.Filename, err)
					continue
				}

				// Download file
				if err = cmd_utils.DownloadFileThroughUrl(localPath, downloadUrl, maxRetries); err != nil {
					log.Errorf("download file %s failed: %v\n", fileName.Filename, err)
					continue
				}

				successCount++
			}

			fmt.Printf("\nDownload completed! \nAll %d / %d files are saved to %s\n", successCount, totalFiles, dstDir)
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().IntVarP(&maxRetries, "max-retries", "r", 3, "maximum number of retries for downloading a file")
	cmd.Flags().StringVarP(&dir, "dir", "d", "", "download specific directory")
	cmd.Flags().StringSliceVar(&fileNames, "files", []string{}, "download specific files (comma-separated)")
	cmd.Flags().BoolVar(&flat, "flat", false, "download directly to the specified directory without creating a subdirectory named with record-id")

	cmd.MarkFlagsMutuallyExclusive("dir", "files")

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

			// Confirm deletion
			if !force {
				fmt.Printf("About to delete %d item(s) from record:\n", len(filesToDelete))
				for _, f := range filesToDelete {
					if strings.HasSuffix(f, "/") {
						fmt.Printf("  - %s (directory - all contents will be deleted)\n", f)
					} else {
						fmt.Printf("  - %s\n", f)
					}
				}
				if confirmed := prompts.PromptYN("Do you want to continue?"); !confirmed {
					fmt.Println("Delete aborted.")
					return
				}
			}

			// Build full resource names for batch delete
			// Server handles recursive deletion for directories
			resourceNames := make([]string, len(filesToDelete))
			for i, fileName := range filesToDelete {
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

			fmt.Printf("Successfully deleted %d item(s).\n", len(filesToDelete))
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
