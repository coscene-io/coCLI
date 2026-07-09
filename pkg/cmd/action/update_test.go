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
	"bytes"
	"strings"
	"testing"

	openv1alpha1commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The update command must register with -p/--project, -f/--file, --dry-run and
// -o/--output.
func TestUpdateCommandFlags(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	updateCmd, _, err := cmd.Find([]string{"update"})
	require.NoError(t, err)
	assert.Equal(t, "update", updateCmd.Name())
	assert.NotEmpty(t, updateCmd.Short)
	assert.NotNil(t, updateCmd.Flag("project"), "flag --project not found")
	assert.NotNil(t, updateCmd.Flag("file"), "flag --file not found")
	assert.NotNil(t, updateCmd.Flag("dry-run"), "flag --dry-run not found")
	assert.NotNil(t, updateCmd.Flag("output"), "flag --output not found")
	assert.Equal(t, "p", updateCmd.Flag("project").Shorthand)
	assert.Equal(t, "f", updateCmd.Flag("file").Shorthand)
	assert.Equal(t, "o", updateCmd.Flag("output").Shorthand)

	// No inline field flags and no --example (plan Unit 9): update is
	// file-based only.
	assert.Nil(t, updateCmd.Flag("example"), "update must not expose --example")
	assert.Nil(t, updateCmd.Flag("name"), "update must not expose inline field flags")
	assert.Nil(t, updateCmd.Flag("image"), "update must not expose inline field flags")
}

// update requires exactly one positional argument (the action resource-name/id).
func TestUpdateCommandRequiresExactlyOneArg(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	updateCmd, _, err := cmd.Find([]string{"update"})
	require.NoError(t, err)
	assert.Error(t, updateCmd.Args(updateCmd, []string{}), "zero args should be rejected")
	assert.NoError(t, updateCmd.Args(updateCmd, []string{"a1"}))
	assert.Error(t, updateCmd.Args(updateCmd, []string{"a1", "a2"}), "two args should be rejected")
}

// cocli always sends exactly update_mask=["spec"] (plan D2/D13/C4).
func TestUpdateMaskIsSpecOnly(t *testing.T) {
	assert.Equal(t, []string{"spec"}, updateMaskSpec)
}

// dumpActionAs renders an Action through the exact printer + printable the get
// command uses, returning the bytes an operator would capture with
// `cocli action get -o <format>`.
func dumpActionAs(t *testing.T, format string, action *openv1alpha1resource.Action) []byte {
	t.Helper()
	p, err := printer.Printer(format, &printer.Options{TableOpts: &table.PrintOpts{Verbose: true}})
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, p.PrintObj(printable.NewSingleAction(action), &out))
	return out.Bytes()
}

// TestUpdateLoaderRoundTripsGetDump is the F1 guard: a `get -o yaml` (and
// `-o json`) dump — which carries output-only fields name/author/timestamps and
// a full spec — must parse cleanly into update's loader and yield the same spec.
// This proves the get -> edit -> update loop round-trips (plan D5/F1). It is the
// central reason update uses a proto-native loader rather than create's strict
// KnownFields YAML loader.
func TestUpdateLoaderRoundTripsGetDump(t *testing.T) {
	action := &openv1alpha1resource.Action{
		Name:   "projects/p1/actions/a1",
		Author: "users/u1",
		Spec: &openv1alpha1commons.ActionSpec{
			Name:        "my-action",
			Description: "desc",
			Labels:      []string{"labels/l1", "labels/l2"},
			Parameters:  map[string]string{"x": "default"},
			Jobs: []*openv1alpha1commons.JobSpec{{
				Name: "main",
				JobKind: &openv1alpha1commons.JobSpec_Container{
					Container: &openv1alpha1commons.ContainerJobSpec{
						Image:   "ubuntu:22.04",
						Command: []string{"python", "run.py"},
						Env:     map[string]string{"COS_KEY": "value"},
					},
				},
			}},
			Quota: &openv1alpha1commons.Quota{
				Cpu:    openv1alpha1commons.Quota_CPU_QUOTA_1C,
				Memory: openv1alpha1commons.Quota_MEMORY_QUOTA_2G,
			},
		},
	}

	for _, format := range []string{"yaml", "json"} {
		t.Run(format+" dump round-trips into loader", func(t *testing.T) {
			dump := dumpActionAs(t, format, action)
			spec, err := loadActionUpdateSpec("-", bytes.NewReader(dump))
			require.NoError(t, err, "get -o %s output must parse cleanly into update's loader", format)

			assert.Equal(t, "my-action", spec.GetName())
			assert.Equal(t, "desc", spec.GetDescription())
			assert.Equal(t, []string{"labels/l1", "labels/l2"}, spec.GetLabels())
			require.Len(t, spec.GetJobs(), 1)
			job := spec.GetJobs()[0]
			assert.Equal(t, "main", job.GetName())
			require.NotNil(t, job.GetContainer())
			assert.Equal(t, "ubuntu:22.04", job.GetContainer().GetImage())
			assert.Equal(t, []string{"python", "run.py"}, job.GetContainer().GetCommand())
			assert.Equal(t, openv1alpha1commons.Quota_CPU_QUOTA_1C, spec.GetQuota().GetCpu())

			// The loaded spec must survive create's proto-level validation so
			// the command would accept it.
			require.NoError(t, validateActionForCreate(&openv1alpha1resource.Action{Spec: spec}))
		})
	}
}

