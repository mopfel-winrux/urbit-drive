#!/bin/bash
# Sign a Windows executable, if a certificate is configured; otherwise leave
# it unsigned and say so. usage: packaging/windows/sign.sh <file.exe>...
#   WINDOWS_PFX           path to a PKCS#12 code-signing certificate
#   WINDOWS_PFX_PASSWORD  its password
#   WINDOWS_TIMESTAMP_URL (default http://timestamp.digicert.com)
# Uses osslsigncode (Linux/macOS) or signtool (Windows).
set -euo pipefail
if [ -z "${WINDOWS_PFX:-}" ]; then
  echo "WINDOWS_PFX not set: $* left unsigned"; exit 0
fi
TS=${WINDOWS_TIMESTAMP_URL:-http://timestamp.digicert.com}
for f in "$@"; do
  if command -v osslsigncode >/dev/null; then
    osslsigncode sign -pkcs12 "$WINDOWS_PFX" -pass "${WINDOWS_PFX_PASSWORD:-}" \
      -n Drive -t "$TS" -h sha256 -in "$f" -out "$f.signed"
    mv "$f.signed" "$f"
  elif command -v signtool >/dev/null || command -v signtool.exe >/dev/null; then
    signtool sign /f "$WINDOWS_PFX" /p "${WINDOWS_PFX_PASSWORD:-}" /fd sha256 /tr "$TS" /td sha256 /d Drive "$f"
  else
    echo "no osslsigncode or signtool to sign $f with" >&2; exit 1
  fi
  echo "signed $f"
done
