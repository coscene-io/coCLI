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

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestActionPrintable(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC))
	actions := []*openv1alpha1resource.Action{
		{
			Name:       "projects/project-a/actions/action-a",
			Spec:       &commons.ActionSpec{Name: "Custom Action"},
			Author:     "users/alice",
			UpdateTime: now,
		},
		{
			Name:       "wftmpls/system-a",
			Spec:       &commons.ActionSpec{Name: "System Action"},
			Author:     "users/system",
			UpdateTime: now,
		},
	}

	obj := NewAction(actions)
	msg := obj.ToProtoMessage().(*openv1alpha1service.ListActionsResponse)
	assert.Equal(t, int64(2), msg.TotalSize)

	tbl := obj.ToTable(&table.PrintOpts{})
	require.Len(t, tbl.Rows, 2)
	assert.Equal(t, "action-a", tbl.Rows[0][0])
	assert.Equal(t, "custom", tbl.Rows[0][1])
	assert.Equal(t, "system-a", tbl.Rows[1][0])
	assert.Equal(t, "system", tbl.Rows[1][1])

	verbose := obj.ToTable(&table.PrintOpts{Verbose: true})
	assert.Equal(t, "projects/project-a/actions/action-a", verbose.Rows[0][0])
}

func TestActionRunPrintable(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC))
	action := &openv1alpha1resource.Action{Spec: &commons.ActionSpec{Name: "Build"}}
	runs := []*openv1alpha1resource.ActionRun{
		{
			Name:       "projects/project-a/actionRuns/run-a",
			CreateTime: now,
			Action:     action,
			State:      enums.ActionRunStateEnum_RUNNING,
			Creator:    &openv1alpha1resource.ActionRun_User{User: "users/alice"},
		},
		{
			Name:       "projects/project-a/actionRuns/run-b",
			CreateTime: now,
			Action:     action,
			State:      enums.ActionRunStateEnum_SUCCEEDED,
			Creator: &openv1alpha1resource.ActionRun_Trigger{
				Trigger: &openv1alpha1resource.Trigger{Spec: &commons.TriggerSpec{Name: "Nightly"}},
			},
		},
	}

	obj := NewActionRun(runs)
	msg := obj.ToProtoMessage().(*openv1alpha1service.ListActionRunsResponse)
	assert.Equal(t, int64(2), msg.TotalSize)

	tbl := obj.ToTable(&table.PrintOpts{})
	require.Len(t, tbl.Rows, 2)
	assert.Equal(t, "run-a", tbl.Rows[0][0])
	assert.Equal(t, "RUNNING", tbl.Rows[0][1])
	assert.Equal(t, "Build", tbl.Rows[0][2])
	assert.Equal(t, "users/alice", tbl.Rows[0][4])
	assert.Equal(t, "trigger: Nightly", tbl.Rows[1][4])
}

func TestFilePrintable(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC))
	files := []*openv1alpha1resource.File{
		{Filename: "data.bin", Size: 2048, CreateTime: now, UpdateTime: now},
		{Filename: "folder/", Size: 2048, CreateTime: now, UpdateTime: now},
	}

	obj := NewFile(files)
	msg := obj.ToProtoMessage().(*openv1alpha1service.ListFilesResponse)
	assert.Equal(t, int64(2), msg.TotalSize)

	tbl := obj.ToTable(&table.PrintOpts{})
	require.Len(t, tbl.Rows, 2)
	assert.Equal(t, "data.bin", tbl.Rows[0][0])
	assert.Equal(t, "2.00 KB", tbl.Rows[0][1])
	assert.Equal(t, "-", tbl.Rows[1][1])
}

func TestEventPrintable(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC))
	events := []*openv1alpha1resource.Event{
		{
			DisplayName: "Hard brake",
			TriggerTime: now,
			Duration:    durationpb.New(90 * time.Second),
		},
	}

	obj := NewEvent(events)
	msg := obj.ToProtoMessage().(*openv1alpha1service.ListRecordEventsResponse)
	assert.Equal(t, int64(1), msg.TotalSize)

	tbl := obj.ToTable(&table.PrintOpts{})
	require.Len(t, tbl.Rows, 1)
	assert.Equal(t, "Hard brake", tbl.Rows[0][0])
	assert.Equal(t, "1m30s", tbl.Rows[0][2])
}

func TestRecordWithMetadataPrintable(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC))
	record := &openv1alpha1resource.Record{
		Name:        "projects/project-a/records/record-a",
		Title:       "Road test",
		Description: "Downtown loop",
		Device:      &openv1alpha1resource.Device{SerialNumber: "SN-001"},
		Labels:      []*openv1alpha1resource.Label{{DisplayName: "city"}},
		Creator:     "users/alice",
		ByteSize:    2048,
		FileSize:    3,
		Summary:     &openv1alpha1resource.RecordSummary{FilesDuration: 10, PlayDuration: 8},
		CreateTime:  now,
		UpdateTime:  now,
		CustomFieldValues: []*commons.CustomFieldValue{
			{
				Property: &commons.Property{Name: "weather", Type: &commons.Property_Text{Text: &commons.TextType{}}},
				Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "sunny"}},
			},
		},
	}

	obj := NewRecordWithMetadata(record, "https://home.coscene.cn/org/project/records/record-a")
	msg := obj.ToProtoMessage().(*structpb.Struct)
	assert.Equal(t, "https://home.coscene.cn/org/project/records/record-a", msg.Fields["url"].GetStringValue())

	tbl := obj.ToTable(&table.PrintOpts{})
	rows := map[string]string{}
	for _, row := range tbl.Rows {
		rows[row[0]] = row[1]
	}

	assert.Equal(t, "record-a", rows["ID:"])
	assert.Equal(t, "Road test", rows["Title:"])
	assert.Equal(t, "SN-001", rows["Device:"])
	assert.Equal(t, "city", rows["Labels:"])
	assert.Contains(t, rows["Custom Field Values:"], "weather")
	assert.Equal(t, "2.00 KB", rows["Byte Size:"])
	assert.Equal(t, "3", rows["File Count:"])
	assert.Equal(t, "https://home.coscene.cn/org/project/records/record-a", rows["URL:"])
}

func TestRecordWithMetadataNilRecord(t *testing.T) {
	obj := NewRecordWithMetadata(nil, "https://example.com/record")

	msg := obj.ToProtoMessage().(*structpb.Struct)
	assert.Empty(t, msg.Fields)

	tbl := obj.ToTable(&table.PrintOpts{})
	require.Equal(t, [][]string{{"URL:", "https://example.com/record"}}, tbl.Rows)
}
