#!/bin/bash
set -e

CONF="/etc/samba/smb.conf"

echo "[*] Checking smb.conf..."
if [ ! -f "$CONF" ]; then
    echo "[*] Init smb.conf from template"
    cp /smb.conf.template "$CONF"
fi

# สร้าง log directory
mkdir -p /var/log/samba
chmod 755 /var/log/samba

echo "[*] Starting nmbd..."
nmbd -F --no-process-group &

echo "[*] Waiting for nmbd to start..."
sleep 3

echo "[*] Starting smbd..."
exec smbd -F --no-process-group

