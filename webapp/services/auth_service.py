import bcrypt
import json
import os
from datetime import datetime, timedelta
from jose import JWTError, jwt
from typing import Optional, Tuple
from models.auth import AdminCredential
from config import settings
import logging

logger = logging.getLogger(__name__)

class AuthService:
    """จัดการ Authentication สำหรับ Admin"""

    @staticmethod
    def hash_password(password: str) -> str:
        """แฮช Password ด้วย bcrypt"""
        salt = bcrypt.gensalt(rounds=12)
        return bcrypt.hashpw(password.encode("utf-8"), salt).decode("utf-8")

    @staticmethod
    def verify_password(plain_password: str, hashed_password: str) -> bool:
        """เปรียบเทียบ Password กับแฮช"""
        try:
            return bcrypt.checkpw(
                plain_password.encode("utf-8"),
                hashed_password.encode("utf-8")
            )
        except Exception as e:
            logger.error(f"Password verification error: {e}")
            return False

    @staticmethod
    def create_access_token(username: str) -> Tuple[str, int]:
        """สร้าง JWT Token"""
        expire_delta = timedelta(minutes=settings.access_token_expire_minutes)
        expire = datetime.utcnow() + expire_delta
        
        payload = {
            "username": username,
            "exp": expire,
            "iat": datetime.utcnow()
        }
        
        token = jwt.encode(
            payload,
            settings.secret_key,
            algorithm=settings.algorithm
        )
        
        return token, int(expire_delta.total_seconds())

    @staticmethod
    def verify_token(token: str) -> Optional[str]:
        """ตรวจสอบ JWT Token"""
        try:
            payload = jwt.decode(
                token,
                settings.secret_key,
                algorithms=[settings.algorithm]
            )
            username = payload.get("username")
            if username:
                return username
        except JWTError as e:
            logger.debug(f"Token verification failed: {e}")
        return None

    @staticmethod
    def _get_admin_file() -> str:
        return settings.admin_credentials_file

    @staticmethod
    def _load_admin() -> Optional[AdminCredential]:
        """โหลด Admin credential จากไฟล์"""
        filepath = AuthService._get_admin_file()
        try:
            if os.path.exists(filepath):
                with open(filepath, 'r') as f:
                    data = json.load(f)
                    return AdminCredential(**data)
        except Exception as e:
            logger.warning(f"Failed to load admin credentials: {e}")
        return None

    @staticmethod
    def _save_admin(admin: AdminCredential) -> bool:
        """บันทึก Admin credential ลงไฟล์"""
        filepath = AuthService._get_admin_file()
        try:
            os.makedirs(os.path.dirname(filepath), exist_ok=True)
            with open(filepath, 'w') as f:
                json.dump({
                    "username": admin.username,
                    "hashed_password": admin.hashed_password,
                    "created_at": admin.created_at.isoformat(),
                    "last_login": admin.last_login.isoformat() if admin.last_login else None
                }, f)
            os.chmod(filepath, 0o600)
            return True
        except Exception as e:
            logger.error(f"Failed to save admin credentials: {e}")
            return False

    @staticmethod
    def initialize_admin(username: str, password: str) -> bool:
        """สร้าง Admin account ครั้งแรก"""
        existing = AuthService._load_admin()
        if existing:
            logger.warning("Admin account already exists")
            return False

        admin = AdminCredential(
            username=username,
            hashed_password=AuthService.hash_password(password),
            created_at=datetime.utcnow()
        )
        return AuthService._save_admin(admin)

    @staticmethod
    def authenticate(username: str, password: str) -> Tuple[bool, Optional[str]]:
        """ตรวจสอบ username + password"""
        admin = AuthService._load_admin()
        
        if not admin:
            return False, "Admin account not initialized"
        
        if admin.username != username:
            return False, "Invalid username"
        
        if not AuthService.verify_password(password, admin.hashed_password):
            return False, "Invalid password"
        
        admin.last_login = datetime.utcnow()
        AuthService._save_admin(admin)
        
        return True, None

    @staticmethod
    def change_password(username: str, old_password: str, new_password: str) -> Tuple[bool, str]:
        """เปลี่ยน Password ของ Admin"""
        admin = AuthService._load_admin()
        
        if not admin or admin.username != username:
            return False, "Admin not found"
        
        if not AuthService.verify_password(old_password, admin.hashed_password):
            return False, "Current password is incorrect"
        
        admin.hashed_password = AuthService.hash_password(new_password)
        if AuthService._save_admin(admin):
            return True, "Password changed successfully"
        
        return False, "Failed to save new password"
