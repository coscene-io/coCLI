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

package action

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/pkg/errors"
)

func TestResolveActionRun(t *testing.T) {
	proj := &name.Project{ProjectID: "11111111-1111-1111-1111-111111111111"}

	t.Run("full resource name", func(t *testing.T) {
		got, err := resolveActionRun("projects/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/actionRuns/bbbb", proj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ProjectID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || got.ID != "bbbb" {
			t.Fatalf("unexpected parse: %+v", got)
		}
	})

	t.Run("bare uuid uses project", func(t *testing.T) {
		got, err := resolveActionRun("22222222-2222-2222-2222-222222222222", proj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ProjectID != proj.ProjectID || got.ID != "22222222-2222-2222-2222-222222222222" {
			t.Fatalf("unexpected: %+v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := resolveActionRun("not-a-name", proj); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestIsRetriable(t *testing.T) {
	retriable := []connect.Code{
		connect.CodeUnknown, connect.CodeInternal, connect.CodeUnavailable,
		connect.CodeAborted, connect.CodeResourceExhausted,
	}
	for _, code := range retriable {
		if !isRetriable(connect.NewError(code, errors.New("x"))) {
			t.Fatalf("code %v should be retriable", code)
		}
	}
	notRetriable := []connect.Code{connect.CodeNotFound, connect.CodeUnauthenticated, connect.CodeInvalidArgument}
	for _, code := range notRetriable {
		if isRetriable(connect.NewError(code, errors.New("x"))) {
			t.Fatalf("code %v should not be retriable", code)
		}
	}
	if isRetriable(errors.New("plain")) {
		t.Fatal("non-connect error should not be retriable")
	}
}

func TestNextDelay(t *testing.T) {
	if got := nextDelay(2 * time.Second); got != 4*time.Second {
		t.Fatalf("got %v, want 4s", got)
	}
	if got := nextDelay(reconnectMaxDelay); got != reconnectMaxDelay {
		t.Fatalf("delay should cap at %v, got %v", reconnectMaxDelay, got)
	}
}

func TestIsNodeNotFound(t *testing.T) {
	if !isNodeNotFound(connect.NewError(connect.CodeInvalidArgument, errors.New("x"))) {
		t.Fatal("InvalidArgument should be treated as node-not-found")
	}
	if isNodeNotFound(connect.NewError(connect.CodeUnavailable, errors.New("x"))) {
		t.Fatal("Unavailable should not be node-not-found")
	}
}
