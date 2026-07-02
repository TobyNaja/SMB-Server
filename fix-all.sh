#!/bin/bash
set -e

cd /mnt/ScriptDataPool/SMB-Server

echo "════════════════════════════════════════════════════════════════"
echo "🔧 MASTER FIX SCRIPT — Fixing All Issues"
echo "════════════════════════════════════════════════════════════════"

# ─────────────────────────────────────────────────────────────────────
# PHASE 1: Restore test file and inspect code
# ─────────────────────────────────────────────────────────────────────
echo ""
echo "📋 PHASE 1: Inspect current state"

if [ -f ./backend/internal/httpapi/shares_test.go.bak ]; then
    echo "✅ Restoring shares_test.go from backup..."
    mv ./backend/internal/httpapi/shares_test.go.bak ./backend/internal/httpapi/shares_test.go
fi

echo "📄 Checking audit service methods..."
grep -n "func.*audit\|type.*Service" ./backend/internal/audit/audit.go | head -10

echo ""
echo "📄 Current shares.go issues:"
echo "   Line 92 (CreateShare chmod): $(sed -n '92p' ./backend/internal/httpapi/shares.go | grep -o 'chmod.*')"
echo "   Line 141 (DeleteShare validation): $(sed -n '141p' ./backend/internal/httpapi/shares.go)"

# ─────────────────────────────────────────────────────────────────────
# PHASE 2: Fix shares.go with Python patcher
# ─────────────────────────────────────────────────────────────────────
echo ""
echo "🔨 PHASE 2: Patching shares.go"

python3 << 'PATCHEOF'
import re

with open('./backend/internal/httpapi/shares.go', 'r') as f:
    content = f.read()

# Fix 1: DeleteShare — return 404 if not found
old_delete = r'func \(h \*sharesHandlers\) handleDeleteShare\(c \*fiber\.Ctx\) error \{.*?h\.parser\(\)\.DeleteShare\(name\)\n'
new_delete = '''func (h *sharesHandlers) handleDeleteShare(c *fiber.Ctx) error {
	name := c.Params("name")

	shares, _ := h.parser().GetShares()
	found := false
	for _, s := range shares {
		if s.Name == name {
			found = true
			break
		}
	}
	if !found {
		return c.Status(404).JSON(fiber.Map{"detail": "Share not found"})
	}

	h.parser().DeleteShare(name)
'''

if 'handleDeleteShare' in content:
    # Find the function and patch it
    start = content.find('func (h *sharesHandlers) handleDeleteShare')
    if start != -1:
        end = content.find('h.parser().DeleteShare(name)', start) + len('h.parser().DeleteShare(name)')
        old_func_part = content[start:end]
        
        # Replace only the logic before DeleteShare call
        new_func_part = old_func_part.replace(
            'func (h *sharesHandlers) handleDeleteShare(c *fiber.Ctx) error {\n\tname := c.Params("name")',
            '''func (h *sharesHandlers) handleDeleteShare(c *fiber.Ctx) error {
\tname := c.Params("name")

\tshares, _ := h.parser().GetShares()
\tfound := false
\tfor _, s := range shares {
\t\tif s.Name == name {
\t\t\tfound = true
\t\t\tbreak
\t\t}
\t}
\tif !found {
\t\treturn c.Status(404).JSON(fiber.Map{"detail": "Share not found"})
\t}'''
        )
        content = content[:start] + new_func_part + content[end:]

# Fix 2: CreateShare — add audit logging after share is created
if '.AddShare(req)' in content and 'CreateShare_WritesAudit' in open('./backend/internal/httpapi/shares_test.go').read():
    content = content.replace(
        'if err := h.parser().AddShare(req); err != nil {',
        '''if err := h.parser().AddShare(req); err != nil {'''
    )
    # Find the ReloadSamba call after AddShare and add audit before it
    content = content.replace(
        'h.exec.ReloadSamba()',
        '''// Audit the share creation
\th.audit.LogAction("create_share", map[string]interface{}{
\t\t"share_name": req.Name,
\t\t"path":       req.Path,
\t\t"comment":    req.Comment,
\t}, c)
\th.exec.ReloadSamba()'''
    )

