#!/bin/bash
# apply-subfolder-acl.sh
# One-shot: switch from "force user = smbshare" to POSIX ACL mode
# + add per-subfolder permission API endpoint.
# Idempotent — safe to re-run. Backs up every file it touches.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$REPO_ROOT"

TEMPLATE="samba/smb.conf.template"
DOCKERFILE="samba/Dockerfile"
SHARES_GO="backend/internal/httpapi/shares.go"
CONF_GO="$(grep -rl 'force user' backend/internal/samba/ 2>/dev/null | grep -v _test | head -1 || true)"
NEW_GO="backend/internal/httpapi/subfolder_permissions.go"
BACKUP_DIR=".backup-acl-$(date +%Y%m%d-%H%M%S)"

ok()   { printf '  \033[32m✅ %s\033[0m\n' "$1"; }
warn() { printf '  \033[33m⚠️  %s\033[0m\n' "$1"; WARNINGS=$((WARNINGS+1)); }
skip() { printf '  \033[36m⏭️  %s (already applied)\033[0m\n' "$1"; }
die()  { printf '  \033[31m❌ %s\033[0m\n' "$1"; exit 1; }
WARNINGS=0

backup() {
    mkdir -p "$BACKUP_DIR/$(dirname "$1")"
    cp "$1" "$BACKUP_DIR/$1"
}

[ -f "$TEMPLATE" ] || die "not at repo root ($TEMPLATE not found)"
[ -f "$SHARES_GO" ] || die "$SHARES_GO not found"

echo "═══════════════════════════════════════════════════"
echo " PART 0: Preflight"
echo "═══════════════════════════════════════════════════"
if ! git diff --quiet 2>/dev/null; then
    warn "working tree has uncommitted changes — backups in $BACKUP_DIR"
fi
ok "repo root: $REPO_ROOT"

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 1: ZFS acltype check"
echo "═══════════════════════════════════════════════════"
ZFS_DATASET="${ZFS_DATASET:-ScriptDataPool}"
if command -v zfs >/dev/null 2>&1; then
    ACLTYPE=$(zfs get -H -o value acltype "$ZFS_DATASET" 2>/dev/null || echo "unknown")
    case "$ACLTYPE" in
        posix|posixacl) ok "ZFS $ZFS_DATASET acltype=$ACLTYPE" ;;
        unknown) warn "dataset '$ZFS_DATASET' not found — set ZFS_DATASET=<name> and re-run, or fix manually" ;;
        *)
            echo "  acltype=$ACLTYPE → enabling posixacl (needs sudo)..."
            sudo zfs set acltype=posixacl "$ZFS_DATASET"
            sudo zfs set xattr=sa "$ZFS_DATASET"
            ok "set acltype=posixacl xattr=sa on $ZFS_DATASET"
            ;;
    esac
else
    warn "zfs command not found — run manually on the storage host:"
    echo "      sudo zfs set acltype=posixacl $ZFS_DATASET && sudo zfs set xattr=sa $ZFS_DATASET"
fi

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 2: smb.conf.template — remove force user, add root preexec"
echo "═══════════════════════════════════════════════════"
backup "$TEMPLATE"
if grep -qE '^\s*force user\s*=' "$TEMPLATE"; then
    sed -i \
        -e '/^\s*#.*file I\/O as this owner/d' \
        -e '/^\s*#.*smbshare/d' \
        -e '/^\s*#.*Set here so existing shares/d' \
        -e '/^\s*#.*not just shares created/d' \
        -e '/^\s*force user\s*=/d' \
        -e '/^\s*force group\s*=/d' \
        "$TEMPLATE"
    ok "removed force user/group from [global]"
else
    skip "force user not in template"
fi

if grep -q 'sync-share-acl.sh' "$TEMPLATE"; then
    skip "root preexec already in template"
elif grep -q '^include = /etc/samba/shares.conf' "$TEMPLATE"; then
    sed -i '/^include = \/etc\/samba\/shares.conf/i\
\
    # Sync base ACLs from valid users/write list on every tree connect.\
    # Cheap (non-recursive); new files inherit via default ACLs.\
    root preexec = /usr/local/bin/sync-share-acl.sh "%S"
