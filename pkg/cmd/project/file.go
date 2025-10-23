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
	"path/filepath"
	"strings"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/fs"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/prompts"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	upload_utils "github.com/coscene-io/cocli/pkg/cmd_utils/upload_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewFileCommand(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Manage files in projects",
	}

	cmd.AddCommand(NewFileListCommand(cfgPath))
	cmd.AddCommand(NewFileDownloadCommand(cfgPath))
	cmd.AddCommand(NewFileUploadCommand(cfgPath))
	cmd.AddCommand(NewFileDeleteCommand(cfgPath))

	return cmd
}

func NewFileListCommand(cfgPath *string) *cobra.Command {
	var (
		verbose      = false
		outputFormat = ""
		recursive    = false
		pageSize     = 0
		page         = 0
		all          = false
		keywords     = ""
	)

	cmd := &cobra.Command{
		Use:                   "list <project-resource-name/slug> [-R] [-v] [--page-size <size>] [--page <number>] [--all] [--keywords <path>]",
		Short:                 "List files in the project",
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

			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			projectName, err := pm.ProjectName(cmd.Context(), args[0])
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			var files []*openv1alpha1resource.File
			var filterParts []string

			// Build filter
			if recursive {
				filterParts = append(filterParts, "recursive=\"true\"")
			}
			if keywords != "" {
				filterParts = append(filterParts, fmt.Sprintf("path=\"%s\"", keywords))
			}
			additionalFilter := strings.Join(filterParts, " && ")

			if all {
				if additionalFilter != "" {
					files, err = pm.ProjectCli().ListAllFilesWithFilter(context.TODO(), projectName, additionalFilter)
				} else {
					files, err = pm.ProjectCli().ListAllFiles(context.TODO(), projectName)
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

				if additionalFilter != "" {
					files, err = pm.ProjectCli().ListFilesWithPaginationAndFilter(context.TODO(), projectName, effectivePageSize, skip, additionalFilter)
				} else {
					files, err = pm.ProjectCli().ListFilesWithPagination(context.TODO(), projectName, effectivePageSize, skip)
				}
				if err != nil {
					log.Fatalf("unable to list files: %v", err)
				}

				if pageSize <= 0 && page > 1 {
					fmt.Fprintf(os.Stderr, "Note: Using default page size of %d files for page %d.\n\n", effectivePageSize, page)
				}
			} else {
				// Default behavior: use MaxPageSize and show note
				defaultPageSize := constants.MaxPageSize
				if additionalFilter != "" {
					files, err = pm.ProjectCli().ListFilesWithPaginationAndFilter(context.TODO(), projectName, defaultPageSize, 0, additionalFilter)
				} else {
					files, err = pm.ProjectCli().ListFilesWithPagination(context.TODO(), projectName, defaultPageSize, 0)
				}
				if err != nil {
					log.Fatalf("unable to list files: %v", err)
				}

				if len(files) == defaultPageSize {
					fmt.Fprintf(os.Stderr, "Note: Showing first %d files (default page size). Use --all to list all files or --page-size to specify page size.\n\n", defaultPageSize)
				}
			}

			// Filter out record files if any
			var projectFiles []*openv1alpha1resource.File
			for _, f := range files {
				if !strings.Contains(f.Name, "/records/") {
					projectFiles = append(projectFiles, f)
				}
			}

			// Print listed files.
			err = printer.Printer(outputFormat, &printer.Options{TableOpts: &table.PrintOpts{
				Verbose: verbose,
			}}).PrintObj(printable.NewFile(projectFiles), os.Stdout)
			if err != nil {
				log.Fatalf("unable to print files: %v", err)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json|yaml)")
	cmd.Flags().BoolVarP(&recursive, "recursive", "R", false, "list files recursively")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "number of files per page (10-100)")
	cmd.Flags().IntVar(&page, "page", 1, "page number (1-based, requires --page-size)")
	cmd.Flags().BoolVar(&all, "all", false, "list all files (overrides default page size)")
	cmd.Flags().StringVar(&keywords, "keywords", "", "filter files by path")

	cmd.MarkFlagsMutuallyExclusive("all", "page-size")
	cmd.MarkFlagsMutuallyExclusive("all", "page")

	return cmd
}

func NewFileDownloadCommand(cfgPath *string) *cobra.Command {
	var (
		maxRetries = 0
	)

	cmd := &cobra.Command{
		Use:                   "download <project-resource-name/slug> <dst-dir>",
		Short:                 "Download files from project to directory.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			// Handle args and flags.
			projectName, err := pm.ProjectName(cmd.Context(), args[0])
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
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

			// List all files in the project.
			files, err := pm.ProjectCli().ListAllFiles(context.TODO(), projectName)
			if err != nil {
				// Check if the error indicates that project-level files are not supported
				if strings.Contains(err.Error(), "invalid_argument") || strings.Contains(err.Error(), "validation error") || strings.Contains(err.Error(), "file_ListFiles_parent_format") {
					log.Fatalf("Project-level file download is not currently supported. The server requires files to be associated with records (validation error: parent must match pattern 'projects/{uuid}/records/{uuid}').")
				}
				log.Fatalf("unable to list project files: %v", err)
			}

			// Filter to get only project-level files (not record files)
			var projectFiles []*openv1alpha1resource.File
			for _, f := range files {
				if !strings.Contains(f.Name, "/records/") {
					projectFiles = append(projectFiles, f)
				}
			}

			if len(projectFiles) == 0 {
				fmt.Println("No project-level files found to download.")
				return
			}

			dstDir := filepath.Join(dirPath, projectName.ProjectID)
			fmt.Println("-------------------------------------------------------------")
			fmt.Printf("Downloading project files from %s\n", projectName.ProjectID)
			projectUrl, err := pm.GetProjectUrl(projectName)
			if err == nil {
				fmt.Println("View project at:", projectUrl)
			} else {
				log.Errorf("unable to get project url: %v", err)
			}
			fmt.Printf("Saving to %s\n", dstDir)

			totalFiles := len(projectFiles)
			successCount := 0
			for fIdx, f := range projectFiles {
				// For project files, we need to parse the file name
				fileName, err := name.NewProjectFile(f.Name)
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

				// Check if local file exists and have the same checksum and size
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

				// Download file with #maxRetries retries
				if err = cmd_utils.DownloadFileThroughUrl(localPath, downloadUrl, maxRetries); err != nil {
					log.Errorf("download file %s failed: %v\n", fileName.Filename, err)
					continue
				}

				successCount++
			}

			fmt.Printf("\nDownload completed! \nAll %d / %d files are saved to %s\n", successCount, totalFiles, dstDir)
		},
	}

	cmd.Flags().IntVarP(&maxRetries, "max-retries", "r", 3, "maximum number of retries for downloading a file")

	return cmd
}

