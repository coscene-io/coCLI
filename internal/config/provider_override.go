// Copyright 2025 coScene
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

// providerOverride allows tests to inject a mock provider
var providerOverride Provider

// SetProviderOverride sets a provider override for testing
// This should only be used in tests!
func SetProviderOverride(p Provider) {
	providerOverride = p
}

// ClearProviderOverride clears the provider override
// This should be called in test cleanup
func ClearProviderOverride() {
	providerOverride = nil
}

// ProvideWithOverride returns the override provider if set, otherwise calls Provide
func ProvideWithOverride(path string) Provider {
	if providerOverride != nil {
		return providerOverride
	}
	return Provide(path)
}
