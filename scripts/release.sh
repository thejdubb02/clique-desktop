#!/usr/bin/env bash
# Build, sign, tag and publish, one step. v0.3.17 shipped without the
# checksum file the in-app updater requires (CLIque.exe.sha256, see
# update.go's sumAsset) because a human typed the asset list by hand for
# `gh release create` and left it off, package.sh had already built it,
# sitting right there in dist/. This uploads dist/* by glob instead, so a
# release can never again ship missing a file that was already on disk.
#
#   scripts/release.sh 0.3.18
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:?usage: release.sh <version>}"
TAG="v${VERSION}"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "working tree is dirty, commit or stash first" >&2
  exit 1
fi
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "$TAG already exists" >&2
  exit 1
fi

scripts/package.sh "$VERSION"

# The one file the updater trusts before it applies anything downloaded.
# Fail loudly here rather than publish a release that quietly can't update
# to itself.
for f in CLIque.exe CLIque.exe.sha256 CLIque-Setup.exe CLIque-Setup.exe.sha256; do
  [ -s "dist/$f" ] || { echo "dist/$f missing or empty" >&2; exit 1; }
done

git tag "$TAG"
git push origin "$TAG"

NOTES="${RELEASE_NOTES:-}"
if [ -z "$NOTES" ]; then
  echo "no RELEASE_NOTES set, publishing with the tag name as the only note" >&2
  echo "set RELEASE_NOTES=\"...\" scripts/release.sh $VERSION to write real ones" >&2
fi

gh release create "$TAG" dist/* --title "$TAG" ${NOTES:+--notes "$NOTES"}

# Trust what got published, not what the upload command claimed: check the
# release itself for the assets a client actually needs, the same gap
# that shipped in v0.3.17.
published="$(gh release view "$TAG" --json assets --jq '.assets[].name')"
for want in CLIque.exe CLIque-Setup.exe CLIque.exe.sha256 CLIque-Setup.exe.sha256; do
  echo "$published" | grep -qx "$want" || {
    echo "published release is missing $want: fix and re-upload, do not tell anyone to update" >&2
    exit 1
  }
done

echo
echo "published: https://github.com/thejdubb02/clique-desktop/releases/tag/$TAG"
echo "assets:"
echo "$published"
