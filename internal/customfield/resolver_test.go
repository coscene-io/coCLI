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
	"testing"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/name"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserClient struct {
	findByNickname map[string][]*openv1alpha1resource.User
	findErr        error
}

func (m *mockUserClient) BatchGetUsers(_ context.Context, _ mapset.Set[name.User]) (map[string]*openv1alpha1resource.User, error) {
	return nil, nil
}

func (m *mockUserClient) ListUsers(_ context.Context, _ *api.ListUsersOptions) (*api.ListUsersResult, error) {
	return &api.ListUsersResult{}, nil
}

func (m *mockUserClient) GetUser(_ context.Context, _ string) (*openv1alpha1resource.User, error) {
	return nil, nil
}

func (m *mockUserClient) FindUsersByNickname(_ context.Context, nickname string) ([]*openv1alpha1resource.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.findByNickname[nickname], nil
}

func newMockUserClient(entries map[string][]*openv1alpha1resource.User) *mockUserClient {
	return &mockUserClient{findByNickname: entries}
}

func testSchema() *commons.CustomFieldSchema {
	return &commons.CustomFieldSchema{
		Properties: []*commons.Property{
			{
				Id:   "prop-text",
				Name: "color",
				Type: &commons.Property_Text{Text: &commons.TextType{}},
			},
			{
				Id:   "prop-num",
				Name: "count",
				Type: &commons.Property_Number{Number: &commons.NumberType{}},
			},
			{
				Id:   "prop-enum",
				Name: "priority",
				Type: &commons.Property_Enums{Enums: &commons.EnumType{
					Values:   map[string]string{"e1": "high", "e2": "medium", "e3": "low"},
					Multiple: false,
				}},
			},
			{
				Id:   "prop-enum-multi",
				Name: "tags",
				Type: &commons.Property_Enums{Enums: &commons.EnumType{
					Values:   map[string]string{"t1": "bug", "t2": "feature", "t3": "docs"},
					Multiple: true,
				}},
			},
			{
				Id:   "prop-time",
				Name: "deadline",
				Type: &commons.Property_Time{Time: &commons.TimeType{}},
			},
			{
				Id:   "prop-user",
				Name: "assignee",
				Type: &commons.Property_User{User: &commons.UserType{Multiple: false}},
			},
			{
				Id:   "prop-user-multi",
				Name: "reviewers",
				Type: &commons.Property_User{User: &commons.UserType{Multiple: true}},
			},
		},
	}
}

func TestResolveText(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	result, err := r.Resolve(context.Background(), []string{"color=blue"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "blue", result[0].GetText().GetValue())
	assert.Equal(t, "color", result[0].Property.Name)
}

func TestResolveNumber(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	result, err := r.Resolve(context.Background(), []string{"count=42.5"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 42.5, result[0].GetNumber().GetValue())
}

func TestResolveNumberInvalid(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	_, err := r.Resolve(context.Background(), []string{"count=abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid number")
}

func TestResolveEnumSingle(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	result, err := r.Resolve(context.Background(), []string{"priority=high"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "e1", result[0].GetEnums().GetId())
}

func TestResolveEnumMultiple(t *testing.T) {
	r := NewResolver(testSchema(), &mockUserClient{})
	result, err := r.Resolve(context.Background(), []string{"tags=bug;feature"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, []string{"t1", "t2"}, result[0].GetEnums().GetIds())
}

func TestResolveEnumInvalid(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	_, err := r.Resolve(context.Background(), []string{"priority=critical"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown enum value")
}

func TestResolveTimeRFC3339(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	result, err := r.Resolve(context.Background(), []string{"deadline=2025-03-15T10:30:00Z"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	ts := result[0].GetTime().GetValue().AsTime()
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, 3, int(ts.Month()))
	assert.Equal(t, 15, ts.Day())
}

func TestResolveTimeDateOnly(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	result, err := r.Resolve(context.Background(), []string{"deadline=2025-03-15"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	ts := result[0].GetTime().GetValue().AsTime().In(time.Local)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, 3, int(ts.Month()))
	assert.Equal(t, 15, ts.Day())
	assert.Equal(t, 0, ts.Hour())
}

func TestResolveTimeInvalid(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	_, err := r.Resolve(context.Background(), []string{"deadline=not-a-date"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time")
}

func TestResolveUser(t *testing.T) {
	mock := newMockUserClient(map[string][]*openv1alpha1resource.User{
		"张三": {{Name: "users/abc-123", Email: "zhang@example.com"}},
	})
	r := NewResolver(testSchema(), mock)
	result, err := r.Resolve(context.Background(), []string{"assignee=张三"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, []string{"abc-123"}, result[0].GetUser().GetIds())
}

func TestResolveUserAmbiguous(t *testing.T) {
	mock := newMockUserClient(map[string][]*openv1alpha1resource.User{
		"张三": {
			{Name: "users/abc-123", Email: "zhang1@example.com"},
			{Name: "users/def-456", Email: "zhang2@example.com"},
		},
	})
	r := NewResolver(testSchema(), mock)
	_, err := r.Resolve(context.Background(), []string{"assignee=张三"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple users match")
	assert.Contains(t, err.Error(), "zhang1@example.com")
	assert.Contains(t, err.Error(), "zhang2@example.com")
}

func TestResolveUserNotFound(t *testing.T) {
	mock := newMockUserClient(map[string][]*openv1alpha1resource.User{})
	r := NewResolver(testSchema(), mock)
	_, err := r.Resolve(context.Background(), []string{"assignee=ghost"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no user found")
}

func TestResolveUnknownField(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	_, err := r.Resolve(context.Background(), []string{"nonexistent=value"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown custom field")
}

func TestResolveInvalidFormat(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	_, err := r.Resolve(context.Background(), []string{"no-equals-sign"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected key=value")
}

func TestResolveMultipleFields(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	result, err := r.Resolve(context.Background(), []string{"color=red", "count=10"})
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "red", result[0].GetText().GetValue())
	assert.Equal(t, float64(10), result[1].GetNumber().GetValue())
}

func TestResolveUserMultiple(t *testing.T) {
	mock := newMockUserClient(map[string][]*openv1alpha1resource.User{
		"张三": {{Name: "users/abc-123", Email: "zhang@example.com"}},
		"李四": {{Name: "users/def-456", Email: "li@example.com"}},
	})
	r := NewResolver(testSchema(), mock)
	result, err := r.Resolve(context.Background(), []string{"reviewers=张三;李四"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	ids := result[0].GetUser().GetIds()
	require.Len(t, ids, 2)
	assert.Equal(t, "abc-123", ids[0])
	assert.Equal(t, "def-456", ids[1])
}

func TestResolveValueContainsEquals(t *testing.T) {
	r := NewResolver(testSchema(), newMockUserClient(nil))
	result, err := r.Resolve(context.Background(), []string{"color=a=b=c"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "a=b=c", result[0].GetText().GetValue())
}
