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

package printable

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/printer/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
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

// ToProtoMessage serializes the full Record proto and injects the URL field.
func (r *RecordWithMetadata) ToProtoMessage() proto.Message {
	if r.Record == nil {
		return &structpb.Struct{}
	}

	jsonBytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(r.Record)
	if err != nil {
		log.Warnf("failed to marshal record: %v", err)
		return &structpb.Struct{}
	}

	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		log.Warnf("failed to unmarshal record JSON: %v", err)
		return &structpb.Struct{}
	}

	if r.URL != "" {
		data["url"] = r.URL
	}

	s, err := structpb.NewStruct(data)
	if err != nil {
		log.Warnf("failed to create struct: %v", err)
		return &structpb.Struct{}
	}
	return s
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

		// Custom field values
		if len(r.Record.CustomFieldValues) > 0 {
			customFieldValues := utils.GetCustomFieldStructs(r.Record.CustomFieldValues)
			rows = append(rows, []string{"Custom Field Values:", strings.Join(lo.Map(customFieldValues, func(c *structpb.Value, _ int) string {
				m := c.AsInterface().(map[string]any)
				return fmt.Sprintf("(%s: %v)", m["property"], m["value"])
			}), ", ")})
		}

		// Creator
		if r.Record.Creator != "" {
			rows = append(rows, []string{"Creator:", r.Record.Creator})
		}

		// Sizes
		rows = append(rows, []string{"Byte Size:", utils.FormatBytes(uint64(r.Record.ByteSize))})
		rows = append(rows, []string{"File Count:", fmt.Sprintf("%d", r.Record.FileSize)})

		// Summary
		if r.Record.Summary != nil {
			rows = append(rows, []string{"Duration:", fmt.Sprintf("%ds", r.Record.Summary.PlayDuration)})
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
