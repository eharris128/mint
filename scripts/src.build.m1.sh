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
GOOS=darwin GOARCH=arm64 go build -mod=vendor -trimpath -ldflags="${LD_FLAGS}" -a -tags "remote containers_image_openpgp containers_image_docker_daemon_stub containers_image_fulcio_stub containers_image_rekor_stub" -o "${BDIR}/bin/mac_m1/mint"
popd

pushd ${BDIR}/cmd/mint-sensor
GOOS=linux GOARCH=arm64 go build -mod=vendor -trimpath -ldflags="${LD_FLAGS}" -a -tags "remote containers_image_openpgp containers_image_docker_daemon_stub containers_image_fulcio_stub containers_image_rekor_stub" -o "$BDIR/bin/linux_arm64/mint-sensor"
chmod a+x "$BDIR/bin/linux_arm64/mint-sensor"
popd

rm -rfv ${BDIR}/dist_mac_m1
mkdir ${BDIR}/dist_mac_m1
cp ${BDIR}/bin/mac_m1/mint ${BDIR}/dist_mac_m1/mint
cp ${BDIR}/bin/linux_arm64/mint-sensor ${BDIR}/dist_mac_m1/mint-sensor
pushd ${BDIR}/dist_mac_m1
ln -s mint docker-slim
ln -s mint slim
popd
pushd ${BDIR}

if hash zip 2> /dev/null; then
	zip -r dist_mac_m1.zip dist_mac_m1 -x "*.DS_Store"
fi

rm -rfv ${BDIR}/bin
