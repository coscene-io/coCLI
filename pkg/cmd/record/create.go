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
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/pkg/cmd_utils/upload_utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCreateCommand(cfgPath *string) *cobra.Command {
	var (
		title             = ""
		description       = ""
		projectSlug       = ""
		labelDisplayNames []string
		thumbnail         = ""
		multiOpts         = &upload_utils.UploadManagerOpts{}
		timeout           time.Duration
	)

	cmd := &cobra.Command{
		Use:                   "create [-t <title>] [-d <description>] [-l <labels>...] [-p <working-project-slug>] [-i <thumbnail>]",
		Short:                 "Create a new record",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := config.Provide(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			// Create record.
			labelEntities := make([]*openv1alpha1resource.Label, 0)
			for _, labelDisplayName := range labelDisplayNames {
				labelEntity, err := pm.LabelCli().GetByDisplayNameOrCreate(context.TODO(), labelDisplayName, proj)
				if err != nil {
					log.Errorf("Failed to get or create label %s: %v", labelDisplayName, err)
				} else {
					labelEntities = append(labelEntities, labelEntity)
				}
			}
			res, err := pm.RecordCli().Create(context.TODO(), proj, title, "", description, labelEntities)
			if err != nil {
				log.Fatalf("Failed to create record: %v", err)
			}

			fmt.Printf("Record created: %v\n", res.Name)
			recordName, _ := name.NewRecord(res.Name)
			recordUrl, err := pm.GetRecordUrl(recordName)
			if err != nil {
				log.Errorf("unable to get record url: %v", err)
			} else {
				fmt.Println("The record url is:", recordUrl)
			}

			if thumbnail != "" {
				// Upload thumbnail.
				thumbnailUploadUrl, err := pm.RecordCli().GenerateRecordThumbnailUploadUrl(context.TODO(), recordName)
				if err != nil {
					log.Fatalf("Failed to generate record thumbnail upload url: %v", err)
				}

				fmt.Println("Uploading thumbnail to pre-signed url...")
				um, err := upload_utils.NewUploadManagerFromConfig(proj, timeout,
					&upload_utils.ApiOpts{SecurityTokenInterface: pm.SecurityTokenCli(), FileInterface: pm.FileCli()}, multiOpts)
				if err != nil {
					log.Fatalf("unable to create upload manager: %v", err)
				}

				err = um.Run(context.TODO(), recordName, &upload_utils.FileOpts{AdditionalUploads: map[string]string{
					thumbnail: thumbnailUploadUrl,
				}})
				if err != nil {
					log.Fatalf("Failed to upload thumbnail: %v", err)
				}
			}
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "cocli created record", "title of the record.")
	cmd.Flags().StringVarP(&description, "description", "d", "", "description of the record.")
	cmd.Flags().StringSliceVarP(&labelDisplayNames, "labels", "l", []string{}, "labels of the record.")
	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&thumbnail, "thumbnail", "i", "", "thumbnail path of the record.")
	cmd.Flags().IntVarP(&multiOpts.Threads, "parallel", "P", 4, "number of uploads (could be part) in parallel")
	cmd.Flags().StringVarP(&multiOpts.PartSize, "part-size", "s", "128Mib", "each part size")
	cmd.Flags().DurationVar(&timeout, "response-timeout", 5*time.Minute, "server response time out")

	return cmd
}
