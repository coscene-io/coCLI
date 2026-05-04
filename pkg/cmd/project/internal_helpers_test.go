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

package project

import (
	"testing"

	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectTableOpts(t *testing.T) {
	format, opts := projectTableOpts(false, "table")
	assert.Equal(t, "table", format)
	assert.False(t, opts.Wide)
	assert.Equal(t, []string{"DISPLAY NAME"}, opts.OmitFields)

	format, opts = projectTableOpts(true, "wide")
	assert.Equal(t, "table", format)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Wide)
	assert.Empty(t, opts.OmitFields)
}

func TestParseTemplateScopes(t *testing.T) {
	got, err := parseTemplateScopes("custom_fields, ACTIONS,triggers, layouts")
	require.NoError(t, err)
	assert.Equal(t, []openv1alpha1service.CreateProjectUsingTemplateRequest_TemplateScope{
		openv1alpha1service.CreateProjectUsingTemplateRequest_CUSTOM_FIELDS,
		openv1alpha1service.CreateProjectUsingTemplateRequest_ACTIONS,
		openv1alpha1service.CreateProjectUsingTemplateRequest_TRIGGERS,
		openv1alpha1service.CreateProjectUsingTemplateRequest_LAYOUTS,
	}, got)

	got, err = parseTemplateScopes("")
	require.NoError(t, err)
	assert.Nil(t, got)

	_, err = parseTemplateScopes("records")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported scope")
}
