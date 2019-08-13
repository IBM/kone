// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"log"

	"github.com/ibm/kone/pkg/commands/options"
	"github.com/spf13/cobra"
)

// addPublish augments our CLI surface with publish.
func addPublish(topLevel *cobra.Command) {
	lo := &options.LocalOptions{}
	no := &options.NameOptions{}
	ta := &options.TagsOptions{}
	do := &options.DebugOptions{}

	publish := &cobra.Command{
		Use:   "publish PATH...",
		Short: "Build and publish container images from the given paths.",
		Long:  `This sub-command containeres the provided nodejs paths and publishes them.`,
		Example: `
  # Publish nodejs app to a Docker
  # Registry as:
  #   ${KO_DOCKER_REPO}/<package name>-<hash of package name>
  # When KO_DOCKER_REPO is ko.local, it is the same as if
  # --local were passed.
  ko publish ./my-app`,
		Args: cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			builder, err := makeBuilder(do)
			if err != nil {
				log.Fatalf("error creating builder: %v", err)
			}
			publisher, err := makePublisher(no, lo, ta)
			if err != nil {
				log.Fatalf("error creating publisher: %v", err)
			}
			images, err := publishImages(args, publisher, builder)
			if err != nil {
				log.Fatalf("failed to publish images: %v", err)
			}
			for _, img := range images {
				fmt.Println(img)
			}
		},
	}
	options.AddLocalArg(publish, lo)
	options.AddNamingArgs(publish, no)
	options.AddTagsArg(publish, ta)
	options.AddDebugArg(publish, do)
	topLevel.AddCommand(publish)
}
