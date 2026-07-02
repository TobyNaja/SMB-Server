#!/bin/bash
# migrate-acl.sh — one-time backfill: recursive ACLs for ALL existing shares
# based on valid users / write list in shares.conf. Idempotent.
set -euo pipefail
CONF="${SHARES_CONF:-/etc/samba/shares.conf}"
[ -f "$CONF" ] || { echo "[!] $CONF not found"; exit 1; }

shares=$(grep -E '^\[' "$CONF" | tr -d '[]') || true
[ -n "$shares" ] || { echo "[*] No shares in $CONF — nothing to do."; exit 0; }

while IFS= read -r S; do
    [ -z "$S" ] && continue
    section=$(awk -v s="[$S]" '$0==s{f=1;next} /^\[/{f=0} f' "$CONF")
    path=$(printf '%s\n' "$section" | grep -E '^\s*path\s*=' | head -1 | sed -E 's/^\s*path\s*=\s*//')
    if [ -z "$path" ] || [ ! -d "$path" ]; then
        echo "[!] [$S] missing dir, skipping: ${path:-<none>}"
        continue
    fi
    echo "[*] [$S] $path"
    printf '%s\n' "$section" \
        | grep -E '^\s*(valid users|write list)\s*=' \
        | sed -E 's/^[^=]+=\s*//' \
        | tr ',' '\n' \
        | sed -E 's/^\s*"?\s*//; s/\s*"?\s*$//' \
        | sort -u \
        | while IFS= read -r u; do
            [ -z "$u" ] && continue
            case "$u" in
                @*) spec="g:${u#@}" ;;
                *)  spec="u:$u" ;;
            esac
            if setfacl -R -m "${spec}:rwX" -m "d:${spec}:rwX" "$path" 2>/dev/null; then
                echo "    + $u -> rwX (recursive + inherit)"
            else
                echo "    ! failed for '$u' (user unknown to NSS? check winbind)"
            fi
        done
done <<< "$shares"
echo "[*] Done."
