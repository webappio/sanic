#!/usr/bin/env sh

go install --ldflags "-X main.version=$(git describe --abbrev=0 --tags)"
