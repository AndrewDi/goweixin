#!/bin/bash -e

git_version=$(git describe --dirty)
 if [[ $git_version == *-dirty ]] ; then
  echo 'Working tree is dirty.'
  exit 1
fi

PROGRAM_NAME=goweixin
PROJECT_PKG="github.com/AndrewDi/goweixin"
PROGRAM_PKG="${PROJECT_PKG}"

export LDFLAGS="-w -X ${PROGRAM_PKG}/version.gitVersion=${git_version}"
export BUILD_OS="${BUILD_OS:-darwin linux windows freebsd}"
export BUILD_ARCH="${BUILD_ARCH:-386 amd64}"

set -x
export GO111MODULE=on
if [[ ! -d "bin" ]];then
  mkdir bin
fi
make -C "$(go env GOPATH)/src/${PROGRAM_PKG}" -j build-all
set +x
