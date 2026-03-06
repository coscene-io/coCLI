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
)

func newTestRole(name, displayName, code, level string) *openv1alpha1resource.Role {
	return &openv1alpha1resource.Role{
		Name:        name,
		DisplayName: displayName,
		Code:        code,
		Level:       level,
	}
}

func TestRole_ToProtoMessage(t *testing.T) {
	roles := []*openv1alpha1resource.Role{
		newTestRole("roles/abc-123", "Org Admin", "ORGANIZATION_ADMIN", "organization"),
	}
	p := NewRole(roles, "next-token")
	msg := p.ToProtoMessage()

	resp, ok := msg.(*openv1alpha1service.ListRolesResponse)
	require.True(t, ok)
	assert.Len(t, resp.Roles, 1)
	assert.Equal(t, "next-token", resp.NextPageToken)
	assert.Equal(t, int64(1), resp.TotalSize)
}

func TestRole_Sorting(t *testing.T) {
	roles := []*openv1alpha1resource.Role{
		newTestRole("roles/3", "Project Writer", "PROJECT_WRITER", "project"),
		newTestRole("roles/1", "Org Admin", "ORGANIZATION_ADMIN", "organization"),
		newTestRole("roles/4", "Project Admin", "PROJECT_ADMIN", "project"),
		newTestRole("roles/2", "Org Reader", "ORGANIZATION_READER", "organization"),
	}
	p := NewRole(roles, "")

	assert.Equal(t, "ORGANIZATION_ADMIN", p.Delegate[0].GetCode())
	assert.Equal(t, "ORGANIZATION_READER", p.Delegate[1].GetCode())
	assert.Equal(t, "PROJECT_ADMIN", p.Delegate[2].GetCode())
	assert.Equal(t, "PROJECT_WRITER", p.Delegate[3].GetCode())
}

func TestRole_ToTable(t *testing.T) {
	roles := []*openv1alpha1resource.Role{
		newTestRole("roles/abc-123", "Org Admin", "ORGANIZATION_ADMIN", "organization"),
		newTestRole("roles/xyz-456", "Project Reader", "PROJECT_READER", "project"),
	}
	p := NewRole(roles, "")

	t.Run("default columns", func(t *testing.T) {
		tbl := p.ToTable(&table.PrintOpts{})
		colNames := roleColumnNames(tbl)
		assert.Contains(t, colNames, "NAME")
		assert.Contains(t, colNames, "DISPLAY NAME")
		assert.Contains(t, colNames, "CODE")
		assert.Contains(t, colNames, "LEVEL")

		require.Len(t, tbl.Rows, 2)
		assert.Equal(t, "abc-123", tbl.Rows[0][0])
		assert.Equal(t, "Org Admin", tbl.Rows[0][1])
		assert.Equal(t, "ORGANIZATION_ADMIN", tbl.Rows[0][2])
		assert.Equal(t, "organization", tbl.Rows[0][3])
	})

	t.Run("verbose shows resource name", func(t *testing.T) {
		opts := &table.PrintOpts{Verbose: true}
		tbl := p.ToTable(opts)
		colNames := roleColumnNamesWithOpts(tbl, opts)
		assert.Contains(t, colNames, "RESOURCE NAME")
		assert.NotContains(t, colNames, "NAME")

		assert.Equal(t, "roles/abc-123", tbl.Rows[0][0])
	})

	t.Run("non-verbose trims roles/ prefix", func(t *testing.T) {
		tbl := p.ToTable(&table.PrintOpts{})
		assert.Equal(t, "abc-123", tbl.Rows[0][0])
		assert.Equal(t, "xyz-456", tbl.Rows[1][0])
	})

	t.Run("omit fields", func(t *testing.T) {
		tbl := p.ToTable(&table.PrintOpts{OmitFields: []string{"NAME"}})
		colNames := roleColumnNames(tbl)
		assert.Contains(t, colNames, "DISPLAY NAME")
		assert.Contains(t, colNames, "CODE")
		assert.Contains(t, colNames, "LEVEL")
		assert.NotContains(t, colNames, "NAME")
		assert.Len(t, tbl.Rows[0], 3)
	})
}

func roleColumnNames(tbl table.Table) []string {
	return roleColumnNamesWithOpts(tbl, &table.PrintOpts{})
}

func roleColumnNamesWithOpts(tbl table.Table, opts *table.PrintOpts) []string {
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
