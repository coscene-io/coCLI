// Copyright 2025 coScene
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

// Package apimocks provides mock implementations for API interfaces.
// It's in a separate package to avoid import cycles with the api package.
package apimocks

import (
	"context"
	"testing"

	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/stretchr/testify/require"
)

// MockProfileManager provides a test-friendly ProfileManager that doesn't require real authentication
type MockProfileManager struct {
	*config.ProfileManager
	recordCli  api.RecordInterface
	fileCli    api.FileInterface
	labelCli   api.LabelInterface
	projectCli api.ProjectInterface
	actionCli  api.ActionInterface
	taskCli    api.TaskInterface
	eventCli   api.EventInterface
}

// NewMockProfileManager creates a ProfileManager suitable for testing
func NewMockProfileManager(t *testing.T) *MockProfileManager {
	t.Helper()

	// Create a minimal valid profile that passes validation
	profile := &config.Profile{
		Name:        "test-profile",
		EndPoint:    "https://openapi.test.com",
		Token:       "test-token",
		ProjectSlug: "test-project",
	}

	pm := &config.ProfileManager{
		CurrentProfile: "test-profile",
		Profiles:       []*config.Profile{profile},
	}

	// Validate to ensure it's properly set up
	err := pm.Validate()
	require.NoError(t, err)

	return &MockProfileManager{
		ProfileManager: pm,
	}
}

// SetRecordClient sets a mock record client for testing
func (m *MockProfileManager) SetRecordClient(client api.RecordInterface) {
	m.recordCli = client
}

// SetFileClient sets a mock file client for testing
func (m *MockProfileManager) SetFileClient(client api.FileInterface) {
	m.fileCli = client
}

// SetLabelClient sets a mock label client for testing
func (m *MockProfileManager) SetLabelClient(client api.LabelInterface) {
	m.labelCli = client
}

// SetProjectClient sets a mock project client for testing
func (m *MockProfileManager) SetProjectClient(client api.ProjectInterface) {
	m.projectCli = client
}

// Override methods to return mock clients
func (m *MockProfileManager) RecordCli() api.RecordInterface {
	if m.recordCli != nil {
		return m.recordCli
	}
	// Return a no-op implementation if not set
	return nil
}

func (m *MockProfileManager) FileCli() api.FileInterface {
	if m.fileCli != nil {
		return m.fileCli
	}
	return nil
}

func (m *MockProfileManager) LabelCli() api.LabelInterface {
	if m.labelCli != nil {
		return m.labelCli
	}
	return nil
}

func (m *MockProfileManager) ProjectCli() api.ProjectInterface {
	if m.projectCli != nil {
		return m.projectCli
	}
	return nil
}

func (m *MockProfileManager) ActionCli() api.ActionInterface {
	if m.actionCli != nil {
		return m.actionCli
	}
	return nil
}

func (m *MockProfileManager) TaskCli() api.TaskInterface {
	if m.taskCli != nil {
		return m.taskCli
	}
	return nil
}

func (m *MockProfileManager) EventCli() api.EventInterface {
	if m.eventCli != nil {
		return m.eventCli
	}
	return nil
}

// ProjectName returns a test project name without requiring real API calls
func (m *MockProfileManager) ProjectName(ctx context.Context, projectSlug string) (*name.Project, error) {
	if projectSlug == "" {
		projectSlug = "test-project"
	}
	return &name.Project{
		ProjectID: projectSlug,
	}, nil
}

// GetRecordUrl returns a test URL
func (m *MockProfileManager) GetRecordUrl(ctx context.Context, recordName *name.Record) (string, error) {
	return "https://test.coscene.io/records/" + recordName.RecordID, nil
}

// CheckAuth always returns true for tests
func (m *MockProfileManager) CheckAuth() bool {
	return true
}

// Auth does nothing for tests
func (m *MockProfileManager) Auth(ctx context.Context) error {
	return nil
}

// GetProjectUrl returns a test URL
func (m *MockProfileManager) GetProjectUrl(ctx context.Context, projectName *name.Project) (string, error) {
	return "https://test.coscene.io/projects/" + projectName.ProjectID, nil
}

// MockProvider implements config.Provider for testing
type MockProvider struct {
	profileManager *MockProfileManager
}

// NewMockProvider creates a test provider with a mock ProfileManager
func NewMockProvider(t *testing.T) *MockProvider {
	return &MockProvider{
		profileManager: NewMockProfileManager(t),
	}
}

// GetProfileManager returns the mock ProfileManager
func (p *MockProvider) GetProfileManager() (*config.ProfileManager, error) {
	return p.profileManager.ProfileManager, nil
}

// Persist is a no-op for tests
func (p *MockProvider) Persist(pm *config.ProfileManager) error {
	return nil
}

// ProfileManager returns the mock ProfileManager (for test setup)
func (p *MockProvider) ProfileManager() *MockProfileManager {
	return p.profileManager
}

// SetProfileManager sets a custom ProfileManager for testing
func (p *MockProvider) SetProfileManager(pm *MockProfileManager) {
	p.profileManager = pm
}
