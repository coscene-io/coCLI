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

package printable

import (
	"fmt"
	"strings"
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// RecordWithMetadata wraps a single record with additional metadata like URL
type RecordWithMetadata struct {
	Record *openv1alpha1resource.Record
	URL    string
}

func NewRecordWithMetadata(record *openv1alpha1resource.Record, url string) *RecordWithMetadata {
	return &RecordWithMetadata{
		Record: record,
		URL:    url,
	}
}

// ToProtoMessage returns a Struct proto message that includes both record and URL
func (r *RecordWithMetadata) ToProtoMessage() proto.Message {
	// Create a struct to hold both record data and URL
	recordStruct, _ := structpb.NewStruct(nil)

	// Convert record fields to struct fields
	if r.Record != nil {
		recordStruct.Fields = map[string]*structpb.Value{
			"name":        structpb.NewStringValue(r.Record.Name),
			"title":       structpb.NewStringValue(r.Record.Title),
			"description": structpb.NewStringValue(r.Record.Description),
			"is_archived": structpb.NewBoolValue(r.Record.IsArchived),
		}

		// Add device name if present
		if r.Record.Device != nil {
			recordStruct.Fields["device"] = structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": structpb.NewStringValue(r.Record.Device.Name),
				},
			})
		}

		// Add labels
		if len(r.Record.Labels) > 0 {
			labelList := make([]*structpb.Value, 0, len(r.Record.Labels))
			for _, label := range r.Record.Labels {
				labelStruct, _ := structpb.NewStruct(map[string]interface{}{
					"display_name": label.DisplayName,
					"name":         label.Name,
				})
				labelList = append(labelList, structpb.NewStructValue(labelStruct))
			}
			recordStruct.Fields["labels"] = structpb.NewListValue(&structpb.ListValue{Values: labelList})
		}

		// Add times
		if r.Record.CreateTime != nil {
			recordStruct.Fields["create_time"] = structpb.NewStringValue(r.Record.CreateTime.AsTime().Format(time.RFC3339))
		}
		if r.Record.UpdateTime != nil {
			recordStruct.Fields["update_time"] = structpb.NewStringValue(r.Record.UpdateTime.AsTime().Format(time.RFC3339))
		}
	}

	// Add URL
	if r.URL != "" {
		recordStruct.Fields["url"] = structpb.NewStringValue(r.URL)
	}

	return recordStruct
}

// ToTable implements the table output format
func (r *RecordWithMetadata) ToTable(opts *table.PrintOpts) table.Table {
	// For table output, we'll create a simple two-column format
	rows := [][]string{}

	if r.Record != nil {
		recordName, _ := name.NewRecord(r.Record.Name)

		// Basic fields
		rows = append(rows, []string{"ID:", recordName.RecordID})
		rows = append(rows, []string{"Name:", r.Record.Name})
		rows = append(rows, []string{"Title:", r.Record.Title})
		rows = append(rows, []string{"Description:", r.Record.Description})

		// Device
		if r.Record.Device != nil {
			rows = append(rows, []string{"Device:", r.Record.Device.Name})
		}

		// Labels
		if len(r.Record.Labels) > 0 {
			labels := lo.Map(r.Record.Labels, func(l *openv1alpha1resource.Label, _ int) string {
				return l.DisplayName
			})
			rows = append(rows, []string{"Labels:", strings.Join(labels, ", ")})
		}

		// Times
		if r.Record.CreateTime != nil {
			rows = append(rows, []string{"Create Time:", r.Record.CreateTime.AsTime().In(time.Local).Format(time.RFC3339)})
		}
		if r.Record.UpdateTime != nil {
			rows = append(rows, []string{"Update Time:", r.Record.UpdateTime.AsTime().In(time.Local).Format(time.RFC3339)})
		}

		// Archived status
		rows = append(rows, []string{"Archived:", fmt.Sprintf("%v", r.Record.IsArchived)})
	}

	// URL
	if r.URL != "" {
		rows = append(rows, []string{"URL:", r.URL})
	}

	// Create column definitions
	columnDefs := []table.ColumnDefinition{
		{FieldName: "Field", TrimSize: 20},
		{FieldName: "Value", TrimSize: 150},
	}

	return table.Table{
		ColumnDefs: columnDefs,
		Rows:       rows,
	}
}
