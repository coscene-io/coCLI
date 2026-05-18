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

import "testing"

func TestNormalizeS3Endpoint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "host", in: "storage-cn-guangzhou.volc.coscene.cn", want: "https://storage-cn-guangzhou.volc.coscene.cn"},
		{name: "shanghai host", in: "storage-cn-shanghai.coscene.cn", want: "https://storage-cn-shanghai.coscene.cn"},
		{name: "us host", in: "storage-us-central-1.coscene.io", want: "https://storage-us-central-1.coscene.io"},
		{name: "https", in: "https://storage-cn-guangzhou.volc.coscene.cn", want: "https://storage-cn-guangzhou.volc.coscene.cn"},
		{name: "http", in: "http://localhost:9000", want: "http://localhost:9000"},
		{name: "trim", in: " storage-cn-hangzhou.coscene.cn ", want: "https://storage-cn-hangzhou.coscene.cn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeS3Endpoint(tt.in); got != tt.want {
				t.Fatalf("normalizeS3Endpoint(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestProjectS3Bucket(t *testing.T) {
	tests := []struct {
		name        string
		orgSlug     string
		projectSlug string
		want        string
	}{
		{name: "normal", orgSlug: "coscene-hy", projectSlug: "demo", want: "coscene-hy.demo"},
		{name: "short project", orgSlug: "coscene-fei-shu", projectSlug: "cs", want: "coscene-fei-shu.cs"},
		{name: "hyphenated", orgSlug: "tii-humanoids", projectSlug: "ego-c", want: "tii-humanoids.ego-c"},
		{name: "empty org", orgSlug: "", projectSlug: "demo", want: ""},
		{name: "empty project", orgSlug: "coscene-hy", projectSlug: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := projectS3Bucket(tt.orgSlug, tt.projectSlug); got != tt.want {
				t.Fatalf("projectS3Bucket(%q, %q) = %q, want %q", tt.orgSlug, tt.projectSlug, got, tt.want)
			}
		})
	}
}
