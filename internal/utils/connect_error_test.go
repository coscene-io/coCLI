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
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
)

func TestIsConnectErrorWithCode_NilError(t *testing.T) {
	assert.False(t, IsConnectErrorWithCode(nil, connect.CodeNotFound))
}

func TestIsConnectErrorWithCode_NonConnectError(t *testing.T) {
	assert.False(t, IsConnectErrorWithCode(fmt.Errorf("plain error"), connect.CodeNotFound))
}

func TestIsConnectErrorWithCode_MatchingCode(t *testing.T) {
	err := connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
	assert.True(t, IsConnectErrorWithCode(err, connect.CodeNotFound))
}

func TestIsConnectErrorWithCode_NonMatchingCode(t *testing.T) {
	err := connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
	assert.False(t, IsConnectErrorWithCode(err, connect.CodePermissionDenied))
}

func TestIsConnectErrorWithCode_WrappedError(t *testing.T) {
	inner := connect.NewError(connect.CodeUnavailable, fmt.Errorf("unavailable"))
	wrapped := fmt.Errorf("outer: %w", inner)
	assert.True(t, IsConnectErrorWithCode(wrapped, connect.CodeUnavailable))
}
