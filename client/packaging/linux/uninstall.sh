#!/bin/sh
# Remove what install.sh put in place. Your synced files, and Drive's
# state (~/.config/drive: pairing, keys), are left alone.
set -eu
PREFIX=${PREFIX:-$HOME/.local}
DATA=${XDG_DATA_HOME:-$HOME/.local/share}
CONF=${XDG_CONFIG_HOME:-$HOME/.config}
if [ -f "$CONF/systemd/user/drive.service" ]; then
  systemctl --user disable --now drive 2>/dev/null || true
  rm -f "$CONF/systemd/user/drive.service"
fi
rm -f "$PREFIX/bin/drive" "$DATA/applications/drive.desktop" \
  "$DATA/icons/hicolor/256x256/apps/drive.png" "$CONF/autostart/drive.desktop"
echo "Drive removed (state in $CONF/drive kept)"
