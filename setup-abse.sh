#!/bin/bash
set -e

PROJECT_DIR="/mnt/ScriptDataPool/smb-manager"
WEBAPP_DIR="$PROJECT_DIR/webapp"

echo "========================================"
echo "  ABSE Setup"
echo "========================================"

# ──────────────────────────────────────────
# 1. Patch config_parser.py (เพิ่ม ABSE)
# ──────────────────────────────────────────
echo "[1/3] Patching config_parser.py..."
cat > "$WEBAPP_DIR/services/config_parser.py" << 'PYEOF'
import os
import logging
from typing import Dict, List, Optional

logger = logging.getLogger(__name__)

SYSTEM_SECTIONS = {'global', 'homes', 'printers', 'print$', 'netlogon', 'sysvol'}

class SmbConfParser:
    def __init__(self, config_path: str):
        self.config_path = config_path
        self.sections: Dict[str, Dict[str, str]] = {}
        self._parse()

    def _parse(self):
        self.sections = {}
        if not os.path.exists(self.config_path):
            logger.warning(f"smb.conf not found: {self.config_path}")
            self.sections = {
                'global': {
                    'workgroup': 'WORKGROUP',
                    'security': 'user',
                    'passdb backend': 'tdbsam'
                }
            }
            return

        with open(self.config_path, 'r') as f:
            section = None
            for line in f:
                line = line.strip()
                if not line or line.startswith(('#', ';')):
                    continue
                if line.startswith('[') and line.endswith(']'):
                    section = line[1:-1].strip()
                    self.sections[section] = {}
                elif section and '=' in line:
                    key, _, val = line.partition('=')
                    self.sections[section][key.strip()] = val.strip()

    def _save(self):
        os.makedirs(os.path.dirname(self.config_path), exist_ok=True)
        with open(self.config_path, 'w') as f:
            f.write("# Auto-managed by SMB Permission Manager\n\n")
            for section, params in self.sections.items():
                f.write(f"[{section}]\n")
                for k, v in params.items():
                    f.write(f"    {k} = {v}\n")
                f.write("\n")
        logger.info("smb.conf saved")

    # ─── Global Settings ──────────────────
    def get_global(self) -> dict:
        """ดู global settings"""
        g = self.sections.get('global', {})
        return {
            "workgroup": g.get('workgroup', 'WORKGROUP'),
            "abse": g.get('access based share enum', 'no').lower() == 'yes',
            "server_string": g.get('server string', 'Samba Server'),
        }

    def set_global_abse(self, enabled: bool) -> bool:
        """เปิด/ปิด ABSE แบบ Global"""
        if 'global' not in self.sections:
            self.sections['global'] = {}
        self.sections['global']['access based share enum'] = 'yes' if enabled else 'no'
        self._save()
        logger.info(f"Global ABSE set to: {enabled}")
        return True

    # ─── Shares ───────────────────────────
    def get_shares(self) -> List[str]:
        return [s for s in self.sections if s not in SYSTEM_SECTIONS]

    def get_share(self, name: str) -> Optional[dict]:
        if name not in self.sections:
            return None
        d = self.sections[name]
        return {
            "name": name,
            "path": d.get('path', ''),
            "comment": d.get('comment', ''),
            "browseable": d.get('browseable', 'yes').lower() == 'yes',
            "read_only": d.get('read only', 'no').lower() == 'yes',
            "guest_ok": d.get('guest ok', 'no').lower() == 'yes',
            "abse": d.get('access based share enum', 'no').lower() == 'yes',
            "valid_users": [u for u in d.get('valid users', '').split() if u],
            "write_list":  [u for u in d.get('write list',  '').split() if u],
            "read_list":   [u for u in d.get('read list',   '').split() if u],
            "admin_users": [u for u in d.get('admin users', '').split() if u],
            "invalid_users": [u for u in d.get('invalid users', '').split() if u],
            "create_mask": d.get('create mask', '0755'),
            "directory_mask": d.get('directory mask', '0755'),
        }

    def get_all_shares(self) -> List[dict]:
        return [self.get_share(s) for s in self.get_shares()]

    def create_share(self, name: str, path: str, comment: str = "") -> bool:
        if name in self.sections:
            return False
        self.sections[name] = {
            'comment': comment or f'{name} share',
            'path': path,
            'browseable': 'yes',
            'read only': 'no',
            'guest ok': 'no',
            'access based share enum': 'no',
            'create mask': '0755',
            'directory mask': '0755',
        }
        self._save()
        return True

    def update_share(self, name: str, updates: dict) -> bool:
        if name not in self.sections:
            return False
        key_map = {
            'read_only':    'read only',
            'guest_ok':     'guest ok',
            'abse':         'access based share enum',
        }
        bool_fields = {'browseable', 'read only', 'guest ok', 'access based share enum'}

        for k, v in updates.items():
            real_key = key_map.get(k, k)
            if real_key in bool_fields:
                self.sections[name][real_key] = 'yes' if v else 'no'
            else:
                self.sections[name][real_key] = str(v)
        self._save()
        return True

    def delete_share(self, name: str) -> bool:
        if name in self.sections:
            del self.sections[name]
            self._save()
            return True
        return False

    def set_valid_users(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['valid users'] = ' '.join(users)
        self._save()
        return True

    def set_write_list(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['write list'] = ' '.join(users)
        self._save()
        return True

    def set_read_list(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['read list'] = ' '.join(users)
        self._save()
        return True

    def set_admin_users(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['admin users'] = ' '.join(users)
        self._save()
        return True

    def set_invalid_users(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['invalid users'] = ' '.join(users)
        self._save()
        return True

    def set_share_abse(self, name: str, enabled: bool) -> bool:
        """เปิด/ปิด ABSE per share"""
        if name not in self.sections:
            return False
        self.sections[name]['access based share enum'] = 'yes' if enabled else 'no'
        self._save()
        logger.info(f"Share '{name}' ABSE set to: {enabled}")
        return True
PYEOF
echo "✅ config_parser.py patched"

# ──────────────────────────────────────────
# 2. Patch routers/shares.py (เพิ่ม ABSE API)
# ──────────────────────────────────────────
echo "[2/3] Patching routers/shares.py..."
cat > "$WEBAPP_DIR/routers/shares.py" << 'PYEOF'
from fastapi import APIRouter, HTTPException, Request
from pydantic import BaseModel
from typing import List, Optional
from services.config_parser import SmbConfParser
from services.docker_executor import DockerExecutor
from services.audit_service import AuditService
from config import settings
import os
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/shares", tags=["shares"])

class ShareCreate(BaseModel):
    name: str
    path: str
    comment: str = ""
    browseable: bool = True
    guest_ok: bool = False
    abse: bool = False

class ShareUpdate(BaseModel):
    comment: Optional[str] = None
    browseable: Optional[bool] = None
    guest_ok: Optional[bool] = None
    read_only: Optional[bool] = None
    abse: Optional[bool] = None

class PermissionUpdate(BaseModel):
    users: List[str] = []
    permission_type: str

class GlobalSettings(BaseModel):
    abse: bool

def get_config():
    return SmbConfParser(os.path.join(settings.samba_config_path, 'smb.conf'))

def get_executor():
    return DockerExecutor(settings.samba_container)

def get_actor(request: Request) -> str:
    return getattr(request.state, 'username', 'unknown')

# ─── Global Settings ──────────────────────
@router.get("/global")
async def get_global_settings():
    """ดู Global SMB settings"""
    try:
        return get_config().get_global()
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.patch("/global")
async def update_global_settings(body: GlobalSettings, request: Request):
    """อัพเดท Global settings (เช่น ABSE)"""
    try:
        cfg = get_config()
        cfg.set_global_abse(body.abse)
        get_executor().reload_samba()

        AuditService.log(
            action="UPDATE_GLOBAL",
            actor=get_actor(request),
            resource_type="SHARE",
            resource_name="global",
            status="success",
            details={"abse": body.abse}
        )
        return {"message": f"Global ABSE set to: {body.abse}"}
    except Exception as e:
        raise HTTPException(500, detail=str(e))

# ─── Shares CRUD ──────────────────────────
@router.get("")
async def list_shares():
    try:
        return {"shares": get_config().get_all_shares()}
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.get("/{share_name}")
async def get_share(share_name: str):
    try:
        share = get_config().get_share(share_name)
        if not share:
            raise HTTPException(404, "Share not found")
        return share
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.post("")
async def create_share(share: ShareCreate, request: Request):
    try:
        ex = get_executor()
        ex.execute(f"mkdir -p {share.path} && chmod 777 {share.path}")

        cfg = get_config()
        if not cfg.create_share(share.name, share.path, share.comment):
            raise HTTPException(400, "Share already exists")

        cfg.update_share(share.name, {
            "browseable": share.browseable,
            "guest_ok": share.guest_ok,
            "abse": share.abse,
        })
        ex.reload_samba()

        AuditService.log(
            action="CREATE",
            actor=get_actor(request),
            resource_type="SHARE",
            resource_name=share.name,
            status="success",
            details={"path": share.path, "abse": share.abse}
        )
        return {"message": f"Share '{share.name}' created"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.patch("/{share_name}")
async def update_share(share_name: str, updates: ShareUpdate, request: Request):
    try:
        cfg = get_config()
        if not cfg.get_share(share_name):
            raise HTTPException(404, "Share not found")

        update_dict = {k: v for k, v in updates.dict().items() if v is not None}
        cfg.update_share(share_name, update_dict)
        get_executor().reload_samba()

        AuditService.log(
            action="UPDATE",
            actor=get_actor(request),
            resource_type="SHARE",
            resource_name=share_name,
            status="success",
            details=update_dict
        )
        return {"message": f"Share '{share_name}' updated"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.delete("/{share_name}")
async def delete_share(share_name: str, request: Request):
    try:
        get_config().delete_share(share_name)
        get_executor().reload_samba()

        AuditService.log(
            action="DELETE",
            actor=get_actor(request),
            resource_type="SHARE",
            resource_name=share_name,
            status="success"
        )
        return {"message": f"Share '{share_name}' deleted"}
    except Exception as e:
        raise HTTPException(500, detail=str(e))

# ─── ABSE Toggle (Per Share) ──────────────
@router.patch("/{share_name}/abse")
async def toggle_abse(share_name: str, enabled: bool, request: Request):
    """เปิด/ปิด ABSE สำหรับ Share นี้"""
    try:
        cfg = get_config()
        if not cfg.get_share(share_name):
            raise HTTPException(404, "Share not found")

        cfg.set_share_abse(share_name, enabled)
        get_executor().reload_samba()

        AuditService.log(
            action="UPDATE",
            actor=get_actor(request),
            resource_type="SHARE",
            resource_name=share_name,
            status="success",
            details={"abse": enabled}
        )
        return {"message": f"ABSE {'enabled' if enabled else 'disabled'} for '{share_name}'"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

# ─── Permissions ──────────────────────────
@router.post("/{share_name}/permissions")
async def update_permissions(share_name: str, perm: PermissionUpdate, request: Request):
    try:
        cfg = get_config()
        if not cfg.get_share(share_name):
            raise HTTPException(404, "Share not found")

        perm_map = {
            'valid_users':   cfg.set_valid_users,
            'write_list':    cfg.set_write_list,
            'read_list':     cfg.set_read_list,
            'admin_users':   cfg.set_admin_users,
            'invalid_users': cfg.set_invalid_users,
        }

        if perm.permission_type not in perm_map:
            raise HTTPException(400, f"Invalid permission_type: {perm.permission_type}")

        perm_map[perm.permission_type](share_name, perm.users)
        get_executor().reload_samba()

        AuditService.log(
            action="PERMISSION_CHANGE",
            actor=get_actor(request),
            resource_type="SHARE",
            resource_name=share_name,
            status="success",
            details={"type": perm.permission_type, "users": perm.users}
        )
        return {"message": "Permissions updated"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))
PYEOF
echo "✅ routers/shares.py patched"

# ──────────────────────────────────────────
# 3. Restart webapp
# ──────────────────────────────────────────
echo "[3/3] Restarting webapp..."
cd "$PROJECT_DIR"
docker compose restart webapp
sleep 5

echo ""
echo "========================================"
echo "  ABSE Setup Complete!"
echo "========================================"
echo ""
echo "NEW API Endpoints:"
echo "  GET  /api/shares/global       <- ดู global settings"
echo "  PATCH /api/shares/global      <- เปิด/ปิด ABSE global"
echo "  PATCH /api/shares/{name}/abse <- เปิด/ปิด ABSE per share"
echo ""
