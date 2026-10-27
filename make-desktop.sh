#!/usr/bin/env bash
set -euo pipefail

APP_NAME="mdedit"
BIN_PATH="/usr/local/bin/mdedit"
DESKTOP_FILE="${HOME}/.local/share/applications/mdedit.desktop"
ICON_DIR="${HOME}/.local/share/icons/hicolor/scalable/apps"
ICON_FILE="${ICON_DIR}/mdedit.svg"

if [[ ! -x "$BIN_PATH" ]]; then
    echo "Warning: $BIN_PATH does not exist or is not executable." >&2
    echo "Continuing anyway; edit BIN_PATH in this script if needed." >&2
fi

mkdir -p "$(dirname "$DESKTOP_FILE")" "$ICON_DIR"

cat > "$ICON_FILE" <<'SVG'
<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128" viewBox="0 0 128 128">
  <defs>
    <linearGradient id="paper" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#ffffff"/>
      <stop offset="1" stop-color="#e7eef8"/>
    </linearGradient>
    <linearGradient id="blue" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#4f9cf9"/>
      <stop offset="1" stop-color="#2457c5"/>
    </linearGradient>
  </defs>
  <rect x="17" y="9" width="82" height="106" rx="11" fill="#14213d" opacity=".15"/>
  <path d="M25 8h49l25 25v76a8 8 0 0 1-8 8H25a8 8 0 0 1-8-8V16a8 8 0 0 1 8-8z" fill="url(#paper)" stroke="#c5d2e3" stroke-width="2"/>
  <path d="M74 8v18a7 7 0 0 0 7 7h18z" fill="#d5e2f3"/>
  <rect x="27" y="44" width="45" height="7" rx="3.5" fill="#a5b7cf"/>
  <rect x="27" y="59" width="59" height="6" rx="3" fill="#c0ccdc"/>
  <rect x="27" y="72" width="49" height="6" rx="3" fill="#c0ccdc"/>
  <rect x="27" y="85" width="36" height="6" rx="3" fill="#c0ccdc"/>
  <circle cx="87" cy="88" r="25" fill="url(#blue)" stroke="#ffffff" stroke-width="3"/>
  <path d="M77 77l-8 11 8 11M97 77l8 11-8 11M91 72l-8 32" fill="none" stroke="#ffffff" stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round"/>
</svg>
SVG

cat > "$DESKTOP_FILE" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=mdedit
Comment=Markdown editor
Exec=/usr/local/bin/mdedit %F
Icon=mdedit
Terminal=false
Categories=Development;Office;Science;
StartupNotify=true
EOF

chmod 644 "$DESKTOP_FILE" "$ICON_FILE"

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "$(dirname "$DESKTOP_FILE")" || true
fi

echo "Created desktop entry: $DESKTOP_FILE"
echo "Created icon:          $ICON_FILE"
echo "You should now find mdedit in your application menu."
