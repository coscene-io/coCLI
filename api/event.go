// Copyright 2024 coScene
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

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/pkg/errors"
)

type EventInterface interface {
	// ObtainEvent creates an event if not found, fetches otherwise.
	ObtainEvent(ctx context.Context, parent string, event *openv1alpha1resource.Event) (*openv1alpha1service.ObtainEventResponse, error)
}

type eventClient struct {
	eventClient openv1alpha1connect.EventServiceClient
}

func NewEventClient(eventServiceClient openv1alpha1connect.EventServiceClient) EventInterface {
	return &eventClient{
		eventClient: eventServiceClient,
	}
}

func (c *eventClient) ObtainEvent(ctx context.Context, parent string, event *openv1alpha1resource.Event) (*openv1alpha1service.ObtainEventResponse, error) {
	createEventReq := connect.NewRequest(&openv1alpha1service.ObtainEventRequest{
		Parent: parent,
		Event:  event,
	})
	createEventRes, err := c.eventClient.ObtainEvent(ctx, createEventReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to obtain event %s", event.DisplayName)
	}

	return createEventRes.Msg, nil
}
