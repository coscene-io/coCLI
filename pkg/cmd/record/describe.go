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

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewDescribeCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		projectSlug  = ""
		outputFormat = ""
	)

	cmd := &cobra.Command{
		Use:                   "describe <record-resource-name/id> [-p <working-project-slug>] [-o <output-format>]",
		Short:                 "Describe record metadata",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Get current profile.
			pm, _ := getProvider(*cfgPath).GetProfileManager()
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}

			// Handle args and flags.
			recordName, err := pm.RecordCli().RecordId2Name(cmd.Context(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				io.Printf("failed to find record: %s in project: %s\n", args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("unable to get record name from %s: %v", args[0], err)
			}

			// Get record details.
			record, err := pm.RecordCli().Get(cmd.Context(), recordName)
			if err != nil {
				log.Fatalf("unable to get record: %v", err)
			}

			// Display record in the requested format
			DisplayRecordWithFormat(cmd.Context(), record, pm, outputFormat, false, io)
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")

	return cmd
}

// DisplayRecordWithFormat displays record details in the specified format
func DisplayRecordWithFormat(ctx context.Context, record *openv1alpha1resource.Record, pm *config.ProfileManager, format string, showSuccessMessage bool, io *iostreams.IOStreams) {
	// Parse record name
	recordName, err := name.NewRecord(record.Name)
	if err != nil {
		log.Warnf("unable to parse record name: %v", err)
		recordName = nil
	}

	// Get record URL
	recordUrl := ""
	if recordName != nil {
		recordUrl, err = pm.GetRecordUrl(ctx, recordName)
		if err != nil {
			log.Warnf("unable to get record url: %v", err)
			recordUrl = ""
		}
	}

	// Create wrapped record with metadata
	recordWithMeta := printable.NewRecordWithMetadata(record, recordUrl)

	// Handle success message for table format
	if showSuccessMessage && format == "table" {
		io.Println("\nRecord created successfully!")
		io.Println("-------------------------------------------------------------")
	}

	// Use the printer pattern
	p := printer.Printer(format, &printer.Options{
		TableOpts: &table.PrintOpts{
			Verbose: false,
		},
	})

	if err := p.PrintObj(recordWithMeta, io.Out); err != nil {
		log.Fatalf("unable to print record: %v", err)
	}

	if showSuccessMessage && format == "table" {
		io.Println("-------------------------------------------------------------")
	}
}
