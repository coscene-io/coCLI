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

package prompts

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectModel(t *testing.T) {
	model := selectModel{prompt: "Pick one", items: []string{"first", "second"}, chosen: -1}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(selectModel)
	require.Nil(t, cmd)
	assert.Equal(t, 1, model.cursor)
	assert.Contains(t, model.View(), "> second")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(selectModel)
	assert.Equal(t, 1, model.chosen)
	assert.NotNil(t, cmd)

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(selectModel)
	assert.True(t, model.quit)
	assert.NotNil(t, cmd)
}

func TestStringModel(t *testing.T) {
	var out bytes.Buffer
	model := stringModel{
		promptMsg:    "Name",
		defaultValue: "default",
		windowWidth:  80,
		io:           iostreams.Test(nil, &out, &out),
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("yz")})
	model = updated.(stringModel)
	require.Nil(t, cmd)
	assert.Equal(t, "yz", model.enteredString)
	assert.Contains(t, model.View(), "yz")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(stringModel)
	assert.Equal(t, "y", model.enteredString)

	model.enteredString = ""
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(stringModel)
	assert.Equal(t, "default", model.enteredString)
	assert.NotNil(t, cmd)

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model = updated.(stringModel)
	assert.True(t, model.quit)
	assert.NotNil(t, cmd)
	assert.Contains(t, out.String(), "Quitting")
}

func TestYNModel(t *testing.T) {
	var out bytes.Buffer
	model := ynModel{promptMsg: "Continue", windowWidth: 80, io: iostreams.Test(nil, &out, &out)}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(ynModel)
	assert.True(t, model.confirmed)
	assert.Equal(t, "y", model.enteredKey)
	assert.NotNil(t, cmd)
	assert.Contains(t, model.View(), "Continue")

	model = ynModel{promptMsg: "Continue", windowWidth: 80, io: iostreams.Test(nil, &out, &out)}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = updated.(ynModel)
	assert.False(t, model.confirmed)
	assert.Equal(t, "n", model.enteredKey)
	assert.NotNil(t, cmd)

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model = updated.(ynModel)
	assert.True(t, model.quit)
	assert.NotNil(t, cmd)
}
