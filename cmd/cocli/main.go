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

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coscene-io/cocli"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := sentry.Init(newSentryClientOptions())
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	// Flush buffered events before the program terminates.
	defer sentry.Flush(2 * time.Second)

	defer func() {
		if r := recover(); r != nil {
			sentry.CurrentHub().Recover(r)
			panic(r)
		}
	}()

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
	log.AddHook(utils.NewSentryHook())

	io := iostreams.System()

	// Cancel the root context on the first SIGINT/SIGTERM so long-lived
	// commands (e.g. streaming `action logs -f`) exit cleanly. A second
	// signal force-kills, matching docker/kubectl behavior.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go watchForceQuit(ctx, forceQuitSignals(), func() {
		sentry.Flush(500 * time.Millisecond) // bounded — user is force-quitting
		os.Exit(130)                         // second signal: force quit (128 + SIGINT)
	})

	if err := cmd.NewCommand(io, config.Provide).ExecuteContext(ctx); err != nil {
		io.Println(err)
		// os.Exit skips deferred sentry.Flush; flush explicitly so errored runs
		// still report telemetry.
		sentry.Flush(2 * time.Second)
		os.Exit(1)
	}
}

// forceQuitSignals returns a channel that fires on the next interrupt/termination
// signal, used to implement double-Ctrl-C force-quit.
func forceQuitSignals() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	return ch
}

// watchForceQuit blocks until the context is cancelled (first signal) and then
// a value arrives on quit (second signal), at which point onForceQuit is invoked.
// Extracted from main for testability — the production callback calls os.Exit.
func watchForceQuit(ctx context.Context, quit <-chan os.Signal, onForceQuit func()) {
	<-ctx.Done() // first signal: graceful cancel requested
	<-quit       // second signal: user insists
	onForceQuit()
}

func newSentryClientOptions() sentry.ClientOptions {
	return sentry.ClientOptions{
		Dsn:     "https://b3bcd9e4d101f927b5f1f7ac67d9b115@sentry.coscene.site/23",
		Release: cocli.GetVersion(),
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for tracing.
		// We recommend adjusting this value in production,
		TracesSampleRate: 1.0,
		AttachStacktrace: true,
	}
}