func NewFileUploadCommand(cfgPath *string) *cobra.Command {
	var (
		isRecursive       = false
		includeHidden     = false
		maxRetries        = 0 // unused, kept for parity
		uploadManagerOpts = &upload_utils.UploadManagerOpts{}
	)

	cmd := &cobra.Command{
		Use:                   "upload <project-resource-name/slug> <directory>",
		Short:                 "Upload files in directory to a project.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			projectName, err := pm.ProjectName(cmd.Context(), args[0])
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			dirPath, err := filepath.Abs(args[1])
			if err != nil {
				log.Fatalf("unable to get absolute path: %v", err)
			}

			if dirInfo, err := os.Stat(dirPath); err != nil {
				log.Fatalf("Error checking source directory: %v", err)
			} else if !dirInfo.IsDir() {
				log.Fatalf("Source is not a directory: %s", dirPath)
			}

			fmt.Println("-------------------------------------------------------------")
			fmt.Printf("Uploading files to project: %s\n", projectName.ProjectID)

			um, err := upload_utils.NewUploadManagerFromConfig(projectName, 0,
				&upload_utils.ApiOpts{SecurityTokenInterface: pm.SecurityTokenCli(), FileInterface: pm.FileCli()}, uploadManagerOpts)
			if err != nil {
				log.Fatalf("unable to create upload manager: %v", err)
			}

			if err := um.Run(cmd.Context(), upload_utils.NewProjectParent(projectName), &upload_utils.FileOpts{Path: dirPath, Recursive: isRecursive, IncludeHidden: includeHidden}); err != nil {
				log.Fatalf("Unable to upload files: %v", err)
			}

			projectUrl, err := pm.GetProjectUrl(projectName)
			if err == nil {
				fmt.Println("View project at:", projectUrl)
			} else {
				log.Errorf("unable to get project url: %v", err)
			}
			_ = maxRetries // silence unused var
		},
	}

	cmd.Flags().BoolVarP(&isRecursive, "recursive", "R", false, "upload files in the current directory recursively")
	cmd.Flags().BoolVarP(&includeHidden, "include-hidden", "H", false, "include hidden files (\"dot\" files) in the upload")
	cmd.Flags().IntVarP(&uploadManagerOpts.Threads, "parallel", "P", 4, "number of uploads (could be part) in parallel")
	cmd.Flags().StringVarP(&uploadManagerOpts.PartSize, "part-size", "s", "128Mib", "each part size")
	cmd.Flags().BoolVar(&uploadManagerOpts.NoTTY, "no-tty", false, "disable interactive mode for headless environments")
	cmd.Flags().BoolVar(&uploadManagerOpts.TTY, "tty", false, "force interactive mode even in headless environments")

	cmd.MarkFlagsMutuallyExclusive("no-tty", "tty")

	return cmd
}

