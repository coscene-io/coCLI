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
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
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
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
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

type actionCreateSpec struct {
	Name               string                      `yaml:"name"`
	Description        string                      `yaml:"description"`
	Labels             []string                    `yaml:"labels"`
	Jobs               []actionCreateJobSpec       `yaml:"jobs"`
	Parameters         map[string]string           `yaml:"parameters"`
	MountOptions       *actionCreateMountOptions   `yaml:"mount_options"`
	MountOptionsJSON   *actionCreateMountOptions   `yaml:"mountOptions"`
	StorageOptions     *actionCreateStorageOptions `yaml:"storage_options"`
	StorageOptionsJSON *actionCreateStorageOptions `yaml:"storageOptions"`
	Quota              *actionCreateQuota          `yaml:"quota"`
	OutputOptions      *actionCreateOutputOptions  `yaml:"output_options"`
	OutputOptionsJSON  *actionCreateOutputOptions  `yaml:"outputOptions"`
}

type actionCreateSpecWrapper struct {
	Spec *actionCreateSpec `yaml:"spec"`
}

type actionCreateJobSpec struct {
	Name      string                     `yaml:"name"`
	Depends   []string                   `yaml:"depends"`
	Container *actionCreateContainerSpec `yaml:"container"`
	HTTP      *actionCreateHTTPSpec      `yaml:"http"`
}

type actionCreateContainerSpec struct {
	Image   string            `yaml:"image"`
	Command actionCreateWords `yaml:"command"`
	Args    actionCreateWords `yaml:"args"`
	Env     map[string]string `yaml:"env"`
}

type actionCreateHTTPSpec struct {
	Method  string                 `yaml:"method"`
	URL     string                 `yaml:"url"`
	Headers map[string]string      `yaml:"headers"`
	Body    map[string]interface{} `yaml:"body"`
	Timeout int32                  `yaml:"timeout"`
}

type actionCreateQuota struct {
	Cpu    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type actionCreateMountOptions struct {
	ReadWrite                 bool  `yaml:"read_write"`
	ReadWriteJSON             bool  `yaml:"readWrite"`
	ContainerStorageBytes     int64 `yaml:"container_storage_bytes"`
	ContainerStorageBytesJSON int64 `yaml:"containerStorageBytes"`
}

type actionCreateStorageOptions struct {
	ContainerStorageBytes     int64                          `yaml:"container_storage_bytes"`
	ContainerStorageBytesJSON int64                          `yaml:"containerStorageBytes"`
	SSDOptions                *actionCreateStorageSSDOptions `yaml:"ssd_options"`
	SSDOptionsJSON            *actionCreateStorageSSDOptions `yaml:"ssdOptions"`
}

type actionCreateStorageSSDOptions struct {
	UseSSD        bool   `yaml:"use_ssd"`
	UseSSDJSON    bool   `yaml:"useSsd"`
	MountPath     string `yaml:"mount_path"`
	MountPathJSON string `yaml:"mountPath"`
}

type actionCreateOutputOptions struct {
	SaveMode     string `yaml:"save_mode"`
	SaveModeJSON string `yaml:"saveMode"`
}

type actionCreateWords []string

func (w *actionCreateWords) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		words, err := splitActionCreateWords(value.Value)
		if err != nil {
			return err
		}
		*w = words
		return nil
	case yaml.SequenceNode:
		words := make([]string, 0, len(value.Content))
		for _, item := range value.Content {
			if item.Kind != yaml.ScalarNode {
				return fmt.Errorf("command and args entries must be strings")
			}
			words = append(words, item.Value)
		}
		*w = words
		return nil
	case 0:
		return nil
	default:
		return fmt.Errorf("command and args must be a string or string array")
	}
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

func buildActionForCreate(opts *actionCreateOptions, cmd *cobra.Command, stdin io.Reader) (*openv1alpha1resource.Action, error) {
	spec := &actionCreateSpec{}
	if opts.filePath != "" {
		loaded, err := loadActionCreateSpec(opts.filePath, stdin)
		if err != nil {
			return nil, err
		}
		spec = loaded
	}

	if err := applyActionCreateOverrides(spec, opts, func(name string) bool { return cmd.Flags().Changed(name) }); err != nil {
		return nil, err
	}
	return lowerActionCreateSpec(spec)
}

