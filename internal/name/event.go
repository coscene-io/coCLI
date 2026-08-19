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

package name

import (
	"fmt"

	"github.com/oriser/regroup"
	"github.com/pkg/errors"
)

type Event struct {
	ProjectID string
	ID        string
}

var eventNameRe = regroup.MustCompile(`^projects/(?P<project>[^/]+)/events/(?P<event>[^/]+)$`)

func NewEvent(event string) (*Event, error) {
	match, err := eventNameRe.Groups(event)
	if err != nil {
		return nil, errors.Wrap(err, "parse event name")
	}

	return &Event{ProjectID: match["project"], ID: match["event"]}, nil
}

func (e Event) Project() *Project {
	return &Project{ProjectID: e.ProjectID}
}

func (e Event) String() string {
	return fmt.Sprintf("projects/%s/events/%s", e.ProjectID, e.ID)
}
