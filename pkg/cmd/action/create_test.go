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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	openv1alpha1commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeActionCreateTestConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(`current-profile: test
profiles:
  - name: test
    endpoint: https://openapi.mock.coscene.com
    token: test-token
    org: test-org
    project: test-project
    project-name: projects/p1
`), 0644)
	require.NoError(t, err)
	return path
}

func TestCreateCommandExample(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{"create", "--example"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "name: my-action")
	assert.Contains(t, out.String(), "quota:")
	assert.Contains(t, out.String(), "cpu: CPU_QUOTA_1C")
	assert.Contains(t, out.String(), "memory: MEMORY_QUOTA_2G")
	// Example parameter key is lowercase (env vars stay uppercase; different namespace).
	assert.Contains(t, out.String(), "x: \"default\"")
	assert.NotContains(t, out.String(), "profile:")
}

func TestCreateCommandDryRunDoesNotRequireConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create",
		"--dry-run",
		"--name", "local-action",
		"--image", "ubuntu:22.04",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"name":"local-action"`)
	// CLI-built single job defaults to "main" (no --job-name flag anymore).
	assert.Contains(t, got, `"name":"main"`)
}

func TestCreateCommandDryRunInlineJSON(t *testing.T) {
	cfgPath := writeActionCreateTestConfig(t)
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create",
		"--dry-run",
		"--name", "inline-action",
		"--image", "ubuntu:22.04",
		"--command", `python "run script.py"`,
		"--env", "FOO=bar",
		"--param", "x=default",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	raw := out.String()
	got := compactActionCreateJSON(raw)
	assert.Contains(t, got, `"name":"inline-action"`)
	assert.Contains(t, got, `"image":"ubuntu:22.04"`)
	assert.Contains(t, raw, `"run script.py"`)
	assert.Contains(t, got, `"FOO":"bar"`)
	assert.Contains(t, got, `"x":"default"`)
}

func TestCreateCommandDryRunTableOutput(t *testing.T) {
	cfgPath := writeActionCreateTestConfig(t)
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create",
		"--dry-run",
		"--name", "table-action",
		"--image", "ubuntu:22.04",
		"-o", "table",
	})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "table-action")
}

// A -f file passes through unchanged in a dry-run (no spec-content flags are
// combined with it — those are rejected; see the flag-conflict tests).
func TestCreateCommandDryRunStdinFilePassthrough(t *testing.T) {
	cfgPath := writeActionCreateTestConfig(t)
	var out bytes.Buffer
	spec := `name: file-action
jobs:
  - name: step
    container:
      image: old-image
      command: ["echo", "{{parameters.x}}"]
parameters:
  x: old
quota:
  cpu: CPU_QUOTA_4C
  memory: MEMORY_QUOTA_8G
`
	ioStreams := iostreams.Test(io.NopCloser(strings.NewReader(spec)), &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create",
		"--dry-run",
		"-f", "-",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"name":"file-action"`)
	assert.Contains(t, got, `"image":"old-image"`)
	assert.Contains(t, got, `"x":"old"`)
	assert.Contains(t, got, `"cpu":"CPU_QUOTA_4C"`)
	assert.Contains(t, got, `"memory":"MEMORY_QUOTA_8G"`)
}

// The create -f loader is the shared proto-native loader: it accepts the full
// protojson Action a `get -o yaml/json` dump emits (spec nested under `spec:`,
// snake_case or camelCase keys) and returns the extracted *ActionSpec.
func TestLoadActionCreateSpecAcceptsProtoJSONWrapper(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.json")
	err := os.WriteFile(path, []byte(`{
  "spec": {
    "name": "json-action",
    "jobs": [
      {
        "name": "job",
        "container": {
          "image": "ubuntu:22.04",
          "command": ["echo", "ok"]
        }
      }
    ],
    "outputOptions": {"saveMode": "APPEND"},
    "storageOptions": {
      "containerStorageBytes": 123,
      "ssdOptions": {"useSsd": true, "mountPath": "/ssd"}
    }
  }
}`), 0644)
	require.NoError(t, err)

	spec, err := loadActionSpecFromFile(path, nil)
	require.NoError(t, err)

	assert.Equal(t, "json-action", spec.GetName())
	assert.Equal(t, openv1alpha1commons.OutputOptions_APPEND, spec.GetOutputOptions().GetSaveMode())
	assert.Equal(t, int64(123), spec.GetStorageOptions().GetContainerStorageBytes())
	assert.True(t, spec.GetStorageOptions().GetSsdOptions().GetUseSsd())
	assert.Equal(t, "/ssd", spec.GetStorageOptions().GetSsdOptions().GetMountPath())
}

