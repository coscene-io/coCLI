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

	"github.com/coscene-io/cocli/internal/printer/table"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestProjectS3InfoToProtoMessage(t *testing.T) {
	info := NewProjectS3Info("https://storage-cn-guangzhou.volc.coscene.cn", "cn-guangzhou", "coscene-hy.demo")
	msg := info.ToProtoMessage()

	st, ok := msg.(*structpb.Struct)
	if !ok {
		t.Fatalf("expected *structpb.Struct, got %T", msg)
	}

	if got := st.Fields["endpoint"].GetStringValue(); got != info.Endpoint {
		t.Fatalf("endpoint = %q, want %q", got, info.Endpoint)
	}
	if got := st.Fields["region"].GetStringValue(); got != info.Region {
		t.Fatalf("region = %q, want %q", got, info.Region)
	}
	if got := st.Fields["bucket"].GetStringValue(); got != info.Bucket {
		t.Fatalf("bucket = %q, want %q", got, info.Bucket)
	}
}

func TestProjectS3InfoToTable(t *testing.T) {
	info := NewProjectS3Info("endpoint", "region", "bucket")
	tbl := info.ToTable(&table.PrintOpts{})

	if len(tbl.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(tbl.Rows))
	}
	if tbl.Rows[0][0] != "ENDPOINT" || tbl.Rows[0][1] != "endpoint" {
		t.Fatalf("unexpected endpoint row: %#v", tbl.Rows[0])
	}
	if tbl.Rows[1][0] != "REGION" || tbl.Rows[1][1] != "region" {
		t.Fatalf("unexpected region row: %#v", tbl.Rows[1])
	}
	if tbl.Rows[2][0] != "BUCKET" || tbl.Rows[2][1] != "bucket" {
		t.Fatalf("unexpected bucket row: %#v", tbl.Rows[2])
	}
}
