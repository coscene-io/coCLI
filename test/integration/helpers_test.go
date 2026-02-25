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

//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/stretchr/testify/require"
)

// Run all integration tests:
//   go test -tags=integration ./test/integration/ -v
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
