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

package cmd_utils

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestAuthCheckAnnotations(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	child := &cobra.Command{Use: "child"}
	root.AddCommand(child)

	assert.True(t, IsAuthCheckEnabled(child))

	DisableAuthCheck(root)
	assert.False(t, IsAuthCheckEnabled(child))
	assert.Equal(t, "true", root.Annotations["skipAuthCheck"])
}

func TestAuthCheckBuiltInCommands(t *testing.T) {
	assert.False(t, IsAuthCheckEnabled(&cobra.Command{Use: "help"}))
	assert.False(t, IsAuthCheckEnabled(&cobra.Command{Use: cobra.ShellCompRequestCmd}))
	assert.False(t, IsAuthCheckEnabled(&cobra.Command{Use: cobra.ShellCompNoDescRequestCmd}))
}

func TestNewTransportUsesSafeDefaults(t *testing.T) {
	timeout := 7 * time.Second

	transport := NewTransport(timeout)

	assert.Equal(t, timeout, transport.ResponseHeaderTimeout)
	assert.Equal(t, time.Minute, transport.IdleConnTimeout)
	assert.Equal(t, 256, transport.MaxIdleConns)
	assert.Equal(t, 16, transport.MaxIdleConnsPerHost)
	assert.True(t, transport.DisableCompression)
	assert.NotNil(t, transport.TLSClientConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
}
