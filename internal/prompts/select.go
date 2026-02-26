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
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coscene-io/cocli/internal/iostreams"
	log "github.com/sirupsen/logrus"
)

type selectModel struct {
	prompt  string
	items   []string
	cursor  int
	chosen  int
	quit    bool
	io      *iostreams.IOStreams
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp, tea.KeyShiftTab:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown, tea.KeyTab:
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			m.chosen = m.cursor
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEscape:
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	s := m.prompt + "\n\n"
	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		s += fmt.Sprintf("%s%s\n", cursor, item)
	}
	s += "\nUse arrow keys to navigate, enter to select, esc to cancel.\n"
	return s
}

// PromptSelect presents a list of items and returns the selected index.
// Returns -1 if the user cancels.
func PromptSelect(prompt string, items []string, io *iostreams.IOStreams) int {
	p := tea.NewProgram(selectModel{prompt: prompt, items: items, chosen: -1, io: io})
	finalModel, err := p.Run()
	if err != nil {
		log.Fatalf("Error running select prompt: %v", err)
	}
	m := finalModel.(selectModel)
	if m.quit {
		os.Exit(1)
	}
	return m.chosen
}
