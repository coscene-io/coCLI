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
	"bytes"
	"encoding/json"
	"io"

	"github.com/coscene-io/cocli/internal/printer/printable"
	"google.golang.org/protobuf/encoding/protojson"
)

type JSONPrinter struct{}

func (p *JSONPrinter) PrintObj(obj printable.Interface, w io.Writer) error {
	// protojson deliberately introduces non-deterministic whitespace (1 or 2
	// spaces after the colon, randomized per process) to discourage reliance on
	// its exact output. Re-indent the raw bytes through encoding/json's
	// json.Indent, which emits stable whitespace ("key": "value", single space)
	// without reordering keys or altering protojson's escaping (it is a lexical
	// re-indent, not a re-marshal through a map).
	raw, err := protojson.Marshal(obj.ToProtoMessage())
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return err
	}
	buf.WriteByte('\n')

	_, err = w.Write(buf.Bytes())
	return err
}
