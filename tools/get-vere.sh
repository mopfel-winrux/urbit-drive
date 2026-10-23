#!/bin/bash
# Print the path of an urbit runtime (vere), downloading the latest official
# release once into tools/.vere/ if there isn't one.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
DIR=$HERE/.vere
if [ ! -x "$DIR/urbit" ]; then
  case "$(uname -s)/$(uname -m)" in
    Linux/x86_64)                P=linux-x86_64 ;;
    Linux/aarch64)               P=linux-aarch64 ;;
    Darwin/x86_64)               P=macos-x86_64 ;;
    Darwin/arm64|Darwin/aarch64) P=macos-aarch64 ;;
    *) echo "get-vere.sh: no vere build for $(uname -sm); set VERE" >&2; exit 1 ;;
  esac
  echo "get-vere.sh: downloading vere ($P)" >&2
  mkdir -p "$DIR"
  curl -fsSL "https://urbit.org/install/$P/latest" | tar -xz -C "$DIR"
  # the archive holds one binary, named for its version
  mv "$(find "$DIR" -maxdepth 1 -type f -name 'vere*' | head -1)" "$DIR/urbit"
  chmod +x "$DIR/urbit"
fi
echo "$DIR/urbit"