// A bare-spec file (just the ActionSpec, no wrapping Action) also loads: the
// spec fields sit at the top level of the Action message, so a spec-shaped doc
// parses identically. This keeps hand-authored specs working too.
func TestUpdateLoaderAcceptsBareSpecShape(t *testing.T) {
	specDump := []byte(`name: my-action
description: desc
jobs:
  - name: main
    container:
      image: ubuntu:22.04
`)
	spec, err := loadActionUpdateSpec("-", bytes.NewReader(specDump))
	require.NoError(t, err)
	assert.Equal(t, "my-action", spec.GetName())
	require.Len(t, spec.GetJobs(), 1)
	assert.Equal(t, "ubuntu:22.04", spec.GetJobs()[0].GetContainer().GetImage())
}

// Output-only fields present in a get dump (createTime/updateTime and any future
// server-only field) are tolerated via DiscardUnknown rather than rejected.
func TestUpdateLoaderToleratesOutputOnlyFields(t *testing.T) {
	dump := []byte(`name: projects/p1/actions/a1
author: users/u1
create_time: "2026-07-08T00:00:00Z"
update_time: "2026-07-08T00:00:00Z"
some_future_server_field: whatever
spec:
  name: my-action
  jobs:
    - name: main
      container:
        image: ubuntu:22.04
`)
	spec, err := loadActionUpdateSpec("-", bytes.NewReader(dump))
	require.NoError(t, err)
	assert.Equal(t, "my-action", spec.GetName())
}

// Empty stdin / empty file is rejected client-side before any wire call.
func TestUpdateLoaderRejectsEmptyInput(t *testing.T) {
	_, err := loadActionUpdateSpec("-", bytes.NewReader([]byte("   \n")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

// A spec with no jobs is rejected client-side (reuses create's validation).
func TestUpdateInvalidSpecRejectedClientSide(t *testing.T) {
	spec, err := loadActionUpdateSpec("-", bytes.NewReader([]byte("name: my-action\n")))
	require.NoError(t, err)
	err = validateActionForCreate(&openv1alpha1resource.Action{Spec: spec})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jobs cannot be empty")
}

// A RESOURCE_EXHAUSTED / NO_SUBSCRIPTION failure is detected as the no-retry
// case so update surfaces the "missing grant, do not retry" message (plan D14).
// This shares create's classifier; pin its behavior for the update path.
func TestUpdateNoSubscriptionErrorClassification(t *testing.T) {
	exhausted := connect.NewError(connect.CodeResourceExhausted, nil)
	assert.True(t, isNoSubscriptionError(exhausted), "ResourceExhausted must be treated as no-retry")

	notFound := connect.NewError(connect.CodeNotFound, nil)
	assert.False(t, isNoSubscriptionError(notFound), "NotFound must not be treated as no-retry")
}

// warnActionUpdateLabelDetach warns only when the submitted spec drops labels
// that the current action has (plan D15/F4).
func TestUpdateLabelDetachWarning(t *testing.T) {
	current := &openv1alpha1resource.Action{
		Name: "projects/p1/actions/a1",
		Spec: &openv1alpha1commons.ActionSpec{Labels: []string{"labels/l1"}},
	}

	fetch := func(a *openv1alpha1resource.Action) func() (*openv1alpha1resource.Action, error) {
		return func() (*openv1alpha1resource.Action, error) { return a, nil }
	}

	t.Run("warns when spec omits labels the action has", func(t *testing.T) {
		var errBuf bytes.Buffer
		io := iostreams.Test(nil, &bytes.Buffer{}, &errBuf)
		warnActionUpdateLabelDetach(&openv1alpha1commons.ActionSpec{}, fetch(current), io)
		assert.Contains(t, errBuf.String(), "DETACH")
	})

	t.Run("silent when spec keeps labels", func(t *testing.T) {
		var errBuf bytes.Buffer
		io := iostreams.Test(nil, &bytes.Buffer{}, &errBuf)
		warnActionUpdateLabelDetach(&openv1alpha1commons.ActionSpec{Labels: []string{"labels/l1"}}, fetch(current), io)
		assert.Empty(t, errBuf.String())
	})

	t.Run("silent when the current action has no labels", func(t *testing.T) {
		var errBuf bytes.Buffer
		io := iostreams.Test(nil, &bytes.Buffer{}, &errBuf)
		warnActionUpdateLabelDetach(&openv1alpha1commons.ActionSpec{}, fetch(&openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{}}), io)
		assert.Empty(t, errBuf.String())
	})
}

// Help text documents the label-detach data-loss path (plan D15/F4) and the
// round-trip contract (F1).
func TestUpdateHelpDocumentsLabelAndRoundTrip(t *testing.T) {
	cfgPath := setupActionCmdTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	updateCmd, _, err := cmd.Find([]string{"update"})
	require.NoError(t, err)
	assert.Contains(t, updateCmd.Long, "label", "help must warn about label detach")
	assert.Contains(t, strings.ToLower(updateCmd.Long), "get")
}
