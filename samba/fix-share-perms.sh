#!/bin/bash
# One-time (idempotent) fixup: re-own every existing share directory to
# smbshare:smbshare 2770. Existing shares were created as 770 root:root by an
# earlier version and are inaccessible to non-root users.
set -euo pipefail

SHARES_CONF="${1:-/etc/samba/shares.conf}"

grep -E '^\s*path\s*=' "$SHARES_CONF" | sed -E 's/^\s*path\s*=\s*//' | while read -r dir; do
    [ -z "$dir" ] && continue
    if [ -d "$dir" ]; then
        echo "[*] Fixing $dir"
        chown smbshare:smbshare "$dir"
        chmod 2770 "$dir"
    else
        echo "[!] Skipping missing dir: $dir"
    fi
done
echo "[*] Done."
