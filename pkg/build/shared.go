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
	"path/filepath"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// Caching wraps a builder implementation in a layer that shares build results
// for the same inputs using a simple "future" implementation.  Cached results
// may be invalidated by calling Invalidate with the same input passed to Build.
type Caching struct {
	inner Interface

	m       sync.Mutex
	results map[string]*future
}

// Caching implements Interface
var _ Interface = (*Caching)(nil)

// NewCaching wraps the provided build.Interface in an implementation that
// shares build results for a given path until the result has been invalidated.
func NewCaching(inner Interface) (*Caching, error) {
	return &Caching{
		inner:   inner,
		results: make(map[string]*future),
	}, nil
}

// Build implements Interface
func (c *Caching) Build(base, dir string) (v1.Image, error) {
	f := func() *future {
		// Lock the map of futures.
		c.m.Lock()
		defer c.m.Unlock()

		key := filepath.Join(base, dir)

		// If a future for "dir" exists, then return it.
		f, ok := c.results[key]
		if ok {
			return f
		}
		// Otherwise create and record a future for a Build of "ip".
		f = newFuture(func() (v1.Image, error) {
			return c.inner.Build(base, dir)
		})
		c.results[key] = f
		return f
	}()

	return f.Get()
}

// IsSupportedReference implements Interface
func (c *Caching) IsSupportedReference(base, dir string) *string {
	return c.inner.IsSupportedReference(base, dir)
}

// Invalidate removes a path's cached results.
func (c *Caching) Invalidate(base, dir string) {
	c.m.Lock()
	defer c.m.Unlock()

	key := filepath.Join(base, dir)
	delete(c.results, key)
}
