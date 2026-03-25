// Copyright 2026 coScene
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
	"testing"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeTestRecord(id, title string, cfvs []*commons.CustomFieldValue) *openv1alpha1resource.Record {
	return &openv1alpha1resource.Record{
		Name:              "projects/p1/records/" + id,
		Title:             title,
		CreateTime:        timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		CustomFieldValues: cfvs,
	}
}

func TestCollectCustomFieldColumns(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
			{Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}}},
			{Property: &commons.Property{Name: "size", Type: &commons.Property_Number{Number: &commons.NumberType{}}}},
		}),
		makeTestRecord("r2", "rec2", []*commons.CustomFieldValue{
			{Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}}},
			{Property: &commons.Property{Name: "status", Type: &commons.Property_Text{Text: &commons.TextType{}}}},
		}),
	}

	cols := collectCustomFieldColumns(records)
	assert.Equal(t, []string{"color", "size", "status"}, cols)
}

func TestCollectCustomFieldColumns_Empty(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", nil),
	}
	cols := collectCustomFieldColumns(records)
	assert.Empty(t, cols)
}

func TestCsvCustomFieldColumnOrder_NilUsesDataOnly(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
			{Property: &commons.Property{Name: "a", Type: &commons.Property_Text{Text: &commons.TextType{}}}},
		}),
	}
	assert.Equal(t, []string{"a"}, csvCustomFieldColumnOrder(nil, records))
}

func TestCsvCustomFieldColumnOrder_SchemaIncludesNeverSetFields(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
			{Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}}},
		}),
		makeTestRecord("r2", "rec2", nil),
	}
	schema := []string{"unused", "color"}
	assert.Equal(t, []string{"unused", "color"}, csvCustomFieldColumnOrder(schema, records))
}

func TestCsvCustomFieldColumnOrder_AppendsOrphanFromRecords(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
			{Property: &commons.Property{Name: "only_in_data", Type: &commons.Property_Text{Text: &commons.TextType{}}}},
		}),
	}
	assert.Equal(t, []string{"in_schema", "only_in_data"}, csvCustomFieldColumnOrder([]string{"in_schema"}, records))
}

func TestCsvCustomFieldColumnOrder_DedupSchemaAndSkipsEmptyNames(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", nil),
	}
	schema := []string{"x", "", "x", "y"}
	assert.Equal(t, []string{"x", "y"}, csvCustomFieldColumnOrder(schema, records))
}

func TestRecord_ToTable_CSV_WithSchemaOrder(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
			{
				Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}},
				Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "blue"}},
			},
		}),
		makeTestRecord("r2", "rec2", nil),
	}

	p := NewRecord(records, "")
	p.CSVCustomFieldSchemaOrder = []string{"ghost", "color"}
	tbl := p.ToTable(&table.PrintOpts{Wide: true, CSV: true})
	headers := getHeaders(tbl)

	ghostIdx, colorIdx := -1, -1
	for i, h := range headers {
		switch h {
		case "ghost":
			ghostIdx = i
		case "color":
			colorIdx = i
		}
	}
	require.NotEqual(t, -1, ghostIdx)
	require.NotEqual(t, -1, colorIdx)
	assert.Less(t, ghostIdx, colorIdx)

	require.Len(t, tbl.Rows, 2)
	assert.Equal(t, "", tbl.Rows[0][ghostIdx])
	assert.Equal(t, "blue", tbl.Rows[0][colorIdx])
	assert.Equal(t, "", tbl.Rows[1][ghostIdx])
	assert.Equal(t, "", tbl.Rows[1][colorIdx])
}

func TestExtractCustomFieldValue_Text(t *testing.T) {
	r := makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
		{
			Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}},
			Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "red"}},
		},
	})
	p := &Record{}
	tableOpts := &table.PrintOpts{}
	assert.Equal(t, "red", p.extractCustomFieldValue(r, "color", tableOpts))
	assert.Equal(t, "", p.extractCustomFieldValue(r, "missing", tableOpts))
}

func TestExtractCustomFieldValue_Number(t *testing.T) {
	r := makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
		{
			Property: &commons.Property{Name: "count", Type: &commons.Property_Number{Number: &commons.NumberType{}}},
			Value:    &commons.CustomFieldValue_Number{Number: &commons.NumberValue{Value: 42.5}},
		},
	})
	p := &Record{}
	assert.Equal(t, "42.5", p.extractCustomFieldValue(r, "count", &table.PrintOpts{}))
}

func TestExtractCustomFieldValue_Enum_Single(t *testing.T) {
	r := makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
		{
			Property: &commons.Property{
				Name: "priority",
				Type: &commons.Property_Enums{Enums: &commons.EnumType{
					Values:   map[string]string{"p1": "High", "p2": "Low"},
					Multiple: false,
				}},
			},
			Value: &commons.CustomFieldValue_Enums{Enums: &commons.EnumValue{Id: "p1"}},
		},
	})
	p := &Record{}
	assert.Equal(t, "High", p.extractCustomFieldValue(r, "priority", &table.PrintOpts{}))
}

