package action

import (
	"testing"

	openv1alpha1commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunCommandValidatesArgs(t *testing.T) {
	cfgPath := ""
	cmd := NewRunCommand(&cfgPath, nil, nil)
	require.NotNil(t, cmd.Args)

	tests := []struct {
		name         string
		args         []string
		recordSearch string
		setSearch    bool
		wantErr      string
	}{
		{name: "no arguments", wantErr: "requires an action argument"},
		{name: "only action", args: []string{"action"}, wantErr: "requires a record argument or --search"},
		{name: "action and record", args: []string{"action", "record"}},
		{name: "action and search", args: []string{"action"}, recordSearch: `{"==":[{"var":"isArchived"},"false"]}`, setSearch: true},
		{name: "record and search", args: []string{"action", "record"}, recordSearch: `{"==":[{"var":"isArchived"},"false"]}`, setSearch: true, wantErr: "mutually exclusive"},
		{name: "blank search", args: []string{"action"}, recordSearch: "  ", setSearch: true, wantErr: "search query must not be empty"},
		{name: "empty search with record", args: []string{"action", "record"}, setSearch: true, wantErr: "search query must not be empty"},
		{name: "invalid search JSON", args: []string{"action"}, recordSearch: `{`, setSearch: true, wantErr: "invalid search JSON"},
		{name: "empty search object", args: []string{"action"}, recordSearch: `{}`, setSearch: true, wantErr: "search query must not be empty"},
		{name: "null search", args: []string{"action"}, recordSearch: `null`, setSearch: true, wantErr: "invalid search JSON"},
		{name: "search array", args: []string{"action"}, recordSearch: `[]`, setSearch: true, wantErr: "invalid search JSON"},
		{name: "too many arguments", args: []string{"action", "record", "extra"}, wantErr: "accepts at most 2 arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRunCommand(&cfgPath, nil, nil)
			if tt.setSearch {
				require.NoError(t, cmd.Flags().Set("search", tt.recordSearch))
			}
			err := cmd.Args(cmd, tt.args)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestParseRecordSearch(t *testing.T) {
	search := `{"and":[{"==":[{"var":"isArchived"},"false"]},{"and":[{"==":[{"var":"customFields.c68621ba-4f29-4da0-8108-d76f612d2dae"},"aede4062-7f1f-4ecf-b3b7-ab604579ff4a"]}]}]}`

	recordQuery, err := parseRecordSearch(search)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"and": []any{
			map[string]any{"==": []any{map[string]any{"var": "isArchived"}, "false"}},
			map[string]any{"and": []any{
				map[string]any{"==": []any{
					map[string]any{"var": "customFields.c68621ba-4f29-4da0-8108-d76f612d2dae"},
					"aede4062-7f1f-4ecf-b3b7-ab604579ff4a",
				}},
			}},
		},
	}, recordQuery.AsMap())
}

func TestRecordNameFromArg(t *testing.T) {
	project := &name.Project{ProjectID: "project-1"}

	t.Run("bare record ID uses the working project", func(t *testing.T) {
		recordName := recordNameFromArg("record-1", project)

		assert.Equal(t, "projects/project-1/records/record-1", recordName.String())
	})

	t.Run("full record name preserves its project", func(t *testing.T) {
		recordName := recordNameFromArg("projects/project-2/records/record-2", project)

		assert.Equal(t, "projects/project-2/records/record-2", recordName.String())
	})
}

func TestPromptActionRunParameters(t *testing.T) {
	defaults := map[string]string{
		"accessKey": "********",
		"region":    "cn",
	}

	t.Run("unchanged values are omitted", func(t *testing.T) {
		overrides := promptActionRunParameters(defaults, func(_ string, defaultValue string) string {
			return defaultValue
		})

		assert.Empty(t, overrides)
	})

	t.Run("changed values are submitted", func(t *testing.T) {
		overrides := promptActionRunParameters(defaults, func(_ string, defaultValue string) string {
			if defaultValue == "********" {
				return "new-access-key"
			}
			return defaultValue
		})

		assert.Equal(t, map[string]string{"accessKey": "new-access-key"}, overrides)
	})
}

func TestNewActionRunAction(t *testing.T) {
	action := &openv1alpha1resource.Action{
		Name: "projects/p1/actions/a1",
		Spec: &openv1alpha1commons.ActionSpec{
			Name:        "server-action",
			Description: "server-description",
			Parameters:  map[string]string{"accessKey": "masked-default", "region": "cn"},
			Jobs: []*openv1alpha1commons.JobSpec{{
				Name: "main",
			}},
		},
	}

	t.Run("omitted overrides preserve the action and leave parameters empty", func(t *testing.T) {
		runAction := newActionRunAction(action, nil)

		assert.Equal(t, "projects/p1/actions/a1", runAction.Name)
		assert.Equal(t, "server-action", runAction.Spec.Name)
		assert.Equal(t, "server-description", runAction.Spec.Description)
		require.Len(t, runAction.Spec.Jobs, 1)
		assert.Equal(t, "main", runAction.Spec.Jobs[0].Name)
		assert.Empty(t, runAction.Spec.Parameters)
		assert.Equal(t, "masked-default", action.Spec.Parameters["accessKey"])
	})

	t.Run("explicit overrides replace submitted parameters only", func(t *testing.T) {
		overrides := map[string]string{"accessKey": "explicit-value"}

		runAction := newActionRunAction(action, overrides)

		assert.Equal(t, overrides, runAction.Spec.Parameters)
		require.Len(t, runAction.Spec.Jobs, 1)
		assert.Equal(t, "main", runAction.Spec.Jobs[0].Name)
		assert.Equal(t, map[string]string{"accessKey": "masked-default", "region": "cn"}, action.Spec.Parameters)
	})
}
