//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Run:
//   go test -tags=integration ./test/integration/ -run TestLive -v
//
// Override config path:
//   COCLI_CONFIG=/path/to/config.yaml go test -tags=integration ./test/integration/ -v

func liveProfileManager(t *testing.T) *config.ProfileManager {
	t.Helper()

	cfgPath := constants.DefaultConfigPath
	if p := os.Getenv("COCLI_CONFIG"); p != "" {
		cfgPath = p
	}

	provider := config.Provide(cfgPath)
	pm, err := provider.GetProfileManager()
	require.NoError(t, err, "failed to load cocli config from %s", cfgPath)

	return pm
}

func liveContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestLiveRecordNotFound(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	projectName, err := name.NewProject(pm.GetCurrentProfile().ProjectName)
	require.NoError(t, err, "current profile has invalid project-name")

	fakeRecord := &name.Record{
		ProjectID: projectName.ProjectID,
		RecordID:  "00000000-0000-0000-0000-000000000000",
	}

	_, err = pm.RecordCli().Get(ctx, fakeRecord)
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err),
		"querying non-existent record should return NOT_FOUND, got: %v", err)
}

func TestLiveRecordInvalidArgument(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	badRecord := &name.Record{
		ProjectID: "not-a-valid-project-id",
		RecordID:  "not-a-valid-record-id",
	}

	_, err := pm.RecordCli().Get(ctx, badRecord)
	require.Error(t, err)
	code := connect.CodeOf(err)
	assert.True(t, code == connect.CodeInvalidArgument || code == connect.CodeNotFound,
		"querying with invalid name format should return INVALID_ARGUMENT or NOT_FOUND, got: %v (%v)", code, err)
}

func TestLiveProjectNotFound(t *testing.T) {
	pm := liveProfileManager(t)
	ctx := liveContext(t)

	fakeProject := &name.Project{
		ProjectID: "00000000-0000-0000-0000-000000000000",
	}

	_, err := pm.ProjectCli().Get(ctx, fakeProject)
	require.Error(t, err)
	code := connect.CodeOf(err)
	assert.True(t, code == connect.CodeNotFound || code == connect.CodePermissionDenied,
		"querying non-existent project should return NOT_FOUND or PERMISSION_DENIED (security), got: %v (%v)", code, err)
}
