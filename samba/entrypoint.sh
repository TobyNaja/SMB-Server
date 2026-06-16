#!/bin/bash
set -e

CONF="/etc/samba/smb.conf"

echo "[*] Checking smb.conf..."
if [ ! -f "$CONF" ]; then
    echo "[*] Init smb.conf from template"
    cp /smb.conf.template "$CONF"
fi

mkdir -p /var/log/samba
chmod 755 /var/log/samba

# Point winbind to TrueNAS host socket
mkdir -p /var/run/samba

echo "[*] Starting nmbd..."
nmbd -F --no-process-group &

echo "[*] Waiting for nmbd..."
sleep 3

echo "[*] Starting smbd (using TrueNAS winbind)..."
exec smbd -F --no-process-group
