#!/usr/bin/env bash

set -eu -o pipefail

docker rm -f sanic-buildkitd >/dev/null 2>&1 || true
docker run -d --name sanic-buildkitd --privileged -p 2149:2149 moby/buildkit:latest --addr tcp://0.0.0.0:2149

echo 'Execute the following:'
echo 'export BUILDKIT_HOST=tcp://0.0.0.0:2149'