' "$TEMPLATE"
    ok "added root preexec = sync-share-acl.sh to [global]"
else
    warn "include line not found in $TEMPLATE — add manually under [global]:"
    echo '      root preexec = /usr/local/bin/sync-share-acl.sh "%S"'
fi

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 3: create samba/sync-share-acl.sh + samba/migrate-acl.sh"
echo "═══════════════════════════════════════════════════"
cat > samba/sync-share-acl.sh <<'EOF'
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
EOF
ok "wrote samba/sync-share-acl.sh"

cat > samba/migrate-acl.sh <<'EOF'
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
EOF
ok "wrote samba/migrate-acl.sh"

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 4: samba/Dockerfile — install acl + copy scripts"
echo "═══════════════════════════════════════════════════"
backup "$DOCKERFILE"
if grep -q 'install -y acl' "$DOCKERFILE"; then
    skip "acl package"
else
    cat >> "$DOCKERFILE" <<'EOF'

# --- POSIX ACL support for per-subfolder permissions ---
RUN apt-get update && apt-get install -y acl && rm -rf /var/lib/apt/lists/*
EOF
    ok "appended acl install"
fi
if grep -q 'sync-share-acl.sh' "$DOCKERFILE"; then
    skip "ACL scripts COPY"
else
    cat >> "$DOCKERFILE" <<'EOF'
COPY sync-share-acl.sh /usr/local/bin/sync-share-acl.sh
COPY migrate-acl.sh /usr/local/bin/migrate-acl.sh
RUN chmod +x /usr/local/bin/sync-share-acl.sh /usr/local/bin/migrate-acl.sh
EOF
    ok "appended COPY for ACL scripts"
fi

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 5: conf.go — remove per-share force user emission"
echo "═══════════════════════════════════════════════════"
if [ -n "$CONF_GO" ]; then
    backup "$CONF_GO"
    sed -i '/force user/d; /force group/d' "$CONF_GO"
    ok "removed force user/group lines from $CONF_GO"
    if grep -rn 'force user' backend/ --include='*_test.go' >/dev/null 2>&1; then
        warn "tests still reference 'force user' — fix these or 'make test' will fail:"
        grep -rn 'force user' backend/ --include='*_test.go' | sed 's/^/      /'
    fi
else
    skip "no non-test Go file emits force user"
fi

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 6: new Go endpoint — subfolder_permissions.go"
echo "═══════════════════════════════════════════════════"
FIBER_IMPORT=$(grep -oE '"github.com/gofiber/fiber/v[0-9]+"' "$SHARES_GO" | head -1 || true)
[ -n "$FIBER_IMPORT" ] || die "cannot detect fiber import in $SHARES_GO"
HANDLER_TYPE=$(grep -oE 'func \(h \*[A-Za-z0-9_]+\)' "$SHARES_GO" | head -1 \
    | sed -E 's/func \(h \*([A-Za-z0-9_]+)\).*/\1/')
[ -n "$HANDLER_TYPE" ] || die "cannot detect handler type in $SHARES_GO"
PKG_NAME=$(head -1 "$SHARES_GO" | awk '{print $2}')
ok "detected: package=$PKG_NAME handler=*$HANDLER_TYPE fiber=$FIBER_IMPORT"

if [ -f "$NEW_GO" ]; then
    skip "$NEW_GO exists"
else
    cat > "$NEW_GO" <<'EOF'
package __PKG__

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	__FIBER__
)

var (
	// perms: ordered subset of rwx (e.g. "r", "rx", "rwx")
	subfolderValidPerms = regexp.MustCompile(`^r?w?x?$`)
	// username: block shell metachars / quotes; allow DOMAIN\user, dots, @, space
	subfolderValidUser = regexp.MustCompile(`^[A-Za-z0-9_.@\\ -]+$`)
)

type subfolderPermissionRequest struct {
	SubfolderPath string `json:"subfolder_path"`
	Username      string `json:"username"`
	Permissions   string `json:"permissions"` // "rwx","rx","r","" or "none" = remove
	Recursive     bool   `json:"recursive"`
}

