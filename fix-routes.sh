#!/bin/bash
set -e
PROJECT_DIR="/mnt/ScriptDataPool/smb-manager"
WEBAPP_DIR="$PROJECT_DIR/webapp"

echo "[1/2] Fixing route order in shares.py..."
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

# ─── ⚠️ FIXED: Static routes MUST come BEFORE /{share_name} ──

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
    """อัพเดท Global ABSE"""
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

# ─── List All Shares ──────────────────────
@router.get("")
async def list_shares():
    try:
        return {"shares": get_config().get_all_shares()}
    except Exception as e:
        raise HTTPException(500, detail=str(e))

# ─── Create Share ─────────────────────────
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
            "guest_ok":   share.guest_ok,
            "abse":       share.abse,
        })
        ex.reload_samba()

        AuditService.log(
            action="CREATE", actor=get_actor(request),
            resource_type="SHARE", resource_name=share.name,
            status="success", details={"path": share.path, "abse": share.abse}
        )
        return {"message": f"Share '{share.name}' created"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

# ─── ABSE per Share ── ⚠️ BEFORE /{share_name} ──
@router.patch("/{share_name}/abse")
async def toggle_share_abse(share_name: str, enabled: bool, request: Request):
    """เปิด/ปิด ABSE สำหรับ Share นี้"""
    try:
        cfg = get_config()
        if not cfg.get_share(share_name):
            raise HTTPException(404, f"Share '{share_name}' not found")

        cfg.set_share_abse(share_name, enabled)
        get_executor().reload_samba()

        AuditService.log(
            action="UPDATE", actor=get_actor(request),
            resource_type="SHARE", resource_name=share_name,
            status="success", details={"abse": enabled}
        )
        return {"message": f"ABSE {'enabled' if enabled else 'disabled'} for '{share_name}'"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

# ─── Permissions ── ⚠️ BEFORE /{share_name} ──
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
            action="PERMISSION_CHANGE", actor=get_actor(request),
            resource_type="SHARE", resource_name=share_name,
            status="success", details={"type": perm.permission_type, "users": perm.users}
        )
        return {"message": "Permissions updated"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

# ─── Get/Update/Delete Share ── ⚠️ LAST (parameterized) ──
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
            action="UPDATE", actor=get_actor(request),
            resource_type="SHARE", resource_name=share_name,
            status="success", details=update_dict
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
            action="DELETE", actor=get_actor(request),
            resource_type="SHARE", resource_name=share_name,
            status="success"
        )
        return {"message": f"Share '{share_name}' deleted"}
    except Exception as e:
        raise HTTPException(500, detail=str(e))
PYEOF

echo "[2/2] Restarting webapp..."
cd "$PROJECT_DIR"
docker compose restart webapp
sleep 5
echo "✅ Done!"

