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

package customfield

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const multiValueSep = ";"

func ResolveCustomFields(ctx context.Context, cfCli api.CustomFieldInterface, userCli api.UserInterface, proj *name.Project, rawInputs []string) ([]*commons.CustomFieldValue, error) {
	schema, err := cfCli.GetRecordCustomFieldSchema(ctx, proj)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom field schema: %w", err)
	}
	return NewResolver(schema, userCli).Resolve(ctx, rawInputs)
}

type Resolver struct {
	schema  *commons.CustomFieldSchema
	userCli api.UserInterface
}

func NewResolver(schema *commons.CustomFieldSchema, userCli api.UserInterface) *Resolver {
	return &Resolver{schema: schema, userCli: userCli}
}

func (r *Resolver) Resolve(ctx context.Context, rawInputs []string) ([]*commons.CustomFieldValue, error) {
	var result []*commons.CustomFieldValue
	for _, input := range rawInputs {
		idx := strings.Index(input, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid custom field format %q: expected key=value", input)
		}
		key := input[:idx]
		value := input[idx+1:]

		prop := r.findProperty(key)
		if prop == nil {
			available := lo.Map(r.schema.Properties, func(p *commons.Property, _ int) string {
				return p.Name
			})
			return nil, fmt.Errorf("unknown custom field %q, available fields: %s", key, strings.Join(available, ", "))
		}

		cfv, err := r.resolveValue(ctx, prop, value)
		if err != nil {
			return nil, fmt.Errorf("custom field %q: %w", key, err)
		}
		result = append(result, cfv)
	}
	return result, nil
}

func (r *Resolver) findProperty(name string) *commons.Property {
	for _, p := range r.schema.Properties {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (r *Resolver) resolveValue(ctx context.Context, prop *commons.Property, value string) (*commons.CustomFieldValue, error) {
	switch prop.GetType().(type) {
	case *commons.Property_Text:
		return resolveText(prop, value), nil
	case *commons.Property_Number:
		return resolveNumber(prop, value)
	case *commons.Property_Enums:
		return resolveEnum(prop, value)
	case *commons.Property_Time:
		return resolveTime(prop, value)
	case *commons.Property_User:
		return r.resolveUser(ctx, prop, value)
	default:
		return nil, fmt.Errorf("unsupported field type")
	}
}

func resolveText(prop *commons.Property, value string) *commons.CustomFieldValue {
	return &commons.CustomFieldValue{
		Property: prop,
		Value:    &commons.CustomFieldValue_Text{Text: &commons.TextValue{Value: value}},
	}
}

func resolveNumber(prop *commons.Property, value string) (*commons.CustomFieldValue, error) {
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q: %w", value, err)
	}
	return &commons.CustomFieldValue{
		Property: prop,
		Value:    &commons.CustomFieldValue_Number{Number: &commons.NumberValue{Value: num}},
	}, nil
}

func resolveEnum(prop *commons.Property, value string) (*commons.CustomFieldValue, error) {
	enumType := prop.GetEnums()
	if enumType == nil {
		return nil, fmt.Errorf("property is not an enum type")
	}

	if enumType.Multiple {
		displayNames := strings.Split(value, multiValueSep)
		var ids []string
		for _, dn := range displayNames {
			dn = strings.TrimSpace(dn)
			id, err := findEnumID(enumType, dn)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return &commons.CustomFieldValue{
			Property: prop,
			Value:    &commons.CustomFieldValue_Enums{Enums: &commons.EnumValue{Ids: ids}},
		}, nil
	}

	id, err := findEnumID(enumType, value)
	if err != nil {
		return nil, err
	}
	return &commons.CustomFieldValue{
		Property: prop,
		Value:    &commons.CustomFieldValue_Enums{Enums: &commons.EnumValue{Id: id}},
	}, nil
}

func findEnumID(enumType *commons.EnumType, displayName string) (string, error) {
	for id, name := range enumType.Values {
		if name == displayName {
			return id, nil
		}
	}
	available := lo.Values(enumType.Values)
	return "", fmt.Errorf("unknown enum value %q, available: %s", displayName, strings.Join(available, ", "))
}

func resolveTime(prop *commons.Property, value string) (*commons.CustomFieldValue, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		t, err := time.ParseInLocation(f, value, time.Local)
		if err == nil {
			return &commons.CustomFieldValue{
				Property: prop,
				Value:    &commons.CustomFieldValue_Time{Time: &commons.TimeValue{Value: timestamppb.New(t)}},
			}, nil
		}
	}
	return nil, fmt.Errorf("invalid time %q: expected RFC3339 (e.g. 2025-01-01T00:00:00Z), datetime (e.g. 2025-01-01T10:30), or date (e.g. 2025-01-01)", value)
}

func (r *Resolver) resolveUser(ctx context.Context, prop *commons.Property, value string) (*commons.CustomFieldValue, error) {
	userType := prop.GetUser()
	if userType == nil {
		return nil, fmt.Errorf("property is not a user type")
	}

	if userType.Multiple {
		nicknames := strings.Split(value, multiValueSep)
		var ids []string
		for _, nn := range nicknames {
			nn = strings.TrimSpace(nn)
			id, err := r.resolveOneUser(ctx, nn)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return &commons.CustomFieldValue{
			Property: prop,
			Value:    &commons.CustomFieldValue_User{User: &commons.UserValue{Ids: ids}},
		}, nil
	}

	id, err := r.resolveOneUser(ctx, strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	return &commons.CustomFieldValue{
		Property: prop,
		Value:    &commons.CustomFieldValue_User{User: &commons.UserValue{Ids: []string{id}}},
	}, nil
}

func (r *Resolver) resolveOneUser(ctx context.Context, nickname string) (string, error) {
	users, err := r.userCli.FindUsersByNickname(ctx, nickname)
	if err != nil {
		return "", fmt.Errorf("failed to find user %q: %w", nickname, err)
	}
	if len(users) == 0 {
		return "", fmt.Errorf("no user found with nickname %q", nickname)
	}
	if len(users) > 1 {
		emails := lo.Map(users, func(u *openv1alpha1resource.User, _ int) string {
			return u.Email
		})
		return "", fmt.Errorf("multiple users match nickname %q: %s — please use a more specific nickname", nickname, strings.Join(emails, ", "))
	}

	// Extract user ID from resource name "users/{uuid}"
	parts := strings.Split(users[0].Name, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected user resource name format: %s", users[0].Name)
	}
	return parts[1], nil
}
