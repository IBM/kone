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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/viper"
)

var (
	defaultBaseImage name.Reference
)

type Package struct {
	Kone KoneOptions `json:"kone"`
}

type KoneOptions struct {
	DefaultBaseImage string `json:"defaultBaseImage"`
}

func getBaseImage(baseDir, s string) (v1.Image, error) {
	ref := defaultBaseImage

	appPath := filepath.Join(baseDir, s, "package.json")
	raw, err := ioutil.ReadFile(appPath)

	if err == nil {
		var pkg = Package{}
		err = json.Unmarshal(raw, &pkg)
		if err == nil || pkg.Kone.DefaultBaseImage != "" {
			newref, ok := name.ParseReference(pkg.Kone.DefaultBaseImage)
			if ok != nil {
				log.Printf("error parsing %q as image reference: %v", ref, ok)
			} else {
				ref = newref
			}
		}
	}

	log.Printf("Using base %s for %s", ref, s)
	return remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}

func getCreationTime() (*v1.Time, error) {
	epoch := os.Getenv("SOURCE_DATE_EPOCH")
	if epoch == "" {
		return nil, nil
	}

	seconds, err := strconv.ParseInt(epoch, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("the environment variable SOURCE_DATE_EPOCH should be the number of seconds since January 1st 1970, 00:00 UTC, got: %v", err)
	}
	return &v1.Time{time.Unix(seconds, 0)}, nil
}

func init() {
	// If omitted, use this base image.
	viper.SetDefault("defaultBaseImage", "node:lts-slim")
	viper.SetConfigName(".ko") // .yaml is implicit

	if override := os.Getenv("KO_CONFIG_PATH"); override != "" {
		viper.AddConfigPath(override)
	}

	viper.AddConfigPath("./")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("error reading config file: %v", err)
		}
	}

	ref := viper.GetString("defaultBaseImage")
	dbi, err := name.ParseReference(ref)
	if err != nil {
		log.Fatalf("'defaultBaseImage': error parsing %q as image reference: %v", ref, err)
	}
	defaultBaseImage = dbi
}
