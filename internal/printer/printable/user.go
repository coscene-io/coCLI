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
	"fmt"
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer/table"
	"google.golang.org/protobuf/proto"
)

const (
	userIdTrimSize       = 36
	userNicknameTrimSize = 20
	userEmailTrimSize    = 30
	userRoleTrimSize     = 20
	userActiveTrimSize   = 6
	userPhoneTrimSize    = 15
	userTimeTrimSize     = len(time.RFC3339)
)

type User struct {
	Delegate      []*openv1alpha1resource.User
	NextPageToken string
}

func NewUser(users []*openv1alpha1resource.User, nextPageToken string) *User {
	return &User{Delegate: users, NextPageToken: nextPageToken}
}

func (p *User) ToProtoMessage() proto.Message {
	return &openv1alpha1service.ListUsersResponse{
		Users:         p.Delegate,
		NextPageToken: p.NextPageToken,
		TotalSize:     int64(len(p.Delegate)),
	}
}

func (p *User) ToTable(opts *table.PrintOpts) table.Table {
	fullColumnDefs := []table.ColumnDefinitionFull[*openv1alpha1resource.User]{
		{
			FieldNameFunc: func(opts *table.PrintOpts) string {
				if opts.Verbose {
					return "RESOURCE NAME"
				}
				return "ID"
			},
			FieldValueFunc: func(u *openv1alpha1resource.User, opts *table.PrintOpts) string {
				if opts.Verbose {
					return u.Name
				}
				userName, _ := name.NewUser(u.Name)
				if userName != nil {
					return userName.UserID
				}
				return u.Name
			},
			TrimSize: userIdTrimSize,
		},
		{
			FieldName: "NICKNAME",
			FieldValueFunc: func(u *openv1alpha1resource.User, opts *table.PrintOpts) string {
				if u.Nickname != nil {
					return *u.Nickname
				}
				return ""
			},
			TrimSize: userNicknameTrimSize,
		},
		{
			FieldName: "EMAIL",
			FieldValueFunc: func(u *openv1alpha1resource.User, opts *table.PrintOpts) string {
				return u.GetEmail()
			},
			TrimSize: userEmailTrimSize,
		},
		{
			FieldName: "ROLE",
			FieldValueFunc: func(u *openv1alpha1resource.User, opts *table.PrintOpts) string {
				if u.GetRole() != nil {
					return u.GetRole().GetCode()
				}
				return ""
			},
			TrimSize: userRoleTrimSize,
		},
		{
			FieldName: "ACTIVE",
			FieldValueFunc: func(u *openv1alpha1resource.User, opts *table.PrintOpts) string {
				return fmt.Sprintf("%v", u.GetActive())
			},
			TrimSize: userActiveTrimSize,
		},
		{
			FieldName: "PHONE",
			FieldValueFunc: func(u *openv1alpha1resource.User, opts *table.PrintOpts) string {
				return u.GetPhoneNumber()
			},
			TrimSize: userPhoneTrimSize,
		},
		{
			FieldName: "CREATE TIME",
			FieldValueFunc: func(u *openv1alpha1resource.User, opts *table.PrintOpts) string {
				if u.GetCreateTime() != nil {
					return u.GetCreateTime().AsTime().In(time.Local).Format(time.RFC3339)
				}
				return ""
			},
			TrimSize: userTimeTrimSize,
		},
	}

	return table.ColumnDefs2Table(fullColumnDefs, p.Delegate, opts)
}

type SingleUser struct {
	Delegate *openv1alpha1resource.User
}

func NewSingleUser(user *openv1alpha1resource.User) *SingleUser {
	return &SingleUser{Delegate: user}
}

func (p *SingleUser) ToProtoMessage() proto.Message {
	return p.Delegate
}

func (p *SingleUser) ToTable(opts *table.PrintOpts) table.Table {
	inner := &User{Delegate: []*openv1alpha1resource.User{p.Delegate}}
	return inner.ToTable(opts)
}
