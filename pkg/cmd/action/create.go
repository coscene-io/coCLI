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
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	openv1alpha1commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// actionCreateSentinelImage is an obvious placeholder used in the --example
// skeleton. If it survives unedited into a real create, the action would be
// junk, so we warn at create-time (see warnActionCreateSentinels).
const actionCreateSentinelImage = "REPLACE-ME/image:tag"

const actionCreateExample = `# ActionSpec - use with: cocli action create --project <slug> -f spec.yaml
name: my-action
description: ""
labels: []
jobs:
  - name: main
    depends: []
    container:
      image: REPLACE-ME/image:tag # <-- replace before creating (see warning)
      command: ["python", "run.py"]
      args: []
      env:
        COS_KEY: "value"
parameters:
  x: "default"
quota:
  cpu: CPU_QUOTA_1C     # CPU_QUOTA_1C|CPU_QUOTA_2C|CPU_QUOTA_4C|CPU_QUOTA_8C
  memory: MEMORY_QUOTA_2G # MEMORY_QUOTA_1G|_2G|_4G|_8G|_16G|_32G|_64G
output_options:
  save_mode: APPEND
`

var (
	actionCreateJobNameRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	actionCreateEnvRe     = regexp.MustCompile(`^[-._a-zA-Z][-._a-zA-Z0-9]*$`)
	actionCreateParamRe   = regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)
)

type actionCreateOptions struct {
	filePath    string
	dryRun      bool
	example     bool
	name        string
	description string
	image       string
	command     string
	env         []string
	params      []string
	quota       string
}

func NewCreateCommand(cfgPath *string, ioStreams *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		opts         actionCreateOptions
		projectSlug  string
		outputFormat string
	)

	cmd := &cobra.Command{
		Use:   "create --project <working-project-slug> (-f spec.yaml|json | --name <name> --image <image>)",
		Short: "Create an action.",
		Long: `Create an action from a spec file (-f) or inline flags.

Resource quota can be set either in the spec file as proto enums
(quota.cpu / quota.memory, e.g. quota: {cpu: CPU_QUOTA_1C, memory: MEMORY_QUOTA_2G})
or via the --quota small|medium|large|xlarge convenience preset, which maps a
t-shirt size to that enum pair and overrides any file quota:.`,
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if opts.example {
				ioStreams.Printf("%s", actionCreateExample)
				return
			}

			action, err := buildActionForCreate(&opts, cmd, ioStreams.In)
			if err != nil {
				log.Fatalf("invalid action spec: %v", err)
			}
			if err = validateActionForCreate(action); err != nil {
				log.Fatalf("invalid action spec: %v", err)
			}

			// Warn (non-fatal) if an unedited --example sentinel survived; both
			// dry-run and real create surface this so agents catch it early.
			warnActionCreateSentinels(action, ioStreams)

			printFormat := outputFormat
			if opts.dryRun && !cmd.Flags().Changed("output") {
				printFormat = "yaml"
			}

			p, err := printer.Printer(printFormat, &printer.Options{TableOpts: &table.PrintOpts{Verbose: true}})
			if err != nil {
				log.Fatal(err)
			}
			if opts.dryRun {
				if err = p.PrintObj(printable.NewActionSpec(action.GetSpec()), ioStreams.Out); err != nil {
					log.Fatalf("failed to print action spec: %v", err)
				}
				return
			}

			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}
			action, err = pm.ActionCli().CreateAction(cmd.Context(), proj.String(), action)
			if err != nil {
				// Quiet exit on Ctrl-C / cancellation (mirror logs.go). No noisy
				// stack — the user already knows they cancelled.
				if errors.Is(err, context.Canceled) {
					os.Exit(1)
				}
				// D14: a ResourceExhausted / NO_SUBSCRIPTION failure is almost
				// always a missing subscription or permission grant, NOT a real
				// quota limit — retrying will not help, so say so and stop.
				if isNoSubscriptionError(err) {
					exitf(ioStreams, "failed to create action: %v\nlikely a missing permission grant, not a quota limit — do not retry", err)
				}
				log.Fatalf("failed to create action: %v", err)
			}
			if err = p.PrintObj(printable.NewSingleAction(action), ioStreams.Out); err != nil {
				log.Fatalf("failed to print action: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&opts.filePath, "file", "f", "", "action spec YAML/JSON file (`-` for stdin)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "validate and print the lowered action without creating it")
	cmd.Flags().BoolVar(&opts.example, "example", false, "print an example action spec")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")

	cmd.Flags().StringVar(&opts.name, "name", "", "action name for inline single-container creation")
	cmd.Flags().StringVar(&opts.description, "description", "", "action description for inline single-container creation")
	cmd.Flags().StringVar(&opts.image, "image", "", "container image for inline single-container creation")
	cmd.Flags().StringVar(&opts.command, "command", "", "container command line (shell-split, quote-aware; e.g. 'python train.py --epochs 10')")
	cmd.Flags().StringArrayVar(&opts.env, "env", []string{}, "container environment variable in key=value format (repeatable)")
	cmd.Flags().StringArrayVarP(&opts.params, "param", "P", []string{}, "action parameter default in key=value format (repeatable)")
	cmd.Flags().StringVar(&opts.quota, "quota", "", "resource preset: small|medium|large|xlarge (convenience; overrides quota.cpu/quota.memory)")

	cmd_utils.DisableAuthCheckForBoolFlags(cmd, "example", "dry-run")

	return cmd
}

