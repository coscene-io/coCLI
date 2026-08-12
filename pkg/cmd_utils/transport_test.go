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
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestNewTransportUsesGeneralTLSPolicy(t *testing.T) {
	transport := NewTransport(15 * time.Second)
	wantCipherSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	}

	if transport.ResponseHeaderTimeout != 15*time.Second {
		t.Fatalf("response header timeout = %s, want 15s", transport.ResponseHeaderTimeout)
	}
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 || transport.TLSClientConfig.MaxVersion != tls.VersionTLS12 {
		t.Fatalf("transport must allow TLS 1.2 only")
	}
	if !reflect.DeepEqual(transport.TLSClientConfig.CipherSuites, wantCipherSuites) {
		t.Fatalf("cipher suites = %#v, want %#v", transport.TLSClientConfig.CipherSuites, wantCipherSuites)
	}
}

func TestNewHTTPClientUsesNewTransport(t *testing.T) {
	client := NewHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.TLSClientConfig.MaxVersion != tls.VersionTLS12 {
		t.Fatalf("maximum TLS version = %x, want TLS 1.2", transport.TLSClientConfig.MaxVersion)
	}
}
