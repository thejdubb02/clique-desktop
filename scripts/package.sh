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

# Windows runs the manifest's background update task on a schedule of its own,
# which in practice can be the better part of a day: reported on 2026-09-18 as
# the app never updating by itself. OnLaunch is what makes an update actually
# arrive. UpdateBlocksActivation stays false, so the app opens straight away and
# Windows fetches behind it; the new version is what opens next time.
#
# Conveyor does not expose this setting, so it is added to the file it writes.
AI=output/clique.appinstaller
python3 - "$AI" <<'ONLAUNCH'
import sys
p = sys.argv[1]
s = open(p, encoding="utf-8").read()
if "<OnLaunch" not in s:
    old = "<AutomaticBackgroundTask />"
    assert s.count(old) == 1, "AutomaticBackgroundTask not found in " + p
    new = ('<OnLaunch HoursBetweenUpdateChecks="0" UpdateBlocksActivation="false" ShowPrompt="false" />\n'
           "        " + old)
    open(p, "w", encoding="utf-8").write(s.replace(old, new))
ONLAUNCH
grep -q "<OnLaunch" "$AI" || { echo "the manifest would never check on launch" >&2; exit 1; }

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
