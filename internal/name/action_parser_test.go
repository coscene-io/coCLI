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

	"github.com/stretchr/testify/require"
)

func TestNewActionRejectsNestedResourcePath(t *testing.T) {
	_, err := NewAction("projects/p1/actions/a1/runs/run1")

	require.Error(t, err)
}

func TestNewActionRejectsNestedWftmplPath(t *testing.T) {
	_, err := NewAction("wftmpls/tmpl1/actions/a1")

	require.Error(t, err)
}

func TestNewActionRunRejectsNestedResourcePath(t *testing.T) {
	_, err := NewActionRun("projects/p1/actionRuns/run1/logs/log1")

	require.Error(t, err)
}
