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

package utils

import (
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentryRunOptionsRun(t *testing.T) {
	done := make(chan struct{})

	SentryRunOptions{RoutineName: "test-routine"}.Run(func(hub *sentry.Hub) {
		require.NotNil(t, hub)
		close(done)
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sentry wrapped goroutine did not run")
	}
}

func TestSentryHook(t *testing.T) {
	hook := NewSentryHook()
	assert.Equal(t, []log.Level{log.FatalLevel, log.PanicLevel}, hook.Levels())

	require.NoError(t, hook.Fire(&log.Entry{Level: log.ErrorLevel, Message: "recoverable"}))
	require.NoError(t, hook.Fire(&log.Entry{Level: log.FatalLevel, Message: "fatal"}))
}
