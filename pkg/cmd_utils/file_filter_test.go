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

package cmd_utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileDirFilter(t *testing.T) {
	tests := []struct {
		name      string
		dir       string
		recursive bool
		want      string
	}{
		{name: "empty", want: ""},
		{name: "recursive only", recursive: true, want: `recursive="true"`},
		{name: "trim trailing slash", dir: "logs/", want: `dir="logs"`},
		{name: "escape quote", dir: `logs" OR recursive="true`, want: `dir="logs\" OR recursive=\"true"`},
		{name: "escape backslash", dir: `logs\raw`, want: `dir="logs\\raw"`},
		{name: "recursive and dir", dir: "logs", recursive: true, want: `recursive="true" AND dir="logs"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FileDirFilter(tt.dir, tt.recursive))
		})
	}
}
