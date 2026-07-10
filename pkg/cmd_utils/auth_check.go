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

package cmd_utils

import (
	"strings"

	"github.com/spf13/cobra"
)

const (
	skipAuthCheckAnnotation          = "skipAuthCheck"
	skipAuthCheckBoolFlagsAnnotation = "skipAuthCheckBoolFlags"
)

func DisableAuthCheck(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}

	cmd.Annotations[skipAuthCheckAnnotation] = "true"
}

func DisableAuthCheckForBoolFlags(cmd *cobra.Command, flags ...string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}

	cmd.Annotations[skipAuthCheckBoolFlagsAnnotation] = strings.Join(flags, ",")
}

func IsAuthCheckEnabled(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return false
	}

	for c := cmd; c.Parent() != nil; c = c.Parent() {
		if c.Annotations == nil {
			continue
		}
		if c.Annotations[skipAuthCheckAnnotation] == "true" {
			return false
		}
		for _, flag := range strings.Split(c.Annotations[skipAuthCheckBoolFlagsAnnotation], ",") {
			flag = strings.TrimSpace(flag)
			if flag == "" {
				continue
			}
			enabled, err := c.Flags().GetBool(flag)
			if err == nil && enabled {
				return false
			}
		}
	}

	return true
}
