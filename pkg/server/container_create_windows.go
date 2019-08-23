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

// No container mounts for windows.
func (c *criService) containerMounts(sandboxID string, config *runtime.ContainerConfig) []*runtime.Mount {
	return nil
}

func (c *criService) containerSpec(id string, sandboxID string, sandboxPid uint32, netNSPath string,
	config *runtime.ContainerConfig, sandboxConfig *runtime.PodSandboxConfig, imageConfig *imagespec.ImageConfig,
	extraMounts []*runtime.Mount, ociRuntime config.Runtime) (*runtimespec.Spec, error) {
	specOpts := []oci.SpecOpts{
		customopts.WithProcessArgs(config, imageConfig),
	}
	if config.GetWorkingDir() != "" {
		specOpts = append(specOpts, oci.WithProcessCwd(config.GetWorkingDir()))
	} else if imageConfig.WorkingDir != "" {
		specOpts = append(specOpts, oci.WithProcessCwd(imageConfig.WorkingDir))
	}

	if config.GetTty() {
		specOpts = append(specOpts, oci.WithTTY)
	}

	// Apply envs from image config first, so that envs from container config
	// can override them.
	env := imageConfig.Env
	for _, e := range config.GetEnvs() {
		env = append(env, e.GetKey()+"="+e.GetValue())
	}
	specOpts = append(specOpts, oci.WithEnv(env))

	specOpts = append(specOpts,
		// Clear the root location since runhcs sets it on the mount path in the guest.
		customopts.WithoutRoot,
		customopts.WithWindowsNetworkNamespace(netNSPath),
	)

	// TODO(windows): Windows mounts.
	specOpts = append(specOpts, customopts.WithMounts(c.os, config, extraMounts, mountLabel))

	// TODO(windows): resources, Username, Credential provider
	if c.config.DisableCgroup {
		specOpts = append(specOpts, customopts.WithDisabledCgroups)
	} else {
		specOpts = append(specOpts, customopts.WithResources(config.GetLinux().GetResources()))
		if sandboxConfig.GetLinux().GetCgroupParent() != "" {
			cgroupsPath := getCgroupsPath(sandboxConfig.GetLinux().GetCgroupParent(), id)
			specOpts = append(specOpts, oci.WithCgroup(cgroupsPath))
		}
	}

	supplementalGroups := securityContext.GetSupplementalGroups()

	for pKey, pValue := range getPassthroughAnnotations(sandboxConfig.Annotations,
		ociRuntime.PodAnnotations) {
		specOpts = append(specOpts, customopts.WithAnnotation(pKey, pValue))
	}

	specOpts = append(specOpts,
		customopts.WithOOMScoreAdj(config, c.config.RestrictOOMScoreAdj),
		customopts.WithPodNamespaces(securityContext, sandboxPid),
		customopts.WithSupplementalGroups(supplementalGroups),
		customopts.WithAnnotation(annotations.ContainerType, annotations.ContainerTypeContainer),
		customopts.WithAnnotation(annotations.SandboxID, sandboxID),
	)

	return runtimeSpec(id, specOpts...)
}
