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
	"context"
	"os"
	"testing"
	"time"

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

func TestForceQuitSignals(t *testing.T) {
	ch := forceQuitSignals()
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	if cap(ch) != 1 {
		t.Fatalf("expected buffered channel of cap 1, got cap %d", cap(ch))
	}
}

func TestWatchForceQuit_InvokesCallbackAfterSecondSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	quit := make(chan os.Signal, 1)
	done := make(chan struct{})

	go watchForceQuit(ctx, quit, func() { close(done) })

	// First signal: cancel the context. Callback must NOT fire yet.
	cancel()
	select {
	case <-done:
		t.Fatal("callback fired after first signal; expected it to wait for the second")
	case <-time.After(50 * time.Millisecond):
	}

	// Second signal: deliver on quit. Callback must fire.
	quit <- os.Interrupt
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("callback did not fire after second signal")
	}
}
