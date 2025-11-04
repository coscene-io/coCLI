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
	"github.com/coscene-io/cocli/internal/printer/table"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// RegistryCredential represents a docker credential pair for printing.
type RegistryCredential struct {
	Username string
	Password string
}

func NewRegistryCredential(username, password string) *RegistryCredential {
	return &RegistryCredential{
		Username: username,
		Password: password,
	}
}

func (c *RegistryCredential) ToProtoMessage() proto.Message {
	data, _ := structpb.NewStruct(map[string]any{
		"username": c.Username,
		"password": c.Password,
	})
	return data
}

func (c *RegistryCredential) ToTable(opts *table.PrintOpts) table.Table {
	rows := [][]string{
		{"USERNAME", c.Username},
		{"PASSWORD", c.Password},
	}

	columnDefs := []table.ColumnDefinition{
		{FieldName: "Field", TrimSize: 20},
		{FieldName: "Value", TrimSize: 120},
	}

	return table.Table{
		ColumnDefs: columnDefs,
		Rows:       rows,
	}
}
