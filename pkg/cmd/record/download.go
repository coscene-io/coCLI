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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/fs"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewDownloadCommand(cfgPath *string, io *iostreams.IOStreams) *cobra.Command {
	var (
		projectSlug    = ""
		maxRetries     = 0
		includeMoments = false
		flat           = false
	)

	cmd := &cobra.Command{
		Use:                   "download <record-resource-name/id> <dst-dir> [-m] [-p <working-project-slug>] [--flat]",
		Short:                 "Download all files from a record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := config.Provide(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			recordName, err := pm.RecordCli().RecordId2Name(cmd.Context(), args[0], proj)
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

			// Download all files recursively
			files, err := pm.RecordCli().ListAllFilesWithFilter(cmd.Context(), recordName, "recursive=\"true\"")
			if err != nil {
				log.Fatalf("unable to list files: %v", err)
			}

			var dstDir string
			if flat {
				dstDir = dirPath
			} else {
				dstDir = filepath.Join(dirPath, recordName.RecordID)
			}
			fmt.Println("-------------------------------------------------------------")
			fmt.Printf("Downloading record %s\n", recordName.RecordID)
			recordUrl, err := pm.GetRecordUrl(cmd.Context(), recordName)
			if err == nil {
				fmt.Println("View record at:", recordUrl)
			} else {
				log.Errorf("unable to get record url: %v", err)
			}
			fmt.Printf("Saving to %s\n", dstDir)

			totalFiles := len(files)
			successCount := 0
			for fIdx, f := range files {
				fileName, _ := name.NewFile(f.Name)
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
				downloadUrl, err := pm.FileCli().GenerateFileDownloadUrl(cmd.Context(), f.Name)
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

			if includeMoments {
				moments, err := pm.RecordCli().ListAllMoments(cmd.Context(), recordName)
				if err != nil {
					// ignore the error and return empty list
					moments = []*api.Moment{}
					log.Errorf("unable to list moments: %v", err)
				}
				totalFiles++
				if err = cmd_utils.SaveMomentsJson(moments, dstDir); err != nil {
					log.Fatalf("unable to save moments: %v", err)
				} else {
					successCount++
				}
			}

			fmt.Printf("\nDownload completed! \nAll %d / %d files are saved to %s\n", successCount, totalFiles, dstDir)
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().IntVarP(&maxRetries, "max-retries", "r", 3, "maximum number of retries for downloading a file")
	cmd.Flags().BoolVarP(&includeMoments, "include-moments", "m", false, "include moments in the download")
	cmd.Flags().BoolVar(&flat, "flat", false, "download directly to the specified directory without creating a subdirectory named with record-id")

	return cmd
}
