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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/fs"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewDownloadCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug    = ""
		maxRetries     = 0
		includeMoments = false
	)

	cmd := &cobra.Command{
		Use:                   "download <record-resource-name/id> <dst-dir> [-m] [-p <working-project-slug>",
		Short:                 "Download files from record to directory.",
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
			dirPath, err := filepath.Abs(args[1])
			if err != nil {
				log.Fatalf("unable to get absolute path: %v", err)
			}
			if dirInfo, err := os.Stat(dirPath); err != nil {
				log.Fatalf("Error checking destination directory: %v", err)
			} else if !dirInfo.IsDir() {
				log.Fatalf("Destination directory is not a directory: %s", dirPath)
			}

			// List all files in the record.
			files, err := pm.RecordCli().ListAllFiles(context.TODO(), recordName)
			if err != nil {
				log.Fatalf("unable to list files: %v", err)
			}

			dstDir := filepath.Join(dirPath, recordName.RecordID)
			fmt.Println("-------------------------------------------------------------")
			fmt.Printf("Downloading record %s\n", recordName.RecordID)
			recordUrl, err := pm.GetRecordUrl(recordName)
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

			if includeMoments {
				if moments, err := pm.RecordCli().ListAllMoments(cmd.Context(), recordName); err != nil {
					log.Errorf("unable to list moments: %v", err)
				} else {
					totalFiles++
					momentPath := filepath.Join(dstDir, "moments.json")
					// Create the file to write the moments to
					momentFile, err := os.Create(momentPath)
					if err != nil {
						log.Fatalf("unable to create moments file %s: %v", momentPath, err)
					} else {
						defer momentFile.Close() // Ensure the file is closed

						type Moments struct {
							Moments []*api.Moment `json:"moments"`
						}

						if jsonData, err := json.MarshalIndent(Moments{Moments: moments}, "", "  "); err != nil {
							log.Fatalf("unable to marshal moments to JSON: %v", err)
						} else {
							if _, err = momentFile.Write(jsonData); err != nil {
								log.Fatalf("unable to write moments to file %s: %v", momentPath, err)
							} else {
								successCount++
								fmt.Printf("Moments saved to %s\n", momentPath)
							}
						}
					}
				}
			}

			fmt.Printf("\nDownload completed! \nAll %d / %d files are saved to %s\n", successCount, totalFiles, dstDir)
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().IntVarP(&maxRetries, "max-retries", "r", 3, "maximum number of retries for downloading a file")
	cmd.Flags().BoolVarP(&includeMoments, "include-moments", "m", false, "include moments in the download")

	return cmd
}
