#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# Node e2e test requires google cloud sdk. 

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"/../..
cd ${ROOT}
. ${ROOT}/hack/versions

# ARTIFACTS is the directory to generate test result.
# RUN_ARGS is the extra args pass to node e2e test.
# TARBALL is the name of the release tar.
ARTIFACTS=${ARTIFACTS:-"/tmp/_artifacts/$(date +%y%m%dT%H%M%S)"}
RUN_ARGS=${RUN_ARGS:-""}
TARBALL=${TARBALL:-"cri-containerd.tar.gz"}

TARBALL_PATH=${ROOT}/_output/${TARBALL}
# TODO(Now): Need to be fixed
PROJECT=$(gcloud config list project --format 'value(core.project)')
PROJECT_HASH=$(echo -n "${PROJECT}" | md5sum | awk '{ print $1 }')
UPLOAD_PATH="cri-containerd-staging-${PROJECT_HASH}"

# Upload tar
if [ ! -e $TARBALL_PATH ]; then
  echo "release tar is built"
  exit 1
fi
if ! gsutil ls "gs://${UPLOAD_PATH}" >/dev/null; then
  gsutil mb "gs://${UPLOAD_PATH}"
fi
gsutil cp ${TARBALL_PATH} gs://${UPLOAD_PATH}
TARBALL_GCS_PATH=https://storage.googleapis.com/${UPLOAD_PATH}/${TARBALL}

# Get kubernetes
KUBERNETES_REPO="https://github.com/kubernetes/kubernetes"
KUBERNETES_PATH="${GOPATH}/src/k8s.io/kubernetes"
if [ ! -d "${KUBERNETES_PATH}" ]; then
  mkdir -p ${KUBERNETES_PATH}
  cd ${KUBERNETES_PATH}
  git clone ${KUBERNETES_REPO}
fi
cd ${KUBERNETES_PATH}
git fetch --all
git checkout ${KUBERNETES_VERSION}

# Run node e2e test
# TODO(random-liu): Add local support.
make test-e2e-node \
	REMOTE=true \
	RUNTIME=remote \
	CONTAINER_RUNTIME_ENDPOINT=/var/run/cri-containerd.sock \
	FOCUS=MirrorPod \
	IMAGES=ubuntu-gke-1604-xenial-v20170420-1 \
	IMAGE_PROJECT=ubuntu-os-gke-cloud \
	INSTANCE_METADATA="userdata<${ROOT}/test/e2e_node/init.yaml,configure-sh<${ROOT}/test/e2e_node/configure.sh,tarball=${TARBALL_GCS_PATH}"

#go run ./test/e2e_node/runner/remote/run_remote.go \
#	--logtostderr \
#	--vmodule=*=4 \
#	--ssh-env=gce \
#	--results-dir="${ARTIFACTS}" \
#	--image-config-file="${ROOT}/test/e2e_node/images.yaml" \
#	--project="${PROJECT}" \
#	--zone="us-central1-f" \
#	--test_args=--kubelet-flags="--cgroups-per-qos=true --cgroup-root=/ --container-runtime=remote --container-runtime-endpoint=" \
#	--ginkgo-flags="--nodes=8 --skip=\"\\[Flaky\\]|\\[Serial\\]\"" \
#	--"userdata<${ROOT}/test/e2e_node/init.yaml,configure-sh<${ROOT}/test/e2e_node/configure.sh,tarball=${TARBALL_GCS_PATH}" \
#	${RUN_ARGS}
