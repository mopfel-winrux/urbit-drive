#!/bin/bash
# Make Drive.app and a dmg from a darwin build. Runs on macOS.
# usage: packaging/macos/package.sh <build/out/darwin-ARCH> <dist dir> <version> <arch>
#
# Signs and notarizes only if configured:
#   MACOS_SIGN_IDENTITY   "Developer ID Application: Name (TEAMID)", in the keychain
#   APPLE_ID, APPLE_TEAM_ID, APPLE_APP_PASSWORD   notarytool credentials
set -euo pipefail
OUT=$1; DIST=$2; VERSION=$3; ARCH=$4
HERE=$(cd "$(dirname "$0")" && pwd)
CLIENT=$(cd "$HERE/../.." && pwd)
[ "$(uname -s)" = Darwin ] || { echo "the macOS package is built on macOS" >&2; exit 2; }
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
APP=$WORK/Drive.app
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$DIST"
cp "$OUT/drive" "$APP/Contents/MacOS/drive"
sed "s|@VERSION@|${VERSION#v}|g" "$HERE/Info.plist" > "$APP/Contents/Info.plist"

# the icon, from internal/icon at every size an icns wants
SET=$WORK/drive.iconset; mkdir -p "$SET"
for n in 16 32 128 256 512; do
  (cd "$CLIENT" && CGO_ENABLED=0 go run ./scripts/genicon -size $n "$SET/icon_${n}x${n}.png")
  (cd "$CLIENT" && CGO_ENABLED=0 go run ./scripts/genicon -size $((n*2)) "$SET/icon_${n}x${n}@2x.png")
done
iconutil -c icns "$SET" -o "$APP/Contents/Resources/drive.icns"

signed=0
if [ -n "${MACOS_SIGN_IDENTITY:-}" ]; then
  codesign --force --options runtime --timestamp --sign "$MACOS_SIGN_IDENTITY" "$APP/Contents/MacOS/drive"
  codesign --force --options runtime --timestamp --sign "$MACOS_SIGN_IDENTITY" "$APP"
  codesign --verify --strict --verbose=2 "$APP"
  signed=1
else
  echo "MACOS_SIGN_IDENTITY not set: Drive.app left unsigned (Gatekeeper will warn)"
fi

DMG=$DIST/drive-$VERSION-darwin-$ARCH.dmg
STAGE=$WORK/dmg; mkdir -p "$STAGE"
cp -R "$APP" "$STAGE/"
ln -s /Applications "$STAGE/Applications"
rm -f "$DMG"
hdiutil create -volname Drive -srcfolder "$STAGE" -ov -format UDZO "$DMG"
[ $signed = 1 ] && codesign --force --timestamp --sign "$MACOS_SIGN_IDENTITY" "$DMG"

if [ $signed = 1 ] && [ -n "${APPLE_ID:-}" ] && [ -n "${APPLE_TEAM_ID:-}" ] && [ -n "${APPLE_APP_PASSWORD:-}" ]; then
  xcrun notarytool submit "$DMG" --apple-id "$APPLE_ID" --team-id "$APPLE_TEAM_ID" \
    --password "$APPLE_APP_PASSWORD" --wait
  xcrun stapler staple "$DMG"
else
  echo "not notarized (needs MACOS_SIGN_IDENTITY, APPLE_ID, APPLE_TEAM_ID, APPLE_APP_PASSWORD)"
fi
(cd "$WORK" && ditto -c -k --keepParent Drive.app "$DIST/drive-$VERSION-darwin-$ARCH.app.zip")
echo "$DMG"
