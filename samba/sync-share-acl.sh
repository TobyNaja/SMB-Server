#!/bin/bash
# sync-share-acl.sh SHARENAME
# Apply base ACLs on a share root from its valid users / write list.
# Called by Samba root preexec on tree connect. Must be fast + silent.
set -u
CONF="${SHARES_CONF:-/etc/samba/shares.conf}"
S="${1:-}"
[ -n "$S" ] || exit 0
[ -f "$CONF" ] || exit 0

section=$(awk -v s="[$S]" '$0==s{f=1;next} /^\[/{f=0} f' "$CONF")
path=$(printf '%s\n' "$section" | grep -E '^\s*path\s*=' | head -1 | sed -E 's/^\s*path\s*=\s*//')
[ -n "$path" ] && [ -d "$path" ] || exit 0

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
        setfacl -m "${spec}:rwX" -m "d:${spec}:rwX" "$path" 2>/dev/null || true
    done
exit 0
