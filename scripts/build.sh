#!/usr/bin/env bash
# Cross-compile the Windows client. Runs anywhere Go runs; no Windows toolchain,
# no cgo, no CI round-trip. That one-second loop is the reason this is Go.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p dist

go test ./...

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -ldflags="-H windowsgui -s -w" -o dist/CLIque.exe .

ls -lh dist/CLIque.exe | awk '{print "dist/CLIque.exe", $5}'
