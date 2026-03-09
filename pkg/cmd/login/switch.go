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

package login

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coscene-io/cocli/internal/config"
	"github.com/coscene-io/cocli/internal/iostreams"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"golang.org/x/term"
)

var errProfileSelectionAborted = errors.New("profile selection aborted")

type profileSelectorFn func(profiles []*config.Profile, currentProfile string) (*config.Profile, error)
type headlessDetectorFn func() bool

func NewSwitchCommand(cfgPath *string, io *iostreams.IOStreams, getProvider func(string) config.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "switch",
		Short:                 "Switch to another login profile.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSwitch(*cfgPath, io, getProvider, promptForProfile, isHeadlessEnvironment)
		},
	}

	return cmd
}

func runSwitch(
	cfgPath string,
	io *iostreams.IOStreams,
	getProvider func(string) config.Provider,
	selectProfile profileSelectorFn,
	isHeadless headlessDetectorFn,
) error {
	cfg := getProvider(cfgPath)
	pm, err := cfg.GetProfileManager()
	if err != nil {
		return fmt.Errorf("failed to get profile manager: %w", err)
	}

	profiles := pm.GetProfiles()
	if len(profiles) == 0 {
		return fmt.Errorf("no login profiles found")
	}

	currentProfileName := ""
	if current := pm.GetCurrentProfile(); current != nil {
		currentProfileName = current.Name
	}

	profile, err := selectProfile(profiles, currentProfileName)
	if err != nil {
		if errors.Is(err, errProfileSelectionAborted) {
			io.Println("Profile switch cancelled.")
			return nil
		}
		if isHeadless() {
			return fmt.Errorf(
				"unable to prompt for profile selection in non-interactive mode, use `cocli login set -n <profile-name>` instead",
			)
		}
		return fmt.Errorf("failed to prompt for select profile: %w", err)
	}

	if err = pm.SwitchProfile(profile.Name); err != nil {
		return fmt.Errorf("failed to switch to profile %s: %w", profile.Name, err)
	}

	if err = cfg.Persist(pm); err != nil {
		return fmt.Errorf("failed to persist profile manager: %w", err)
	}

	curProfile := pm.GetCurrentProfile()
	io.Printf("Successfully switched to profile:\n%s\n", curProfile)
	return nil
}

type selectProfileModel struct {
	profiles   []*config.Profile
	initCursor int
	cursor     int
	selected   int
}

func (m selectProfileModel) Init() tea.Cmd {
	return nil
}

func (m selectProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.profiles)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectProfileModel) View() string {
	if m.selected >= 0 {
		return ""
	}
	var s string
	s += "Use the arrow keys to navigate, press enter to select a profile, and press q to quit.\n\n"
	for i, choice := range m.profiles {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		initIndicator := ""
		if m.initCursor == i {
			initIndicator = "(*)"
		}
		s += fmt.Sprintf("%s %s %s\n", cursor, choice.Name, initIndicator)
	}

	s += fmt.Sprintf("\n--------- Info ----------\n%s", m.profiles[m.cursor])

	return s
}

func promptForProfile(profiles []*config.Profile, currentProfile string) (*config.Profile, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found")
	}
	if len(profiles) == 1 {
		return profiles[0], nil
	}

	profileNames := lo.Map(profiles, func(p *config.Profile, _ int) string { return p.Name })
	cursor := slices.Index(profileNames, currentProfile)
	if cursor < 0 {
		cursor = 0
	}

	p := tea.NewProgram(selectProfileModel{
		profiles:   profiles,
		initCursor: cursor,
		cursor:     cursor,
		selected:   -1,
	})
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	selected := finalModel.(selectProfileModel).selected
	if selected < 0 {
		return nil, errProfileSelectionAborted
	}
	return profiles[selected], nil
}

func isHeadlessEnvironment() bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return true
	}
	if os.Getenv("CI") == "true" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	return false
}
