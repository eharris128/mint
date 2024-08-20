#!/usr/bin/env bash

set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
BDIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

export CGO_ENABLED=0

pushd $BDIR

BUILD_TIME="$(date -u '+%Y-%m-%d_%I:%M:%S%p')"
TAG="current"
REVISION="current"
if hash git 2>/dev/null && [ -e $BDIR/.git ]; then
  TAG="$(git describe --tags --always)"
  REVISION="$(git rev-parse HEAD)"
fi

LD_FLAGS="-s -w -X github.com/mintoolkit/mint/pkg/version.appVersionTag=${TAG} -X github.com/mintoolkit/mint/pkg/version.appVersionRev=${REVISION} -X github.com/mintoolkit/mint/pkg/version.appVersionTime=${BUILD_TIME}"

go generate github.com/mintoolkit/mint/pkg/appbom

pushd ${BDIR}/cmd/mint
GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags="${LD_FLAGS}" -tags "remote containers_image_openpgp containers_image_docker_daemon_stub containers_image_fulcio_stub containers_image_rekor_stub" -o "${BDIR}/bin/linux/mint"
popd

pushd ${BDIR}/cmd/mint-sensor
GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags="${LD_FLAGS}" -tags "remote containers_image_openpgp containers_image_docker_daemon_stub containers_image_fulcio_stub containers_image_rekor_stub" -o "${BDIR}/bin/linux/mint-sensor"
chmod a+x "${BDIR}/bin/linux/mint-sensor"
popd

rm -rfv ${BDIR}/dist_linux
mkdir ${BDIR}/dist_linux
cp ${BDIR}/bin/linux/mint ${BDIR}/dist_linux/mint
cp ${BDIR}/bin/linux/mint-sensor ${BDIR}/dist_linux/mint-sensor
pushd ${BDIR}/dist_linux
ln -s mint docker-slim
ln -s mint slim
popd
pushd ${BDIR}
tar -czvf dist_linux.tar.gz dist_linux
popd

rm -rfv ${BDIR}/bin
