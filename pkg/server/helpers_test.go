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
	"testing"

	"github.com/containerd/containerd/reference"
	"github.com/stretchr/testify/assert"

	"github.com/kubernetes-incubator/cri-containerd/pkg/metadata"
)

func TestGetSandbox(t *testing.T) {
	c := newTestCRIContainerdService()
	testID := "abcdefg"
	testSandbox := metadata.SandboxMetadata{
		ID:   testID,
		Name: "test-name",
	}
	assert.NoError(t, c.sandboxStore.Create(testSandbox))
	assert.NoError(t, c.sandboxIDIndex.Add(testID))

	for desc, test := range map[string]struct {
		id       string
		expected *metadata.SandboxMetadata
	}{
		"full id": {
			id:       testID,
			expected: &testSandbox,
		},
		"partial id": {
			id:       testID[:3],
			expected: &testSandbox,
		},
		"non-exist id": {
			id:       "gfedcba",
			expected: nil,
		},
	} {
		t.Logf("TestCase %q", desc)
		sb, err := c.getSandbox(test.id)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, sb)
	}
}

func TestNormalizeImageRef(t *testing.T) {
	for _, ref := range []string{
		"busybox",        // has nothing
		"busybox:latest", // only has tag
		"busybox@sha256:e6693c20186f837fc393390135d8a598a96a833917917789d63766cab6c59582", // only has digest
		"library/busybox",                  // only has path
		"docker.io/busybox",                // only has hostname
		"docker.io/library/busybox",        // has no tag
		"docker.io/busybox:latest",         // has no path
		"library/busybox:latest",           // has no hostname
		"docker.io/library/busybox:latest", // full reference
		"gcr.io/library/busybox",           // gcr reference
	} {
		t.Logf("TestCase %q", ref)
		normalized, err := normalizeImageRef(ref)
		assert.NoError(t, err)
		_, err = reference.Parse(normalized.String())
		assert.NoError(t, err, "%q should be containerd supported reference", normalized)
	}
}
