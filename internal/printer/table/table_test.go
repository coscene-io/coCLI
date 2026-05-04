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

package table

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColumnDefinitionFull_ToColumnDefinition(t *testing.T) {
	fieldName := func(opts *PrintOpts) string {
		if opts.Wide {
			return "WIDE"
		}
		return "NARROW"
	}

	full := ColumnDefinitionFull[string]{
		FieldName:     "fallback",
		FieldNameFunc: fieldName,
		TrimSize:      12,
	}

	got := full.ToColumnDefinition()
	assert.Equal(t, "fallback", got.FieldName)
	assert.Equal(t, 12, got.TrimSize)
	assert.Equal(t, "WIDE", got.FieldNameFunc(&PrintOpts{Wide: true}))
}

func TestColumnDefs2Table(t *testing.T) {
	type row struct {
		Name string
		Age  int
	}

	defs := []ColumnDefinitionFull[row]{
		{
			FieldName: "NAME",
			FieldValueFunc: func(r row, opts *PrintOpts) string {
				return r.Name
			},
			TrimSize: 8,
		},
		{
			FieldNameFunc: func(opts *PrintOpts) string {
				if opts.Wide {
					return "AGE"
				}
				return "YEARS"
			},
			FieldValueFunc: func(r row, opts *PrintOpts) string {
				return strconv.Itoa(r.Age)
			},
			TrimSize: 3,
		},
	}

	tbl := ColumnDefs2Table(defs, []row{{Name: "Ada", Age: 37}}, &PrintOpts{})
	require.Len(t, tbl.ColumnDefs, 2)
	assert.Equal(t, "NAME", tbl.ColumnDefs[0].FieldName)
	assert.Equal(t, "YEARS", tbl.ColumnDefs[1].FieldNameFunc(&PrintOpts{}))
	assert.Equal(t, [][]string{{"Ada", "37"}}, tbl.Rows)

	wideWithoutAge := ColumnDefs2Table(defs, []row{{Name: "Ada", Age: 37}}, &PrintOpts{
		Wide:       true,
		OmitFields: []string{"AGE"},
	})
	require.Len(t, wideWithoutAge.ColumnDefs, 1)
	assert.Equal(t, [][]string{{"Ada"}}, wideWithoutAge.Rows)
}
