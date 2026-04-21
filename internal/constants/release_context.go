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

package constants

import (
	"net/url"
	"strings"
)

func ReleaseChannelDisplayName() string {
	return releaseChannelDisplayName(BaseApiEndpoint, DownloadBaseUrl)
}

func releaseChannelDisplayName(baseAPIEndpoint, downloadBaseURL string) string {
	apiHost := hostFromURL(baseAPIEndpoint)
	downloadHost := hostFromURL(downloadBaseURL)

	switch {
	case apiHost == "openapi.coscene.cn" && downloadHost == "download.coscene.cn":
		return "CN"
	case apiHost == "openapi.coscene.io" && downloadHost == "download.coscene.io":
		return "IO"
	default:
		return "Custom"
	}
}

func hostFromURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Host)
	}

	trimmed := strings.TrimSpace(rawURL)
	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	return strings.ToLower(strings.TrimSuffix(trimmed, "/"))
}
