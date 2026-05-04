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
	"testing"

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockContainerRegistryServiceClient struct {
	openv1alpha1connect.ContainerRegistryServiceClient
	createBasicCredentialFunc func(context.Context, *connect.Request[openv1alpha1service.CreateBasicCredentialRequest]) (*connect.Response[openv1alpha1service.BasicCredential], error)
}

func (m *mockContainerRegistryServiceClient) CreateBasicCredential(ctx context.Context, req *connect.Request[openv1alpha1service.CreateBasicCredentialRequest]) (*connect.Response[openv1alpha1service.BasicCredential], error) {
	return m.createBasicCredentialFunc(ctx, req)
}

type mockCustomFieldServiceClient struct {
	openv1alpha1connect.CustomFieldServiceClient
	getRecordCustomFieldSchemaFunc func(context.Context, *connect.Request[openv1alpha1service.GetRecordCustomFieldSchemaRequest]) (*connect.Response[commons.CustomFieldSchema], error)
}

func (m *mockCustomFieldServiceClient) GetRecordCustomFieldSchema(ctx context.Context, req *connect.Request[openv1alpha1service.GetRecordCustomFieldSchemaRequest]) (*connect.Response[commons.CustomFieldSchema], error) {
	return m.getRecordCustomFieldSchemaFunc(ctx, req)
}

type mockEventServiceClient struct {
	openv1alpha1connect.EventServiceClient
	obtainEventFunc func(context.Context, *connect.Request[openv1alpha1service.ObtainEventRequest]) (*connect.Response[openv1alpha1service.ObtainEventResponse], error)
}

func (m *mockEventServiceClient) ObtainEvent(ctx context.Context, req *connect.Request[openv1alpha1service.ObtainEventRequest]) (*connect.Response[openv1alpha1service.ObtainEventResponse], error) {
	return m.obtainEventFunc(ctx, req)
}

type mockOrganizationServiceClient struct {
	openv1alpha1connect.OrganizationServiceClient
	getOrganizationFunc func(context.Context, *connect.Request[openv1alpha1service.GetOrganizationRequest]) (*connect.Response[openv1alpha1resource.Organization], error)
}

func (m *mockOrganizationServiceClient) GetOrganization(ctx context.Context, req *connect.Request[openv1alpha1service.GetOrganizationRequest]) (*connect.Response[openv1alpha1resource.Organization], error) {
	return m.getOrganizationFunc(ctx, req)
}

type mockRoleServiceClient struct {
	openv1alpha1connect.RoleServiceClient
	listRolesFunc func(context.Context, *connect.Request[openv1alpha1service.ListRolesRequest]) (*connect.Response[openv1alpha1service.ListRolesResponse], error)
}

func (m *mockRoleServiceClient) ListRoles(ctx context.Context, req *connect.Request[openv1alpha1service.ListRolesRequest]) (*connect.Response[openv1alpha1service.ListRolesResponse], error) {
	return m.listRolesFunc(ctx, req)
}

type mockTaskServiceClient struct {
	openv1alpha1connect.TaskServiceClient
	upsertTaskFunc func(context.Context, *connect.Request[openv1alpha1service.UpsertTaskRequest]) (*connect.Response[openv1alpha1resource.Task], error)
	syncTaskFunc   func(context.Context, *connect.Request[openv1alpha1service.SyncTaskRequest]) (*connect.Response[openv1alpha1resource.Task], error)
}

func (m *mockTaskServiceClient) UpsertTask(ctx context.Context, req *connect.Request[openv1alpha1service.UpsertTaskRequest]) (*connect.Response[openv1alpha1resource.Task], error) {
	return m.upsertTaskFunc(ctx, req)
}

func (m *mockTaskServiceClient) SyncTask(ctx context.Context, req *connect.Request[openv1alpha1service.SyncTaskRequest]) (*connect.Response[openv1alpha1resource.Task], error) {
	return m.syncTaskFunc(ctx, req)
}

func TestContainerRegistryClientCreateBasicCredential(t *testing.T) {
	mock := &mockContainerRegistryServiceClient{
		createBasicCredentialFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.CreateBasicCredentialRequest]) (*connect.Response[openv1alpha1service.BasicCredential], error) {
			return connect.NewResponse(&openv1alpha1service.BasicCredential{Username: "robot", Password: "secret"}), nil
		},
	}

	credential, err := NewContainerRegistryClient(mock).CreateBasicCredential(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "robot", credential.Username)
	assert.Equal(t, "secret", credential.Password)
}

