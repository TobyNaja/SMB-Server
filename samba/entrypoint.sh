#!/bin/bash
set -e

CONF="/etc/samba/smb.conf"

# บังคับก๊อปปี้ทับไปเลย ไม่ต้องเช็คว่ามีไฟล์อยู่ไหม
echo "[*] Provisioning smb.conf from template..."
cp /smb.conf.template "$CONF"

mkdir -p /var/log/samba
chmod 755 /var/log/samba
mkdir -p /var/run/samba

echo "[*] Starting nmbd..."
nmbd -F --no-process-group &
sleep 2

echo "[*] Starting winbindd..."
winbindd -F --no-process-group &
sleep 3

echo "[*] Starting smbd..."
exec smbd -F --no-process-group
