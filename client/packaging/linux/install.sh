#!/bin/sh
# Install Drive for this user (no root): the binary in ~/.local/bin, a menu
# entry and its icon. Options:
#   --autostart   start the tray at login (same as its "Start at login")
#   --service     install the headless systemd user service instead
# PREFIX (default ~/.local) and XDG_* are honoured. Undo with uninstall.sh.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
PREFIX=${PREFIX:-$HOME/.local}
DATA=${XDG_DATA_HOME:-$HOME/.local/share}
CONF=${XDG_CONFIG_HOME:-$HOME/.config}
BINDIR=$PREFIX/bin
autostart=0; service=0
for a in "$@"; do
  case "$a" in
    --autostart) autostart=1 ;;
    --service)   service=1 ;;
    *) echo "usage: $0 [--autostart] [--service]" >&2; exit 2 ;;
  esac
done

install -d "$BINDIR" "$DATA/applications" "$DATA/icons/hicolor/256x256/apps"
install -m 755 "$HERE/drive" "$BINDIR/drive"
sed "s|@BINDIR@|$BINDIR|g" "$HERE/drive.desktop" > "$DATA/applications/drive.desktop"
install -m 644 "$HERE/drive.png" "$DATA/icons/hicolor/256x256/apps/drive.png"
echo "installed $BINDIR/drive"

if [ $autostart = 1 ]; then
  install -d "$CONF/autostart"
  sed -e "s|@BINDIR@|$BINDIR|g" -e '$a X-GNOME-Autostart-enabled=true' "$HERE/drive.desktop" > "$CONF/autostart/drive.desktop"
  echo "starts at login ($CONF/autostart/drive.desktop)"
fi
if [ $service = 1 ]; then
  install -d "$CONF/systemd/user"
  sed "s|@BINDIR@|$BINDIR|g" "$HERE/drive.service" > "$CONF/systemd/user/drive.service"
  echo "service installed: pair with '$BINDIR/drive pair ...', then 'systemctl --user enable --now drive'"
fi
case ":$PATH:" in *":$BINDIR:"*) ;; *) echo "note: $BINDIR isn't on your PATH" ;; esac
echo "run 'drive' (or Drive from your menu) to pair with your planet"
