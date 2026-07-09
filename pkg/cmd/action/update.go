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
	"encoding/json"
	"io"
	"os"

	openv1alpha1commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/coscene-io/cocli/internal/printer/table"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

// updateMaskSpec is the only update_mask path the backend accepts (matrix
// acPathToSel has a single entry, "spec"); it full-replaces the spec. cocli
// always sends exactly this — the mask is never user-crafted (plan D2/D13/C4).
var updateMaskSpec = []string{"spec"}

func NewUpdateCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var (
		projectSlug  = ""
		filePath     = ""
		dryRun       = false
		outputFormat = "table"
	)

	cmd := &cobra.Command{
		Use:   "update <action-resource-name/id> -f <spec.yaml|json> [-p <working-project-slug>] [--dry-run] [-o <output-format>]",
		Short: "Update an action's spec from a file.",
		Long: "Update an action's spec, replacing it wholesale from a YAML/JSON file (`-` for stdin).\n\n" +
			"The file is the full protojson Action produced by `cocli action get -o yaml/json`, so the\n" +
			"get -> edit -> update loop round-trips. Output-only fields (name, author, timestamps) in the\n" +
			"file are ignored; only the spec is written, selected by the positional action name/id.\n\n" +
			"WARNING: update replaces the ENTIRE spec, including spec.labels. A spec that omits labels\n" +
			"DETACHES all labels currently on the action. Keep the labels from the `get` dump (or set them\n" +
			"explicitly) to preserve them.\n\n" +
			"Masked secrets appear as `********` placeholders in a `get` dump; leave them unchanged and the\n" +
			"backend restores the real values. --dry-run prints the spec that would be sent without any\n" +
			"wire call.",
		Example: "  # Round-trip: fetch, edit, then update:\n" +
			"  cocli action get my-action -p my-project -o yaml > spec.yaml\n" +
			"  # ...edit spec.yaml...\n" +
			"  cocli action update my-action -p my-project -f spec.yaml\n\n" +
			"  # Update from stdin:\n" +
			"  cat spec.yaml | cocli action update my-action -p my-project -f -\n\n" +
			"  # Preview the spec that would be sent without mutating anything:\n" +
			"  cocli action update my-action -p my-project -f spec.yaml --dry-run",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if filePath == "" {
				log.Fatalf("a spec file is required: pass -f <file> (or -f - for stdin)")
			}

			// Load the spec with a proto-native loader that matches the format
			// `action get -o yaml/json` emits, so the get -> edit -> update loop
			// round-trips (plan D5/F1). The same loader backs `action create -f`,
			// so both commands share one file schema.
			spec, err := loadActionSpecFromFile(filePath, io.In)
			if err != nil {
				log.Fatalf("invalid action spec: %v", err)
			}

			// Reuse create's proto-level validation. Note (F7): it defaults an
			// unnamed single job to "main" — acceptable here.
			if err = validateActionForCreate(&openv1alpha1resource.Action{Spec: spec}); err != nil {
				log.Fatalf("invalid action spec: %v", err)
			}

			printFormat := outputFormat
			if dryRun && !cmd.Flags().Changed("output") {
				printFormat = "yaml"
			}
			p, err := printer.Printer(printFormat, &printer.Options{TableOpts: &table.PrintOpts{Verbose: true}})
			if err != nil {
				log.Fatal(err)
			}

			// --dry-run: print the spec that would be sent, make NO wire call.
			if dryRun {
				if err = p.PrintObj(printable.NewActionSpec(spec), io.Out); err != nil {
					log.Fatalf("failed to print action spec: %v", err)
				}
				return
			}

			// Get current profile and resolve the action name/id (client-side,
			// resolve-first — clean not-found message; plan D7/F2).
			pm := cmd_utils.ProfileManager(cmd, getProvider, *cfgPath)
			proj, err := pm.ProjectName(cmd.Context(), projectSlug)
			if err != nil {
				log.Fatalf("unable to get project name: %v", err)
			}
			actionName, err := pm.ActionCli().ActionId2Name(cmd.Context(), args[0], proj)
			if utils.IsConnectErrorWithCode(err, connect.CodeNotFound) {
				printActionNotFound(io, args[0], proj)
				return
			} else if err != nil {
				log.Fatalf("failed to convert action id to name: %v", err)
			}

			// Labels warning (plan D15/F4): the submitted spec replaces
			// spec.labels wholesale. If it omits labels but the current action
			// has some, the update DETACHES them — warn loudly before mutating.
			warnActionUpdateLabelDetach(spec, func() (*openv1alpha1resource.Action, error) {
				return pm.ActionCli().GetByName(cmd.Context(), actionName)
			}, io)

			// Update the action. Name selects the row; only spec is written.
			updated, err := pm.ActionCli().UpdateAction(cmd.Context(), actionName, spec, updateMaskSpec)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					os.Exit(1)
				}
				// A ResourceExhausted / NO_SUBSCRIPTION failure is almost always
				// a missing permission grant, NOT a real quota limit — retrying
				// will not help (plan D14, reused from create).
				if isNoSubscriptionError(err) {
					exitf(io, "failed to update action: %v\nlikely a missing permission grant, not a quota limit — do not retry", err)
				}
				log.Fatalf("failed to update action: %v", err)
			}

			// After a successful update, re-fetch to confirm the action is still
			// retrievable (plan D14/F3). matrix UpdateAction echoes the request
			// rather than the persisted row, and its blind WHERE-id update can
			// silently write to a soft-deleted row and still return 200. A
			// follow-up NotFound is the only visible signal of that footgun.
			refetched, refetchErr := pm.ActionCli().GetByName(cmd.Context(), actionName)
			if utils.IsConnectErrorWithCode(refetchErr, connect.CodeNotFound) {
				io.Eprintf("warning: update reported success but the action is no longer retrievable; it may have been deleted.\n")
				return
			} else if refetchErr != nil {
				// Non-NotFound re-fetch error: fall back to the echoed response
				// so the caller still sees what was sent.
				io.Eprintf("warning: update succeeded but re-fetching the action failed: %v\n", refetchErr)
				refetched = updated
			}

			if err = p.PrintObj(printable.NewSingleAction(refetched), io.Out); err != nil {
				log.Fatalf("failed to print action: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&projectSlug, "project", "p", "", "the slug of the working project")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "action spec YAML/JSON file (`-` for stdin), as produced by `cocli action get -o yaml`")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and print the spec that would be sent without updating")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")

	return cmd
}

