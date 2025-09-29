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

package utils

import (
	"errors"
	"strings"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/structpb"
)

func GetCustomFieldStructs(customFieldValues []*commons.CustomFieldValue) []*structpb.Value {
	return lo.Map(customFieldValues, func(customFieldValue *commons.CustomFieldValue, _ int) *structpb.Value {
		customFieldStruct, _ := getCustomFieldStruct(customFieldValue)
		return structpb.NewStructValue(customFieldStruct)
	})
}

func getCustomFieldStruct(customFieldValue *commons.CustomFieldValue) (*structpb.Struct, error) {
	switch customFieldValue.Property.GetType().(type) {
	case *commons.Property_Text:
		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    customFieldValue.GetText().Value,
		})
	case *commons.Property_Number:
		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    customFieldValue.GetNumber().Value,
		})
	case *commons.Property_Enums:
		if customFieldValue.Property.GetEnums().Multiple {
			enumNames := lo.Map(customFieldValue.GetEnums().Ids, func(id string, _ int) string {
				if v, ok := customFieldValue.Property.GetEnums().Values[id]; ok {
					return v
				}
				// Fallback to id if display name not found
				return id
			})
			return structpb.NewStruct(map[string]interface{}{
				"property": customFieldValue.Property.Name,
				"value":    strings.Join(enumNames, ", "),
			})
		} else {
			var enumName string
			if v, ok := customFieldValue.Property.GetEnums().Values[customFieldValue.GetEnums().Id]; ok {
				enumName = v
			} else {
				// Fallback to id if display name not found
				enumName = customFieldValue.GetEnums().Id
			}
			return structpb.NewStruct(map[string]interface{}{
				"property": customFieldValue.Property.Name,
				"value":    enumName,
			})
		}
	case *commons.Property_Time:
		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    customFieldValue.GetTime().Value.AsTime().Format(time.RFC3339),
		})
	case *commons.Property_User:
		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    strings.Join(customFieldValue.GetUser().Ids, ", "),
		})
	default:
		return nil, errors.New("unknown custom field type")
	}
}