func loadActionCreateSpec(path string, stdin io.Reader) (*actionCreateSpec, error) {
	var data []byte
	var err error
	if path == "-" {
		if stdin == nil {
			return nil, errors.New("stdin is not available")
		}
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, errors.Wrap(err, "read action spec")
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("action spec is empty")
	}

	var spec actionCreateSpec
	if err = decodeActionCreateYAML(data, &spec); err != nil {
		var wrapper actionCreateSpecWrapper
		if wrapperErr := decodeActionCreateYAML(data, &wrapper); wrapperErr != nil || wrapper.Spec == nil {
			return nil, errors.Wrap(err, "parse action spec")
		}
		spec = *wrapper.Spec
	}
	normalizeActionCreateAliases(&spec)
	return &spec, nil
}

func decodeActionCreateYAML(data []byte, out interface{}) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(out)
}

func applyActionCreateOverrides(spec *actionCreateSpec, opts *actionCreateOptions, changed func(string) bool) error {
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
	job, err := singleActionCreateJob(spec)
	if err != nil {
		return err
	}
	if changed("image") || changed("command") || changed("env") {
		if job.Container == nil {
			job.Container = &actionCreateContainerSpec{}
		}
		job.HTTP = nil
	}
	if changed("image") {
		job.Container.Image = opts.image
	}
	if changed("command") {
		words, err := splitActionCreateWords(opts.command)
		if err != nil {
			return errors.Wrap(err, "parse command")
		}
		job.Container.Command = words
	}
	if changed("env") {
		env, err := parseActionCreateKeyValues(opts.env, "env")
		if err != nil {
			return err
		}
		if job.Container.Env == nil {
			job.Container.Env = map[string]string{}
		}
		for k, v := range env {
			job.Container.Env[k] = v
		}
	}
	return nil
}

func singleActionCreateJob(spec *actionCreateSpec) (*actionCreateJobSpec, error) {
	switch len(spec.Jobs) {
	case 0:
		spec.Jobs = []actionCreateJobSpec{{Name: "main"}}
	case 1:
	default:
		return nil, errors.New("inline job flags can only be used with zero or one job")
	}
	return &spec.Jobs[0], nil
}

func normalizeActionCreateAliases(spec *actionCreateSpec) {
	if spec.MountOptions == nil {
		spec.MountOptions = spec.MountOptionsJSON
	}
	if spec.StorageOptions == nil {
		spec.StorageOptions = spec.StorageOptionsJSON
	}
	if spec.OutputOptions == nil {
		spec.OutputOptions = spec.OutputOptionsJSON
	}
	if spec.MountOptions != nil {
		spec.MountOptions.normalizeAliases()
	}
	if spec.StorageOptions != nil {
		spec.StorageOptions.normalizeAliases()
	}
	if spec.OutputOptions != nil {
		spec.OutputOptions.normalizeAliases()
	}
}

func (opts *actionCreateMountOptions) normalizeAliases() {
	if opts.ReadWriteJSON {
		opts.ReadWrite = true
	}
	if opts.ContainerStorageBytes == 0 {
		opts.ContainerStorageBytes = opts.ContainerStorageBytesJSON
	}
}

func (opts *actionCreateStorageOptions) normalizeAliases() {
	if opts.ContainerStorageBytes == 0 {
		opts.ContainerStorageBytes = opts.ContainerStorageBytesJSON
	}
	if opts.SSDOptions == nil {
		opts.SSDOptions = opts.SSDOptionsJSON
	}
	if opts.SSDOptions != nil {
		opts.SSDOptions.normalizeAliases()
	}
}

func (opts *actionCreateStorageSSDOptions) normalizeAliases() {
	if opts.UseSSDJSON {
		opts.UseSSD = true
	}
	if opts.MountPath == "" {
		opts.MountPath = opts.MountPathJSON
	}
}

func (opts *actionCreateOutputOptions) normalizeAliases() {
	if opts.SaveMode == "" {
		opts.SaveMode = opts.SaveModeJSON
	}
}