// loadActionSpecFromFile reads a YAML/JSON action document (or stdin for `-`)
// and extracts its ActionSpec via a proto-native path: parse to a generic
// value, re-encode as JSON, then protojson.Unmarshal into an Action with
// DiscardUnknown so output-only fields a `get -o yaml/json` dump carries (name,
// author, create/update times) — and any unknown keys — are tolerated rather
// than rejected. This is the exact format `action get` emits, so the
// get -> edit -> update loop round-trips (plan D5/F1). Both `action update` and
// `action create` load their -f spec through this single tolerant loader.
func loadActionSpecFromFile(path string, stdin io.Reader) (*openv1alpha1commons.ActionSpec, error) {
	var (
		data []byte
		err  error
	)
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

	// Parse as YAML into a generic value. JSON is a subset of YAML, so this
	// accepts both `get -o yaml` (snake_case protojson) and `get -o json`.
	var generic interface{}
	if err = yaml.Unmarshal(data, &generic); err != nil {
		return nil, errors.Wrap(err, "parse action spec")
	}
	// yaml.v3 decodes mappings into map[string]interface{} when keys are
	// strings, but nested non-string keys would break json.Marshal; normalize
	// defensively.
	normalized, err := normalizeActionUpdateYAML(generic)
	if err != nil {
		return nil, errors.Wrap(err, "parse action spec")
	}

	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, errors.Wrap(err, "encode action spec")
	}

	// Preferred shape: the full protojson Action a `get -o yaml/json` dump
	// emits (spec nested under a `spec:` key, alongside output-only fields).
	action := &openv1alpha1resource.Action{}
	if err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(jsonBytes, action); err != nil {
		return nil, errors.Wrap(err, "parse action spec")
	}
	if action.GetSpec() != nil {
		return action.GetSpec(), nil
	}

	// Fallback: a bare ActionSpec document (spec fields at the top level, no
	// wrapping Action). Hand-authored specs commonly take this shape.
	spec := &openv1alpha1commons.ActionSpec{}
	if err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(jsonBytes, spec); err != nil {
		return nil, errors.Wrap(err, "parse action spec")
	}
	if isEmptyActionSpec(spec) {
		return nil, errors.New("action spec must not be empty")
	}
	return spec, nil
}

// isEmptyActionSpec reports whether an ActionSpec has no meaningful content —
// used to reject a document that parsed but carried nothing an update could
// write (e.g. only output-only Action fields, or an empty file body).
func isEmptyActionSpec(spec *openv1alpha1commons.ActionSpec) bool {
	return spec.GetName() == "" &&
		spec.GetDescription() == "" &&
		len(spec.GetLabels()) == 0 &&
		len(spec.GetJobs()) == 0 &&
		len(spec.GetParameters()) == 0 &&
		spec.GetQuota() == nil &&
		spec.GetMountOptions() == nil &&
		spec.GetStorageOptions() == nil &&
		spec.GetOutputOptions() == nil
}

// normalizeActionUpdateYAML converts any map[interface{}]interface{} that
// yaml.v3 might produce into map[string]interface{} so json.Marshal accepts it.
func normalizeActionUpdateYAML(v interface{}) (interface{}, error) {
	switch typed := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for k, val := range typed {
			nv, err := normalizeActionUpdateYAML(val)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(typed))
		for k, val := range typed {
			ks, ok := k.(string)
			if !ok {
				return nil, errors.Errorf("non-string map key %v", k)
			}
			nv, err := normalizeActionUpdateYAML(val)
			if err != nil {
				return nil, err
			}
			out[ks] = nv
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			nv, err := normalizeActionUpdateYAML(item)
			if err != nil {
				return nil, err
			}
			out = append(out, nv)
		}
		return out, nil
	default:
		return v, nil
	}
}

// warnActionUpdateLabelDetach warns to stderr when the submitted spec carries no
// labels but the current action does — that update would DETACH all labels
// (plan D15/F4). The check is best-effort: fetchCurrent is only called when the
// submitted spec drops labels, and any fetch error stays silent (the update
// proceeds regardless).
func warnActionUpdateLabelDetach(spec *openv1alpha1commons.ActionSpec, fetchCurrent func() (*openv1alpha1resource.Action, error), io *iostreams.IOStreams) {
	if len(spec.GetLabels()) > 0 {
		return
	}
	current, err := fetchCurrent()
	if err != nil || current == nil || current.GetSpec() == nil {
		return
	}
	if len(current.GetSpec().GetLabels()) > 0 {
		io.Eprintf("warning: the submitted spec has no labels but the action currently has %d; update will DETACH all of them.\n", len(current.GetSpec().GetLabels()))
	}
}
