#!/bin/bash
# Build libmesa's core (no host: Go does the I/O) for this machine, into
# third_party/libmesa/zig-out, where the Go binding's #cgo lines look.
# Runs once; delete third_party/libmesa/zig-out to rebuild.
set -euo pipefail
CLIENT=$(cd "$(dirname "$0")/.." && pwd)
LM=$CLIENT/third_party/libmesa
if [ ! -f "$LM/build.zig" ]; then
  echo "libmesa.sh: fetching the libmesa submodule" >&2
  git -C "$CLIENT" submodule update --init third_party/libmesa
fi
[ -f "$LM/zig-out/lib/libmesa.a" ] && exit 0
ZIG=$("$CLIENT/scripts/zig.sh")
echo "libmesa.sh: building libmesa" >&2
(cd "$LM" && "$ZIG" build -Dno-host=true -Doptimize=ReleaseFast)
