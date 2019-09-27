#!/usr/bin/env bash

set -eu -o pipefail

if [ -z "${GIT_COMMIT:-}" ]; then
	export GIT_COMMIT="$(git rev-parse HEAD)"
fi

if [ -z "${GIT_TAG:-}" ]; then
	if git describe --abbrev=0 >/dev/null; then
		export GIT_TAG="$(git describe --abbrev=0)"
	else
		export GIT_TAG="master@${GIT_COMMIT}"
	fi
fi
echo "Building..."
go install -v --ldflags "-X main.version=${GIT_TAG}" -mod=vendor
