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

package action

import (
	"fmt"

	"github.com/coscene-io/cocli/internal/name"
)

// resolveActionRun accepts a full action-run resource name or a bare UUID.
func resolveActionRun(arg string, proj *name.Project) (*name.ActionRun, error) {
	if actionRun, err := name.NewActionRun(arg); err == nil {
		return actionRun, nil
	}
	if name.IsUUID(arg) {
		return &name.ActionRun{ProjectID: proj.ProjectID, ID: arg}, nil
	}
	return nil, fmt.Errorf("invalid action run name or id: %s", arg)
}
