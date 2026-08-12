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

package api_utils

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"

	"golang.org/x/net/http2"
)

func TestNewConnectClientUsesOpenAPITLSPolicy(t *testing.T) {
	got := NewConnectClient()
	client, ok := got.(*http.Client)
	if !ok {
		t.Fatalf("client type = %T, want *http.Client", got)
	}
	transport, ok := client.Transport.(*http2.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http2.Transport", client.Transport)
	}
	wantCipherSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}

	config := transport.TLSClientConfig
	if config.MinVersion != tls.VersionTLS12 || config.MaxVersion != tls.VersionTLS12 {
		t.Fatalf("OpenAPI transport must allow TLS 1.2 only")
	}
	if !reflect.DeepEqual(config.CipherSuites, wantCipherSuites) {
		t.Fatalf("cipher suites = %#v, want %#v", config.CipherSuites, wantCipherSuites)
	}
}
