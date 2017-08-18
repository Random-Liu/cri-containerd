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

# Dependencies:
# runc:
# - libseccomp-dev(Ubuntu,Debian)/libseccomp-devel(Fedora, CentOS, RHEL). Note that
# libseccomp in ubuntu <=trusty and debian <=jessie is not new enough, backport
# is required.
# - libapparmor-dev(Ubuntu,Debian)/libapparmor-devel(Fedora, CentOS, RHEL)
# containerd:
# - btrfs-tools(Ubuntu,Debian)/btrfs-progs-devel(Fedora, CentOS, RHEL)

set -o errexit
set -o nounset
set -o pipefail

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"/..
. ${ROOT}/hack/versions

# DESTDIR is the dest path to install dependencies.
DESTDIR=${DESTDIR:-"/"}
# Convert to absolute path if it's relative.
if [[ ${DESTDIR} != /* ]]; then
	DESTDIR=${ROOT}/${DESTDIR}
fi

CONTAINERD_DIR=${DESTDIR}/usr/local
RUNC_DIR=${DESTDIR}
CNI_DIR=${DESTDIR}/opt/cni
CNI_CONFIG_DIR=${DESTDIR}/etc/cni/net.d

RUNC_PKG=github.com/opencontainers/runc
CNI_PKG=github.com/containernetworking/cni
CONTAINERD_PKG=github.com/containerd/containerd

# Install runc
go get -d ${RUNC_PKG}/...
cd ${GOPATH}/src/${RUNC_PKG}
git fetch --all
git checkout ${RUNC_VERSION}
make
sudo make install -e DESTDIR=${RUNC_DIR}

# Install cni
go get -d ${CNI_PKG}/...
cd ${GOPATH}/src/${CNI_PKG}
git fetch --all
git checkout ${CNI_VERSION}
./build
sudo mkdir -p ${CNI_DIR}
sudo cp -r ./bin ${CNI_DIR}
sudo mkdir -p ${CNI_CONFIG_DIR}
sudo bash -c 'cat >'${CNI_CONFIG_DIR}'/10-containerd-bridge.conf <<EOF
{
	"cniVersion": "0.2.0",
	"name": "containerd-bridge",
	"type": "bridge",
	"bridge": "cni0",
	"isGateway": true,
	"ipMasq": true,
	"ipam": {
		"type": "host-local",
		"subnet": "10.88.0.0/16",
		"routes": [
			{ "dst": "0.0.0.0/0" }
		]
	}
}
EOF'
sudo bash -c 'cat >'${CNI_CONFIG_DIR}'/99-loopback.conf <<EOF
{
	"cniVersion": "0.2.0",
	"type": "loopback"
}
EOF'

# Install containerd
go get -d ${CONTAINERD_PKG}/...
cd ${GOPATH}/src/${CONTAINERD_PKG}
git fetch --all
git checkout ${CONTAINERD_VERSION}
make
sudo make install -e DESTDIR=${CONTAINERD_DIR}