// resolveSubfolder validates the path stays inside the share root.
func (h *__HANDLER__) resolveSubfolder(name, sub string) (base, target, rel string, err error) {
	p := h.parser()
	share, err := p.GetShare(name)
	if err != nil || share == nil {
		return "", "", "", fmt.Errorf("share not found")
	}
	base = share.Path
	target = filepath.Join(base, filepath.Clean("/"+sub))
	rel, err = filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", "", "", fmt.Errorf("invalid subfolder path")
	}
	return base, target, rel, nil
}

func (h *__HANDLER__) updateSubfolderPermissions(c *fiber.Ctx) error {
	name := c.Params("name")
	var req subfolderPermissionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}

	username := strings.TrimSpace(req.Username)
	if username == "" || !subfolderValidUser.MatchString(username) {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid username"})
	}
	perms := strings.TrimSpace(req.Permissions)
	if perms != "" && perms != "none" && !subfolderValidPerms.MatchString(perms) {
		return c.Status(400).JSON(fiber.Map{"detail": "permissions must be an ordered subset of rwx"})
	}

	basePath, targetPath, rel, err := h.resolveSubfolder(name, req.SubfolderPath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}

	qUser := "'" + username + "'"
	qPath := "'" + targetPath + "'"
	recursiveFlag := ""
	if req.Recursive {
		recursiveFlag = "-R "
	}

	var cmd string
	if perms == "" || perms == "none" {
		// remove both access ACL and default ACL (no ghost perms on new files)
		cmd = fmt.Sprintf(
			"setfacl %s-x u:%s %s 2>/dev/null; setfacl %s-x d:u:%s %s 2>/dev/null; true",
			recursiveFlag, qUser, qPath,
			recursiveFlag, qUser, qPath,
		)
	} else {
		// access ACL + default ACL (inheritance) in one call
		cmd = fmt.Sprintf(
			"setfacl %s-m u:%s:%s,d:u:%s:%s %s",
			recursiveFlag, qUser, perms, qUser, perms, qPath,
		)
		// grant traverse (x) up the parent chain so the user can reach it
		dir := filepath.Dir(targetPath)
		for strings.HasPrefix(dir, basePath) && dir != basePath {
			cmd += fmt.Sprintf(" && setfacl -m u:%s:x '%s'", qUser, dir)
			dir = filepath.Dir(dir)
		}
		cmd += fmt.Sprintf(" && setfacl -m u:%s:x '%s'", qUser, basePath)
	}

	if out, err := h.exec.Execute(cmd); err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": "setfacl failed: " + out})
	}
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("permissions for %q on %q updated", username, rel),
	})
}

func (h *__HANDLER__) getSubfolderPermissions(c *fiber.Ctx) error {
	name := c.Params("name")
	sub := c.Query("path", ".")

	_, targetPath, rel, err := h.resolveSubfolder(name, sub)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}

	out, err := h.exec.Execute("getfacl -p '" + targetPath + "'")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": "getfacl failed: " + out})
	}

	type aclEntry struct {
		Type    string `json:"type"`    // user | group
		Name    string `json:"name"`    // empty = owner/owning-group
		Perms   string `json:"perms"`   // e.g. rwx, r-x
		Default bool   `json:"default"` // inheritance entry
	}
	var entries []aclEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		isDefault := strings.HasPrefix(line, "default:")
		line = strings.TrimPrefix(line, "default:")
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		kind := parts[0]
		if kind != "user" && kind != "group" {
			continue
		}
		// name may itself contain ':' (DOMAIN\user won't, but be safe):
		nameField := strings.Join(parts[1:len(parts)-1], ":")
		permField := parts[len(parts)-1]
		// strip getfacl effective-rights comments, e.g. "rwx\t#effective:r-x"
		if i := strings.IndexAny(permField, " \t#"); i >= 0 {
			permField = permField[:i]
		}
		entries = append(entries, aclEntry{
			Type: kind, Name: nameField, Perms: permField, Default: isDefault,
		})
	}
	return c.JSON(fiber.Map{"share": name, "path": rel, "entries": entries})
}
EOF
    sed -i \
        -e "s/__PKG__/$PKG_NAME/" \
        -e "s|__FIBER__|$FIBER_IMPORT|" \
        -e "s/__HANDLER__/$HANDLER_TYPE/g" \
        "$NEW_GO"
    ok "wrote $NEW_GO"
