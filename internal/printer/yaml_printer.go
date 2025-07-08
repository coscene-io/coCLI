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
	"encoding/json"
	"io"

	"github.com/coscene-io/cocli/internal/printer/printable"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

type YAMLPrinter struct{}

func (p *YAMLPrinter) PrintObj(obj printable.Interface, w io.Writer) error {
	// First convert proto message to JSON
	jsonBytes, err := protojson.MarshalOptions{
		UseProtoNames: true,
	}.Marshal(obj.ToProtoMessage())
	if err != nil {
		return err
	}

	// Convert JSON to map for YAML marshaling
	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return err
	}

	// Marshal to YAML
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}
