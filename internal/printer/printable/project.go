// Copyright 2024 coScene
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
	"strings"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer/table"
	"google.golang.org/protobuf/proto"
)

const (
	projectIdTrimSize          = 36
	projectSlugTrimSize        = 30
	projectDisplayNameTrimSize = 40
	projectRegionTrimSize      = 15
	projectFileSystemTrimSize  = 20
)

type Project struct {
	Delegate       []*openv1alpha1resource.Project
	FileSystemInfo map[string]*openv1alpha1resource.FileSystem
}

func NewProject(projects []*openv1alpha1resource.Project) *Project {
	return &Project{Delegate: projects}
}

func NewProjectWithFileSystemInfo(projects []*openv1alpha1resource.Project, fsInfo map[string]*openv1alpha1resource.FileSystem) *Project {
	return &Project{Delegate: projects, FileSystemInfo: fsInfo}
}

func (p *Project) ToProtoMessage() proto.Message {
	return &openv1alpha1service.ListProjectsResponse{
		Projects:  p.Delegate,
		TotalSize: int64(len(p.Delegate)),
	}
}

func (p *Project) resolveRegion(proj *openv1alpha1resource.Project) string {
	if proj.Region != 0 {
		return api.FormatRegion(proj.Region)
	}
	if p.FileSystemInfo != nil {
		if fs, ok := p.FileSystemInfo[proj.FileSystem]; ok {
			return api.FormatRegion(fs.Region)
		}
		for k, fs := range p.FileSystemInfo {
			if strings.HasSuffix(k, "/"+proj.FileSystem) || strings.HasSuffix(k, "/fileSystems/"+strings.TrimPrefix(proj.FileSystem, "fileSystems/")) {
				return api.FormatRegion(fs.Region)
			}
		}
	}
	return ""
}

func (p *Project) resolveFileSystem(fsName string) string {
	if p.FileSystemInfo != nil {
		if fs, ok := p.FileSystemInfo[fsName]; ok && fs.DisplayName != "" {
			return fs.DisplayName
		}
		for k, fs := range p.FileSystemInfo {
			if strings.HasSuffix(k, "/"+fsName) || strings.HasSuffix(k, "/fileSystems/"+fsName) {
				if fs.DisplayName != "" {
					return fs.DisplayName
				}
			}
		}
	}
	if idx := strings.LastIndex(fsName, "/fileSystems/"); idx >= 0 {
		return fsName[idx+len("/fileSystems/"):]
	}
	return strings.TrimPrefix(fsName, "fileSystems/")
}

func (p *Project) ToTable(opts *table.PrintOpts) table.Table {
	fullColumnDefs := []table.ColumnDefinitionFull[*openv1alpha1resource.Project]{
		{
			FieldNameFunc: func(opts *table.PrintOpts) string {
				if opts.Verbose {
					return "RESOURCE NAME"
				}
				return "ID"
			},
			FieldValueFunc: func(proj *openv1alpha1resource.Project, opts *table.PrintOpts) string {
				if opts.Verbose {
					return proj.Name
				}
				projectName, _ := name.NewProject(proj.Name)
				return projectName.ProjectID
			},
			TrimSize: projectIdTrimSize,
		},
		{
			FieldName: "SLUG",
			FieldValueFunc: func(proj *openv1alpha1resource.Project, opts *table.PrintOpts) string {
				return proj.Slug
			},
			TrimSize: projectSlugTrimSize,
		},
		{
			FieldName: "DISPLAY NAME",
			FieldValueFunc: func(proj *openv1alpha1resource.Project, opts *table.PrintOpts) string {
				return proj.DisplayName
			},
			TrimSize: projectDisplayNameTrimSize,
		},
		{
			FieldName: "REGION",
			FieldValueFunc: func(proj *openv1alpha1resource.Project, opts *table.PrintOpts) string {
				return p.resolveRegion(proj)
			},
			TrimSize: projectRegionTrimSize,
		},
		{
			FieldName: "FILE SYSTEM",
			FieldValueFunc: func(proj *openv1alpha1resource.Project, opts *table.PrintOpts) string {
				return p.resolveFileSystem(proj.FileSystem)
			},
			TrimSize: projectFileSystemTrimSize,
		},
	}

	return table.ColumnDefs2Table(fullColumnDefs, p.Delegate, opts)
}
