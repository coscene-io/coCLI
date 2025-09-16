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

package registry

import (
	"context"
	"fmt"

	"github.com/coscene-io/cocli/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCreateCredentialCommand(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "create-credential",
		Short:                 "Generate a temporary docker credential",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			pm, err := config.Provide(*cfgPath).GetProfileManager()
			if err != nil {
				log.Fatalf("failed to load profile manager: %v", err)
			}

			cred, err := pm.ContainerRegistryCli().CreateBasicCredential(context.TODO())
			if err != nil {
				log.Fatalf("failed to create basic credential: %v", err)
			}

			fmt.Printf("username: %s\n", cred.GetUsername())
			fmt.Printf("password: %s\n", cred.GetPassword())
		},
	}

	return cmd
}