func lowerActionCreateSpec(spec *actionCreateSpec) (*openv1alpha1resource.Action, error) {
	normalizeActionCreateAliases(spec)

	actionSpec := &openv1alpha1commons.ActionSpec{
		Name:        spec.Name,
		Description: spec.Description,
		Labels:      append([]string(nil), spec.Labels...),
		Parameters:  copyStringMap(spec.Parameters),
	}

	for i := range spec.Jobs {
		job, err := lowerActionCreateJob(&spec.Jobs[i])
		if err != nil {
			return nil, err
		}
		actionSpec.Jobs = append(actionSpec.Jobs, job)
	}

	if spec.MountOptions != nil {
		actionSpec.MountOptions = &openv1alpha1commons.MountOptions{
			ReadWrite:             spec.MountOptions.ReadWrite,
			ContainerStorageBytes: spec.MountOptions.ContainerStorageBytes,
		}
	}
	if spec.StorageOptions != nil {
		actionSpec.StorageOptions = &openv1alpha1commons.StorageOptions{
			ContainerStorageBytes: spec.StorageOptions.ContainerStorageBytes,
		}
		if spec.StorageOptions.SSDOptions != nil {
			actionSpec.StorageOptions.SsdOptions = &openv1alpha1commons.StorageOptions_SSDOptions{
				UseSsd:    spec.StorageOptions.SSDOptions.UseSSD,
				MountPath: spec.StorageOptions.SSDOptions.MountPath,
			}
		}
	}
	if spec.Quota != nil {
		quota, err := lowerActionCreateQuota(spec.Quota)
		if err != nil {
			return nil, err
		}
		actionSpec.Quota = quota
	}
	if spec.OutputOptions != nil {
		saveMode, err := parseActionCreateSaveMode(spec.OutputOptions.SaveMode)
		if err != nil {
			return nil, err
		}
		actionSpec.OutputOptions = &openv1alpha1commons.OutputOptions{SaveMode: saveMode}
	}

	return &openv1alpha1resource.Action{Spec: actionSpec}, nil
}

func lowerActionCreateJob(job *actionCreateJobSpec) (*openv1alpha1commons.JobSpec, error) {
	if job.Container != nil && job.HTTP != nil {
		return nil, fmt.Errorf("job %q must set only one of container or http", job.Name)
	}
	out := &openv1alpha1commons.JobSpec{
		Name:    job.Name,
		Depends: append([]string(nil), job.Depends...),
	}
	if job.Container != nil {
		out.JobKind = &openv1alpha1commons.JobSpec_Container{
			Container: &openv1alpha1commons.ContainerJobSpec{
				Image:   job.Container.Image,
				Command: append([]string(nil), job.Container.Command...),
				Args:    append([]string(nil), job.Container.Args...),
				Env:     copyStringMap(job.Container.Env),
			},
		}
	}
	if job.HTTP != nil {
		method, err := parseActionCreateHTTPMethod(job.HTTP.Method)
		if err != nil {
			return nil, err
		}
		httpSpec := &openv1alpha1commons.HttpJobSpec{
			Method:  method,
			Url:     job.HTTP.URL,
			Headers: copyStringMap(job.HTTP.Headers),
			Timeout: job.HTTP.Timeout,
		}
		if job.HTTP.Body != nil {
			body, err := structpb.NewStruct(normalizeActionCreateStruct(job.HTTP.Body))
			if err != nil {
				return nil, errors.Wrapf(err, "invalid http body for job %q", job.Name)
			}
			httpSpec.Body = body
		}
		out.JobKind = &openv1alpha1commons.JobSpec_Http{
			Http: httpSpec,
		}
	}
	return out, nil
}

