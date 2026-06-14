from fastapi import APIRouter, Request
from services.audit_service import AuditService
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/audit", tags=["audit"])

@router.get("/logs")
async def get_audit_logs(
    limit: int = 100,
    action: str = None,
    actor: str = None,
    request: Request = None
):
    """📌 GET /api/audit/logs - ดึง Audit Logs"""
    logs = AuditService.get_logs(
        limit=limit,
        action_filter=action,
        actor_filter=actor
    )
    
    return {
        "logs": logs,
        "count": len(logs)
    }
