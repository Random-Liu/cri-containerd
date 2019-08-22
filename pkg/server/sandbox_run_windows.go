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

package server

import (
	"fmt"
	"os"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/plugin"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"github.com/containerd/cri/pkg/annotations"
	criconfig "github.com/containerd/cri/pkg/config"
	customopts "github.com/containerd/cri/pkg/containerd/opts"
	osinterface "github.com/containerd/cri/pkg/os"
)

func (c *criService) generateSandboxContainerSpec(id string, config *runtime.PodSandboxConfig,
	imageConfig *imagespec.ImageConfig, nsPath string, runtimePodAnnotations []string) (*runtimespec.Spec, error) {
	// Creates a spec Generator with the default spec.
	specOpts := []oci.SpecOpts{
		oci.WithEnv(imageConfig.Env),
		oci.WithHostname(config.GetHostname()),
	}
	if imageConfig.WorkingDir != "" {
		specOpts = append(specOpts, oci.WithProcessCwd(imageConfig.WorkingDir))
	}

	if len(imageConfig.Entrypoint) == 0 && len(imageConfig.Cmd) == 0 {
		// Pause image must have entrypoint or cmd.
		return nil, errors.Errorf("invalid empty entrypoint and cmd in image config %+v", imageConfig)
	}
	specOpts = append(specOpts, oci.WithProcessArgs(append(imageConfig.Entrypoint, imageConfig.Cmd...)...))

	specOpts = append(specOpts,
		// Clear the root location since runhcs sets it on the mount path in the guest.
		customopts.WithoutRoot,
		customopts.WithWindowsNetworkNamespace(nsPath),
	)

	for pKey, pValue := range getPassthroughAnnotations(config.Annotations,
		runtimePodAnnotations) {
		specOpts = append(specOpts, customopts.WithAnnotation(pKey, pValue))
	}

	specOpts = append(specOpts,
		customopts.WithAnnotation(annotations.ContainerType, annotations.ContainerTypeSandbox),
		customopts.WithAnnotation(annotations.SandboxID, id),
		customopts.WithAnnotation(annotations.SandboxLogDir, config.GetLogDirectory()),
	)

	return runtimeSpec(id, specOpts...)
}

// No sandbox files needed for windows.
func (c *criService) setupSandboxFiles(id string, config *runtime.PodSandboxConfig) error {
	return nil
}

// No sandbox files needed for windows.
func (c *criService) cleanupSandboxFiles(id string, config *runtime.PodSandboxConfig) error {
	return nil
}

// No sandbox task options needed for windows.
func (c *criService) sandboxTaskOpts(r criconfig.Runtime) []containerd.NewTaskOpts {
	return nil
}
