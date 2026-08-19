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

package api

import (
	"context"
	"errors"
	"testing"

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

const eventTestName = "projects/11111111-1111-1111-1111-111111111111/events/22222222-2222-2222-2222-222222222222"

type mockEventServiceClient struct {
	openv1alpha1connect.EventServiceClient
	getEventFunc    func(context.Context, *connect.Request[openv1alpha1service.GetEventRequest]) (*connect.Response[openv1alpha1resource.Event], error)
	deleteEventFunc func(context.Context, *connect.Request[openv1alpha1service.DeleteEventRequest]) (*connect.Response[emptypb.Empty], error)
}

func (m *mockEventServiceClient) GetEvent(ctx context.Context, req *connect.Request[openv1alpha1service.GetEventRequest]) (*connect.Response[openv1alpha1resource.Event], error) {
	return m.getEventFunc(ctx, req)
}

func (m *mockEventServiceClient) DeleteEvent(ctx context.Context, req *connect.Request[openv1alpha1service.DeleteEventRequest]) (*connect.Response[emptypb.Empty], error) {
	return m.deleteEventFunc(ctx, req)
}

func TestEventClientGetEvent(t *testing.T) {
	eventName, err := name.NewEvent(eventTestName)
	require.NoError(t, err)

	mock := &mockEventServiceClient{
		getEventFunc: func(_ context.Context, req *connect.Request[openv1alpha1service.GetEventRequest]) (*connect.Response[openv1alpha1resource.Event], error) {
			assert.Equal(t, eventTestName, req.Msg.GetName())
			return connect.NewResponse(&openv1alpha1resource.Event{Name: eventTestName, DisplayName: "Collision"}), nil
		},
	}

	event, err := NewEventClient(mock).GetEvent(context.Background(), eventName)
	require.NoError(t, err)
	assert.Equal(t, "Collision", event.GetDisplayName())
}

func TestEventClientGetEventPreservesConnectError(t *testing.T) {
	eventName, err := name.NewEvent(eventTestName)
	require.NoError(t, err)

	mock := &mockEventServiceClient{
		getEventFunc: func(context.Context, *connect.Request[openv1alpha1service.GetEventRequest]) (*connect.Response[openv1alpha1resource.Event], error) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("missing"))
		},
	}

	_, err = NewEventClient(mock).GetEvent(context.Background(), eventName)
	require.Error(t, err)
	assert.True(t, utils.IsConnectErrorWithCode(err, connect.CodeNotFound))
}

func TestEventClientDeleteEvent(t *testing.T) {
	eventName, err := name.NewEvent(eventTestName)
	require.NoError(t, err)

	mock := &mockEventServiceClient{
		deleteEventFunc: func(_ context.Context, req *connect.Request[openv1alpha1service.DeleteEventRequest]) (*connect.Response[emptypb.Empty], error) {
			assert.Equal(t, eventTestName, req.Msg.GetName())
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
	}

	require.NoError(t, NewEventClient(mock).DeleteEvent(context.Background(), eventName))
}

func TestEventClientDeleteEventPreservesConnectError(t *testing.T) {
	eventName, err := name.NewEvent(eventTestName)
	require.NoError(t, err)

	mock := &mockEventServiceClient{
		deleteEventFunc: func(context.Context, *connect.Request[openv1alpha1service.DeleteEventRequest]) (*connect.Response[emptypb.Empty], error) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("denied"))
		},
	}

	err = NewEventClient(mock).DeleteEvent(context.Background(), eventName)
	require.Error(t, err)
	assert.True(t, utils.IsConnectErrorWithCode(err, connect.CodePermissionDenied))
}
