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

package options

import (
	"github.com/ibm/kone/pkg/publish"
	"github.com/spf13/cobra"
)

// NameOptions represents options for the kone binary.
type NameOptions struct {
	// PreservePackageName preserves the package name after KO_DOCKER_REPO, whithout MD5 hash
	// PreservePackageName bool
}

func AddNamingArgs(cmd *cobra.Command, no *NameOptions) {
	//cmd.Flags().BoolVarP(&no.PreservePackageName, "preserve-package-name", "P", no.PreservePackageName,
	//	"Whether to preserve the package name after KO_DOCKER_REPO.")
}

// func packageWithMD5(packageName string) string {
// 	hasher := md5.New()
// 	hasher.Write([]byte(packageName))
// 	return filepath.Base(packageName) + "-" + hex.EncodeToString(hasher.Sum(nil))
// }

func preservePackageName(packageName string) string {
	return packageName
}

func MakeNamer(no *NameOptions) publish.Namer {
	//	if no.PreservePackageName {
	return preservePackageName
	// }
	//	return packageWithMD5
}
