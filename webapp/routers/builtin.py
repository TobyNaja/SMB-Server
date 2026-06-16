from fastapi import APIRouter, HTTPException, Request
from pydantic import BaseModel
from services.docker_executor import DockerExecutor
from services.audit_service import AuditService
from config import settings
import json
import os
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/builtin", tags=["builtin-groups"])

# ไฟล์เก็บ memberships ของแต่ละ group
BUILTIN_STORE = "/mnt/shared/builtin_groups.json"

BUILTIN_GROUPS = {
    "Administrators": {
        "description": "Full control over Samba server — เข้าได้ทุก Share อัตโนมัติ",
        "color": "danger",
        "icon": "shield-fill-check"
    },
    "Users": {
        "description": "Standard users — user ทั่วไปที่ login ได้",
        "color": "primary",
        "icon": "people-fill"
    },
    "Guests": {
        "description": "Guest access — เข้าได้โดยไม่ต้อง login",
        "color": "secondary",
        "icon": "person-dash-fill"
    },
    "Power Users": {
        "description": "ระหว่าง Admin กับ User — สิทธิ์พิเศษบางอย่าง",
        "color": "warning",
        "icon": "lightning-fill"
    },
    "Backup Operators": {
        "description": "Backup access — สำหรับทำ Backup",
        "color": "info",
        "icon": "archive-fill"
    },
    "Print Operators": {
        "description": "Printer management — จัดการ Printer",
        "color": "dark",
        "icon": "printer-fill"
    },
}

class MemberAction(BaseModel):
    username: str

def executor():
    return DockerExecutor(settings.samba_container)

def get_actor(request: Request) -> str:
    return getattr(request.state, 'username', 'unknown')

def load_store() -> dict:
    """โหลด memberships จากไฟล์"""
    try:
        if os.path.exists(BUILTIN_STORE):
            with open(BUILTIN_STORE, 'r') as f:
                return json.load(f)
    except Exception as e:
        logger.warning(f"Load store failed: {e}")
    # Default: empty members
    return {g: [] for g in BUILTIN_GROUPS}

def save_store(data: dict) -> bool:
    """บันทึก memberships ลงไฟล์"""
    try:
        os.makedirs(os.path.dirname(BUILTIN_STORE), exist_ok=True)
        with open(BUILTIN_STORE, 'w') as f:
            json.dump(data, f, indent=2)
        return True
    except Exception as e:
        logger.error(f"Save store failed: {e}")
        return False

def apply_to_samba(group_name: str, username: str, action: str) -> dict:
    """
    Apply membership to Samba via net groupmap
    action: 'add' หรือ 'del'
    """
    ex = executor()

    # Try net sam command first
    cmd = f"net sam {'add' if action == 'add' else 'del'}mem 'BUILTIN\\\\{group_name}' '{username}' 2>&1 || true"
    result = ex.execute(cmd)
    logger.info(f"Samba {action} {username} to/from {group_name}: {result['output'][:100]}")
    return result

@router.get("")
async def list_builtin_groups():
    """ดู Builtin Groups ทั้งหมดพร้อม members"""
    store = load_store()
    result = []
    for group_name, info in BUILTIN_GROUPS.items():
        members = store.get(group_name, [])
        result.append({
            "name": group_name,
            "full_name": f"BUILTIN\\{group_name}",
            "description": info["description"],
            "color": info["color"],
            "icon": info["icon"],
            "members": members,
            "member_count": len(members)
        })
    return {"groups": result}

@router.get("/{group_name}/members")
async def get_group_members(group_name: str):
    """ดู members ของ Builtin Group"""
    if group_name not in BUILTIN_GROUPS:
        raise HTTPException(404, f"Builtin group '{group_name}' not found")

    store = load_store()
    members = store.get(group_name, [])
    info = BUILTIN_GROUPS[group_name]

    return {
        "group": group_name,
        "full_name": f"BUILTIN\\{group_name}",
        "description": info["description"],
        "members": members,
        "member_count": len(members)
    }

@router.post("/{group_name}/members")
async def add_member(group_name: str, body: MemberAction, request: Request):
    """เพิ่ม user เข้า Builtin Group"""
    if group_name not in BUILTIN_GROUPS:
        raise HTTPException(404, f"Builtin group '{group_name}' not found")

    username = body.username.strip()
    if not username:
        raise HTTPException(400, "Username is required")

    store = load_store()
    members = store.get(group_name, [])

    if username in members:
        raise HTTPException(400, f"'{username}' is already in {group_name}")

    # บันทึก
    members.append(username)
    store[group_name] = members
    save_store(store)

    # Apply to Samba (best effort)
    apply_to_samba(group_name, username, 'add')

    AuditService.log(
        action="BUILTIN_ADD_MEMBER",
        actor=get_actor(request),
        resource_type="BUILTIN_GROUP",
        resource_name=group_name,
        status="success",
        details={"username": username}
    )

    return {
        "message": f"Added '{username}' to BUILTIN\\{group_name}",
        "members": members
    }

@router.delete("/{group_name}/members/{username}")
async def remove_member(group_name: str, username: str, request: Request):
    """ลบ user ออกจาก Builtin Group"""
    if group_name not in BUILTIN_GROUPS:
        raise HTTPException(404, f"Builtin group '{group_name}' not found")

    store = load_store()
    members = store.get(group_name, [])

    if username not in members:
        raise HTTPException(400, f"'{username}' is not in {group_name}")

    # ลบ
    members = [m for m in members if m != username]
    store[group_name] = members
    save_store(store)

    # Apply to Samba (best effort)
    apply_to_samba(group_name, username, 'del')

    AuditService.log(
        action="BUILTIN_REMOVE_MEMBER",
        actor=get_actor(request),
        resource_type="BUILTIN_GROUP",
        resource_name=group_name,
        status="success",
        details={"username": username}
    )

    return {
        "message": f"Removed '{username}' from BUILTIN\\{group_name}",
        "members": members
    }