// lowerActionCreateQuota parses the proto-native CPU/memory quota enum strings
// (the same form `action get -o yaml`/`action update` use) into a proto Quota.
// Empty cpu AND memory leaves the quota effectively unset (server-defaulted).
func lowerActionCreateQuota(quota *actionCreateQuota) (*openv1alpha1commons.Quota, error) {
	if quota.Cpu == "" && quota.Memory == "" {
		return &openv1alpha1commons.Quota{}, nil
	}

	out := &openv1alpha1commons.Quota{}
	if quota.Cpu != "" {
		key := strings.ToUpper(strings.TrimSpace(quota.Cpu))
		v, ok := openv1alpha1commons.Quota_CPUQuota_value[key]
		if !ok {
			return nil, fmt.Errorf("unknown quota cpu %q (valid: %s)", quota.Cpu, quotaEnumNames(openv1alpha1commons.Quota_CPUQuota_name))
		}
		out.Cpu = openv1alpha1commons.Quota_CPUQuota(v)
	}
	if quota.Memory != "" {
		key := strings.ToUpper(strings.TrimSpace(quota.Memory))
		v, ok := openv1alpha1commons.Quota_MemoryQuota_value[key]
		if !ok {
			return nil, fmt.Errorf("unknown quota memory %q (valid: %s)", quota.Memory, quotaEnumNames(openv1alpha1commons.Quota_MemoryQuota_name))
		}
		out.Memory = openv1alpha1commons.Quota_MemoryQuota(v)
	}
	return out, nil
}

// actionCreateQuotaPreset maps a t-shirt resource preset (small|medium|large|
// xlarge) to the proto CPU/memory quota enum pair. It backs the --quota
// convenience flag, which is a shortcut for setting the spec's quota.cpu /
// quota.memory enums (the flag overrides any file quota:).
func actionCreateQuotaPreset(preset string) (*actionCreateQuota, error) {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "small":
		return &actionCreateQuota{Cpu: "CPU_QUOTA_1C", Memory: "MEMORY_QUOTA_2G"}, nil
	case "medium":
		return &actionCreateQuota{Cpu: "CPU_QUOTA_2C", Memory: "MEMORY_QUOTA_4G"}, nil
	case "large":
		return &actionCreateQuota{Cpu: "CPU_QUOTA_4C", Memory: "MEMORY_QUOTA_8G"}, nil
	case "xlarge":
		return &actionCreateQuota{Cpu: "CPU_QUOTA_8C", Memory: "MEMORY_QUOTA_16G"}, nil
	default:
		return nil, fmt.Errorf("unknown quota preset %q (valid: small, medium, large, xlarge)", preset)
	}
}

// quotaEnumNames renders the enum value names (ordered by number) for a clear
// error message when an unknown cpu/memory value is supplied.
func quotaEnumNames(names map[int32]string) string {
	nums := make([]int32, 0, len(names))
	for n := range names {
		nums = append(nums, n)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	out := make([]string, 0, len(nums))
	for _, n := range nums {
		out = append(out, names[n])
	}
	return strings.Join(out, ", ")
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

func parseActionCreateHTTPMethod(value string) (openv1alpha1commons.HttpJobSpec_HttpMethod, error) {
	if value == "" {
		return openv1alpha1commons.HttpJobSpec_HTTP_METHOD_UNSPECIFIED, nil
	}
	key := strings.ToUpper(strings.TrimSpace(value))
	if v, ok := openv1alpha1commons.HttpJobSpec_HttpMethod_value[key]; ok {
		return openv1alpha1commons.HttpJobSpec_HttpMethod(v), nil
	}
	return openv1alpha1commons.HttpJobSpec_HTTP_METHOD_UNSPECIFIED, fmt.Errorf("unknown HTTP method %q", value)
}

func parseActionCreateSaveMode(value string) (openv1alpha1commons.OutputOptions_SaveMode, error) {
	if value == "" {
		return openv1alpha1commons.OutputOptions_SAVE_MODE_UNSPECIFIED, nil
	}
	key := strings.ToUpper(strings.TrimSpace(value))
	if v, ok := openv1alpha1commons.OutputOptions_SaveMode_value[key]; ok {
		return openv1alpha1commons.OutputOptions_SaveMode(v), nil
	}
	return openv1alpha1commons.OutputOptions_SAVE_MODE_UNSPECIFIED, fmt.Errorf("unknown output save mode %q", value)
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

func normalizeActionCreateStruct(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = normalizeActionCreateValue(v)
	}
	return out
}

func normalizeActionCreateValue(v interface{}) interface{} {
	switch typed := v.(type) {
	case map[string]interface{}:
		return normalizeActionCreateStruct(typed)
	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeActionCreateValue(item))
		}
		return out
	default:
		return typed
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