# Fix 3: ToggleABSE — add chmod 0750 and audit logging
if 'handleToggleABSE' in content:
    # Find ToggleABSE and ensure it has chmod logic
    toggle_start = content.find('func (h *sharesHandlers) handleToggleABSE')
    if toggle_start != -1:
        toggle_end = content.find('h.exec.ReloadSamba()', toggle_start)
        toggle_section = content[toggle_start:toggle_end]
        
        # Add chmod 0750 if enabled and audit logging
        if 'chmod 0750' not in toggle_section:
            insert_pos = content.find('h.exec.ReloadSamba()', toggle_start)
            content = (content[:insert_pos] + 
                '''\tif req.Enabled {
\t\tshares, _ := h.parser().GetShares()
\t\tfor _, s := range shares {
\t\t\tif s.Name == name {
\t\t\t\tif s.Path != "" {
\t\t\t\t\th.exec.Execute(fmt.Sprintf("chmod 0750 %s", s.Path))
\t\t\t\t}
\t\t\t\tbreak
\t\t\t}
\t\t}
\t}

\t// Audit
\th.audit.LogAction("toggle_abse", map[string]interface{}{
\t\t"share_name": name,
\t\t"enabled":    req.Enabled,
\t}, c)

\t''' + content[insert_pos:])

with open('./backend/internal/httpapi/shares.go', 'w') as f:
    f.write(content)

print("✅ shares.go patched successfully")
PATCHEOF

# ─────────────────────────────────────────────────────────────────────
# PHASE 3: Get correct service name and rebuild
# ─────────────────────────────────────────────────────────────────────
echo ""
echo "🐳 PHASE 3: Rebuilding Docker services"

SERVICE_NAME=$(sudo docker compose config --services | grep -i -E "web|app|server|smb" | head -1)
echo "📦 Found service: $SERVICE_NAME"

# Test build first
echo "🔨 Building..."
sudo docker compose build --no-cache "$SERVICE_NAME" 2>&1 | tail -20

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "🚀 Starting service..."
    sudo docker compose up -d "$SERVICE_NAME"
    sleep 5
else
    echo "❌ Build failed! Running diagnostics..."
    grep "FAIL\|Error" <(sudo docker compose build "$SERVICE_NAME" 2>&1) | head -10
    exit 1
fi

# ─────────────────────────────────────────────────────────────────────
# PHASE 4: Fix Samba configuration and permissions
# ─────────────────────────────────────────────────────────────────────
echo ""
echo "🔧 PHASE 4: Fixing Samba share configuration"

sudo docker exec samba-server /bin/bash -c '
  # Create shares directory
  mkdir -p /etc/samba/shares
  
  # Fix share path permissions
  mkdir -p /mnt/share/test01
  chown root:root /mnt/share/test01
  chmod 0755 /mnt/share/test01
  
  echo "📁 Share permissions:"
  ls -la /mnt/share/test01
'

echo ""
echo "🔄 PHASE 5: Checking/fixing Winbind for AD authentication"

sudo docker exec samba-server /bin/bash -c '
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "Winbind Status Check"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  
  # Check if winbind is running
  ps aux | grep -E "winbind|smbd" | grep -v grep
  
  echo ""
  echo "Checking domain connection..."
  net ads testjoin -S 10.70.37.143 2>&1 || echo "⚠️ Domain join test failed"
  
  echo ""
  echo "Checking if user can be resolved..."
  getent passwd IT/it67070109 || echo "⚠️ User not resolvable via winbind"
  
  echo ""
  echo "Checking wbinfo..."
  wbinfo -u 2>&1 | head -5 || echo "⚠️ wbinfo failed"
'

# ─────────────────────────────────────────────────────────────────────
# PHASE 6: Update smb.conf for proper AD authentication
# ─────────────────────────────────────────────────────────────────────
echo ""
echo "📝 PHASE 6: Updating smb.conf for AD user support"

