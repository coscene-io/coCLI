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

package name

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProject(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{"valid", "projects/abc-123", "abc-123", false},
		{"valid uuid", "projects/d9b9d56b-0d43-4719-b7cc-0d7e6616bb8a", "d9b9d56b-0d43-4719-b7cc-0d7e6616bb8a", false},
		{"empty id", "projects/", "", false},
		{"invalid prefix", "repos/abc", "", true},
		{"empty string", "", "", true},
		{"no slash", "projects", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProject(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, p.ProjectID)
			assert.Equal(t, tt.input, p.String())
		})
	}
}

func TestNewRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantProj string
		wantRec  string
		wantErr  bool
	}{
		{"valid", "projects/p1/records/r1", "p1", "r1", false},
		{"uuid ids", "projects/aaa-bbb/records/ccc-ddd", "aaa-bbb", "ccc-ddd", false},
		{"empty project", "projects//records/r1", "", "", true},
		{"empty record", "projects/p1/records/", "", "", true},
		{"missing records segment", "projects/p1/r1", "", "", true},
		{"empty string", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRecord(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantProj, r.ProjectID)
			assert.Equal(t, tt.wantRec, r.RecordID)
			assert.Equal(t, tt.input, r.String())
			assert.Equal(t, tt.wantProj, r.Project().ProjectID)
		})
	}
}

func TestNewFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantProj string
		wantRec  string
		wantFile string
		wantErr  bool
	}{
		{"valid", "projects/p1/records/r1/files/data.bin", "p1", "r1", "data.bin", false},
		{"nested path", "projects/p1/records/r1/files/dir/sub/file.txt", "p1", "r1", "dir/sub/file.txt", false},
		{"nested reserved segments", "projects/p1/records/r1/files/dir/files/records/data.bin", "p1", "r1", "dir/files/records/data.bin", false},
		{"missing files segment", "projects/p1/records/r1/data.bin", "", "", "", true},
		{"empty", "", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFile(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantProj, f.ProjectID)
			assert.Equal(t, tt.wantRec, f.RecordID)
			assert.Equal(t, tt.wantFile, f.Filename)
			assert.Equal(t, tt.input, f.String())
			assert.Equal(t, tt.wantProj, f.Project().ProjectID)
		})
	}
}

func TestNewAction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantProj string
		wantID   string
		wantErr  bool
	}{
		{"project action", "projects/p1/actions/act1", "p1", "act1", false},
		{"wftmpl format", "wftmpls/tmpl-123", "", "tmpl-123", false},
		{"invalid format", "actions/act1", "", "", true},
		{"empty", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewAction(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantProj, a.ProjectID)
			assert.Equal(t, tt.wantID, a.ID)
			assert.Equal(t, tt.input, a.String())
		})
	}
}

func TestAction_IsWftmpl(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   bool
	}{
		{"wftmpl", Action{ProjectID: "", ID: "tmpl-1"}, true},
		{"project action", Action{ProjectID: "p1", ID: "act1"}, false},
		{"empty", Action{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.action.IsWftmpl())
		})
	}
}

func TestAction_String(t *testing.T) {
	assert.Equal(t, "wftmpls/tmpl-1", Action{ID: "tmpl-1"}.String())
	assert.Equal(t, "projects/p1/actions/a1", Action{ProjectID: "p1", ID: "a1"}.String())
}

func TestNewActionRun(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantProj string
		wantID   string
		wantErr  bool
	}{
		{"valid", "projects/p1/actionRuns/run-1", "p1", "run-1", false},
		{"invalid format", "projects/p1/runs/run-1", "", "", true},
		{"empty", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ar, err := NewActionRun(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantProj, ar.ProjectID)
			assert.Equal(t, tt.wantID, ar.ID)
			assert.Equal(t, tt.input, ar.String())
			assert.Equal(t, tt.wantProj, ar.Project().ProjectID)
		})
	}
}

func TestNewOrganization(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{"valid", "organizations/my-org", "my-org", false},
		{"invalid prefix", "orgs/my-org", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := NewOrganization(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, o.OrgID)
			assert.Equal(t, tt.input, o.String())
		})
	}
}

func TestNewUser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{"valid", "users/user-123", "user-123", false},
		{"invalid prefix", "people/user-123", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := NewUser(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, u.UserID)
			assert.Equal(t, tt.input, u.String())
		})
	}
}

func TestNewProjectFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantProj string
		wantFile string
		wantErr  bool
	}{
		{"valid", "projects/p1/files/readme.md", "p1", "readme.md", false},
		{"nested path", "projects/p1/files/dir/sub/file.txt", "p1", "dir/sub/file.txt", false},
		{"nested files segment", "projects/p1/files/dir/files/readme.md", "p1", "dir/files/readme.md", false},
		{"missing files segment", "projects/p1/readme.md", "", "", true},
		{"empty", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf, err := NewProjectFile(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantProj, pf.ProjectID)
			assert.Equal(t, tt.wantFile, pf.Filename)
			assert.Equal(t, tt.input, pf.String())
			assert.Equal(t, tt.wantProj, pf.Project().ProjectID)
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"d9b9d56b-0d43-4719-b7cc-0d7e6616bb8a", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"D9B9D56B-0D43-4719-B7CC-0D7E6616BB8A", false}, // uppercase not matched
		{"not-a-uuid", false},
		{"d9b9d56b-0d43-4719-b7cc", false},
		{"", false},
		{"d9b9d56b0d434719b7cc0d7e6616bb8a", false}, // no dashes
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, IsUUID(tt.input))
		})
	}
}
