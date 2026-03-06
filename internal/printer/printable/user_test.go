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

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newTestUser(name, email string, nickname *string, role *openv1alpha1resource.Role) *openv1alpha1resource.User {
	u := &openv1alpha1resource.User{
		Name:       name,
		Email:      email,
		Nickname:   nickname,
		Active:     true,
		CreateTime: timestamppb.Now(),
	}
	if role != nil {
		u.Role = role
	}
	return u
}

func strPtr(s string) *string {
	return &s
}

func TestUser_ToProtoMessage(t *testing.T) {
	users := []*openv1alpha1resource.User{
		newTestUser("users/abc-123", "alice@example.com", strPtr("Alice"), nil),
	}
	p := NewUser(users, "next-token")
	msg := p.ToProtoMessage()

	resp, ok := msg.(*openv1alpha1service.ListUsersResponse)
	require.True(t, ok)
	assert.Len(t, resp.Users, 1)
	assert.Equal(t, "next-token", resp.NextPageToken)
	assert.Equal(t, int64(1), resp.TotalSize)
}

func TestUser_ToTable(t *testing.T) {
	role := &openv1alpha1resource.Role{Code: "ORGANIZATION_ADMIN"}
	users := []*openv1alpha1resource.User{
		newTestUser("users/abc-def-123", "alice@example.com", strPtr("Alice"), role),
		newTestUser("users/xyz-789-456", "bob@example.com", nil, nil),
	}
	p := NewUser(users, "")

	t.Run("default columns", func(t *testing.T) {
		tbl := p.ToTable(&table.PrintOpts{})
		colNames := columnNames(tbl)
		assert.Contains(t, colNames, "ID")
		assert.Contains(t, colNames, "NICKNAME")
		assert.Contains(t, colNames, "EMAIL")
		assert.Contains(t, colNames, "ROLE")
		assert.Contains(t, colNames, "ACTIVE")

		require.Len(t, tbl.Rows, 2)
		assert.Equal(t, "abc-def-123", tbl.Rows[0][0])
		assert.Equal(t, "Alice", tbl.Rows[0][1])
		assert.Equal(t, "ORGANIZATION_ADMIN", tbl.Rows[0][roleColumnIndex(tbl)])
		assert.Equal(t, "", tbl.Rows[1][1])
		assert.Equal(t, "", tbl.Rows[1][roleColumnIndex(tbl)])
	})

	t.Run("verbose shows resource name", func(t *testing.T) {
		opts := &table.PrintOpts{Verbose: true}
		tbl := p.ToTable(opts)
		colNames := columnNamesWithOpts(tbl, opts)
		assert.Contains(t, colNames, "RESOURCE NAME")
		assert.NotContains(t, colNames, "ID")

		assert.Equal(t, "users/abc-def-123", tbl.Rows[0][0])
	})

	t.Run("omit fields", func(t *testing.T) {
		tbl := p.ToTable(&table.PrintOpts{OmitFields: []string{"EMAIL", "PHONE", "CREATE TIME"}})
		colNames := columnNames(tbl)
		assert.Contains(t, colNames, "ID")
		assert.Contains(t, colNames, "NICKNAME")
		assert.Contains(t, colNames, "ROLE")
		assert.Contains(t, colNames, "ACTIVE")
		assert.NotContains(t, colNames, "EMAIL")
		assert.NotContains(t, colNames, "PHONE")
		assert.NotContains(t, colNames, "CREATE TIME")
	})
}

func TestSingleUser_ToTable(t *testing.T) {
	role := &openv1alpha1resource.Role{Code: "PROJECT_READER"}
	u := newTestUser("users/single-id", "single@example.com", strPtr("Single"), role)
	p := NewSingleUser(u)

	tbl := p.ToTable(&table.PrintOpts{})
	require.Len(t, tbl.Rows, 1)
	assert.Equal(t, "single-id", tbl.Rows[0][0])
	assert.Equal(t, "PROJECT_READER", tbl.Rows[0][roleColumnIndex(tbl)])
}

func TestSingleUser_ToProtoMessage(t *testing.T) {
	u := newTestUser("users/abc", "a@b.com", nil, nil)
	p := NewSingleUser(u)
	msg := p.ToProtoMessage()

	got, ok := msg.(*openv1alpha1resource.User)
	require.True(t, ok)
	assert.Equal(t, "users/abc", got.Name)
}

func columnNames(tbl table.Table) []string {
	return columnNamesWithOpts(tbl, &table.PrintOpts{})
}

func columnNamesWithOpts(tbl table.Table, opts *table.PrintOpts) []string {
	names := make([]string, len(tbl.ColumnDefs))
	for i, cd := range tbl.ColumnDefs {
		if cd.FieldNameFunc != nil {
			names[i] = cd.FieldNameFunc(opts)
		} else {
			names[i] = cd.FieldName
		}
	}
	return names
}

func roleColumnIndex(tbl table.Table) int {
	for i, cd := range tbl.ColumnDefs {
		if cd.FieldName == "ROLE" {
			return i
		}
	}
	return -1
}
