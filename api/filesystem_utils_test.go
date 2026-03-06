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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatRegion(t *testing.T) {
	t.Run("empty returns unspecified", func(t *testing.T) {
		assert.Equal(t, "unspecified", FormatRegion(""))
	})

	t.Run("non-empty returns as-is", func(t *testing.T) {
		assert.Equal(t, "cn-hangzhou", FormatRegion("cn-hangzhou"))
		assert.Equal(t, "cn-shanghai", FormatRegion("cn-shanghai"))
	})
}
