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

package tlsconfig

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestOpenAPIPolicy(t *testing.T) {
	want := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}
	assertPolicy(t, NewOpenAPI(), want)
}

func TestGeneralPolicy(t *testing.T) {
	want := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	}
	assertPolicy(t, NewGeneral(), want)
}

func TestPoliciesReturnIndependentConfigs(t *testing.T) {
	first := NewGeneral()
	second := NewGeneral()
	first.MinVersion = tls.VersionTLS11
	first.CipherSuites[0] = tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA

	if second.MinVersion != tls.VersionTLS12 {
		t.Fatalf("mutating one config changed another config's minimum TLS version")
	}
	if second.CipherSuites[0] != tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 {
		t.Fatalf("mutating one config changed another config's cipher suites")
	}
}

func TestGeneralPolicyTLSHandshake(t *testing.T) {
	for _, cipherSuite := range generalCipherSuites {
		t.Run("allows "+tls.CipherSuiteName(cipherSuite), func(t *testing.T) {
			server := newTLSServer(t, tls.VersionTLS12, tls.VersionTLS12, []uint16{cipherSuite})
			defer server.Close()

			client := clientForTestServer(server, NewGeneral())
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("request with allowed TLS policy failed: %v", err)
			}
			_ = resp.Body.Close()
			if resp.TLS.Version != tls.VersionTLS12 {
				t.Fatalf("negotiated TLS version %x, want TLS 1.2", resp.TLS.Version)
			}
			if resp.TLS.CipherSuite != cipherSuite {
				t.Fatalf("negotiated cipher suite %x, want %x", resp.TLS.CipherSuite, cipherSuite)
			}
		})
	}

	t.Run("rejects TLS 1.3", func(t *testing.T) {
		server := newTLSServer(t, tls.VersionTLS13, tls.VersionTLS13, nil)
		defer server.Close()

		_, err := clientForTestServer(server, NewGeneral()).Get(server.URL)
		if err == nil {
			t.Fatal("expected TLS 1.3-only server to be rejected")
		}
	})

	for _, cipherSuite := range []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	} {
		t.Run("rejects "+tls.CipherSuiteName(cipherSuite), func(t *testing.T) {
			server := newTLSServer(t, tls.VersionTLS12, tls.VersionTLS12, []uint16{cipherSuite})
			defer server.Close()

			_, err := clientForTestServer(server, NewGeneral()).Get(server.URL)
			if err == nil {
				t.Fatalf("expected server limited to %s to be rejected", tls.CipherSuiteName(cipherSuite))
			}
		})
	}
}

func TestOpenAPIPolicyRejectsStaticRSA(t *testing.T) {
	server := newTLSServer(t, tls.VersionTLS12, tls.VersionTLS12, []uint16{
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	})
	defer server.Close()

	_, err := clientForTestServer(server, NewOpenAPI()).Get(server.URL)
	if err == nil {
		t.Fatal("expected OpenAPI policy to reject static RSA cipher suite")
	}
}

func assertPolicy(t *testing.T, config *tls.Config, wantCipherSuites []uint16) {
	t.Helper()
	if config.MinVersion != tls.VersionTLS12 || config.MaxVersion != tls.VersionTLS12 {
		t.Fatalf("TLS versions = %x-%x, want TLS 1.2 only", config.MinVersion, config.MaxVersion)
	}
	if !reflect.DeepEqual(config.CipherSuites, wantCipherSuites) {
		t.Fatalf("cipher suites = %#v, want %#v", config.CipherSuites, wantCipherSuites)
	}
}

func newTLSServer(t *testing.T, minVersion, maxVersion uint16, cipherSuites []uint16) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = &tls.Config{
		MinVersion:   minVersion,
		MaxVersion:   maxVersion,
		CipherSuites: cipherSuites,
	}
	server.StartTLS()
	return server
}

func clientForTestServer(server *httptest.Server, config *tls.Config) *http.Client {
	config.RootCAs = server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	return &http.Client{Transport: &http.Transport{TLSClientConfig: config}}
}
