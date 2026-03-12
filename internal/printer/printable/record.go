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
	"strconv"
	"strings"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
)

const (
	recordIdTrimSize      = 36
	recordArchiveTrimSize = 8
	recordTitleTrimSize   = 40
	recordLabelsTrimSize  = 25
	recordTimeTrimSize    = len(time.RFC3339)

	multiValueSepTable = ", "
	multiValueSepCSV   = ";"
)

type Record struct {
	Delegate      []*openv1alpha1resource.Record
	NextPageToken string
}

func NewRecord(records []*openv1alpha1resource.Record, nextPageToken string) *Record {
	return &Record{
		Delegate:      records,
		NextPageToken: nextPageToken,
	}
}

func (p *Record) ToProtoMessage() proto.Message {
	return &openv1alpha1service.ListRecordsResponse{
		Records:       p.Delegate,
		TotalSize:     int64(len(p.Delegate)),
		NextPageToken: p.NextPageToken,
	}
}

func (p *Record) ToTable(opts *table.PrintOpts) table.Table {
	fullColumnDefs := []table.ColumnDefinitionFull[*openv1alpha1resource.Record]{
		{
			FieldNameFunc: func(opts *table.PrintOpts) string {
				if opts.Verbose {
					return "RESOURCE NAME"
				}
				return "ID"
			},
			FieldValueFunc: func(r *openv1alpha1resource.Record, opts *table.PrintOpts) string {
				if opts.Verbose {
					return r.Name
				}
				recordName, _ := name.NewRecord(r.Name)
				return recordName.RecordID
			},
			TrimSize: recordIdTrimSize,
		},
		{
			FieldName: "ARCHIVED",
			FieldValueFunc: func(r *openv1alpha1resource.Record, opts *table.PrintOpts) string {
				return strconv.FormatBool(r.IsArchived)
			},
			TrimSize: recordArchiveTrimSize,
		},
		{
			FieldName: "TITLE",
			FieldValueFunc: func(r *openv1alpha1resource.Record, opts *table.PrintOpts) string {
				return r.Title
			},
			TrimSize: recordTitleTrimSize,
		},
		{
			FieldName: "LABELS",
			FieldValueFunc: func(r *openv1alpha1resource.Record, opts *table.PrintOpts) string {
				labels := lo.Map(r.Labels, func(l *openv1alpha1resource.Label, _ int) string {
					return l.DisplayName
				})
				return strings.Join(labels, multiValueSep(opts))
			},
			TrimSize: recordLabelsTrimSize,
		},
		{
			FieldName: "CREATE TIME",
			FieldValueFunc: func(r *openv1alpha1resource.Record, opts *table.PrintOpts) string {
				return r.CreateTime.AsTime().In(time.Local).Format(time.RFC3339)
			},
			TrimSize: recordTimeTrimSize,
		},
	}

	if opts.Wide {
		cfColumns := collectCustomFieldColumns(p.Delegate)
		for _, col := range cfColumns {
			colName := col
			fullColumnDefs = append(fullColumnDefs, table.ColumnDefinitionFull[*openv1alpha1resource.Record]{
				FieldName: colName,
				FieldValueFunc: func(r *openv1alpha1resource.Record, opts *table.PrintOpts) string {
					return extractCustomFieldValue(r, colName, opts)
				},
				TrimSize: recordTitleTrimSize,
			})
		}
	}

	return table.ColumnDefs2Table(fullColumnDefs, p.Delegate, opts)
}

func collectCustomFieldColumns(records []*openv1alpha1resource.Record) []string {
	seen := map[string]bool{}
	var columns []string
	for _, r := range records {
		for _, cfv := range r.CustomFieldValues {
			n := cfv.Property.GetName()
			if n != "" && !seen[n] {
				seen[n] = true
				columns = append(columns, n)
			}
		}
	}
	return columns
}

func extractCustomFieldValue(r *openv1alpha1resource.Record, propertyName string, opts *table.PrintOpts) string {
	sep := multiValueSep(opts)
	for _, cfv := range r.CustomFieldValues {
		if cfv.Property.GetName() != propertyName {
			continue
		}
		switch cfv.Property.GetType().(type) {
		case *commons.Property_Text:
			return cfv.GetText().GetValue()
		case *commons.Property_Number:
			return fmt.Sprintf("%g", cfv.GetNumber().GetValue())
		case *commons.Property_Enums:
			if cfv.Property.GetEnums().GetMultiple() {
				names := lo.Map(cfv.GetEnums().GetIds(), func(id string, _ int) string {
					if v, ok := cfv.Property.GetEnums().GetValues()[id]; ok {
						return v
					}
					return id
				})
				return strings.Join(names, sep)
			}
			if v, ok := cfv.Property.GetEnums().GetValues()[cfv.GetEnums().GetId()]; ok {
				return v
			}
			return cfv.GetEnums().GetId()
		case *commons.Property_Time:
			if cfv.GetTime().GetValue() != nil {
				return cfv.GetTime().GetValue().AsTime().In(time.Local).Format(time.RFC3339)
			}
		case *commons.Property_User:
			return strings.Join(cfv.GetUser().GetIds(), sep)
		}
	}
	return ""
}

func multiValueSep(opts *table.PrintOpts) string {
	if opts.CSV {
		return multiValueSepCSV
	}
	return multiValueSepTable
}