sudo docker exec samba-server /bin/bash -c '
  # Backup original
  cp /etc/samba/smb.conf /etc/samba/smb.conf.bak.$(date +%s)
  
  # Make sure/etc/samba/shares.conf exists and is included
  if ! grep -q "include = /etc/samba/shares.conf" /etc/samba/smb.conf; then
    echo "include = /etc/samba/shares.conf" >> /etc/samba/smb.conf
  fi
  
  # Verify critical settings for AD are present
  cat >> /etc/samba/smb.conf << '\''SMB_EOF'\''

# Ensure proper AD integration settings
[global]
  # Ensure winbind is configured for AD user resolution
  idmap config IT : backend = rid
  idmap config IT : range = 100000-999999
  
  # Offline logon and ticket refresh
  winbind offline logon = yes
  winbind refresh tickets = yes
  
  # Use default domain for username resolution
  winbind use default domain = yes
  
  # Enum for proper user/group visibility
  winbind enum users = yes
  winbind enum groups = yes
SMB_EOF

  # Reload config
  smbcontrol smbd reload-config
  sleep 1
  
  echo "✅ smb.conf updated"
  echo ""
  echo "Current test01 share config:"
  testparm -s 2>&1 | grep -A 12 "\[test01\]"
'

# ─────────────────────────────────────────────────────────────────────
# PHASE 7: Verify API and test access
# ─────────────────────────────────────────────────────────────────────
echo ""
echo "✅ PHASE 7: Final verification"

echo ""
echo "1️⃣  API Status Check:"
curl -s http://localhost:8080/api/health 2>/dev/null && echo "" || echo "❌ API not responding"

echo ""
echo "2️⃣  AD Connection Status:"
curl -s -c /tmp/test_cookies.txt -b /tmp/test_cookies.txt \
  http://localhost:8080/api/ad/status 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "⚠️ Need to login first"

echo ""
echo "3️⃣  Build Test Results:"
echo "Checking if tests pass now..."
sudo docker exec $(sudo docker compose ps -q ${SERVICE_NAME}) /bin/bash -c 'go test ./... 2>&1' | grep -E "ok|FAIL|PASS" | tail -10

echo ""
echo "4️⃣  Samba Share List:"
sudo docker exec samba-server /bin/bash -c 'smbclient -L 127.0.0.1 -N 2>&1' | grep "\[test01\]" && echo "✅ test01 share visible" || echo "❌ Share not visible"

# ─────────────────────────────────────────────────────────────────────
# PHASE 8: Troubleshooting guide
# ─────────────────────────────────────────────────────────────────────
echo ""
echo "════════════════════════════════════════════════════════════════"
echo "📋 Next Steps / Troubleshooting"
echo "════════════════════════════════════════════════════════════════"

cat << 'GUIDE'

❌ If build still fails:
   → Check logs: sudo docker compose build $SERVICE_NAME 2>&1 | grep -i error
   
❌ If "NT_STATUS_LOGON_FAILURE" still appears:
   Action 1: Check winbind is running
      sudo docker exec samba-server ps aux | grep winbind
   
   Action 2: Check machine account
      sudo docker exec samba-server net ads status
   
   Action 3: Rejoin domain
      sudo docker exec samba-server /bin/bash -c '
        net ads leave -U administrator@IT.KMITL.AC.TH
        sleep 2
        net ads join -U administrator@IT.KMITL.AC.TH
      '
   
   Action 4: Restart daemons
      sudo docker compose restart samba-server
      sleep 5
      sudo docker exec samba-server systemctl restart smbd winbind

❌ If share permissions still wrong:
   → Run: sudo docker exec samba-server chmod -R 0775 /mnt/share/test01

✅ To test Windows access:
   net use Z: \\10.0.36.11\test01 /user:IT\it67070109 "YourPassword"

✅ To test Linux CLI:
   sudo docker exec samba-server smbclient //127.0.0.1/test01 \
     -U "IT\\it67070109%YourPassword" -c "ls; quit"

GUIDE

echo ""
echo "✅ Master fix script completed!"
