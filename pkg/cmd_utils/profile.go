// Copyright 2024 coScene
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

package cmd_utils

import (
	"github.com/coscene-io/cocli/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// ProfileManager returns the resolved profile manager for a command.
//
// When the root PersistentPreRun has already resolved and stashed the manager
// on the command context (the common, auth-checked path), that instance is
// reused — avoiding a second resolution and a redundant network authentication.
// For commands that skip the root auth check (e.g. registry), it resolves now,
// honoring the global --profile flag and COS_* env precedence. On a resolution
// error (e.g. --profile names an unknown profile) it logs fatally, surfacing the
// clear message rather than letting callers dereference a nil manager.
func ProfileManager(cmd *cobra.Command, getProvider func(string) config.Provider, cfgPath string) *config.ProfileManager {
	if pm, ok := config.ProfileManagerFromContext(cmd.Context()); ok {
		return pm
	}
	override, _ := cmd.Flags().GetString("profile")
	pm, _, err := config.ResolveProfileManager(cmd.Context(), getProvider(cfgPath), override)
	if err != nil {
		log.Fatalf("Failed to resolve profile: %v", err)
	}
	return pm
}
