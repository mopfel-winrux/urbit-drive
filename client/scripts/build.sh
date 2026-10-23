#!/bin/bash
# Build the drive client for one target, against libmesa built for that target.
# usage: scripts/build.sh <goos> <goarch>      e.g. scripts/build.sh linux arm64
#
# The libmesa Go binding links ${SRCDIR}/../../zig-out (the native build), so
# for any target we copy the binding into build/gobind/<target>, point its two
# #cgo lines at build/libmesa/<target>, and build through a go.work whose
# replace overrides the one in go.mod. libmesa itself is never touched.
#
# env: LIBMESA  libmesa checkout (default ../../libmesa/libmesa from client/)
#      ZIG      zig 0.15.2 (default: zig on PATH)
#      VERSION  stamped into the binary (default: git describe)
set -euo pipefail
GOOS_T=$1; GOARCH_T=$2
CLIENT=$(cd "$(dirname "$0")/.." && pwd)
LIBMESA=$(cd "${LIBMESA:-$CLIENT/../../libmesa/libmesa}" && pwd)
ZIG=${ZIG:-zig}
VERSION=${VERSION:-$(git -C "$CLIENT" describe --tags --always --dirty 2>/dev/null || echo dev)}

case "$GOOS_T/$GOARCH_T" in
  linux/amd64)   T=x86_64-linux-musl ;;
  linux/arm64)   T=aarch64-linux-musl ;;
  windows/amd64) T=x86_64-windows-gnu ;;
  darwin/arm64)  T=aarch64-macos ;;
  darwin/amd64)  T=x86_64-macos ;;
  *) echo "unsupported target $GOOS_T/$GOARCH_T" >&2; exit 2 ;;
esac
PREFIX=$CLIENT/build/libmesa/$T
BIND=$CLIENT/build/gobind/$T
OUT=$CLIENT/build/out/$GOOS_T-$GOARCH_T
EXE=drive; [ "$GOOS_T" = windows ] && EXE=drive.exe

# 1. libmesa's core (no host: Go does the I/O) for the target
if [ ! -f "$PREFIX/include/mesa/engine.h" ]; then
  echo "libmesa -> $PREFIX"
  (cd "$LIBMESA" && "$ZIG" build -Dno-host=true -Doptimize=ReleaseFast -Dtarget="$T" -p "$PREFIX")
fi

# 2. the binding, linking that prefix
rm -rf "$BIND"; mkdir -p "$BIND"
cp "$LIBMESA"/bindings/go/{go.mod,go.sum,shim.c,shim.h} "$BIND"/
for f in "$LIBMESA"/bindings/go/*.go; do
  case "$f" in *_test.go) continue ;; esac
  sed "s|\${SRCDIR}/../../zig-out|$PREFIX|g" "$f" > "$BIND/$(basename "$f")"
done
cat > "$BIND/go.work" <<EOF
go 1.26.0

use $CLIENT

replace github.com/mopfel-winrux/libmesa/bindings/go => $BIND
EOF

# 3. the client. On a Mac, build natively (fyne/systray needs the macOS SDK's
#    Cocoa, which zig doesn't ship); everywhere else, cross-compile with zig cc.
LDFLAGS="-s -w -X main.version=$VERSION"
case "$GOOS_T" in
  linux)   LDFLAGS="$LDFLAGS -linkmode external -extldflags '-static -s'" ;;
  windows) LDFLAGS="$LDFLAGS -H=windowsgui" ;;
esac
if [ "$GOOS_T" = darwin ]; then
  [ "$(uname -s)" = Darwin ] || { echo "darwin builds run on macOS (see packaging/macos/README)" >&2; exit 2; }
  CC_T="clang -arch $([ "$GOARCH_T" = arm64 ] && echo arm64 || echo x86_64)"
else
  CC_T="$ZIG cc -target $T"
fi
mkdir -p "$OUT"
echo "drive $VERSION -> $OUT/$EXE"
(cd "$CLIENT" && GOWORK="$BIND/go.work" CGO_ENABLED=1 GOOS=$GOOS_T GOARCH=$GOARCH_T CC="$CC_T" \
  go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/$EXE" ./cmd/drive)
