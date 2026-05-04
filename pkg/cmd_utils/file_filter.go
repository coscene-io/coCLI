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
	"fmt"
	"strconv"
	"strings"
)

func FileDirFilter(dir string, recursive bool) string {
	var filterParts []string
	if recursive {
		filterParts = append(filterParts, `recursive="true"`)
	}
	if dir != "" {
		normalizedDir := strings.TrimSuffix(dir, "/")
		filterParts = append(filterParts, fmt.Sprintf("dir=%s", strconv.Quote(normalizedDir)))
	}
	return strings.Join(filterParts, " AND ")
}
