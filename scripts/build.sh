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
  -manifest app.manifest versioninfo.json

# -trimpath keeps the build reproducible: without it the binary carries absolute
# paths from whichever machine built it, two builds of identical source hash
# differently, and the published checksum stops meaning anything verifiable.
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -trimpath -ldflags="-H windowsgui -s -w -X main.Version=${VERSION}" -o dist/CLIque.exe .

# The window asks for its icon by resource id (main.go's IconId), and nothing
# at runtime says so when it is missing: the window simply has no icon, which
# is only visible to someone looking at a taskbar. Windows shipped that way
# once already. Id 2, not 1: RT_MANIFEST claims id 1, which pushes the icon
# group to 2 — keep this in sync with main.go's IconId if that ever changes.
if ! command -v wrestool >/dev/null; then
  echo "need icoutils for the icon check: apt-get install icoutils" >&2
  exit 1
fi
# -l and grep, not -x: extracting a resource that is not there still exits 0.
wrestool -l dist/CLIque.exe 2>/dev/null | grep -q -- '--type=14 --name=2 --language' || {
  echo "no icon group 2 in dist/CLIque.exe, so the window would have no icon" >&2
  exit 1
}

sha256sum dist/CLIque.exe > dist/CLIque.exe.sha256

ls -lh dist/CLIque.exe | awk '{print "dist/CLIque.exe", $5}'
cat dist/CLIque.exe.sha256
