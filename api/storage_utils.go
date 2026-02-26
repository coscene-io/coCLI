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

package api

import (
	openv1alpha1enums "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/enums"
)

var regionDisplayName = map[openv1alpha1enums.RegionEnum_Region]string{
	openv1alpha1enums.RegionEnum_REGION_UNSPECIFIED: "unspecified",
	openv1alpha1enums.RegionEnum_CN_HANGZHOU:        "cn-hangzhou",
	openv1alpha1enums.RegionEnum_CN_SHANGHAI:        "cn-shanghai",
}

// FormatRegion returns a human-readable region string.
func FormatRegion(region openv1alpha1enums.RegionEnum_Region) string {
	if r, ok := regionDisplayName[region]; ok {
		return r
	}
	return "unspecified"
}
