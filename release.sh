#!/bin/bash
# Usage: ./release.sh vX.Y.Z [release title]
# Publishes the release if missing, then pushes the formula bump to main.
set -euo pipefail

VERSION="${1:?usage: ./release.sh vX.Y.Z [title]}"
TITLE="${2:-$VERSION}"
REPO="o-o-o-o-o/caddy-hot-reloader"
FORMULA="Formula/caddy-hot-reloader.rb"
URL="https://github.com/$REPO/archive/refs/tags/$VERSION.tar.gz"

[ -z "$(git status --porcelain)" ] || { echo "Error: working tree is not clean." >&2; exit 1; }
[ "$(git branch --show-current)" = main ] || { echo "Error: not on main." >&2; exit 1; }

git push origin main
gh release view "$VERSION" >/dev/null 2>&1 || gh release create "$VERSION" --target main --title "$TITLE" --generate-notes

SHA="$(curl -sfL --retry 3 "$URL" | shasum -a 256 | awk '{print $1}')"
[ ${#SHA} -eq 64 ] || { echo "Error: could not hash $URL" >&2; exit 1; }

sed -i '' "s|url \".*\"|url \"$URL\"|; s|sha256 \".*\"|sha256 \"$SHA\"|" "$FORMULA"
if git diff --quiet; then
    echo "Formula already at $VERSION."
    exit 0
fi
git add "$FORMULA"
git commit -m "chore: update formula to $VERSION"
git push origin main
echo "Released $VERSION."
