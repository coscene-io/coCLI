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
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The get command must register with the -p/--project and -o/--output flags.
func TestGetCommandFlags(t *testing.T) {
	cfgPath := setupGetTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	getCmd, _, err := cmd.Find([]string{"get"})
	require.NoError(t, err)
	assert.Equal(t, "get", getCmd.Name())
	assert.NotEmpty(t, getCmd.Short)
	assert.NotNil(t, getCmd.Flag("project"), "flag --project not found")
	assert.NotNil(t, getCmd.Flag("output"), "flag --output not found")
	// -p / -o shorthands must be wired.
	assert.Equal(t, "p", getCmd.Flag("project").Shorthand)
	assert.Equal(t, "o", getCmd.Flag("output").Shorthand)
}

// get requires exactly one positional argument (the action resource-name/id).
func TestGetCommandRequiresExactlyOneArg(t *testing.T) {
	cfgPath := setupGetTestConfigPath(t)
	var buf bytes.Buffer
	io := iostreams.Test(nil, &buf, &buf)
	cmd := NewRootCommand(&cfgPath, io, config.Provide)

	getCmd, _, err := cmd.Find([]string{"get"})
	require.NoError(t, err)
	assert.Error(t, getCmd.Args(getCmd, []string{}), "zero args should be rejected")
	assert.NoError(t, getCmd.Args(getCmd, []string{"a1"}))
	assert.Error(t, getCmd.Args(getCmd, []string{"a1", "a2"}), "two args should be rejected")
}

// Contract (F1): the -o yaml / -o json output produced by `get` is the full
// protojson Action (name, author, spec) that `action update -f` will later
// consume. These format tests pin that shape so the get -> edit -> update loop
// round-trips. They exercise the exact printer + printable used by the command.
func TestGetCommandOutputFormats(t *testing.T) {
	action := &openv1alpha1resource.Action{
		Name:   "projects/p1/actions/a1",
		Author: "users/u1",
		Spec: &openv1alpha1commons.ActionSpec{
			Name:        "my-action",
			Description: "desc",
			Jobs: []*openv1alpha1commons.JobSpec{{
				Name: "main",
				JobKind: &openv1alpha1commons.JobSpec_Container{
					Container: &openv1alpha1commons.ContainerJobSpec{Image: "ubuntu:22.04"},
				},
			}},
		},
	}

	t.Run("json is protojson Action", func(t *testing.T) {
		out := renderSingleAction(t, "json", action)
		compact := strings.Join(strings.Fields(out), "")
		assert.Contains(t, compact, `"name":"projects/p1/actions/a1"`)
		assert.Contains(t, compact, `"author":"users/u1"`)
		assert.Contains(t, compact, `"image":"ubuntu:22.04"`)
	})

	t.Run("yaml is protojson Action", func(t *testing.T) {
		out := renderSingleAction(t, "yaml", action)
		assert.Contains(t, out, "projects/p1/actions/a1")
		assert.Contains(t, out, "users/u1")
		assert.Contains(t, out, "ubuntu:22.04")
	})

	t.Run("table renders the action title", func(t *testing.T) {
		out := renderSingleAction(t, "table", action)
		assert.Contains(t, out, "ID")
		assert.NotContains(t, out, "RESOURCE NAME")
		assert.NotContains(t, out, "projects/p1/actions/a1")
		assert.Contains(t, out, "a1")
		assert.Contains(t, out, "my-action")
	})

	t.Run("unsupported format errors", func(t *testing.T) {
		_, err := printer.Printer("xml", &printer.Options{TableOpts: &table.PrintOpts{Verbose: true}})
		require.Error(t, err)
	})
}

// renderSingleAction prints an action through the exact printer + printable
// pair the get command uses, returning the rendered string.
func renderSingleAction(t *testing.T, format string, action *openv1alpha1resource.Action) string {
	t.Helper()
	p, err := printer.Printer(format, &printer.Options{TableOpts: &table.PrintOpts{}})
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, p.PrintObj(printable.NewSingleAction(action), &out))
	return out.String()
}

// When GetByName returns a connect NotFound (a deleted/absent action), the get
// command prints the shared clean not-found message rather than the raw
// `not_found` connect error. ActionId2Name resolves a plain `get <name>` with
// no server call, so the final GetByName guard is the one that catches this —
// and it prints via printActionNotFound, the exact helper delete/update use, so
// the wording stays identical across all three commands.
func TestPrintActionNotFoundMessage(t *testing.T) {
	proj := &name.Project{ProjectID: "p1"}
	var out bytes.Buffer
	io := iostreams.Test(nil, &out, &bytes.Buffer{})

	printActionNotFound(io, "my-action", proj)

	got := out.String()
	assert.Contains(t, got, "failed to find action")
	assert.Contains(t, got, "my-action")
	assert.Contains(t, got, proj.String())
	// The message is not the raw connect NotFound error surfaced by log.Fatalf.
	assert.NotContains(t, got, "not_found")
	// Pin the exact format shared by get/delete/update so their wording matches.
	assert.Equal(t, "failed to find action: my-action in project: "+proj.String()+"\n", got)
}

func setupGetTestConfigPath(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(t.TempDir()) + "/test-config.yaml"
}