// A misspelled/unknown key in the create -f file is tolerated (ignored), not
// rejected — the tolerant DiscardUnknown contract shared with `action update`.
func TestLoadActionCreateSpecToleratesUnknownKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	err := os.WriteFile(path, []byte(`name: typo-action
descriptionn: oops   # misspelled key, must be ignored not rejected
jobs:
  - name: main
    container:
      image: ubuntu:22.04
      command: ["echo", "ok"]
`), 0644)
	require.NoError(t, err)

	spec, err := loadActionSpecFromFile(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "typo-action", spec.GetName())
	assert.Empty(t, spec.GetDescription())
	require.Len(t, spec.GetJobs(), 1)
	assert.Equal(t, "ubuntu:22.04", spec.GetJobs()[0].GetContainer().GetImage())
}

// The --quota preset (t-shirt size) is a convenience that maps to the proto
// CPU/memory quota enum pair and sets the spec's quota.
func TestCreateCommandQuotaPresetFlag(t *testing.T) {
	cfgPath := writeActionCreateTestConfig(t)
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create", "--dry-run",
		"--name", "preset-action",
		"--image", "ubuntu:22.04",
		"--quota", "small",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"cpu":"CPU_QUOTA_1C"`)
	assert.Contains(t, got, `"memory":"MEMORY_QUOTA_2G"`)
}

// A quota: set in the -f file survives unchanged in a dry-run (the --quota
// flag can no longer be combined with -f — that is rejected; see the
// flag-conflict tests). A file spec is authoritative.
func TestCreateCommandQuotaFromFilePreserved(t *testing.T) {
	cfgPath := writeActionCreateTestConfig(t)
	var out bytes.Buffer
	spec := `name: file-action
jobs:
  - name: main
    container:
      image: ubuntu:22.04
      command: ["echo", "ok"]
quota:
  cpu: CPU_QUOTA_1C
  memory: MEMORY_QUOTA_2G
`
	ioStreams := iostreams.Test(io.NopCloser(strings.NewReader(spec)), &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create", "--dry-run",
		"-f", "-",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"cpu":"CPU_QUOTA_1C"`)
	assert.Contains(t, got, `"memory":"MEMORY_QUOTA_2G"`)
}

// The --quota preset maps a t-shirt size to the proto CPU/memory quota enum
// pair; an invalid preset fails with a clear error listing the valid sizes.
func TestActionCreateQuotaPreset(t *testing.T) {
	for _, tc := range []struct {
		preset string
		cpu    openv1alpha1commons.Quota_CPUQuota
		memory openv1alpha1commons.Quota_MemoryQuota
	}{
		{"small", openv1alpha1commons.Quota_CPU_QUOTA_1C, openv1alpha1commons.Quota_MEMORY_QUOTA_2G},
		{"medium", openv1alpha1commons.Quota_CPU_QUOTA_2C, openv1alpha1commons.Quota_MEMORY_QUOTA_4G},
		{"large", openv1alpha1commons.Quota_CPU_QUOTA_4C, openv1alpha1commons.Quota_MEMORY_QUOTA_8G},
		{"xlarge", openv1alpha1commons.Quota_CPU_QUOTA_8C, openv1alpha1commons.Quota_MEMORY_QUOTA_16G},
	} {
		t.Run(tc.preset, func(t *testing.T) {
			q, err := actionCreateQuotaPreset(tc.preset)
			require.NoError(t, err)
			assert.Equal(t, tc.cpu, q.GetCpu())
			assert.Equal(t, tc.memory, q.GetMemory())
		})
	}

	_, err := actionCreateQuotaPreset("huge")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown quota preset")
	assert.Contains(t, err.Error(), "small, medium, large, xlarge")
}

