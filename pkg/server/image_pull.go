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
	"encoding/json"
	"fmt"

	"github.com/containerd/containerd/content"
	containerdimages "github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/reference"
	"github.com/golang/glog"
	imagedigest "github.com/opencontainers/go-digest"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/kubelet/api/v1alpha1/runtime"

	"github.com/kubernetes-incubator/cri-containerd/pkg/metadata"
)

// PullImage pulls an image with authentication config.
// TODO(mikebrow): add authentication
// TODO(mikebrow): harden api (including figuring out at what layer we should be blocking on duplicate requests.)
// TODO(random-liu): Handle cleanup when failed to pull image.
// TODO(random-liu): Synchronize with RemoveImage when we actually remove image.
func (c *criContainerdService) PullImage(ctx context.Context, r *runtime.PullImageRequest) (retRes *runtime.PullImageResponse, retErr error) {
	glog.V(2).Infof("PullImage %q with auth config %+v", r.GetImage().GetImage(), r.GetAuth())
	defer func() {
		if retErr == nil {
			glog.V(2).Infof("PullImage returns image reference %q", retRes.GetImageRef())
		}
	}()

	namedRef, err := normalizeImageRef(r.GetImage().GetImage())
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference %q: %v", r.GetImage().GetImage(), err)
	}
	image := namedRef.String()
	if r.GetImage().GetImage() != image {
		glog.V(4).Info("PullImage using normalized image ref: %q", image)
	}

	// TODO(random-liu): [P1] Schema 1 image is not supported in containerd now, we need to support
	// it for backward compatiblity.
	chainID, digest, size, err := c.pullImage(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image %q: %v", image, err)
	}
	glog.V(4).Info("Pulled image %q with chainID %q, digest %q, size %d", image, chainID, digset, size)

	repoTag, repoDigest := getRepoTagAndDigest(namedRef, digest)

	// TODO(mikebrow): add truncIndex for image id
	// Use the chainID as image id to make sure image id is unique for the same image content.
	// Note that for the same image content, digest may be different.
	meta, err := c.imageMetadataStore.Get(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image %q metadata: %v", chainID, err)
	}
	// There is a known race here because the image metadata could be created after `Get`.
	// TODO(random-liu): [P0] Add CreateOrUpdate operation in metadata store to create or update in one
	// transaction.
	if meta == nil {
		// Create corresponding image metadata.
		newMeta := &metadata.ImageMetadata{
			ID:   chainID,
			Size: uint64(size),
		}
		updateImageMetadata(newMeta, repoTag, repoDigest)
		if err = c.imageMetadataStore.Create(newMeta); err != nil {
			return nil, fmt.Errorf("failed to create image %q metadata: %v", chainID, err)
		}
	} else {
		// Update existing image metadata.
		if err := c.imageMetadataStore.Update(imageID, func(m metadata.ImageMetadata) (metadata.ImageMetadata, error) {
			updateImageMetadata(&m, repoTag, repoDigest)
			return m, nil
		}); err != nil {
			return nil, fmt.Errorf("failed to update image %q metadata: %v", chainID, err)
		}
	}

	return &runtime.PullImageResponse{ImageRef: chahinID}, err
}

// pullImage pulls image and returns image chainID, digest and compressed size.
func (c *criContainerdService) pullImage(ctx context.Context, ref string) (imagedigest.Digest, imagedigest.Digest, int64, error) {
	// Resolve the image reference to get descriptor and fetcher.
	resolver := docker.NewResolver()
	resolvedImageName, desc, fetcher, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to resolve ref %q: %v", ref, err)
	}
	// Currently, resolvedImageName is the same with ref in docker resolver,
	// but they may be different in the future.
	// TODO(random-liu): Store resolvedImageName in the metadata store if they are
	// different in the future, because we need it to delete corresponding image store
	// entry.

	// Put the image information into containerd image store.
	if err := c.imageStoreService.Put(ctx, resolvedImageName, desc); err != nil {
		return "", "", 0, fmt.Errorf("failed to put image %q desc %v into containerd image store: %v",
			resolvedImageName, desc, err)
	}

	// Fetch all image resources into content store.
	// Dispatch a handler for a sequence of handlers which:
	// 1) fetch the object using a FetchHandler;
	// 2) recurse through any sub-layers via a ChildrenHandler.
	err = containerdimages.Dispatch(
		ctx,
		containerdimages.Handlers(
			remotes.FetchHandler(c.contentIngester, fetcher),
			containerdimages.ChildrenHandler(c.contentProvider)),
		desc)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to fetch image %q desc %+v: %v", ref, desc, err)
	}

	// Read the image manifest from content store.
	image, err := c.imageStoreService.Get(ctx, resolvedImageName)
	if err != nil {
		return "", "", 0,
			fmt.Errorf("failed to get image %q from containerd image store: %v", resolvedImageName, err)
	}
	digest := image.Target.Digest
	p, err := content.ReadBlob(ctx, c.contentProvider, digest)
	if err != nil {
		return "", "", 0,
			fmt.Errorf("readblob failed for digest %q: %v", digest, err)
	}
	var manifest imagespec.Manifest
	if err := json.Unmarshal(p, &manifest); err != nil {
		return "", "", 0,
			fmt.Errorf("unmarshal blob to manifest failed for digest %q: %v", digest, err)
	}

	// Unpack the image layers into snapshots.
	chainID, err := c.rootfsUnpacker.Unpack(ctx, manifest.Layers)
	if err != nil {
		return "", "", 0,
			fmt.Errorf("unpack failed for manifest layers %v: %v", manifest.Layers, err)
	}
	// TODO(random-liu): Considering how to deal with content disk usage.

	// TODO(random-liu): Get uncompressed size from snapshot service.
	// Get compressed image size.
	size, err = image.Size(ctx, c.contentProvider)
	if err != nil {
		return "", "", 0,
			fmt.Errorf("size failed for image %q: %v", ref, err)
	}
	return chainID, digest, size, nil
}

// updateImageMetadata updates existing image meta with new repoTag and repoDigest.
func updateImageMetadata(meta *metadata.ImageMetadata, repoTag, repoDigest string) {
	if repoTag != "" {
		meta.RepoTags = insertToStringSlice(meta.RepoTags, repoTag)
	}
	if repoDigest != "" {
		meta.RepoDigests = insertToStringSlice(meta.RepoDigests, repoDigest)
	}
}

// getRepoTagAndDigest returns image repoTag and repoDigest of the named image reference.
// Note that repoTag could be empty string if the image reference is not tagged, but repoDigest
// must not be empty.
func getRepoTagAndDigest(namedRef reference.Named, digest imagedigest.Digest) (string, string) {
	var repoTag string
	namedTagged, ok := namedRef.(reference.NamedTagged)
	if ok {
		repoTag = namedTagged.Name() + ":" + namedTagged.Tag()
	}
	repoDigest := namedRef.Name() + "@" + digest.String()
	return repoTag, repoDigest
}
