#!/usr/bin/env bash
# Cross-compile the Windows client. Runs anywhere Go runs; no Windows toolchain,
# no cgo, no CI round-trip. That one-second loop is the reason this is Go.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p dist

VERSION="${1:-dev}"

go test ./...

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -ldflags="-H windowsgui -s -w -X main.Version=${VERSION}" -o dist/CLIque.exe .

sha256sum dist/CLIque.exe > dist/CLIque.exe.sha256

ls -lh dist/CLIque.exe | awk '{print "dist/CLIque.exe", $5}'
cat dist/CLIque.exe.sha256
