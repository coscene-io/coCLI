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
	"github.com/coscene-io/cocli/internal/printer/table"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// ProjectS3Info is the S3 connection information shown in project overview.
type ProjectS3Info struct {
	Endpoint string
	Region   string
	Bucket   string
}

func NewProjectS3Info(endpoint, region, bucket string) *ProjectS3Info {
	return &ProjectS3Info{
		Endpoint: endpoint,
		Region:   region,
		Bucket:   bucket,
	}
}

func (i *ProjectS3Info) ToProtoMessage() proto.Message {
	data, _ := structpb.NewStruct(map[string]any{
		"endpoint": i.Endpoint,
		"region":   i.Region,
		"bucket":   i.Bucket,
	})
	return data
}

func (i *ProjectS3Info) ToTable(opts *table.PrintOpts) table.Table {
	rows := [][]string{
		{"ENDPOINT", i.Endpoint},
		{"REGION", i.Region},
		{"BUCKET", i.Bucket},
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
