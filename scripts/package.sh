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

# The package must start CLIque, not the packaging tool's update checker.
# Conveyor makes updatecheck.exe the entry point by default and only knows how
# to hand off to a JVM app afterwards, so for this binary it launched nothing at
# all. Turned off in conveyor.conf, and checked here because a Conveyor upgrade
# could put the default back without anything saying so.
MSIX="$(ls -1 output/*.msix | head -1)"
ENTRY="$(unzip -p "$MSIX" AppxManifest.xml | grep -oE 'Executable="[^"]*"' | head -1)"
if [ "$ENTRY" != 'Executable="CLIque.exe"' ]; then
  echo "the package would start $ENTRY instead of CLIque.exe" >&2
  exit 1
fi

ls -1 output/
