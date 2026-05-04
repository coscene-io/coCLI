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

package cocli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	oldVersion := version
	oldGitCommit := gitCommit
	oldGitTag := gitTag
	oldGitTreeState := gitTreeState
	t.Cleanup(func() {
		version = oldVersion
		gitCommit = oldGitCommit
		gitTag = oldGitTag
		gitTreeState = oldGitTreeState
	})

	version = "v1.2.3"
	gitCommit = "abcdef123456"
	gitTag = ""
	gitTreeState = "dirty"
	assert.Equal(t, "v1.2.3+abcdef1.dirty", GetVersion())

	gitTreeState = "clean"
	assert.Equal(t, "v1.2.3+abcdef1", GetVersion())

	gitTag = "v1.2.4"
	assert.Equal(t, "v1.2.4", GetVersion())

	gitCommit = "abc"
	gitTag = ""
	assert.Equal(t, "v1.2.3+unknown", GetVersion())
}
