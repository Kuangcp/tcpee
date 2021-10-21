#!/bin/sh

TOOLCHAIN='x86_64-linux-musl'

CC="${TOOLCHAIN}-cc" LD="${TOOLCHAIN}-ldd" CGO_ENABLED=0 go build -trimpath -a -v -tags 'netgo osusergo static_build' -ldflags '-s -w -extldflags "-static"' -gcflags '-l=4' -o tcpee cmd/tcpee/main.go
