#!/usr/bin/env bash
# Build the Windows package Justin actually installs: an MSIX that Windows keeps
# current in the background, plus the certificate and the one-line PowerShell
# installer that trusts it. Cross-built from Linux, same as the exe.
#
#   scripts/package.sh 0.3.0
#
# The version is given once and reaches both the binary and the package, because
# a package whose manifest disagrees with the exe inside it updates on a
# schedule nobody can explain.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:?usage: package.sh <version>}"

scripts/build.sh "$VERSION"

conveyor -Kapp.version="$VERSION" make site

ls -1 output/