// buildActionForCreate loads the -f spec (if any) through the shared
// proto-native loader used by `action update`, then applies inline flag
// overrides directly on the parsed *ActionSpec. With no -f, it starts from an
// empty ActionSpec so flags-only create still works.
func buildActionForCreate(opts *actionCreateOptions, cmd *cobra.Command, stdin io.Reader) (*openv1alpha1resource.Action, error) {
	spec := &openv1alpha1commons.ActionSpec{}
	if opts.filePath != "" {
		loaded, err := loadActionSpecFromFile(opts.filePath, stdin)
		if err != nil {
			return nil, err
		}
		spec = loaded
	}

	if err := applyActionCreateOverrides(spec, opts, func(name string) bool { return cmd.Flags().Changed(name) }); err != nil {
		return nil, err
	}

	// Default the three option messages to empty (but non-nil) so every
	// cocli-created action carries them. A minimal create otherwise leaves
	// spec.MountOptions/StorageOptions/OutputOptions nil, which panics the
	// matrix backend at run time; the proper fix is in matrix (separate,
	// needs a deploy), but this default gives immediate relief and matches
	// the user's manual `action update` workaround of filling empty objects.
	// Empty means backend defaults — no field is set inside them. Runs for
	// both -f file and flags-only creates.
	if spec.MountOptions == nil {
		spec.MountOptions = &openv1alpha1commons.MountOptions{}
	}
	if spec.StorageOptions == nil {
		spec.StorageOptions = &openv1alpha1commons.StorageOptions{}
	}
	if spec.OutputOptions == nil {
		spec.OutputOptions = &openv1alpha1commons.OutputOptions{}
	}
	return &openv1alpha1resource.Action{Spec: spec}, nil
}

