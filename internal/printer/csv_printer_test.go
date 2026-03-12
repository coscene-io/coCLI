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

package printer

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCSVPrinter_PrintObj(t *testing.T) {
	records := []*openv1alpha1resource.Record{
		{
			Name:       "projects/p1/records/r1",
			Title:      "Record One",
			CreateTime: timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			CustomFieldValues: []*commons.CustomFieldValue{
				{
					Property: &commons.Property{Name: "note", Type: &commons.Property_Text{Text: &commons.TextType{}}},
					Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "hello, world"}},
				},
			},
		},
		{
			Name:       "projects/p1/records/r2",
			Title:      "Record Two",
			CreateTime: timestamppb.New(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
			CustomFieldValues: []*commons.CustomFieldValue{
				{
					Property: &commons.Property{Name: "note", Type: &commons.Property_Text{Text: &commons.TextType{}}},
					Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "simple"}},
				},
			},
		},
	}

	p := &CSVPrinter{Opts: &table.PrintOpts{Wide: true}}
	obj := printable.NewRecord(records, "")

	var buf bytes.Buffer
	err := p.PrintObj(obj, &buf)
	require.NoError(t, err)

	reader := csv.NewReader(strings.NewReader(buf.String()))
	rows, err := reader.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3) // header + 2 data rows

	headers := rows[0]
	assert.Contains(t, headers, "note")

	noteIdx := -1
	for i, h := range headers {
		if h == "note" {
			noteIdx = i
			break
		}
	}
	require.NotEqual(t, -1, noteIdx)
	assert.Equal(t, "hello, world", rows[1][noteIdx])
	assert.Equal(t, "simple", rows[2][noteIdx])
}

func TestCSVPrinter_ViaFactory(t *testing.T) {
	p := Printer("csv", &Options{TableOpts: &table.PrintOpts{}})
	_, ok := p.(*CSVPrinter)
	assert.True(t, ok, "csv format should produce CSVPrinter")

	csvP := p.(*CSVPrinter)
	assert.True(t, csvP.Opts.Wide, "CSVPrinter should have Wide=true")
	assert.True(t, csvP.Opts.CSV, "CSVPrinter should have CSV=true")
}

func TestTableWide_ViaFactory(t *testing.T) {
	p := Printer("table,wide", &Options{TableOpts: &table.PrintOpts{}})
	tp, ok := p.(*TablePrinter)
	assert.True(t, ok, "table,wide should produce TablePrinter")
	assert.True(t, tp.Opts.Wide, "table,wide should have Wide=true")
}
