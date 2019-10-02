#!/usr/bin/env sh

set -eu -o pipefail

echo "Building..."
go install -v --ldflags "-X main.version=$(git describe --abbrev=0 --tags)" -mod=vendor
