#!/usr/bin/env bash
# Build, sign and package the Windows distribution: the loose CLIque.exe (what
# the in-app updater downloads) and CLIque-Setup.exe (an NSIS installer for a
# first install, with a Start Menu entry and a real uninstaller). Cross-built
# from Linux, no Windows machine, no paid tooling: NSIS's makensis and
# osslsigncode are both free and both apt packages.
#
#   scripts/package.sh 0.3.14
#
# Signing certificate/key: generated once with openssl, self-signed (no CA, so
# no cost, and no renewal to track — see CLAUDE.md). Windows still runs its
# own separate SmartScreen reputation check on a first download regardless of
# signing, which a self-signed cert cannot clear; that is expected, and it is
# a one-time click on first install only, never on an update.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:?usage: package.sh <version>}"

CERT="${CLIQUE_SIGNING_CERT:-/etc/clique-desktop/signing/codesign.pem}"
KEY="${CLIQUE_SIGNING_KEY:-/etc/clique-desktop/signing/codesign.key}"
[ -f "$CERT" ] && [ -f "$KEY" ] || {
  echo "no signing cert/key at $CERT / $KEY" >&2
  exit 1
}

scripts/build.sh "$VERSION"

sign() {
  osslsigncode sign -certs "$CERT" -key "$KEY" \
    -n "CLIque" -i "https://useclique.dev" \
    -t http://timestamp.digicert.com \
    -in "$1" -out "$1.signed"
  mv "$1.signed" "$1"
}

sign dist/CLIque.exe

makensis -DVERSION="$VERSION" \
  -DSRC="$(pwd)/dist/CLIque.exe" \
  -DOUT="$(pwd)/dist/CLIque-Setup.exe" \
  installer.nsi

sign dist/CLIque-Setup.exe

# Recomputed after signing: the published checksum has to match the bytes
# actually downloaded, and signing changes those bytes.
sha256sum dist/CLIque.exe | awk '{print $1, "CLIque.exe"}' OFS='  ' > dist/CLIque.exe.sha256

ls -lh dist/
