#!/bin/bash
# fix-after-test.sh — fix 3 issues found by `make up` test run. Idempotent.
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

NEW_GO="backend/internal/httpapi/subfolder_permissions.go"
CONF_TEST="backend/internal/samba/conf_test.go"
ok()   { printf '  \033[32m✅ %s\033[0m\n' "$1"; }
warn() { printf '  \033[33m⚠️  %s\033[0m\n' "$1"; }
die()  { printf '  \033[31m❌ %s\033[0m\n' "$1"; exit 1; }

echo "── FIX 1: delete obsolete test (asserts removed force-user behavior) ──"
if grep -q 'func TestCreateShare_WritesForceUserAndGroup' "$CONF_TEST"; then
    awk '
        /^func TestCreateShare_WritesForceUserAndGroup\(/ {del=1}
        del { if ($0 == "}") del=0; next }
        { print }
    ' "$CONF_TEST" > "$CONF_TEST.tmp" && mv "$CONF_TEST.tmp" "$CONF_TEST"
    ok "removed TestCreateShare_WritesForceUserAndGroup from $CONF_TEST"
else
    ok "test already removed"
fi

echo "── FIX 2: strip residual force user/group from live shares.conf ──"
if grep -qE '^\s*force (user|group)\s*=' samba/shares.conf; then
    sed -i '/^\s*force user\s*=/d; /^\s*force group\s*=/d' samba/shares.conf
    ok "cleaned samba/shares.conf (test02 had them)"
else
    ok "shares.conf already clean"
fi

echo "── FIX 3: adapt Go code to single-return Execute ──"
IFACE_FILE=$(grep -rln 'type Executor interface' backend/ | head -1)
[ -n "$IFACE_FILE" ] || die "Executor interface not found"
RET=$(grep -E '^\s*Execute\(' "$IFACE_FILE" | head -1 | sed -E 's/.*\)\s*//')
ok "interface: $IFACE_FILE — Execute returns '$RET'"

# 3a. call sites in new endpoint -> ExecuteOutput
sed -i \
    -e 's/if out, err := h\.exec\.Execute(cmd)/if out, err := h.exec.ExecuteOutput(cmd)/' \
    -e 's/out, err := h\.exec\.Execute("getfacl/out, err := h.exec.ExecuteOutput("getfacl/' \
    "$NEW_GO"
ok "call sites in $NEW_GO now use ExecuteOutput"

# 3b. add ExecuteOutput to the interface
if ! grep -q 'ExecuteOutput' "$IFACE_FILE"; then
    sed -i '/type Executor interface/,/^}/ s/^\(\s*Execute(.*\)$/\1\n\tExecuteOutput(command string) (string, error)/' "$IFACE_FILE"
    ok "added ExecuteOutput to interface"
fi

# 3c. add ExecuteOutput to every implementation (incl. fakes/mocks in tests)
grep -rln ') Execute(' backend/ --include='*.go' | while read -r f; do
    grep -q 'func .* ExecuteOutput(' "$f" && continue
    DECL=$(grep -oE 'func \([a-zA-Z_]+ \*?[A-Za-z0-9_]+\) Execute\(' "$f" | head -1)
    [ -n "$DECL" ] || continue
    RECV=$(echo "$DECL" | sed -E 's/func \((.*)\) Execute\(.*/\1/')   # e.g. "e *ShellExecutor"
    VAR=${RECV%% *}
    if grep -q 'exec\.Command' "$f"; then
        # real shell executor -> capture output properly
        cat >> "$f" <<GOEOF

func ($RECV) ExecuteOutput(command string) (string, error) {
	out, err := exec.Command("sh", "-c", command).CombinedOutput()
	return string(out), err
}
GOEOF
        ok "$f: real ExecuteOutput (CombinedOutput)"
    else
        # fake/mock -> delegate, keeps interface satisfied
        case "$RET" in
            error)  BODY="return \"\", $VAR.Execute(command)" ;;
            string) BODY="return $VAR.Execute(command), nil" ;;
            *)      warn "$f: unknown return '$RET' — add ExecuteOutput manually"; continue ;;
        esac
        cat >> "$f" <<GOEOF

func ($RECV) ExecuteOutput(command string) (string, error) {
	$BODY
}
GOEOF
        ok "$f: delegating ExecuteOutput"
    fi
done

command -v gofmt >/dev/null 2>&1 && gofmt -w backend/ && ok "gofmt done" || true
echo ""
echo "Now run: make up && make migrate-acl"
