#!/bin/sh

CGO_ENABLED=0 go build -trimpath -v -tags 'netgo osusergo static_build kvformat' -ldflags '-s -w -extldflags "-static"' -gcflags '-l=4 -lbudget=1000' -o tcpee ./cmd/tcpee
