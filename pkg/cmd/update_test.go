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

package cmd

import (
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewUpdateRequesterUsesGeneralTLSPolicy(t *testing.T) {
	requester := newUpdateRequester()
	transport, ok := requester.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", requester.client.Transport)
	}
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 || transport.TLSClientConfig.MaxVersion != tls.VersionTLS12 {
		t.Fatalf("self-update transport must allow TLS 1.2 only")
	}
	if got := len(transport.TLSClientConfig.CipherSuites); got != 4 {
		t.Fatalf("self-update cipher suite count = %d, want 4", got)
	}
}

func TestUpdateRequesterFetch(t *testing.T) {
	t.Run("returns successful response body", func(t *testing.T) {
		body := &trackingReadCloser{Reader: strings.NewReader("update")}
		requester := &updateRequester{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: body}, nil
		})}}

		got, err := requester.Fetch("https://updates.example.test/version")
		if err != nil {
			t.Fatalf("Fetch returned error: %v", err)
		}
		if got != body {
			t.Fatalf("Fetch returned an unexpected body")
		}
		if body.closed {
			t.Fatal("successful response body was closed before the caller could read it")
		}
		_ = got.Close()
	})

	t.Run("closes non-200 response body", func(t *testing.T) {
		body := &trackingReadCloser{Reader: strings.NewReader("failure")}
		requester := &updateRequester{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Body: body}, nil
		})}}

		got, err := requester.Fetch("https://updates.example.test/version")
		if err == nil || !strings.Contains(err.Error(), "502 Bad Gateway") {
			t.Fatalf("Fetch error = %v, want HTTP status error", err)
		}
		if got != nil {
			t.Fatalf("Fetch body = %v, want nil", got)
		}
		if !body.closed {
			t.Fatal("non-200 response body was not closed")
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}