func TestExtractCustomFieldValue_Enum_Multiple(t *testing.T) {
	r := makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
		{
			Property: &commons.Property{
				Name: "tags",
				Type: &commons.Property_Enums{Enums: &commons.EnumType{
					Values:   map[string]string{"t1": "Alpha", "t2": "Beta"},
					Multiple: true,
				}},
			},
			Value: &commons.CustomFieldValue_Enums{Enums: &commons.EnumValue{Ids: []string{"t1", "t2"}}},
		},
	})
	p := &Record{}
	assert.Equal(t, "Alpha, Beta", p.extractCustomFieldValue(r, "tags", &table.PrintOpts{}))
	assert.Equal(t, "Alpha;Beta", p.extractCustomFieldValue(r, "tags", &table.PrintOpts{CSV: true}))
}

func TestExtractCustomFieldValue_Time(t *testing.T) {
	ts := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	r := makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
		{
			Property: &commons.Property{Name: "due", Type: &commons.Property_Time{Time: &commons.TimeType{}}},
			Value:    &commons.CustomFieldValue_Time{Time: &commons.TimeValue{Value: timestamppb.New(ts)}},
		},
	})
	p := &Record{}
	expected := ts.In(time.Local).Format(time.RFC3339)
	assert.Equal(t, expected, p.extractCustomFieldValue(r, "due", &table.PrintOpts{}))
}

func TestExtractCustomFieldValue_User(t *testing.T) {
	r := makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
		{
			Property: &commons.Property{Name: "assignee", Type: &commons.Property_User{User: &commons.UserType{}}},
			Value:    &commons.CustomFieldValue_User{User: &commons.UserValue{Ids: []string{"u1", "u2"}}},
		},
	})

	p := &Record{UserNames: map[string]string{"u1": "Alice", "u2": "Bob"}}
	assert.Equal(t, "Alice, Bob", p.extractCustomFieldValue(r, "assignee", &table.PrintOpts{}))
	assert.Equal(t, "Alice;Bob", p.extractCustomFieldValue(r, "assignee", &table.PrintOpts{CSV: true}))

	pNoNames := &Record{}
	assert.Equal(t, "u1, u2", pNoNames.extractCustomFieldValue(r, "assignee", &table.PrintOpts{}))
}

func getHeaders(tbl table.Table) []string {
	headers := make([]string, len(tbl.ColumnDefs))
	for i, col := range tbl.ColumnDefs {
		if col.FieldName != "" {
			headers[i] = col.FieldName
		}
	}
	return headers
}

func TestRecord_ToTable_Wide(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", nil),
	}

	p := NewRecord(records, "")
	tbl := p.ToTable(&table.PrintOpts{Wide: true})
	headers := getHeaders(tbl)

	assert.Contains(t, headers, "CREATOR")
	assert.Contains(t, headers, "BYTE SIZE")
	assert.Contains(t, headers, "PLAY DURATION")

	assert.NotContains(t, headers, "DEVICE")
	assert.NotContains(t, headers, "DESCRIPTION")
	assert.NotContains(t, headers, "FILE COUNT")
	assert.NotContains(t, headers, "FILES DURATION")
}

func TestRecord_ToTable_CSV(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
			{
				Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}},
				Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "blue"}},
			},
		}),
		makeTestRecord("r2", "rec2", []*commons.CustomFieldValue{
			{
				Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}},
				Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "green"}},
			},
			{
				Property: &commons.Property{Name: "size", Type: &commons.Property_Number{Number: &commons.NumberType{}}},
				Value:    &commons.CustomFieldValue_Number{Number: &commons.NumberValue{Value: 10}},
			},
		}),
	}

	p := NewRecord(records, "")
	tbl := p.ToTable(&table.PrintOpts{Wide: true, CSV: true})
	headers := getHeaders(tbl)

	assert.Contains(t, headers, "DEVICE")
	assert.Contains(t, headers, "DESCRIPTION")
	assert.Contains(t, headers, "FILE COUNT")
	assert.Contains(t, headers, "FILES DURATION")
	assert.Contains(t, headers, "color")
	assert.Contains(t, headers, "size")
	require.Len(t, tbl.Rows, 2)

	colorIdx := -1
	sizeIdx := -1
	for i, h := range headers {
		if h == "color" {
			colorIdx = i
		}
		if h == "size" {
			sizeIdx = i
		}
	}
	require.NotEqual(t, -1, colorIdx)
	require.NotEqual(t, -1, sizeIdx)

	assert.Equal(t, "blue", tbl.Rows[0][colorIdx])
	assert.Equal(t, "", tbl.Rows[0][sizeIdx])
	assert.Equal(t, "green", tbl.Rows[1][colorIdx])
	assert.Equal(t, "10", tbl.Rows[1][sizeIdx])
}

func TestRecord_ToTable_NoWide_NoCFColumns(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		makeTestRecord("r1", "rec1", []*commons.CustomFieldValue{
			{
				Property: &commons.Property{Name: "color", Type: &commons.Property_Text{Text: &commons.TextType{}}},
				Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "blue"}},
			},
		}),
	}

	p := NewRecord(records, "")
	tbl := p.ToTable(&table.PrintOpts{Wide: false})
	headers := getHeaders(tbl)

	assert.NotContains(t, headers, "color")
	assert.NotContains(t, headers, "DEVICE")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int64
		expected string
	}{
		{0, "0s"},
		{59, "59s"},
		{60, "1m0s"},
		{137, "2m17s"},
		{3599, "59m59s"},
		{3600, "1h0m0s"},
		{3661, "1h1m1s"},
		{86400, "24h0m0s"},
		{1000000, "277h46m40s"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatDuration(tt.seconds), "formatDuration(%d)", tt.seconds)
	}
}
