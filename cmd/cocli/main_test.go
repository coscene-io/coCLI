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

package main

import (
	"testing"

	"github.com/coscene-io/cocli"
)

func TestNewSentryClientOptions(t *testing.T) {
	opts := newSentryClientOptions()

	if opts.Dsn == "" {
		t.Fatalf("expected DSN to be set")
	}
	if opts.Release != cocli.GetVersion() {
		t.Fatalf("expected release %q, got %q", cocli.GetVersion(), opts.Release)
	}
	if opts.TracesSampleRate != 1.0 {
		t.Fatalf("expected TracesSampleRate 1.0, got %v", opts.TracesSampleRate)
	}
	if !opts.AttachStacktrace {
		t.Fatalf("expected AttachStacktrace to be true")
	}
}
