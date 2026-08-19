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

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEvent_ToTable(t *testing.T) {
	triggerTime := time.Date(2026, 6, 4, 3, 21, 17, 0, time.UTC)
	events := []*openv1alpha1resource.Event{
		{
			Name:        "projects/project-id/events/25ed8af0-4baf-43fa-8b2e-1e26f7b395b2",
			DisplayName: "test",
			TriggerTime: timestamppb.New(triggerTime),
			Duration:    durationpb.New(0),
		},
	}

	tbl := NewEvent(events).ToTable(&table.PrintOpts{})

	require.Len(t, tbl.ColumnDefs, 4)
	assert.Equal(t, []string{"ID", "DISPLAY NAME", "TRIGGER TIME", "DURATION"}, []string{
		tbl.ColumnDefs[0].FieldName,
		tbl.ColumnDefs[1].FieldName,
		tbl.ColumnDefs[2].FieldName,
		tbl.ColumnDefs[3].FieldName,
	})
	require.Len(t, tbl.Rows, 1)
	assert.Equal(t, []string{
		"25ed8af0-4baf-43fa-8b2e-1e26f7b395b2",
		"test",
		triggerTime.In(time.Local).Format(time.RFC3339),
		"0s",
	}, tbl.Rows[0])
}