fi

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 7: register routes"
echo "═══════════════════════════════════════════════════"
ROUTES_FILE=$(grep -rl 'registerSharesRoutes' backend/internal/httpapi/ 2>/dev/null | grep -v _test | head -1 || true)
if [ -z "$ROUTES_FILE" ]; then
    # routes may be registered in shares.go itself
    ROUTES_FILE=$(grep -rln 'g\.Post("/:name' backend/internal/httpapi/*.go 2>/dev/null | grep -v _test | head -1 || true)
fi
if [ -z "$ROUTES_FILE" ]; then
    warn "route registration file not found — add these two lines manually:"
    echo '      g.Get("/:name/subfolders/permissions", h.getSubfolderPermissions)'
    echo '      g.Post("/:name/subfolders/permissions", h.updateSubfolderPermissions)'
elif grep -q 'subfolders/permissions' "$ROUTES_FILE"; then
    skip "routes already registered in $ROUTES_FILE"
else
    backup "$ROUTES_FILE"
    ANCHOR=$(grep -nE 'g\.(Post|Put|Delete|Get)\("/:name' "$ROUTES_FILE" | tail -1 | cut -d: -f1)
    if [ -n "$ANCHOR" ]; then
        sed -i "${ANCHOR}a\\	g.Get(\"/:name/subfolders/permissions\", h.getSubfolderPermissions)\\
	g.Post(\"/:name/subfolders/permissions\", h.updateSubfolderPermissions)" "$ROUTES_FILE"
        ok "registered routes in $ROUTES_FILE (after line $ANCHOR)"
    else
        warn "no ':name' route anchor in $ROUTES_FILE — add routes manually (see above)"
    fi
fi

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 8: Makefile — migrate-acl target"
echo "═══════════════════════════════════════════════════"
if [ -f Makefile ]; then
    if grep -q '^migrate-acl:' Makefile; then
        skip "migrate-acl target"
    else
        backup Makefile
        cat >> Makefile <<'EOF'

migrate-acl:
	docker exec samba-server /usr/local/bin/migrate-acl.sh
EOF
        ok "added 'make migrate-acl'"
    fi
else
    warn "Makefile not found — run migration with: docker exec samba-server /usr/local/bin/migrate-acl.sh"
fi

echo ""
echo "═══════════════════════════════════════════════════"
echo " PART 9: sanity checks"
echo "═══════════════════════════════════════════════════"
# LF line endings for shell scripts (critical: they run inside Linux container)
for f in samba/sync-share-acl.sh samba/migrate-acl.sh; do
    if file "$f" 2>/dev/null | grep -q CRLF; then
        sed -i 's/\r$//' "$f"
        ok "converted $f to LF"
    fi
done
chmod +x samba/sync-share-acl.sh samba/migrate-acl.sh 2>/dev/null || true

if command -v gofmt >/dev/null 2>&1; then
    gofmt -w "$NEW_GO" && ok "gofmt $NEW_GO"
else
    warn "gofmt not on host — will be checked during docker build"
fi

# leftover force user anywhere?
LEFT=$(grep -rn 'force user' samba/ backend/ --include='*.go' --include='*.template' 2>/dev/null | grep -v _test | grep -v Binary || true)
if [ -n "$LEFT" ]; then
    warn "residual 'force user' references:"
    echo "$LEFT" | sed 's/^/      /'
else
    ok "no residual 'force user' in template/Go code"
fi

echo ""
echo "═══════════════════════════════════════════════════"
printf ' DONE — %d warning(s). Backups in %s\n' "$WARNINGS" "$BACKUP_DIR"
echo "═══════════════════════════════════════════════════"
cat <<'EOF'

 Next steps (in order):
   1. git diff                     # review every change
   2. make test                    # Go must compile & pass
   3. make up                      # rebuild both images
   4. make migrate-acl             # backfill ACLs for existing shares
   5. Regression: smbclient //localhost/test01 -U <user> -c 'ls'   # old shares must still work
   6. Test new endpoint:
      curl -X POST http://localhost:<port>/api/shares/test01/subfolders/permissions \
        -H 'Content-Type: application/json' \
        -d '{"subfolder_path":"docs","username":"user01","permissions":"rx","recursive":true}'
EOF