// A realistic `get -o yaml` dump (full Action with output-only name/author/
// timestamps, array command, quota as proto enum strings) round-trips through
// the create -f loader: DiscardUnknown tolerates the output-only fields and the
// spec fields survive.
func TestLoadActionCreateSpecRoundTripsGetDump(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dump.yaml")
	err := os.WriteFile(path, []byte(`name: organizations/o1/projects/p1/actions/a1
author: users/u1
createTime: "2026-01-02T03:04:05Z"
updateTime: "2026-01-02T03:04:05Z"
spec:
  name: dump-action
  jobs:
    - name: main
      container:
        image: ubuntu:22.04
        command: ["echo", "{{parameters.x}}"]
  parameters:
    x: "default"
  quota:
    cpu: CPU_QUOTA_1C
    memory: MEMORY_QUOTA_2G
`), 0644)
	require.NoError(t, err)

	spec, err := loadActionSpecFromFile(path, nil)
	require.NoError(t, err)

	// Output-only Action fields were tolerated; only the nested spec survives.
	assert.Equal(t, "dump-action", spec.GetName())
	assert.Equal(t, "default", spec.GetParameters()["x"])
	require.Len(t, spec.GetJobs(), 1)
	assert.Equal(t, []string{"echo", "{{parameters.x}}"}, spec.GetJobs()[0].GetContainer().GetCommand())
	assert.Equal(t, openv1alpha1commons.Quota_CPU_QUOTA_1C, spec.GetQuota().GetCpu())
	assert.Equal(t, openv1alpha1commons.Quota_MEMORY_QUOTA_2G, spec.GetQuota().GetMemory())
}

// The --example skeleton is valid proto YAML: it parses through the shared
// create -f loader and yields a spec that validateActionForCreate accepts.
func TestCreateExampleParsesThroughLoader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.yaml")
	require.NoError(t, os.WriteFile(path, []byte(actionCreateExample), 0644))

	spec, err := loadActionSpecFromFile(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "my-action", spec.GetName())
	require.Len(t, spec.GetJobs(), 1)
	assert.Equal(t, "main", spec.GetJobs()[0].GetName())
	assert.Equal(t, []string{"python", "run.py"}, spec.GetJobs()[0].GetContainer().GetCommand())
	assert.Equal(t, openv1alpha1commons.Quota_CPU_QUOTA_1C, spec.GetQuota().GetCpu())
	assert.Equal(t, openv1alpha1commons.Quota_MEMORY_QUOTA_2G, spec.GetQuota().GetMemory())
	assert.Equal(t, openv1alpha1commons.OutputOptions_APPEND, spec.GetOutputOptions().GetSaveMode())
	// Validation passes (the sentinel image is not itself a validation error).
	require.NoError(t, validateActionForCreate(&openv1alpha1resource.Action{Spec: spec}))
}

// A single unnamed job in the file defaults to "main" via validateActionForCreate.
func TestCreateSingleUnnamedJobDefaultsToMain(t *testing.T) {
	action := &openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{
		Name: "unnamed-job",
		Jobs: []*openv1alpha1commons.JobSpec{{
			JobKind: &openv1alpha1commons.JobSpec_Container{Container: &openv1alpha1commons.ContainerJobSpec{
				Image: "ubuntu:22.04",
			}},
		}},
	}}
	require.NoError(t, validateActionForCreate(action))
	assert.Equal(t, "main", action.Spec.Jobs[0].Name)
}

