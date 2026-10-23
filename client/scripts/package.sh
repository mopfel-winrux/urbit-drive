#!/bin/bash
# Package a build from scripts/build.sh into dist/.
# usage: scripts/package.sh <goos> <goarch>
#
#   linux    drive-VERSION-linux-ARCH.tar.gz (binary, menu entry, icon,
#            systemd user unit, install.sh), and a .deb if dpkg-deb exists
#   windows  drive-VERSION-windows-amd64.zip (drive.exe signed if a PFX is
#            configured, drive.ico); the installer is built on Windows from
#            packaging/windows/drive.iss
#   darwin   Drive.app in a dmg and a zip (packaging/macos/package.sh, on macOS)
set -euo pipefail
GOOS_T=$1; GOARCH_T=$2
CLIENT=$(cd "$(dirname "$0")/.." && pwd)
VERSION=${VERSION:-$(git -C "$CLIENT" describe --tags --always --dirty 2>/dev/null || echo dev)}
OUT=$CLIENT/build/out/$GOOS_T-$GOARCH_T
DIST=$CLIENT/dist
PKG=$CLIENT/packaging
NAME=drive-$VERSION-$GOOS_T-$GOARCH_T
mkdir -p "$DIST"
genicon() { (cd "$CLIENT" && CGO_ENABLED=0 GOOS= GOARCH= go run ./scripts/genicon "$@"); }

case "$GOOS_T" in
linux)
  STAGE=$CLIENT/build/stage/$NAME; rm -rf "$STAGE"; mkdir -p "$STAGE"
  cp "$OUT/drive" "$PKG"/linux/{drive.desktop,drive.service,install.sh,uninstall.sh} "$STAGE"/
  genicon -size 256 "$STAGE/drive.png"
  cat > "$STAGE/README" <<EOF
Drive $VERSION: sync folders with your Urbit planet.

  ./install.sh               install for this user (~/.local), with a menu entry
  ./install.sh --autostart   ... and start the tray at login
  ./install.sh --service     ... or run headless as a systemd user service

Then run 'drive' to pair with your planet. ./uninstall.sh removes it.
Use the tray (and --autostart) or the service, not both: one device runs once.
EOF
  tar -C "$(dirname "$STAGE")" -czf "$DIST/$NAME.tar.gz" "$NAME"
  echo "$DIST/$NAME.tar.gz"

  if command -v dpkg-deb >/dev/null; then
    V=${VERSION#v}
    case "$V" in [0-9]*) ;; *) V=0.0.0~$V ;; esac
    V=${V//-/\~}
    DEB=$CLIENT/build/stage/deb-$GOARCH_T; rm -rf "$DEB"
    install -d "$DEB/DEBIAN" "$DEB/usr/bin" "$DEB/usr/share/applications" \
      "$DEB/usr/share/icons/hicolor/256x256/apps" "$DEB/usr/lib/systemd/user"
    install -m 755 "$OUT/drive" "$DEB/usr/bin/drive"
    sed "s|@BINDIR@|/usr/bin|g" "$PKG/linux/drive.desktop" > "$DEB/usr/share/applications/drive.desktop"
    sed "s|@BINDIR@|/usr/bin|g" "$PKG/linux/drive.service" > "$DEB/usr/lib/systemd/user/drive.service"
    cp "$STAGE/drive.png" "$DEB/usr/share/icons/hicolor/256x256/apps/drive.png"
    chmod 644 "$DEB/usr/share/applications/drive.desktop" "$DEB/usr/lib/systemd/user/drive.service" \
      "$DEB/usr/share/icons/hicolor/256x256/apps/drive.png"
    cat > "$DEB/DEBIAN/control" <<EOF
Package: drive-urbit
Version: $V
Architecture: $GOARCH_T
Maintainer: Drive for Urbit <noreply@example.invalid>
Section: net
Priority: optional
Description: Sync folders with your Urbit planet
 A desktop client for the %drive Urbit app. The computer becomes a moon of
 your planet and syncs a folder with it over mesa.
EOF
    dpkg-deb --root-owner-group --build "$DEB" "$DIST/drive-urbit_${V}_$GOARCH_T.deb" >/dev/null
    echo "$DIST/drive-urbit_${V}_$GOARCH_T.deb"
  fi
  ;;
windows)
  genicon -ico "$OUT/drive.ico"
  "$PKG/windows/sign.sh" "$OUT/drive.exe"
  STAGE=$CLIENT/build/stage/$NAME; rm -rf "$STAGE"; mkdir -p "$STAGE"
  cp "$OUT/drive.exe" "$OUT/drive.ico" "$STAGE"/
  (cd "$(dirname "$STAGE")" && rm -f "$DIST/$NAME.zip" && zip -qr "$DIST/$NAME.zip" "$NAME")
  echo "$DIST/$NAME.zip"
  if command -v iscc >/dev/null || command -v iscc.exe >/dev/null; then
    (cd "$PKG/windows" && iscc "/DAppVersion=${VERSION#v}" "/DSourceDir=$OUT" "/DOutputDir=$DIST" drive.iss)
    "$PKG/windows/sign.sh" "$DIST"/drive-*-setup.exe
  else
    echo "no iscc (Inno Setup): installer not built; build it on Windows from packaging/windows/drive.iss"
  fi
  ;;
darwin)
  "$PKG/macos/package.sh" "$OUT" "$DIST" "$VERSION" "$GOARCH_T"
  ;;
*) echo "unsupported $GOOS_T" >&2; exit 2 ;;
esac
