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

package constants

import (
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

const (
	// CLIName is the name of the CLI
	CLIName = "cocli"

	// ConfigFilename is the name of the configuration file
	ConfigFilename = ".cocli.yaml"

	// CurrentOrgNameStr is the string for the current profile
	CurrentOrgNameStr = "organizations/current"

	// MaxPageSize is the maximum page size for the api
	MaxPageSize = 100
)

var (
	// DownloadBaseUrl can be overridden at build time for region-specific releases.
	DownloadBaseUrl = "https://download.coscene.cn/"

	// BaseApiEndpoint can be overridden at build time for region-specific releases.
	BaseApiEndpoint = "https://openapi.coscene.cn"

	DefaultConfigPath      = defaultConfigPath()
	DefaultUploaderDirPath = defaultUploaderDirPath()
)

func defaultConfigPath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("unable to read current user home")
	}
	return path.Join(homedir, ConfigFilename)
}

func defaultUploaderDirPath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("unable to read current user home")
	}
	return path.Join(homedir, ".cache", "cocli")
}
