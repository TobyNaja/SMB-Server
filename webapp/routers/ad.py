from fastapi import APIRouter, HTTPException, Request, Query
from services.ldap_service import LdapService
from services.audit_service import AuditService
from config import settings
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/ad", tags=["active-directory"])

def get_actor(request: Request) -> str:
    return getattr(request.state, 'username', 'unknown')

@router.get("/status")
async def get_status():
    """ดูสถานะ LDAP Connection"""
    result = LdapService.test_connection()
    return {
        "ldap_server": settings.ldap_server,
        "domain": settings.ldap_domain,
        "base_dn": settings.ldap_base_dn,
        "bind_dn": settings.ldap_bind_dn,
        "connected": result.get("ok", False),
        "error": result.get("error", None)
    }

@router.get("/users")
async def search_users(
    q: str = Query("", description="ค้นหาด้วย username หรือ ชื่อ"),
    ou: str = Query(None, description="OU เฉพาะ เช่น OU=Lecturer"),
    limit: int = Query(50, le=200)
):
    """ค้นหา AD Users"""
    try:
        users = LdapService.search_users(query=q, ou=ou, limit=limit)
        return {"users": users, "count": len(users), "domain": settings.ldap_domain}
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.get("/users/{username}")
async def get_user(username: str):
    """ดู AD User คนเดียว"""
    user = LdapService.get_user(username)
    if not user:
        raise HTTPException(404, detail=f"User '{username}' not found in AD")
    return user

@router.get("/groups")
async def search_groups(
    q: str = Query("", description="ค้นหาด้วยชื่อ group"),
    limit: int = Query(50, le=200)
):
    """ค้นหา AD Groups"""
    try:
        groups = LdapService.search_groups(query=q, limit=limit)
        return {"groups": groups, "count": len(groups), "domain": settings.ldap_domain}
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.get("/ous")
async def list_ous():
    """ดู OUs ทั้งหมด"""
    return {
        "ous": [
            {"name": "Lecturer",   "dn": f"OU=Lecturer,{settings.ldap_base_dn}",   "description": "อาจารย์"},
            {"name": "Staff",      "dn": f"OU=Staff,{settings.ldap_base_dn}",       "description": "เจ้าหน้าที่"},
            {"name": "Student",    "dn": f"OU=Student,{settings.ldap_base_dn}",     "description": "นักศึกษา"},
            {"name": "Group",      "dn": f"OU=Group,{settings.ldap_base_dn}",       "description": "Groups"},
            {"name": "Service Accounts", "dn": f"OU=Service Accounts,{settings.ldap_base_dn}", "description": "Service Accounts"},
        ],
        "domain": settings.ldap_domain
    }