func NewFileDeleteCommand(cfgPath *string) *cobra.Command {
	var (
		force     = false
		fileNames []string
	)

	cmd := &cobra.Command{
		Use:                   "delete <project-resource-name/slug> [<filename>] [--files <file1,file2,...>] [-f]",
		Short:                 "Delete file(s) or directory from a project",
		DisableFlagsInUseLine: true,
		Args:                  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := config.Provide(*cfgPath).GetProfileManager()

			projectName, err := pm.ProjectName(cmd.Context(), args[0])
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			var filesToDelete []string

			// Collect files to delete from args and flags
			if len(args) == 2 {
				filesToDelete = append(filesToDelete, args[1])
			}
			if len(fileNames) > 0 {
				filesToDelete = append(filesToDelete, fileNames...)
			}

			if len(filesToDelete) == 0 {
				log.Fatalf("must specify at least one file to delete")
			}

			// Expand directories: if name ends with / or is a directory prefix, list and collect
			var finalFilesToDelete []string
			for _, fileName := range filesToDelete {
				if strings.HasSuffix(fileName, "/") {
					// Directory: list all files with this prefix
					allFiles, err := pm.ProjectCli().ListAllFiles(context.TODO(), projectName)
					if err != nil {
						log.Fatalf("unable to list project files: %v", err)
					}
					for _, f := range allFiles {
						// Extract filename from resource name
						projectFile, err := name.NewProjectFile(f.Name)
						if err != nil {
							continue
						}
						if strings.HasPrefix(projectFile.Filename, fileName) {
							finalFilesToDelete = append(finalFilesToDelete, projectFile.Filename)
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
				fmt.Printf("About to delete %d file(s) from project:\n", len(finalFilesToDelete))
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
				resourceNames[i] = name.ProjectFile{ProjectID: projectName.ProjectID, Filename: fileName}.String()
			}

			// Always use batch delete for consistency
			if err := pm.FileCli().BatchDeleteFiles(context.TODO(), projectName.String(), resourceNames); err != nil {
				log.Fatalf("failed to delete files: %v", err)
			}

			fmt.Printf("Successfully deleted %d file(s).\n", len(finalFilesToDelete))
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", force, "Force delete without confirmation")
	cmd.Flags().StringSliceVar(&fileNames, "files", []string{}, "additional files to delete (comma-separated)")

	return cmd
}