func TestValidateActionForCreate(t *testing.T) {
	action := &openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{
		Name: "valid",
		Jobs: []*openv1alpha1commons.JobSpec{{
			JobKind: &openv1alpha1commons.JobSpec_Container{Container: &openv1alpha1commons.ContainerJobSpec{
				Image:   "img",
				Command: []string{"echo {{parameters.x}}"},
			}},
		}},
		Parameters: map[string]string{"x": "default"},
	}}
	require.NoError(t, validateActionForCreate(action))
	assert.Equal(t, "main", action.Spec.Jobs[0].Name)

	action.Spec.Parameters = nil
	err := validateActionForCreate(action)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `parameter "x" is referenced but not defined`)

	// Empty name is rejected.
	err = validateActionForCreate(&openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{
		Jobs: []*openv1alpha1commons.JobSpec{{
			Name:    "main",
			JobKind: &openv1alpha1commons.JobSpec_Container{Container: &openv1alpha1commons.ContainerJobSpec{Image: "img"}},
		}},
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")

	// No jobs is rejected.
	err = validateActionForCreate(&openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{Name: "no-jobs"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jobs cannot be empty")
}

func TestValidateActionForCreateRejectsInvalidEnvNames(t *testing.T) {
	for _, envName := range []string{"1BAD", "BAD!"} {
		t.Run(envName, func(t *testing.T) {
			action := &openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{
				Name: "valid",
				Jobs: []*openv1alpha1commons.JobSpec{{
					Name: "main",
					JobKind: &openv1alpha1commons.JobSpec_Container{Container: &openv1alpha1commons.ContainerJobSpec{
						Image: "img",
						Env:   map[string]string{envName: "value"},
					}},
				}},
			}}

			err := validateActionForCreate(action)
			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("env name %q", envName))
		})
	}
}

func TestSplitActionCreateWords(t *testing.T) {
	words, err := splitActionCreateWords(`python "run script.py" --flag=value`)
	require.NoError(t, err)
	assert.Equal(t, []string{"python", "run script.py", "--flag=value"}, words)

	_, err = splitActionCreateWords(`python "unterminated`)
	assert.Error(t, err)
}

// --- -f is authoritative: spec-content flags conflict with -f ------------

// findActionCreateCmd returns the parsed `create` subcommand so tests can flip
// individual flags and exercise actionCreateFlagConflict directly. The conflict
// check lives in the command's Run behind exitf (os.Exit), which a test can't
// intercept via Execute(), so we test the extracted checker on the real flagset.
func findActionCreateCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	ioStreams := iostreams.Test(nil, &bytes.Buffer{}, &bytes.Buffer{})
	root := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	createCmd, _, err := root.Find([]string{"create"})
	require.NoError(t, err)
	return createCmd
}

// -f alone (no spec-content flag) is fine — the file is the whole spec.
func TestActionCreateFlagConflictFileAlone(t *testing.T) {
	cmd := findActionCreateCmd(t)
	require.NoError(t, cmd.Flags().Set("file", "spec.yaml"))
	require.NoError(t, actionCreateFlagConflict(cmd))
}

// Flags-only (no -f) still scaffolds a spec — no conflict.
func TestActionCreateFlagConflictFlagsOnly(t *testing.T) {
	cmd := findActionCreateCmd(t)
	require.NoError(t, cmd.Flags().Set("name", "x"))
	require.NoError(t, cmd.Flags().Set("image", "img"))
	require.NoError(t, cmd.Flags().Set("command", "run"))
	require.NoError(t, cmd.Flags().Set("quota", "small"))
	require.NoError(t, actionCreateFlagConflict(cmd))
}

// -f with a single spec-content flag errors and names exactly that flag.
func TestActionCreateFlagConflictSingleOffender(t *testing.T) {
	for _, tc := range []struct {
		flag, value, want string
	}{
		{"name", "x", "--name"},
		{"description", "d", "--description"},
		{"image", "img", "--image"},
		{"command", "run", "--command"},
		{"env", "A=B", "--env"},
		{"param", "x=1", "--param"},
		{"quota", "small", "--quota"},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			cmd := findActionCreateCmd(t)
			require.NoError(t, cmd.Flags().Set("file", "spec.yaml"))
			require.NoError(t, cmd.Flags().Set(tc.flag, tc.value))
			err := actionCreateFlagConflict(cmd)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Contains(t, err.Error(), "-f/--file")
			assert.Contains(t, err.Error(), "authoritative")
		})
	}
}

