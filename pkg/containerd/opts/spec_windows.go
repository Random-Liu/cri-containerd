// +build windows

/*
Copyright The containerd Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package opts

import (
	"context"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

// WithWindowsNetworkNamespace sets windows network namespace for container.
// TODO(random-liu): Move this into container/containerd.
func WithWindowsNetworkNamespace(path string) oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, c *containers.Container, s *runtimespec.Spec) error {
		if s.Windows == nil {
			s.Windows = &runtimespec.Windows{}
		}
		if s.Windows.Network == nil {
			s.Windows.Network = &runtimespec.WindowsNetwork{}
		}
		s.Windows.Network.NetworkNamespace = path
		return nil
	}
}
