import json
import os
from datetime import datetime
from typing import List, Dict, Optional
from config import settings
import logging

logger = logging.getLogger(__name__)

class AuditService:
    """บันทึก Audit Log ทั้งหมด"""

    @staticmethod
    def _get_log_file() -> str:
        return settings.audit_log_path

    @staticmethod
    def _ensure_log_file():
        """สร้างไฟล์ถ้าไม่มี"""
        filepath = AuditService._get_log_file()
        os.makedirs(os.path.dirname(filepath), exist_ok=True)
        if not os.path.exists(filepath):
            with open(filepath, 'w') as f:
                json.dump([], f)

    @staticmethod
    def log(
        action: str,
        actor: str,
        resource_type: str,
        resource_name: str,
        status: str = "success",
        details: Optional[Dict] = None,
        client_ip: Optional[str] = None
    ) -> bool:
        """บันทึก Audit Log"""
        try:
            AuditService._ensure_log_file()
            
            log_entry = {
                "timestamp": datetime.utcnow().isoformat(),
                "action": action,
                "actor": actor,
                "resource_type": resource_type,
                "resource_name": resource_name,
                "status": status,
                "details": details or {},
                "client_ip": client_ip
            }
            
            filepath = AuditService._get_log_file()
            
            logs = []
            try:
                with open(filepath, 'r') as f:
                    logs = json.load(f)
            except (json.JSONDecodeError, FileNotFoundError):
                logs = []
            
            logs.append(log_entry)
            logs = logs[-10000:]
            
            with open(filepath, 'w') as f:
                json.dump(logs, f, indent=2)
            
            logger.info(f"[AUDIT] {action} on {resource_type}/{resource_name} by {actor}: {status}")
            return True
            
        except Exception as e:
            logger.error(f"Audit logging failed: {e}")
            return False

    @staticmethod
    def get_logs(
        limit: int = 100,
        action_filter: Optional[str] = None,
        actor_filter: Optional[str] = None
    ) -> List[Dict]:
        """ดึง Audit Logs"""
        try:
            limit = min(limit, 500)
            AuditService._ensure_log_file()
            
            filepath = AuditService._get_log_file()
            with open(filepath, 'r') as f:
                logs = json.load(f)
            
            if action_filter:
                logs = [l for l in logs if l.get("action") == action_filter]
            if actor_filter:
                logs = [l for l in logs if l.get("actor") == actor_filter]
            
            return logs[-limit:][::-1]
            
        except Exception as e:
            logger.error(f"Failed to retrieve audit logs: {e}")
            return []
