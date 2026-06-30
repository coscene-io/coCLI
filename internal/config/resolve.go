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

package config

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ResolveProfileManager resolves the active profile manager applying the global
// --profile override and COS_* env-inject on top of the on-disk config.
//
// Precedence, highest first:
//
//		--profile NAME  >  complete COS_* env profile  >  config current-profile
//
//	  - override set: select the named profile from config; hard error if absent.
//	    If a complete env profile is also present, it is ignored with a warning.
//	  - else complete COS_* env: build an ephemeral profile from the env; if the
//	    config is non-empty, warn that the env profile overrides the configured
//	    current-profile.
//	  - else: use the config current-profile.
//
// The returned ephemeral bool is true whenever resolution was driven by the
// override or the env profile. When ephemeral, the result MUST NOT be persisted
// (doing so would let concurrent invocations clobber the shared config file),
// and the overridden profile is authenticated in memory before returning so
// downstream commands see a populated Org / ProjectName.
func ResolveProfileManager(ctx context.Context, p Provider, override string) (pm *ProfileManager, ephemeral bool, err error) {
	pm, err = p.GetProfileManager()
	if err != nil {
		return nil, false, err
	}

	envProfile, err := buildEnvProfileFromOS()
	if err != nil {
		return nil, false, err
	}

	// 1. --profile NAME wins.
	if override != "" {
		if pm == nil || !pm.hasProfile(override) {
			return nil, false, errors.Errorf("profile %q not found in config", override)
		}
		pm.CurrentProfile = override
		if envProfile != nil {
			log.Warnf("COS_* env profile ignored; --profile %q takes precedence", override)
		}
		if err = ephemeralAuth(ctx, pm); err != nil {
			return nil, false, err
		}
		return pm, true, nil
	}

	// 2. Complete COS_* env profile.
	if envProfile != nil {
		// Validate the env-built profile (endpoint format etc.) so a malformed
		// COS_ENDPOINT fails with a clear message rather than surfacing later
		// as a confusing network error — matching pre-refactor behavior.
		if err = envProfile.Validate(); err != nil {
			return nil, false, errors.Wrap(err, "invalid COS_* env profile")
		}
		if pm != nil && !pm.IsEmpty() {
			log.Warnf("COS_* env profile overrides configured current-profile %q", pm.CurrentProfile)
		}
		envPM := &ProfileManager{
			CurrentProfile: envProfile.Name,
			Profiles:       []*Profile{envProfile},
		}
		if err = ephemeralAuth(ctx, envPM); err != nil {
			return nil, false, err
		}
		return envPM, true, nil
	}

	// 3. Config current-profile (persist allowed).
	return pm, false, nil
}

// ephemeralAuth is the in-memory authentication step for resolved profiles.
// It is a package variable so tests can stub out the network call.
var ephemeralAuth = authEphemeral

// authEphemeral authenticates the current profile in memory when it is not
// already authenticated, so an overridden / env-inject profile has its Org and
// ProjectName populated without touching disk.
func authEphemeral(ctx context.Context, pm *ProfileManager) error {
	if pm.CheckAuth() {
		return nil
	}
	if err := pm.Auth(ctx); err != nil {
		return errors.Wrap(err, "unable to authenticate resolved profile")
	}
	return nil
}

// hasProfile reports whether a profile with the given name exists.
func (pm *ProfileManager) hasProfile(name string) bool {
	for _, p := range pm.Profiles {
		if p.Name == name {
			return true
		}
	}
	return false
}