// -P is the shorthand for --param and still conflicts, reported as --param.
func TestActionCreateFlagConflictParamShorthand(t *testing.T) {
	cmd := findActionCreateCmd(t)
	require.NoError(t, cmd.Flags().Set("file", "spec.yaml"))
	require.NoError(t, cmd.Flags().Set("param", "x=1")) // -P binds to the "param" flag
	err := actionCreateFlagConflict(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--param")
}

// Multiple offenders are all listed, in the declared (stable) order.
func TestActionCreateFlagConflictMultipleOffenders(t *testing.T) {
	cmd := findActionCreateCmd(t)
	require.NoError(t, cmd.Flags().Set("file", "spec.yaml"))
	require.NoError(t, cmd.Flags().Set("name", "x"))
	require.NoError(t, cmd.Flags().Set("quota", "small"))
	err := actionCreateFlagConflict(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name")
	assert.Contains(t, err.Error(), "--quota")
	// Deterministic order: --name is declared before --quota.
	assert.Less(t, strings.Index(err.Error(), "--name"), strings.Index(err.Error(), "--quota"))
}

// Operational flags (-p/--project, --dry-run, -o/--output) stay allowed with -f.
func TestActionCreateFlagConflictOperationalFlagsAllowed(t *testing.T) {
	cmd := findActionCreateCmd(t)
	require.NoError(t, cmd.Flags().Set("file", "spec.yaml"))
	require.NoError(t, cmd.Flags().Set("project", "proj"))
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))
	require.NoError(t, cmd.Flags().Set("output", "json"))
	require.NoError(t, actionCreateFlagConflict(cmd))
}

// End-to-end OK path: -f from stdin combined with operational flags only
// (--dry-run, -o json) succeeds and prints the file's spec unchanged.
func TestCreateCommandFileWithOperationalFlagsSucceeds(t *testing.T) {
	cfgPath := writeActionCreateTestConfig(t)
	var out bytes.Buffer
	spec := `name: file-only-action
jobs:
  - name: main
    container:
      image: ubuntu:22.04
      command: ["echo", "ok"]
`
	ioStreams := iostreams.Test(io.NopCloser(strings.NewReader(spec)), &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create", "--dry-run",
		"-f", "-",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"name":"file-only-action"`)
	assert.Contains(t, got, `"image":"ubuntu:22.04"`)
}

func compactActionCreateJSON(input string) string {
	return strings.Join(strings.Fields(input), "")
}

// A minimal flags-only create must default the three option messages to empty
// (but non-nil) so the created spec is runnable — a nil MountOptions /
// StorageOptions / OutputOptions panics the matrix backend at run time. Empty
// means backend defaults; no field is set inside them.
func TestBuildActionForCreateDefaultsOptions(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	ioStreams := iostreams.Test(nil, &bytes.Buffer{}, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	createCmd, _, err := cmd.Find([]string{"create"})
	require.NoError(t, err)

	opts := &actionCreateOptions{
		name:    "minimal",
		image:   "img",
		command: "run",
		quota:   "small",
	}
	for _, f := range []string{"name", "image", "command", "quota"} {
		require.NoError(t, createCmd.Flags().Set(f, flagValueFor(opts, f)))
	}

	action, err := buildActionForCreate(opts, createCmd, nil)
	require.NoError(t, err)
	spec := action.GetSpec()
	// All three option messages are defaulted to non-nil so the spec is runnable.
	require.NotNil(t, spec.MountOptions, "MountOptions must be defaulted to non-nil")
	require.NotNil(t, spec.StorageOptions, "StorageOptions must be defaulted to non-nil")
	require.NotNil(t, spec.OutputOptions, "OutputOptions must be defaulted to non-nil")
	// Empty means backend defaults — no field is set inside them.
	assert.Equal(t, openv1alpha1commons.OutputOptions_SAVE_MODE_UNSPECIFIED, spec.OutputOptions.GetSaveMode())
}

func flagValueFor(opts *actionCreateOptions, flag string) string {
	switch flag {
	case "name":
		return opts.name
	case "image":
		return opts.image
	case "command":
		return opts.command
	case "quota":
		return opts.quota
	default:
		return ""
	}
}

// End-to-end: a minimal flags-only dry-run create emits the three empty option
// objects in the JSON, so every cocli-created action carries them.
func TestCreateCommandDryRunDefaultsOptionsInJSON(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create", "--dry-run",
		"--name", "minimal",
		"--image", "img",
		"--command", "run",
		"--quota", "small",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"mountOptions":{}`)
	assert.Contains(t, got, `"storageOptions":{}`)
	assert.Contains(t, got, `"outputOptions":{}`)
}

