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

package name

import (
	"fmt"

	"github.com/oriser/regroup"
	"github.com/pkg/errors"
)

// ProjectFile represents a file resource directly under a project (not under a record).
// Format: projects/{project}/files/{filename}
type ProjectFile struct {
	ProjectID string
	Filename  string
}

var (
	projectFileRe = regroup.MustCompile(`^projects/(?P<project>.*)/files/(?P<file>.*)$`)
)

func NewProjectFile(file string) (*ProjectFile, error) {
	if match, err := projectFileRe.Groups(file); err != nil {
		return nil, errors.Wrap(err, "parse project file name")
	} else {
		return &ProjectFile{ProjectID: match["project"], Filename: match["file"]}, nil
	}
}

func (f ProjectFile) Project() Project {
	return Project{ProjectID: f.ProjectID}
}

func (f ProjectFile) String() string {
	return fmt.Sprintf("projects/%s/files/%s", f.ProjectID, f.Filename)
}
