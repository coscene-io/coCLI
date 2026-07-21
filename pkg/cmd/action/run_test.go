package action

import (
	"testing"

	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
)

func TestNewActionRunAction(t *testing.T) {
	actionName := &name.Action{ProjectID: "p1", ID: "a1"}

	t.Run("omitted overrides leave parameters empty for server defaults", func(t *testing.T) {
		action := newActionRunAction(actionName, nil)

		assert.Equal(t, "projects/p1/actions/a1", action.Name)
		assert.Empty(t, action.Spec.Parameters)
		assert.Empty(t, action.Spec.Jobs)
	})

	t.Run("explicit overrides are the only submitted parameters", func(t *testing.T) {
		overrides := map[string]string{"accessKey": "explicit-value"}

		action := newActionRunAction(actionName, overrides)

		assert.Equal(t, overrides, action.Spec.Parameters)
		assert.Empty(t, action.Spec.Jobs)
	})
}
