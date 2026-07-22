package action

import (
	"testing"

	openv1alpha1commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
