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

package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

const (
	appDir             = "/ko-app"
	defaultAppFilename = "main.js"
)

// GetBase takes an filepath and returns a base v1.Image.
type GetBase func(string) (v1.Image, error)
type builder func(string, string) (string, error)

type gobuild struct {
	getBase      GetBase
	creationTime v1.Time
	build        builder
}

// Option is a functional option for NewGo.
type Option func(*gobuildOpener) error

type gobuildOpener struct {
	getBase      GetBase
	creationTime v1.Time
	build        builder
}

func (gbo *gobuildOpener) Open() (Interface, error) {
	if gbo.getBase == nil {
		return nil, errors.New("a way of providing base images must be specified, see build.WithBaseImages")
	}
	return &gobuild{
		getBase:      gbo.getBase,
		creationTime: gbo.creationTime,
		build:        gbo.build,
	}, nil
}

// NewGo returns a build.Interface implementation that:
//  1. builds go binaries named by importpath,
//  2. containerizes the binary on a suitable base,
func NewGo(options ...Option) (Interface, error) {
	gbo := &gobuildOpener{
		build: build,
	}

	for _, option := range options {
		if err := option(gbo); err != nil {
			return nil, err
		}
	}
	return gbo.Open()
}

type Package struct {
	Name string `json:"name"`
}

// IsSupportedReference implements build.Interface
func (g *gobuild) IsSupportedReference(base, s string) *string {
	appPath := filepath.Join(base, s, "package.json")
	raw, err := ioutil.ReadFile(appPath)
	if err != nil {
		return nil
	}
	var pkg = Package{}
	err = json.Unmarshal(raw, &pkg)
	if err != nil || pkg.Name == "" {
		return nil
	}
	return &pkg.Name
}

func build(base, path string) (string, error) {
	file := filepath.Join(base, path, "package.json")
	return file, nil
}

func appFilename(path string) string {
	// For now assume main.js.
	return defaultAppFilename
}

func tarAddDirectories(tw *tar.Writer, dir string) error {
	if dir == "." || dir == string(filepath.Separator) {
		return nil
	}

	// Write parent directories first
	if err := tarAddDirectories(tw, filepath.Dir(dir)); err != nil {
		return err
	}

	// write the directory header to the tarball archive
	if err := tw.WriteHeader(&tar.Header{
		Name:     dir,
		Typeflag: tar.TypeDir,
		// Use a fixed Mode, so that this isn't sensitive to the directory and umask
		// under which it was created. Additionally, windows can only set 0222,
		// 0444, or 0666, none of which are executable.
		Mode: 0555,
	}); err != nil {
		return err
	}

	return nil
}

func (g *gobuild) tarNodeApp(name string) (*bytes.Buffer, error) {
	return g.tarData(name, appDir)
}

func (g *gobuild) kodataPath(s string) (string, error) {
	return filepath.Join(s, "kodata"), nil
}

// Where kodata lives in the image.
const kodataRoot = "/var/run/ko"

func (g *gobuild) tarKoData(name string) (*bytes.Buffer, error) {
	root, err := g.kodataPath(name)
	if err != nil {
		return nil, err
	}

	return g.tarData(root, kodataRoot)
}

func (g *gobuild) tarData(src, dst string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	// Compress this before calling tarball.LayerFromOpener, since it eagerly
	// calculates digests and diffids. This prevents us from double compressing
	// the layer when we have to actually upload the blob.
	//
	// https://github.com/google/go-containerregistry/issues/413
	gw, _ := gzip.NewWriterLevel(buf, gzip.BestSpeed)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if path == src {
			// Add an entry for dst
			return tw.WriteHeader(&tar.Header{
				Name:     dst,
				Typeflag: tar.TypeDir,
				// Use a fixed Mode, so that this isn't sensitive to the directory and umask
				// under which it was created. Additionally, windows can only set 0222,
				// 0444, or 0666, none of which are executable.
				Mode: 0555,
			})
		}
		if err != nil {
			return err
		}

		// Skip other directories.
		if info.Mode().IsDir() {
			return nil
		}

		// Chase symlinks.
		info, err = os.Stat(path)
		if err != nil {
			return err
		}

		// Open the file to copy it into the tarball.
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy the file into the image tarball.
		newPath := filepath.Join(dst, path[len(src):])

		if err := tw.WriteHeader(&tar.Header{
			Name:     newPath,
			Size:     info.Size(),
			Typeflag: tar.TypeReg,
			// Use a fixed Mode, so that this isn't sensitive to the directory and umask
			// under which it was created. Additionally, windows can only set 0222,
			// 0444, or 0666, none of which are executable.
			Mode: 0555,
		}); err != nil {
			return err
		}
		_, err = io.Copy(tw, file)
		return err
	})
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Build implements build.Interface
func (gb *gobuild) Build(basedir, s string) (v1.Image, error) {
	var layers []mutate.Addendum

	// target app path
	appPath := filepath.Join(appDir, appFilename(s))

	// Construct a tarball with the nodejs and produce a layer.
	nodeappLayerBuf, err := gb.tarNodeApp(filepath.Join(basedir, s))
	if err != nil {
		return nil, err
	}
	nodeappLayerBytes := nodeappLayerBuf.Bytes()
	nodeappLayer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(nodeappLayerBytes)), nil
	})
	if err != nil {
		return nil, err
	}
	layers = append(layers, mutate.Addendum{
		Layer: nodeappLayer,
		History: v1.History{
			Author:    "kone",
			CreatedBy: "kone publish " + s,
			Created:   v1.Time{Time: time.Now()},
		},
	})

	// Determine the appropriate base image for this filepath
	base, err := gb.getBase(s)
	if err != nil {
		return nil, err
	}

	// Augment the base image with our application layer.
	withApp, err := mutate.Append(base, layers...)
	if err != nil {
		return nil, err
	}

	// Start from a copy of the base image's config file, and set
	// the entrypoint to our app.
	cfg, err := withApp.ConfigFile()
	if err != nil {
		return nil, err
	}

	cfg = cfg.DeepCopy()
	cfg.Config.Entrypoint = []string{"node", appPath}
	cfg.Config.WorkingDir = appDir
	cfg.Config.Env = append(cfg.Config.Env, "KONE_DATA_PATH="+kodataRoot)
	cfg.ContainerConfig = cfg.Config
	cfg.Author = "github.com/ibm/kone"

	image, err := mutate.ConfigFile(withApp, cfg)
	if err != nil {
		return nil, err
	}

	empty := v1.Time{}
	if gb.creationTime != empty {
		return mutate.CreatedAt(image, gb.creationTime)
	}
	return image, nil
}
