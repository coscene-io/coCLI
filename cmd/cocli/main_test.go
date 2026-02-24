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