// --- fix-wave (Worker E) tests --------------------------------------------

// The --example skeleton must ship the obvious sentinel so an unedited example
// is easy to spot (warnActionCreateSentinels keys on exactly this string).
func TestCreateCommandExampleUsesSentinel(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{"create", "--example"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), actionCreateSentinelImage)
}

// A surviving sentinel image warns to stderr but does not fail the command.
func TestWarnActionCreateSentinels(t *testing.T) {
	t.Run("sentinel warns", func(t *testing.T) {
		var errOut bytes.Buffer
		ioStreams := iostreams.Test(nil, &bytes.Buffer{}, &errOut)
		action := &openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{
			Name: "a",
			Jobs: []*openv1alpha1commons.JobSpec{{
				Name: "job",
				JobKind: &openv1alpha1commons.JobSpec_Container{Container: &openv1alpha1commons.ContainerJobSpec{
					Image: actionCreateSentinelImage,
				}},
			}},
		}}
		warnActionCreateSentinels(action, ioStreams)
		assert.Contains(t, errOut.String(), "sentinel")
		assert.Contains(t, errOut.String(), `job "job"`)
	})

	t.Run("real image is silent", func(t *testing.T) {
		var errOut bytes.Buffer
		ioStreams := iostreams.Test(nil, &bytes.Buffer{}, &errOut)
		action := &openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{
			Name: "a",
			Jobs: []*openv1alpha1commons.JobSpec{{
				Name: "job",
				JobKind: &openv1alpha1commons.JobSpec_Container{Container: &openv1alpha1commons.ContainerJobSpec{
					Image: "ubuntu:22.04",
				}},
			}},
		}}
		warnActionCreateSentinels(action, ioStreams)
		assert.Empty(t, errOut.String())
	})
}

// Dry-run with the sentinel image emits the stderr warning end-to-end.
func TestCreateCommandDryRunWarnsOnSentinel(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	var out, errOut bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &errOut)
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create", "--dry-run",
		"--name", "sentinel-action",
		"--image", actionCreateSentinelImage,
		"-o", "json",
	})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, errOut.String(), "sentinel")
}

// D14: a ResourceExhausted / NO_SUBSCRIPTION connect error is classified as
// "do not retry"; unrelated errors are not.
func TestIsNoSubscriptionError(t *testing.T) {
	// ResourceExhausted code -> true, even wrapped (mirrors api/action.go's %w).
	re := connect.NewError(connect.CodeResourceExhausted, errors.New("quota exceeded"))
	assert.True(t, isNoSubscriptionError(fmt.Errorf("failed to create action: %w", re)))

	// NO_SUBSCRIPTION marker in the message -> true, regardless of code.
	ns := connect.NewError(connect.CodeFailedPrecondition, errors.New("reason: NO_SUBSCRIPTION for org"))
	assert.True(t, isNoSubscriptionError(ns))

	// A different connect code without the marker -> false.
	nf := connect.NewError(connect.CodeNotFound, errors.New("missing"))
	assert.False(t, isNoSubscriptionError(nf))

	// A non-connect error -> false.
	assert.False(t, isNoSubscriptionError(errors.New("boom")))
}

// -P is the cocli shorthand for --param (D6). --command is shell-split
// (quote-aware) into the job's command token list, so flags ride inside the
// command line instead of a separate --args flag.
func TestCreateCommandParamShorthandAndCommandSplit(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create", "--dry-run",
		"--name", "flag-action",
		"--image", "ubuntu:22.04",
		"--command", "echo hi --verbose",
		"-P", "x=default",
		"-o", "json",
	})
	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"x":"default"`)
	assert.Contains(t, got, `"--verbose"`)
}
