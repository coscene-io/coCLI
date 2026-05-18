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

package project

import (
	"context"
	"strings"

	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/printer"
	"github.com/coscene-io/cocli/internal/printer/printable"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewS3InfoCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:                   "s3-info [project-resource-name/slug]",
		Short:                 "Show S3 connection information for a project",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pm, _ := getProvider(*cfgPath).GetProfileManager()
			profile := pm.GetCurrentProfile()
			projectName, err := resolveS3InfoProjectName(cmd.Context(), pm, args)
			if err != nil {
				log.Fatalf("unable to resolve project: %v", err)
			}

			project, err := pm.ProjectCli().Get(cmd.Context(), projectName)
			if err != nil {
				log.Fatalf("unable to get project: %v", err)
			}

			securityToken, err := pm.SecurityTokenCli().GenerateSecurityToken(cmd.Context(), projectName.String())
			if err != nil {
				log.Fatalf("unable to get S3 endpoint: %v", err)
			}

			endpoint := normalizeS3Endpoint(securityToken.GetEndpoint())
			if endpoint == "" {
				log.Fatalf("unable to get S3 endpoint: empty endpoint")
			}

			bucket := projectS3Bucket(profile.Org, project.GetSlug())
			if bucket == "" {
				log.Fatalf("unable to get S3 bucket: missing organization or project slug")
			}

			info := printable.NewProjectS3Info(endpoint, project.GetRegion(), bucket)
			p, err := printer.Printer(outputFormat, &printer.Options{})
			if err != nil {
				log.Fatal(err)
			}
			if err = p.PrintObj(info, io.Out); err != nil {
				log.Fatalf("unable to print S3 connection information: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json|yaml)")

	return cmd
}

func resolveS3InfoProjectName(ctx context.Context, pm *config.ProfileManager, args []string) (*name.Project, error) {
	if len(args) == 0 || args[0] == "" {
		return pm.ProjectName(ctx, "")
	}
	if strings.Contains(args[0], "/") {
		projectName, err := name.NewProject(args[0])
		if err != nil {
			return nil, errors.Wrap(err, "parse project resource name")
		}
		return projectName, nil
	}
	return pm.ProjectName(ctx, args[0])
}

func normalizeS3Endpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	return "https://" + endpoint
}

func projectS3Bucket(orgSlug, projectSlug string) string {
	if orgSlug == "" || projectSlug == "" {
		return ""
	}
	return orgSlug + "." + projectSlug
}
