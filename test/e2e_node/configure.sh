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

# Get tarball gcs path from the `tarball` metadata item.
TARBALL_GCS_PATH=$(curl --fail --retry 5 --retry-delay 3 --silent --show-error \
	-H "X-Google-Metadata-Request: True" \
	http://metadata.google.internal/computeMetadata/v1/instance/attributes/tarball)
TARBALL="cri-containerd.tar.gz"

# Download and untar the release tar ball.
curl -f --ipv4 -Lo "${TARBALL}" --connect-timeout 20 --max-time 300 --retry 6 --retry-delay 10 "${TARBALL_GCS_PATH}"
tar xvf "${TARBALL}"

# Add binary path into PATH.
echo "PATH=${PWD}/usr/local/bin:${PWD}/usr/local/sbin:\${PATH}" > /etc/profile.d/cri-containerd.sh

IS_GCI=$(cat /etc/os-release | grep "ID=cos" || true)
if [ -z "${IS_GCI}" ]; then
	exit
fi
# GCI specfic setup.
CONTAINERIZED_MOUNTER_HOME="/home/kubernetes/containerized_mounter"

mount /tmp /tmp -o remount,exec,suid
mkdir -p ${CONTAINERIZED_MOUNTER_HOME}/rootfs
mount --bind ${CONTAINERIZED_MOUNTER_HOME}/ ${CONTAINERIZED_MOUNTER_HOME}/
mount -o remount, exec ${CONTAINERIZED_MOUNTER_HOME}/
wget https://storage.googleapis.com/kubernetes-release/gci-mounter/mounter.tar -O /tmp/mounter.tar
tar xvf /tmp/mounter.tar -C ${CONTAINERIZED_MOUNTER_HOME}/rootfs
mkdir -p ${CONTAINERIZED_MOUNTER_HOME}/rootfs/var/lib/kubelet
mount --rbind /var/lib/kubelet ${CONTAINERIZED_MOUNTER_HOME}/rootfs/var/lib/kubelet
mount --make-rshared ${CONTAINERIZED_MOUNTER_HOME}/rootfs/var/lib/kubelet
mount --bind /proc ${CONTAINERIZED_MOUNTER_HOME}/rootfs/proc
mount --bind /dev ${CONTAINERIZED_MOUNTER_HOME}/rootfs/dev
