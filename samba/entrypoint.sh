#!/bin/bash

set -e

CONF="/etc/samba/smb.conf"
SHARES_CONF="/etc/samba/shares.conf"

# ✅ Global config: copy template เสมอ (AD settings ต้องถูกต้องทุก start)
echo "[*] Provisioning smb.conf (global) from template..."
cp /smb.conf.template "$CONF"

# ✅ Shares config: สร้างเฉพาะถ้ายังไม่มี (Web UI จัดการไฟล์นี้)
if [ ! -f "$SHARES_CONF" ]; then
    echo "[*] Creating empty shares.conf (first run)..."
    cat > "$SHARES_CONF" << 'EOF'
# ===========================================
# SMB Shares - Managed by SMB Manager Web UI
# DO NOT EDIT MANUALLY
# ===========================================
EOF
fi

# ตรวจ config ก่อน start
echo "[*] Validating smb.conf..."
testparm -s "$CONF" > /dev/null 2>&1 && echo "[*] Config OK" || echo "[!] Config WARNING - check logs"

mkdir -p /var/log/samba
chmod 755 /var/log/samba
mkdir -p /var/run/samba

echo "[*] Starting nmbd..."
nmbd -F --no-process-group &
sleep 2

echo "[*] Starting winbindd..."
winbindd -F --no-process-group &
sleep 3

# รอ winbind พร้อม (สำคัญสำหรับ AD)
echo "[*] Waiting for winbind to be ready..."
for i in $(seq 1 10); do
    if wbinfo --ping-dc > /dev/null 2>&1; then
        echo "[*] Winbind connected to DC ✓"
        break
    fi
    echo "[*] Waiting... ($i/10)"
    sleep 2
done

echo "[*] Starting smbd..."
exec smbd -F --no-process-group

