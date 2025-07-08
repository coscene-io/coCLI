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
	"strings"
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// RecordWithMetadata wraps the API record response with additional computed fields
type RecordWithMetadata struct {
	*openv1alpha1resource.Record `yaml:",inline"`
	URL                          string `json:"url" yaml:"url"`
}

func NewDescribeCommand(cfgPath *string) *cobra.Command {
	var (
		projectSlug  = ""
		outputFormat = ""
	)

	cmd := &cobra.Command{
		Use:                   "describe <record-resource-name/id> [-p <working-project-slug>] [-o <output-format>]",
		Short:                 "Describe record metadata.",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
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

			// Get record details.
			record, err := pm.RecordCli().Get(context.TODO(), recordName)
			if err != nil {
				log.Fatalf("unable to get record: %v", err)
			}

			// Get record URL.
			recordUrl, err := pm.GetRecordUrl(recordName)
			if err != nil {
				log.Warnf("unable to get record url: %v", err)
				recordUrl = ""
			}

			// Create wrapped record with metadata.
			recordWithMeta := &RecordWithMetadata{
				Record: record,
				URL:    recordUrl,
			}

			// Output based on format.
			switch outputFormat {
			case "json":
				if err := outputJSON(recordWithMeta); err != nil {
					log.Fatalf("unable to output JSON: %v", err)
				}
			case "yaml":
				if err := outputYAML(recordWithMeta); err != nil {
					log.Fatalf("unable to output YAML: %v", err)
				}
			default:
				outputTable(recordWithMeta)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")

	return cmd
}

// DisplayRecord displays record details with URL, handling URL fetching internally
func DisplayRecord(record *openv1alpha1resource.Record, pm *config.ProfileManager) {
	// Parse record name
	recordName, err := name.NewRecord(record.Name)
	if err != nil {
		log.Warnf("unable to parse record name: %v", err)
		DisplayRecordDetails(record, "")
		return
	}

	// Get record URL
	recordUrl, err := pm.GetRecordUrl(recordName)
	if err != nil {
		log.Warnf("unable to get record url: %v", err)
		recordUrl = ""
	}

	// Display record details
	DisplayRecordDetails(record, recordUrl)
}

// DisplayRecordDetails prints record details in table format with a provided URL
func DisplayRecordDetails(record *openv1alpha1resource.Record, recordUrl string) {
	// Convert record to a map for easier handling
	data, err := convertToMap(record)
	if err != nil {
		log.Fatalf("unable to convert record: %v", err)
	}

	// Extract record name parts
	recordName, _ := name.NewRecord(getString(data, "name"))

	// Print formatted output
	fmt.Printf("%-20s %s\n", "ID:", recordName.RecordID)
	fmt.Printf("%-20s %s\n", "Name:", getString(data, "name"))
	fmt.Printf("%-20s %s\n", "Title:", getString(data, "title"))
	fmt.Printf("%-20s %s\n", "Description:", getString(data, "description"))

	// Device
	if device := getMap(data, "device"); device != nil {
		fmt.Printf("%-20s %s\n", "Device:", getString(device, "name"))
	}

	// Labels
	if labels := getSlice(data, "labels"); len(labels) > 0 {
		labelNames := []string{}
		for _, label := range labels {
			if labelMap, ok := label.(map[string]interface{}); ok {
				labelNames = append(labelNames, getString(labelMap, "display_name"))
			}
		}
		fmt.Printf("%-20s %s\n", "Labels:", strings.Join(labelNames, ", "))
	}

	// Times
	fmt.Printf("%-20s %s\n", "Create Time:", formatTimeFromMap(getMap(data, "create_time")))
	fmt.Printf("%-20s %s\n", "Update Time:", formatTimeFromMap(getMap(data, "update_time")))

	// Duration
	if duration := getString(data, "duration"); duration != "" {
		fmt.Printf("%-20s %s\n", "Duration:", duration)
	}

	// Archived status
	fmt.Printf("%-20s %v\n", "Archived:", getBool(data, "is_archived"))

	// Total size
	if totalSize := getFloat64(data, "total_size"); totalSize > 0 {
		fmt.Printf("%-20s %.2f MB\n", "Total Size:", totalSize/1024/1024)
	}

	// URL
	if recordUrl != "" {
		fmt.Printf("%-20s %s\n", "URL:", recordUrl)
	}
}

func outputTable(recordWithMeta interface{}) {
	// Convert to RecordWithMetadata to access both record and URL
	rwm, ok := recordWithMeta.(*RecordWithMetadata)
	if !ok {
		log.Fatalf("unable to cast to RecordWithMetadata")
	}

	DisplayRecordDetails(rwm.Record, rwm.URL)
}

func outputJSON(record interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(record)
}

func outputYAML(record interface{}) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	return encoder.Encode(record)
}

func convertToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mapVal, ok := v.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return nil
}

func getSlice(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key]; ok {
		if slice, ok := v.([]interface{}); ok {
			return slice
		}
	}
	return nil
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func formatTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}

	// Try to parse the time string
	t, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		// Try other formats
		t, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return timeStr // Return as-is if can't parse
		}
	}

	return t.In(time.Local).Format(time.RFC3339)
}

func formatTimeFromMap(timeMap map[string]interface{}) string {
	if timeMap == nil {
		return ""
	}

	seconds := getFloat64(timeMap, "seconds")
	nanos := getFloat64(timeMap, "nanos")

	if seconds == 0 {
		return ""
	}

	t := time.Unix(int64(seconds), int64(nanos))
	return t.In(time.Local).Format(time.RFC3339)
}