func TestCustomFieldClientGetRecordCustomFieldSchema(t *testing.T) {
	mock := &mockCustomFieldServiceClient{
		getRecordCustomFieldSchemaFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetRecordCustomFieldSchemaRequest]) (*connect.Response[commons.CustomFieldSchema], error) {
			assert.Equal(t, "projects/project-a", req.Msg.Project)
			return connect.NewResponse(&commons.CustomFieldSchema{
				Properties: []*commons.Property{{Name: "weather"}},
			}), nil
		},
	}

	schema, err := NewCustomFieldClient(mock).GetRecordCustomFieldSchema(context.Background(), &name.Project{ProjectID: "project-a"})

	require.NoError(t, err)
	require.Len(t, schema.Properties, 1)
	assert.Equal(t, "weather", schema.Properties[0].Name)
}

func TestEventClientObtainEvent(t *testing.T) {
	event := &openv1alpha1resource.Event{DisplayName: "Hard brake"}
	mock := &mockEventServiceClient{
		obtainEventFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ObtainEventRequest]) (*connect.Response[openv1alpha1service.ObtainEventResponse], error) {
			assert.Equal(t, "projects/project-a", req.Msg.Parent)
			assert.Equal(t, event, req.Msg.Event)
			return connect.NewResponse(&openv1alpha1service.ObtainEventResponse{Event: event}), nil
		},
	}

	res, err := NewEventClient(mock).ObtainEvent(context.Background(), "projects/project-a", event)

	require.NoError(t, err)
	assert.Equal(t, event, res.Event)
}

func TestOrganizationClientSlug(t *testing.T) {
	mock := &mockOrganizationServiceClient{
		getOrganizationFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.GetOrganizationRequest]) (*connect.Response[openv1alpha1resource.Organization], error) {
			assert.Equal(t, "organizations/current", req.Msg.Name)
			return connect.NewResponse(&openv1alpha1resource.Organization{Slug: "coScene"}), nil
		},
	}

	org, err := name.NewOrganization("organizations/current")
	require.NoError(t, err)
	slug, err := NewOrganizationClient(mock).Slug(context.Background(), org)

	require.NoError(t, err)
	assert.Equal(t, "coScene", slug)
}

func TestRoleClientListRoles(t *testing.T) {
	mock := &mockRoleServiceClient{
		listRolesFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.ListRolesRequest]) (*connect.Response[openv1alpha1service.ListRolesResponse], error) {
			assert.Equal(t, int32(20), req.Msg.PageSize)
			assert.Equal(t, "next", req.Msg.PageToken)
			assert.Equal(t, `level="project"`, req.Msg.Filter)
			return connect.NewResponse(&openv1alpha1service.ListRolesResponse{
				Roles:         []*openv1alpha1resource.Role{{Name: "roles/admin", Code: "admin"}},
				NextPageToken: "after",
				TotalSize:     1,
			}), nil
		},
	}

	res, err := NewRoleClient(mock).ListRoles(context.Background(), &ListRolesOptions{
		Level:     "project",
		PageSize:  20,
		PageToken: "next",
	})

	require.NoError(t, err)
	assert.Equal(t, "after", res.NextPageToken)
	assert.Equal(t, int64(1), res.TotalSize)
	require.Len(t, res.Roles, 1)
	assert.Equal(t, "admin", res.Roles[0].Code)
}

func TestTaskClientUpsertAndSync(t *testing.T) {
	task := &openv1alpha1resource.Task{Name: "projects/project-a/tasks/task-a", Title: "Review"}
	mock := &mockTaskServiceClient{
		upsertTaskFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.UpsertTaskRequest]) (*connect.Response[openv1alpha1resource.Task], error) {
			assert.Equal(t, "projects/project-a", req.Msg.Parent)
			assert.Equal(t, task, req.Msg.Task)
			return connect.NewResponse(task), nil
		},
		syncTaskFunc: func(ctx context.Context, req *connect.Request[openv1alpha1service.SyncTaskRequest]) (*connect.Response[openv1alpha1resource.Task], error) {
			assert.Equal(t, "projects/project-a/tasks/task-a", req.Msg.Name)
			return connect.NewResponse(task), nil
		},
	}
	client := NewTaskClient(mock)

	upserted, err := client.UpsertTask(context.Background(), "projects/project-a", task)
	require.NoError(t, err)
	assert.Equal(t, task, upserted)

	synced, err := client.SyncTask(context.Background(), "projects/project-a/tasks/task-a")
	require.NoError(t, err)
	assert.Equal(t, task, synced)
}
