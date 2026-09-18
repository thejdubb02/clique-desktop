#!/usr/bin/env bash
# Cross-compile the Windows client. Runs anywhere Go runs; no Windows toolchain,
# no cgo, no CI round-trip. That one-second loop is the reason this is Go.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p dist

VERSION="${1:-dev}"

go test ./...

# The icon and the file properties are a Windows resource, not something a
# Go build produces. Without this the taskbar shows the generic application
# icon and the SmartScreen warning has no name to show.
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 \
  -o resource_windows.syso -product-version "${VERSION}" -file-version "${VERSION}" \
  versioninfo.json

# -trimpath keeps the build reproducible: without it the binary carries absolute
# paths from whichever machine built it, two builds of identical source hash
# differently, and the published checksum stops meaning anything verifiable.
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -trimpath -ldflags="-H windowsgui -s -w -X main.Version=${VERSION}" -o dist/CLIque.exe .

sha256sum dist/CLIque.exe > dist/CLIque.exe.sha256

ls -lh dist/CLIque.exe | awk '{print "dist/CLIque.exe", $5}'
cat dist/CLIque.exe.sha256
