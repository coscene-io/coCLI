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

package registry

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewLoginCommand(cfgPath *string, io *iostreams.IOStreams) *cobra.Command {
	var registry string

	cmd := &cobra.Command{
		Use:                   "login",
		Short:                 "Log in to the coScene container registry",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			pm, err := config.Provide(*cfgPath).GetProfileManager()
			if err != nil {
				log.Fatalf("failed to load profile manager: %v", err)
			}

			endpoint := pm.GetCurrentProfile().EndPoint
			host, err := inferRegistryHost(endpoint, registry)
			if err != nil {
				log.Fatalf("%v", err)
			}

			cred, err := pm.ContainerRegistryCli().CreateBasicCredential(cmd.Context())
			if err != nil {
				log.Fatalf("failed to create basic credential: %v", err)
			}

			if err := dockerLogin(host, cred.GetUsername(), cred.GetPassword()); err != nil {
				log.Fatalf("docker login failed: %v", err)
			}

			io.Printf("Logged in to %s as %s\n", host, cred.GetUsername())
		},
	}

	cmd.Flags().StringVar(&registry, "registry", "", "override registry host (e.g. cr.coscene.cn)")

	return cmd
}

// inferRegistryHost determines the registry host from the profile endpoint unless overridden.
func inferRegistryHost(endpoint, override string) (string, error) {
	if override != "" {
		return override, nil
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %v", err)
	}
	host := u.Host

	switch host {
	case "openapi.coscene.cn":
		return "cr.coscene.cn", nil
	case "openapi.staging.coscene.cn":
		return "cr.staging.coscene.cn", nil
	case "openapi.api.coscene.dev", "api.dev.coscene.cn":
		return "cr.dev.coscene.cn", nil
	}

	if after, found := strings.CutPrefix(host, "openapi."); found {
		return "cr." + after, nil
	}

	return "", fmt.Errorf("unable to infer registry host from endpoint '%s'; please specify --registry", endpoint)
}

func dockerLogin(registry, username, password string) error {
	// docker login <registry> --username <username> --password-stdin
	cmd := exec.Command("docker", "login", registry, "--username", username, "--password-stdin")
	cmd.Stdin = bytes.NewBufferString(password + "\n")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		// include stderr for better diagnostics
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(errOut.String()))
	}
	return nil
}
