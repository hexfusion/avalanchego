#!/bin/bash

set -x

# TODO: error if !linux

export PKG_CONFIG_PATH="/usr/lib/pkgconfig"

# install deps
apt-get install -y \
  runc libsystemd-dev \
  libgpgme-dev libseccomp-dev \
  libbtrfs-dev libdevmapper-dev \
  build-essential btrfs-progs \
  crun go-md2man \
  iptables libassuan-dev \
  libbtrfs-dev libc6-dev \
  libglib2.0-dev libgpg-error-dev \
  libprotobuf-dev libprotobuf-c-dev \
  libseccomp-dev libselinux1-dev \
  libsystemd-dev pkg-config \
  uidmap catatonit

GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
BINDIR=${BINDIR:-/usr/local/bin}

# Podman requires conmon which monitors OCI runtimes
function install_conmon {
  local version=2.1.7
  local download_url="https://github.com/containers/conmon/releases/download/v${version}/conmon.${GOARCH}"
  local download_path=/tmp/conmon

  curl --fail -L ${download_url} -o ${download_path}  
  echo "installing conmon to ${BINDIR}"
  chmod +x ${download_path}
  mv ${download_path} ${BINDIR}/

  conmon --version
}

function install_podman {
  local version=4.5.1
  local github_url=https://github.com/containers/podman
  local download_path=/tmp/podman/

  git clone ${github_url} ${download_path}
  cd "${download_path}"
  git checkout v${version}
  make BUILDTAGS="selinux seccomp" PREFIX=/usr  
  make install PREFIX=/usr
  podman --version
}

install_conmon

install_podman

echo "install complete..."
echo "run the below as non root user"
echo "systemctl enable --user podman.socket"
echo "systemctl start --user podman.socket"
