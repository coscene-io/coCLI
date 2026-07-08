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
	assert.Contains(t, out.String(), "profile: small")
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
	assert.Contains(t, compactActionCreateJSON(out.String()), `"name":"local-action"`)
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
		"--param", "X=default",
		"--quota", "small",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	raw := out.String()
	got := compactActionCreateJSON(raw)
	assert.Contains(t, got, `"name":"inline-action"`)
	assert.Contains(t, got, `"image":"ubuntu:22.04"`)
	assert.Contains(t, raw, `"run script.py"`)
	assert.Contains(t, got, `"FOO":"bar"`)
	assert.Contains(t, got, `"cpu":"CPU_QUOTA_1C"`)
	assert.Contains(t, got, `"memory":"MEMORY_QUOTA_2G"`)
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

func TestCreateCommandDryRunStdinAndFlagOverrides(t *testing.T) {
	cfgPath := writeActionCreateTestConfig(t)
	var out bytes.Buffer
	spec := `name: file-action
jobs:
  - name: step
    container:
      image: old-image
      command: ["echo", "{{parameters.X}}"]
parameters:
  X: old
quota:
  profile: large
`
	ioStreams := iostreams.Test(io.NopCloser(strings.NewReader(spec)), &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create",
		"--dry-run",
		"-f", "-",
		"--name", "flag-action",
		"--image", "new-image",
		"--env", "A=B",
		"-o", "json",
	})

	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"name":"flag-action"`)
	assert.Contains(t, got, `"image":"new-image"`)
	assert.Contains(t, got, `"A":"B"`)
	assert.Contains(t, got, `"X":"old"`)
	assert.Contains(t, got, `"cpu":"CPU_QUOTA_4C"`)
	assert.NotContains(t, got, "old-image")
}

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

	spec, err := loadActionCreateSpec(path, nil)
	require.NoError(t, err)
	action, err := lowerActionCreateSpec(spec)
	require.NoError(t, err)

	assert.Equal(t, "json-action", action.GetSpec().GetName())
	assert.Equal(t, openv1alpha1commons.OutputOptions_APPEND, action.GetSpec().GetOutputOptions().GetSaveMode())
	assert.Equal(t, int64(123), action.GetSpec().GetStorageOptions().GetContainerStorageBytes())
	assert.True(t, action.GetSpec().GetStorageOptions().GetSsdOptions().GetUseSsd())
	assert.Equal(t, "/ssd", action.GetSpec().GetStorageOptions().GetSsdOptions().GetMountPath())
}

func TestLowerActionCreateQuota(t *testing.T) {
	quota, err := lowerActionCreateQuota(&actionCreateQuota{Profile: "xlarge"})
	require.NoError(t, err)
	assert.Equal(t, openv1alpha1commons.Quota_CPU_QUOTA_8C, quota.Cpu)
	assert.Equal(t, openv1alpha1commons.Quota_MEMORY_QUOTA_16G, quota.Memory)

	_, err = lowerActionCreateQuota(&actionCreateQuota{Profile: "huge"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid: small, medium, large, xlarge")
}

func TestLowerActionCreateRejectsBothJobKinds(t *testing.T) {
	_, err := lowerActionCreateSpec(&actionCreateSpec{
		Name: "bad",
		Jobs: []actionCreateJobSpec{{
			Name:      "job",
			Container: &actionCreateContainerSpec{Image: "img"},
			HTTP:      &actionCreateHTTPSpec{Method: "GET", URL: "https://example.com"},
		}},
	})
	assert.Error(t, err)
}

func TestValidateActionForCreate(t *testing.T) {
	action := &openv1alpha1resource.Action{Spec: &openv1alpha1commons.ActionSpec{
		Name: "valid",
		Jobs: []*openv1alpha1commons.JobSpec{{
			JobKind: &openv1alpha1commons.JobSpec_Container{Container: &openv1alpha1commons.ContainerJobSpec{
				Image:   "img",
				Command: []string{"echo {{parameters.X}}"},
			}},
		}},
		Parameters: map[string]string{"X": "default"},
	}}
	require.NoError(t, validateActionForCreate(action))
	assert.Equal(t, "job", action.Spec.Jobs[0].Name)

	action.Spec.Parameters = nil
	err := validateActionForCreate(action)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `parameter "X" is referenced but not defined`)
}

func TestSplitActionCreateWords(t *testing.T) {
	words, err := splitActionCreateWords(`python "run script.py" --flag=value`)
	require.NoError(t, err)
	assert.Equal(t, []string{"python", "run script.py", "--flag=value"}, words)

	_, err = splitActionCreateWords(`python "unterminated`)
	assert.Error(t, err)
}

func compactActionCreateJSON(input string) string {
	return strings.Join(strings.Fields(input), "")
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

// -P is the cocli shorthand for --param (D6); --args is the reconciled plan
// flag name (was --arg). Both flow through to the lowered spec.
func TestCreateCommandParamShorthandAndArgs(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	var out bytes.Buffer
	ioStreams := iostreams.Test(nil, &out, &bytes.Buffer{})
	cmd := NewRootCommand(&cfgPath, ioStreams, config.Provide)
	cmd.SetArgs([]string{
		"create", "--dry-run",
		"--name", "flag-action",
		"--image", "ubuntu:22.04",
		"--command", "echo hi",
		"-P", "X=default",
		"--args=--verbose",
		"-o", "json",
	})
	require.NoError(t, cmd.Execute())
	got := compactActionCreateJSON(out.String())
	assert.Contains(t, got, `"X":"default"`)
	assert.Contains(t, got, `"--verbose"`)
}
