#!/bin/bash
# Print the path of a zig 0.15.2 (libmesa needs exactly this version):
# $ZIG if it is one, else `zig` on PATH if it is one, else a copy
# downloaded once into build/zig-0.15.2, checked against ziglang.org's
# published SHA-256.
set -euo pipefail
WANT=0.15.2
CLIENT=$(cd "$(dirname "$0")/.." && pwd)
ok() { [ -x "$(command -v "$1" 2>/dev/null)" ] && [ "$("$1" version 2>/dev/null)" = "$WANT" ]; }
for z in "${ZIG:-}" zig; do
  if [ -n "$z" ] && ok "$z"; then command -v "$z"; exit 0; fi
done
DIR=$CLIENT/build/zig-$WANT
if ! ok "$DIR/zig"; then
  case "$(uname -s)/$(uname -m)" in
    Linux/x86_64)              K=x86_64-linux ;;
    Linux/aarch64)             K=aarch64-linux ;;
    Darwin/x86_64)             K=x86_64-macos ;;
    Darwin/arm64|Darwin/aarch64) K=aarch64-macos ;;
    *) echo "zig.sh: no zig $WANT download for $(uname -sm); install it and set ZIG" >&2; exit 1 ;;
  esac
  echo "zig.sh: downloading zig $WANT ($K)" >&2
  read -r URL SUM < <(curl -fsSL https://ziglang.org/download/index.json | python3 -c \
    "import json,sys; d=json.load(sys.stdin)['$WANT']['$K']; print(d['tarball'], d['shasum'])")
  TMP=$(mktemp -d); trap 'rm -rf "${TMP:?}"' EXIT
  curl -fsSL "$URL" -o "$TMP/zig.tar.xz"
  GOT=$( (sha256sum "$TMP/zig.tar.xz" 2>/dev/null || shasum -a 256 "$TMP/zig.tar.xz") | cut -d' ' -f1)
  [ "$GOT" = "$SUM" ] || { echo "zig.sh: checksum mismatch for $URL" >&2; exit 1; }
  mkdir -p "$DIR"
  tar -xJf "$TMP/zig.tar.xz" -C "$DIR" --strip-components=1
fi
echo "$DIR/zig"
