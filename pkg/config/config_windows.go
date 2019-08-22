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

package config

import (
	"github.com/containerd/containerd"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
)

// DefaultConfig returns default configurations of cri plugin.
func DefaultConfig() PluginConfig {
	return PluginConfig{
		// TODO(random-liu): Add CNI support for windows.
		CniConfig: CniConfig{
			NetworkPluginBinDir:       "",
			NetworkPluginConfDir:      "",
			NetworkPluginMaxConfNum:   0,
			NetworkPluginConfTemplate: "",
		},
		ContainerdConfig: ContainerdConfig{
			Snapshotter:        containerd.DefaultSnapshotter,
			DefaultRuntimeName: "runhcs",
			NoPivot:            false,
			Runtimes: map[string]Runtime{
				"runhcs": {
					Type: "io.containerd.runhcs.v1",
				},
			},
		},
		DisableTCPService:   true,
		StreamServerAddress: "127.0.0.1",
		StreamServerPort:    "0",
		StreamIdleTimeout:   streaming.DefaultConfig.StreamIdleTimeout.String(), // 4 hour
		EnableTLSStreaming:  false,
		X509KeyPairStreaming: X509KeyPairStreaming{
			TLSKeyFile:  "",
			TLSCertFile: "",
		},
		SandboxImage:            "e2eteam/pause:3.1", // TODO(random-liu): Should we use this image?
		StatsCollectPeriod:      10,
		MaxContainerLogLineSize: 16 * 1024,
		Registry: Registry{
			Mirrors: map[string]Mirror{
				"docker.io": {
					Endpoints: []string{"https://registry-1.docker.io"},
				},
			},
		},
		MaxConcurrentDownloads: 3,
	}
}
