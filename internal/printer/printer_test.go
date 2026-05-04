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
	"encoding/json"
	"errors"
	"testing"

	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type testPrintable struct {
	table table.Table
}

func (p testPrintable) ToProtoMessage() proto.Message {
	msg, _ := structpb.NewStruct(map[string]any{
		"name": "demo",
		"size": float64(42),
	})
	return msg
}

func (p testPrintable) ToTable(*table.PrintOpts) table.Table {
	return p.table
}

func TestPrinterFactory(t *testing.T) {
	tests := []struct {
		format string
		want   any
	}{
		{"", &TablePrinter{}},
		{"table", &TablePrinter{}},
		{"json", &JSONPrinter{}},
		{"yaml", &YAMLPrinter{}},
		{"csv", &CSVPrinter{}},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			p, err := Printer(tt.format, nil)
			require.NoError(t, err)
			assert.IsType(t, tt.want, p)
		})
	}
}

func TestJSONPrinter_PrintObj(t *testing.T) {
	var buf bytes.Buffer
	err := (&JSONPrinter{}).PrintObj(testPrintable{}, &buf)

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "demo", got["name"])
	assert.Equal(t, float64(42), got["size"])
	assert.True(t, bytes.HasSuffix(buf.Bytes(), []byte("\n")))
}

func TestYAMLPrinter_PrintObj(t *testing.T) {
	var buf bytes.Buffer
	err := (&YAMLPrinter{}).PrintObj(testPrintable{}, &buf)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "name: demo")
	assert.Contains(t, buf.String(), "size: 42")
}

func TestTablePrinter_PrintObj(t *testing.T) {
	obj := testPrintable{table: table.Table{
		ColumnDefs: []table.ColumnDefinition{
			{FieldName: "NAME", TrimSize: 8},
			{FieldName: "DESCRIPTION", TrimSize: 10},
		},
		Rows: [][]string{{"alpha", "abcdefghijklmnopqrstuvwxyz"}},
	}}

	var buf bytes.Buffer
	err := (&TablePrinter{Opts: &table.PrintOpts{}}).PrintObj(obj, &buf)

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "abcdefg...")
	assert.NotContains(t, out, "abcdefghijklmnopqrstuvwxyz")
}

func TestTablePrinter_PrintObjReturnsWriterError(t *testing.T) {
	obj := testPrintable{table: table.Table{
		ColumnDefs: []table.ColumnDefinition{{FieldName: "NAME", TrimSize: 8}},
	}}

	err := (&TablePrinter{Opts: &table.PrintOpts{}}).PrintObj(obj, errWriter{})
	require.Error(t, err)
}

func TestGetColumnFormat(t *testing.T) {
	assert.Equal(t, "%s ", getColumnFormat(true, 10, "anything"))
	assert.Equal(t, "%-15s", getColumnFormat(false, 10, "abc"))
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
