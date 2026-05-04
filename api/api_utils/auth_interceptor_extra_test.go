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
	"context"
	"encoding/base64"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestAuthInterceptorAddsAPIKeyHeaders(t *testing.T) {
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("apikey:secret"))
	interceptor := AuthInterceptor("secret")

	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		assert.Equal(t, want, req.Header().Get("Authorization"))
		assert.Equal(t, want, req.Header().Get("x-cos-auth-token"))
		return connect.NewResponse(&emptypb.Empty{}), nil
	})

	_, err := wrapped(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	require.NoError(t, err)
}

func TestAuthInterceptorKeepsJWTBearerToken(t *testing.T) {
	interceptor := AuthInterceptor("header.payload.signature")

	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		assert.Equal(t, "Bearer header.payload.signature", req.Header().Get("Authorization"))
		assert.Equal(t, "Bearer header.payload.signature", req.Header().Get("x-cos-auth-token"))
		return connect.NewResponse(&emptypb.Empty{}), nil
	})

	_, err := wrapped(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	require.NoError(t, err)
}

func TestNewConnectClient(t *testing.T) {
	client, ok := NewConnectClient().(*http.Client)
	require.True(t, ok)
	_, ok = client.Transport.(*http2.Transport)
	assert.True(t, ok)
}
