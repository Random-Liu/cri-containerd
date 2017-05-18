/*
Copyright 2017 The Kubernetes Authors.

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

	"github.com/containerd/containerd/images"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/kubelet/api/v1alpha1/runtime"

	"github.com/kubernetes-incubator/cri-containerd/pkg/metadata"
)

// RemoveImage removes the image.
// TODO(mikebrow): harden api
// TODO(random-liu): Update CRI to pass image reference instead of ImageSpec.
// TODO(random-liu): We should change CRI to distinguish image id and image spec.
// Remove the whole image no matter the it's image id or reference. This is the
// semantic defined in CRI now.
func (c *criContainerdService) RemoveImage(ctx context.Context, r *runtime.RemoveImageRequest) (retRes *runtime.RemoveImageResponse, retErr error) {
	glog.V(2).Infof("RemoveImage %q", r.GetImage().GetImage())
	defer func() {
		if retErr == nil {
			glog.V(2).Infof("RemoveImage %q returns successfully", r.GetImage().GetImage())
		}
	}()
	imageID, err := c.localResolve(ctx, r.GetImage().GetImage())
	glog.V(0).Infof("Image ref to remove %q", imageID)
	if err != nil {
		return nil, fmt.Errorf("can not resolve %q locally: %v", r.GetImage().GetImage(), err)
	}
	if imageID == "" {
		// return empty without error when image not found.
		return &runtime.RemoveImageResponse{}, nil
	}
	meta, err := c.imageMetadataStore.Get(imageID)
	if err != nil {
		if metadata.IsNotExistError(err) {
			return &runtime.RemoveImageResponse{}, nil
		}
		return nil, fmt.Errorf("an error occurred when get image %q metadata: %v", imageID, err)
	}
	// Also include repo digest, because if user pull image with digest,
	// there will also be a corresponding repo digest reference.
	for _, ref := range append(meta.RepoTags, meta.RepoDigests...) {
		// TODO(random-liu): Containerd should schedule a garbage collection immediately,
		// and we may want to wait for the garbage collection to be over here.
		err = c.imageStoreService.Delete(ctx, ref)
		if err == nil || images.IsNotFound(err) {
			continue
		}
		return nil, fmt.Errorf("failed to delete image reference %q for image %q: %v", ref, imageID, err)
	}
	err = c.imageMetadataStore.Delete(imageID)
	if err != nil {
		if metadata.IsNotExistError(err) {
			return &runtime.RemoveImageResponse{}, nil
		}
		return nil, fmt.Errorf("an error occurred when delete image %q matadata: %v", imageID, err)
	}
	return &runtime.RemoveImageResponse{}, nil
}
