#!/usr/bin/env bash
#
# Installs dnsctl-helper as a root LaunchDaemon so unprivileged dnsctl (and the
# future GUI) can perform privileged DNS/hosts changes without per-action sudo.
#
# Usage: sudo packaging/install-helper.sh [path-to-dnsctl-helper-binary]
#
set -euo pipefail

LABEL="com.github.nycjv321.dnsctl-helper"
HELPER_SRC="${1:-bin/dnsctl-helper}"
HELPER_DST="/usr/local/libexec/dnsctl-helper"
PLIST_SRC="$(cd "$(dirname "$0")" && pwd)/${LABEL}.plist"
PLIST_DST="/Library/LaunchDaemons/${LABEL}.plist"
SOCKET="/var/run/dnsctl-helper.sock"
HOSTS_FILE="/etc/hosts"

# Authorize the user invoking sudo (so they get password-less changes). Falls
# back to the current UID when not run via sudo.
ALLOW_UID="${SUDO_UID:-$(id -u)}"

if [ "$(id -u)" -ne 0 ]; then
    echo "error: must run as root (use sudo)" >&2
    exit 1
fi
if [ ! -f "$HELPER_SRC" ]; then
    echo "error: helper binary not found at '$HELPER_SRC' (run 'make build-helper')" >&2
    exit 1
fi

echo "Installing helper binary -> $HELPER_DST"
install -d -m 0755 /usr/local/libexec
install -m 0755 "$HELPER_SRC" "$HELPER_DST"

echo "Writing LaunchDaemon -> $PLIST_DST (authorized UID: $ALLOW_UID)"
sed -e "s|@HELPER_BIN@|${HELPER_DST}|g" \
    -e "s|@SOCKET@|${SOCKET}|g" \
    -e "s|@HOSTS_FILE@|${HOSTS_FILE}|g" \
    -e "s|@ALLOW_UIDS@|${ALLOW_UID}|g" \
    "$PLIST_SRC" > "$PLIST_DST"
chown root:wheel "$PLIST_DST"
chmod 0644 "$PLIST_DST"

echo "Loading service via launchd"
launchctl bootout system "$PLIST_DST" 2>/dev/null || true
launchctl bootstrap system "$PLIST_DST"
launchctl enable "system/${LABEL}"

echo "Done. dnsctl-helper is running and listening on ${SOCKET}."
