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

package record

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordTableOpts(t *testing.T) {
	format, opts := recordTableOpts(false, "table", []string{"ARCHIVED"})
	assert.Equal(t, "table", format)
	assert.False(t, opts.Wide)
	assert.Equal(t, []string{"ARCHIVED"}, opts.OmitFields)

	format, opts = recordTableOpts(true, "wide", nil)
	assert.Equal(t, "table", format)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Wide)
	assert.Empty(t, opts.OmitFields)

	format, opts = recordTableOpts(false, "json", nil)
	assert.Equal(t, "json", format)
	assert.False(t, opts.Wide)
}
