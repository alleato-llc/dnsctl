#!/usr/bin/env bash
#
# Removes the dnsctl-helper LaunchDaemon and binary.
#
# Usage: sudo packaging/uninstall-helper.sh
#
set -euo pipefail

LABEL="com.github.nycjv321.dnsctl-helper"
HELPER_DST="/usr/local/libexec/dnsctl-helper"
PLIST_DST="/Library/LaunchDaemons/${LABEL}.plist"
SOCKET="/var/run/dnsctl-helper.sock"

if [ "$(id -u)" -ne 0 ]; then
    echo "error: must run as root (use sudo)" >&2
    exit 1
fi

echo "Unloading service"
launchctl bootout system "$PLIST_DST" 2>/dev/null || true

echo "Removing files"
rm -f "$PLIST_DST" "$HELPER_DST" "$SOCKET"

echo "Done. dnsctl-helper removed."