// applyActionCreateOverrides mutates the parsed *ActionSpec in place from the
// inline flags. --name/--description/--param set top-level fields;
// --image/--command/--env operate on the single job's container (creating a
// "main" container job when the spec has none); --quota sets the proto Quota
// enum pair and overrides any file quota:.
func applyActionCreateOverrides(spec *openv1alpha1commons.ActionSpec, opts *actionCreateOptions, changed func(string) bool) error {
	if changed("name") {
		spec.Name = opts.name
	}
	if changed("description") {
		spec.Description = opts.description
	}
	if changed("param") {
		params, err := parseActionCreateKeyValues(opts.params, "param")
		if err != nil {
			return err
		}
		if spec.Parameters == nil {
			spec.Parameters = map[string]string{}
		}
		for k, v := range params {
			spec.Parameters[k] = v
		}
	}
	if changed("quota") {
		quota, err := actionCreateQuotaPreset(opts.quota)
		if err != nil {
			return err
		}
		// The --quota preset is a convenience that overrides any file quota:
		// (cpu/memory) — the flag wins.
		spec.Quota = quota
	}
	if !changed("image") && !changed("command") && !changed("env") {
		return nil
	}

	container, err := singleActionCreateContainer(spec)
	if err != nil {
		return err
	}
	if changed("image") {
		container.Image = opts.image
	}
	if changed("command") {
		words, err := splitActionCreateWords(opts.command)
		if err != nil {
			return errors.Wrap(err, "parse command")
		}
		container.Command = words
	}
	if changed("env") {
		env, err := parseActionCreateKeyValues(opts.env, "env")
		if err != nil {
			return err
		}
		if container.Env == nil {
			container.Env = map[string]string{}
		}
		for k, v := range env {
			container.Env[k] = v
		}
	}
	return nil
}

// singleActionCreateContainer returns the container of the spec's single job so
// the inline --image/--command/--env flags can mutate it. Inline job flags are
// only valid with zero or one job (preserving the original guard). With zero
// jobs it creates a "main" container job; with one job it forces that job to be
// a container job (dropping any http kind the file set).
func singleActionCreateContainer(spec *openv1alpha1commons.ActionSpec) (*openv1alpha1commons.ContainerJobSpec, error) {
	switch len(spec.Jobs) {
	case 0:
		container := &openv1alpha1commons.ContainerJobSpec{}
		spec.Jobs = []*openv1alpha1commons.JobSpec{{
			Name:    "main",
			JobKind: &openv1alpha1commons.JobSpec_Container{Container: container},
		}}
		return container, nil
	case 1:
		job := spec.Jobs[0]
		container := job.GetContainer()
		if container == nil {
			container = &openv1alpha1commons.ContainerJobSpec{}
			job.JobKind = &openv1alpha1commons.JobSpec_Container{Container: container}
		}
		return container, nil
	default:
		return nil, errors.New("inline job flags can only be used with zero or one job")
	}
}

// actionCreateQuotaPreset maps a t-shirt resource preset (small|medium|large|
// xlarge) to the proto CPU/memory quota enum pair. It backs the --quota
// convenience flag, which is a shortcut for setting the spec's quota.cpu /
// quota.memory enums (the flag overrides any file quota:).
func actionCreateQuotaPreset(preset string) (*openv1alpha1commons.Quota, error) {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "small":
		return &openv1alpha1commons.Quota{Cpu: openv1alpha1commons.Quota_CPU_QUOTA_1C, Memory: openv1alpha1commons.Quota_MEMORY_QUOTA_2G}, nil
	case "medium":
		return &openv1alpha1commons.Quota{Cpu: openv1alpha1commons.Quota_CPU_QUOTA_2C, Memory: openv1alpha1commons.Quota_MEMORY_QUOTA_4G}, nil
	case "large":
		return &openv1alpha1commons.Quota{Cpu: openv1alpha1commons.Quota_CPU_QUOTA_4C, Memory: openv1alpha1commons.Quota_MEMORY_QUOTA_8G}, nil
	case "xlarge":
		return &openv1alpha1commons.Quota{Cpu: openv1alpha1commons.Quota_CPU_QUOTA_8C, Memory: openv1alpha1commons.Quota_MEMORY_QUOTA_16G}, nil
	default:
		return nil, fmt.Errorf("unknown quota preset %q (valid: small, medium, large, xlarge)", preset)
	}
}

