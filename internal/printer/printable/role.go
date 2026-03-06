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
	"sort"
	"strings"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/internal/printer/table"
	"google.golang.org/protobuf/proto"
)

const (
	roleIdTrimSize          = 36
	roleDisplayNameTrimSize = 30
	roleCodeTrimSize        = 25
	roleLevelTrimSize       = 15
)

type Role struct {
	Delegate      []*openv1alpha1resource.Role
	NextPageToken string
}

func NewRole(roles []*openv1alpha1resource.Role, nextPageToken string) *Role {
	sorted := make([]*openv1alpha1resource.Role, len(roles))
	copy(sorted, roles)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].GetLevel() != sorted[j].GetLevel() {
			return sorted[i].GetLevel() < sorted[j].GetLevel()
		}
		return sorted[i].GetCode() < sorted[j].GetCode()
	})
	return &Role{Delegate: sorted, NextPageToken: nextPageToken}
}

func (p *Role) ToProtoMessage() proto.Message {
	return &openv1alpha1service.ListRolesResponse{
		Roles:         p.Delegate,
		NextPageToken: p.NextPageToken,
		TotalSize:     int64(len(p.Delegate)),
	}
}

func (p *Role) ToTable(opts *table.PrintOpts) table.Table {
	fullColumnDefs := []table.ColumnDefinitionFull[*openv1alpha1resource.Role]{
		{
			FieldNameFunc: func(opts *table.PrintOpts) string {
				if opts.Verbose {
					return "RESOURCE NAME"
				}
				return "NAME"
			},
			FieldValueFunc: func(r *openv1alpha1resource.Role, opts *table.PrintOpts) string {
				if opts.Verbose {
					return r.GetName()
				}
				return strings.TrimPrefix(r.GetName(), "roles/")
			},
			TrimSize: roleIdTrimSize,
		},
		{
			FieldName: "DISPLAY NAME",
			FieldValueFunc: func(r *openv1alpha1resource.Role, opts *table.PrintOpts) string {
				return r.GetDisplayName()
			},
			TrimSize: roleDisplayNameTrimSize,
		},
		{
			FieldName: "CODE",
			FieldValueFunc: func(r *openv1alpha1resource.Role, opts *table.PrintOpts) string {
				return r.GetCode()
			},
			TrimSize: roleCodeTrimSize,
		},
		{
			FieldName: "LEVEL",
			FieldValueFunc: func(r *openv1alpha1resource.Role, opts *table.PrintOpts) string {
				return r.GetLevel()
			},
			TrimSize: roleLevelTrimSize,
		},
	}

	return table.ColumnDefs2Table(fullColumnDefs, p.Delegate, opts)
}
