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

package cmd

import (
	"bytes"
	"testing"

	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/stretchr/testify/assert"
)

func TestRootDescriptionsIncludeReleaseContext(t *testing.T) {
	assert.Contains(t, rootLongDescription(), "Release channel:")
	assert.Contains(t, rootLongDescription(), "Base API endpoint:")
	assert.Contains(t, rootVersionString(), "Download base:")
}

func TestUpdateCommandSkipsAuth(t *testing.T) {
	var buf bytes.Buffer
	cmd := NewUpdateCommand(iostreams.Test(nil, &buf, &buf))

	assert.Equal(t, "update", cmd.Use)
	assert.False(t, cmd_utils.IsAuthCheckEnabled(cmd))
	assert.NoError(t, cmd.Args(cmd, nil))
	assert.Error(t, cmd.Args(cmd, []string{"extra"}))
}
