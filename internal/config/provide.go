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
	"os"
	"strings"

	"dario.cat/mergo"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/pkg/errors"
)

// Provider is an interface for providing the configuration
type Provider interface {
	GetProfileManager() (*ProfileManager, error)
	Persist(pm *ProfileManager) error
}

// globalConfig implements Provider
type globalConfig struct {
	path           string     `koanf:"-"`
	CurrentProfile string     `koanf:"current-profile"`
	Profiles       []*Profile `koanf:"profiles"`
}

func Provide(path string) Provider {
	return &globalConfig{path: path}
}

// EnvProfileName is the synthetic profile name assigned to a profile that is
// built entirely from COS_* environment variables (env-inject). It never
// appears in the on-disk config and is never persisted.
const EnvProfileName = "ENV_LOADED_PROFILE"

// GetProfileManager loads the profile manager from the config file only.
//
// Env-inject (building a profile from COS_ENDPOINT / COS_TOKEN / COS_PROJECT)
// and the --profile override are handled by ResolveProfileManager, which wraps
// this method. Callers that need override/env precedence (all data-plane
// commands and the root auth check) must go through ResolveProfileManager.
// Config-plane commands (login *) call this directly so they always operate on
// the real on-disk config.
func (cfg *globalConfig) GetProfileManager() (*ProfileManager, error) {
	if err := cfg.loadYaml("current-profile", &cfg.CurrentProfile); err != nil {
		return nil, errors.Wrapf(err, "unable to load current-profile from %s", cfg.path)
	}
	if err := cfg.loadYaml("profiles", &cfg.Profiles); err != nil {
		return nil, errors.Wrapf(err, "unable to load profiles from %s", cfg.path)
	}

	pm := new(ProfileManager)
	pm.CurrentProfile = cfg.CurrentProfile
	pm.Profiles = cfg.Profiles

	if err := pm.Validate(); err != nil {
		return nil, errors.Wrapf(err, "profile validation failed")
	}

	return pm, nil
}

// buildEnvProfileFromOS builds a profile from the COS_* environment variables.
// It returns (nil, nil) when the env set is incomplete — i.e. any of endpoint,
// token, or project slug is missing — so partial COS_* is silently ignored.
// The COS_PROJECTID hack fills the project name directly when present.
func buildEnvProfileFromOS() (*Profile, error) {
	k := koanf.New(".")
	if err := k.Load(
		env.Provider(
			"COS",
			"_",
			func(s string) string {
				return strings.ToLower(strings.TrimPrefix(s, "COS_"))
			},
		),
		nil,
	); err != nil {
		return nil, errors.Wrap(err, "load config from env")
	}

	p := &Profile{}
	if err := k.Unmarshal("", p); err != nil {
		return nil, errors.Wrap(err, "unmarshal env")
	}
	// Force the synthetic name after unmarshal so a stray COS_NAME cannot
	// rename (and thereby alias) the env-inject profile.
	p.Name = EnvProfileName

	// Hack to prioritize filling project name with project id
	if projectID := os.Getenv("COS_PROJECTID"); projectID != "" {
		p.ProjectName = name.Project{ProjectID: projectID}.String()
	}

	if p.EndPoint == "" || p.Token == "" || p.ProjectSlug == "" {
		return nil, nil
	}

	return p, nil
}

// Persist saves the profile manager to the config file
func (cfg *globalConfig) Persist(pm *ProfileManager) error {
	cfg.CurrentProfile = pm.CurrentProfile
	cfg.Profiles = pm.Profiles
	return cfg.persist()
}

func (cfg *globalConfig) loadYaml(path string, any interface{}) error {
	k := koanf.New(".")
	if err := k.Load(file.Provider(cfg.path), yaml.Parser()); err != nil {
		return errors.Wrapf(err, "unable to load config from yaml %s", cfg.path)
	}

	if err := k.Unmarshal(path, any); err != nil {
		return errors.Wrapf(err, "unable to unmarshal config from %s", cfg.path)
	}

	return nil
}

// persist saves the current config as an update to the original config file
func (cfg *globalConfig) persist() error {
	// Load original config
	originalConfig := &globalConfig{path: cfg.path}
	err := cfg.loadYaml("", originalConfig)
	if err != nil {
		return errors.Wrapf(err, "unable to load config from %s", cfg.path)
	}

	// Update original with current
	err = mergo.Merge(originalConfig, cfg, mergo.WithOverride)
	if err != nil {
		return errors.Wrapf(err, "unable to merge config")
	}

	k := koanf.New(".")

	// load updated originalConfig to k
	err = k.Load(structs.Provider(originalConfig, "koanf"), nil)
	if err != nil {
		return errors.Wrapf(err, "unable to load config to k from original config")
	}
	// marshal k to yamlStr
	yamlStr, err := k.Marshal(yaml.Parser())
	if err != nil {
		return errors.Wrapf(err, "unable to marshal k to yaml")
	}

	// write yamlStr to globalConfig.path
	err = os.WriteFile(originalConfig.path, yamlStr, 0644)
	if err != nil {
		return errors.Wrapf(err, "unable to write yaml to %s", originalConfig.path)
	}
	return nil
}
