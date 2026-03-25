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

package printer

import (
	"fmt"
	"io"

	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/mattn/go-runewidth"
)

const (
	trimPadding = 5
)

type TablePrinter struct {
	Opts *table.PrintOpts
}

func (p *TablePrinter) PrintObj(obj printable.Interface, w io.Writer) (err error) {
	t := obj.ToTable(p.Opts)
	return p.printFixed(t, w)
}

func (p *TablePrinter) printFixed(t table.Table, w io.Writer) (err error) {
	for _, columnDef := range t.ColumnDefs {
		fieldName := columnDef.FieldName
		if columnDef.FieldNameFunc != nil {
			fieldName = columnDef.FieldNameFunc(p.Opts)
		}
		format := getColumnFormat(p.Opts.Verbose, columnDef.TrimSize, fieldName)
		if _, err = fmt.Fprintf(w, format, fieldName); err != nil {
			return err
		}
	}
	if _, err = fmt.Fprintln(w); err != nil {
		return err
	}

	for _, row := range t.Rows {
		for idx, columnDef := range t.ColumnDefs {
			item := row[idx]
			if !p.Opts.Verbose && runewidth.StringWidth(item) > columnDef.TrimSize {
				item = runewidth.Truncate(item, columnDef.TrimSize, "...")
			}
			format := getColumnFormat(p.Opts.Verbose, columnDef.TrimSize, item)
			if _, err = fmt.Fprintf(w, format, item); err != nil {
				return err
			}
		}
		if _, err = fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func getColumnFormat(verbose bool, trimSize int, value string) string {
	if verbose {
		return "%s "
	}
	return fmt.Sprintf("%%-%ds", trimSize+trimPadding+runewidth.StringWidth(value)-len(value))
}
