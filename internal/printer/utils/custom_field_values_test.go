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

package utils

import (
	"testing"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetCustomFieldStructs(t *testing.T) {
	ts := time.Date(2026, 5, 4, 1, 2, 3, 0, time.UTC)
	values := []*commons.CustomFieldValue{
		{
			Property: &commons.Property{Name: "note", Type: &commons.Property_Text{Text: &commons.TextType{}}},
			Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: "hello"}},
		},
		{
			Property: &commons.Property{Name: "score", Type: &commons.Property_Number{Number: &commons.NumberType{}}},
			Value:    &commons.CustomFieldValue_Number{Number: &commons.NumberValue{Value: 9.5}},
		},
		{
			Property: &commons.Property{Name: "status", Type: &commons.Property_Enums{Enums: &commons.EnumType{Values: map[string]string{"ok": "OK"}}}},
			Value:    &commons.CustomFieldValue_Enums{Enums: &commons.EnumValue{Id: "missing"}},
		},
		{
			Property: &commons.Property{Name: "tags", Type: &commons.Property_Enums{Enums: &commons.EnumType{Multiple: true, Values: map[string]string{"a": "Alpha"}}}},
			Value:    &commons.CustomFieldValue_Enums{Enums: &commons.EnumValue{Ids: []string{"a", "b"}}},
		},
		{
			Property: &commons.Property{Name: "when", Type: &commons.Property_Time{Time: &commons.TimeType{}}},
			Value:    &commons.CustomFieldValue_Time{Time: &commons.TimeValue{Value: timestamppb.New(ts)}},
		},
		{
			Property: &commons.Property{Name: "owners", Type: &commons.Property_User{User: &commons.UserType{}}},
			Value:    &commons.CustomFieldValue_User{User: &commons.UserValue{Ids: []string{"u1", "u2"}}},
		},
	}

	got := GetCustomFieldStructs(values)

	require.Len(t, got, 6)
	assert.Equal(t, map[string]any{"property": "note", "value": "hello"}, got[0].AsInterface())
	assert.Equal(t, map[string]any{"property": "score", "value": float64(9.5)}, got[1].AsInterface())
	assert.Equal(t, map[string]any{"property": "status", "value": "missing"}, got[2].AsInterface())
	assert.Equal(t, map[string]any{"property": "tags", "value": "Alpha, b"}, got[3].AsInterface())
	assert.Equal(t, map[string]any{"property": "when", "value": ts.Format(time.RFC3339)}, got[4].AsInterface())
	assert.Equal(t, map[string]any{"property": "owners", "value": "u1, u2"}, got[5].AsInterface())
}

func TestGetCustomFieldStructUnknownType(t *testing.T) {
	_, err := getCustomFieldStruct(&commons.CustomFieldValue{
		Property: &commons.Property{Name: "unknown"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown custom field type")
}
