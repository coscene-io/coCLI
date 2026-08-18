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

func TestNewEvent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantProject string
		wantEvent   string
		wantErr     bool
	}{
		{
			name:        "valid",
			input:       "projects/11111111-1111-1111-1111-111111111111/events/22222222-2222-2222-2222-222222222222",
			wantProject: "11111111-1111-1111-1111-111111111111",
			wantEvent:   "22222222-2222-2222-2222-222222222222",
		},
		{name: "missing events segment", input: "projects/p1/e1", wantErr: true},
		{name: "extra segment", input: "projects/p1/events/e1/details", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := NewEvent(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantProject, event.ProjectID)
			assert.Equal(t, tt.wantEvent, event.ID)
			assert.Equal(t, tt.input, event.String())
			assert.Equal(t, tt.wantProject, event.Project().ProjectID)
		})
	}
}