func validateActionForCreate(action *openv1alpha1resource.Action) error {
	if action == nil || action.Spec == nil {
		return errors.New("action spec must not be empty")
	}
	spec := action.Spec
	var msgs []string
	if spec.Name == "" {
		msgs = append(msgs, "name cannot be empty")
	}
	if len(spec.Jobs) == 0 {
		msgs = append(msgs, "jobs cannot be empty")
	}
	if len(spec.Jobs) == 1 && spec.Jobs[0].Name == "" {
		spec.Jobs[0].Name = "main"
	}
	for _, job := range spec.Jobs {
		if job.Name == "" {
			msgs = append(msgs, "job name cannot be empty")
		}
		if !actionCreateJobNameRe.MatchString(job.Name) {
			msgs = append(msgs, fmt.Sprintf("job name %q must match %s", job.Name, actionCreateJobNameRe.String()))
		}
		if job.JobKind == nil {
			msgs = append(msgs, fmt.Sprintf("job %q must define container or http", job.Name))
		}
		containerJob, ok := job.JobKind.(*openv1alpha1commons.JobSpec_Container)
		if !ok || containerJob.Container == nil {
			continue
		}
		for env := range containerJob.Container.Env {
			if !actionCreateEnvRe.MatchString(env) {
				msgs = append(msgs, fmt.Sprintf("env name %q must match %s", env, actionCreateEnvRe.String()))
			}
		}
		for _, cmd := range containerJob.Container.Command {
			for _, match := range actionCreateParamRe.FindAllStringSubmatch(cmd, -1) {
				param := strings.TrimSpace(match[1])
				if !strings.HasPrefix(param, "parameters.") {
					msgs = append(msgs, fmt.Sprintf("parameter reference %q must start with parameters.", param))
					continue
				}
				param = strings.TrimPrefix(param, "parameters.")
				if param == "" {
					msgs = append(msgs, "parameter reference cannot be empty")
					continue
				}
				if _, ok := spec.Parameters[param]; !ok {
					msgs = append(msgs, fmt.Sprintf("parameter %q is referenced but not defined", param))
				}
			}
		}
	}
	if len(msgs) > 0 {
		return errors.New(strings.Join(msgs, "; "))
	}
	return nil
}

// isNoSubscriptionError reports whether a CreateAction failure is the D14
// "no subscription / missing permission grant" case rather than a genuine
// transient quota limit. The backend signals this with a ResourceExhausted
// connect code and/or a NO_SUBSCRIPTION detail; either way, retrying will not
// help. Mirrors the connect-code inspection style in logs.go (errors.As +
// connErr.Code()).
func isNoSubscriptionError(err error) bool {
	var connErr *connect.Error
	if !errors.As(err, &connErr) {
		return false
	}
	if strings.Contains(strings.ToUpper(err.Error()), "NO_SUBSCRIPTION") {
		return true
	}
	return connErr.Code() == connect.CodeResourceExhausted
}

// warnActionCreateSentinels warns to stderr when an unedited --example sentinel
// (e.g. the placeholder image) survives into the spec. It does not block: the
// server-side validation is authoritative, but a surviving sentinel almost
// always means the user forgot to fill in the skeleton and would otherwise
// create a junk action.
func warnActionCreateSentinels(action *openv1alpha1resource.Action, io *iostreams.IOStreams) {
	if action == nil || action.GetSpec() == nil {
		return
	}
	for _, job := range action.GetSpec().GetJobs() {
		container := job.GetContainer()
		if container == nil {
			continue
		}
		if container.GetImage() == actionCreateSentinelImage {
			io.Eprintf("warning: job %q still uses the example sentinel image %q — edit the spec before creating a real action\n", job.GetName(), actionCreateSentinelImage)
		}
	}
}

func parseActionCreateKeyValues(items []string, flagName string) (map[string]string, error) {
	values := map[string]string{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("--%s expects key=value, got %q", flagName, item)
		}
		values[key] = value
	}
	return values, nil
}

func splitActionCreateWords(input string) ([]string, error) {
	var words []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words, nil
}
