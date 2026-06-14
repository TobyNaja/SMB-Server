#!/bin/bash
# =====================================================
#  SMB Manager Phase 1 - Enterprise Security Setup
#  Complete installation script for Docker
# =====================================================

POOL="${1:-ScriptDataPool}"
PROJECT_DIR="/mnt/$POOL/smb-manager"
WEBAPP_DIR="$PROJECT_DIR/webapp"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}  SMB Manager Phase 1 - Setup${NC}"
echo -e "${BLUE}════════════════════════════════════════${NC}"
echo ""

# ─── Step 1: Create Directory Structure ────────────
echo -e "${YELLOW}[1/10]${NC} Creating directory structure..."
mkdir -p "$WEBAPP_DIR"/{models,services,routers,middleware,templates,static,data}
echo -e "${GREEN}✅ Directories created${NC}"

# ─── Step 2: Create requirements.txt ───────────────
echo -e "${YELLOW}[2/10]${NC} Creating requirements.txt..."
cat > "$WEBAPP_DIR/requirements.txt" << 'EOF'
fastapi==0.104.1
uvicorn==0.24.0
docker==6.1.3
requests==2.31.0
urllib3==1.26.18
pydantic==2.5.0
pydantic-settings==2.1.0
jinja2==3.1.2
python-multipart==0.0.6
bcrypt==4.1.1
python-jose[cryptography]==3.3.0
passlib[bcrypt]==1.7.4
python-dotenv==1.0.0
EOF
echo -e "${GREEN}✅ requirements.txt created${NC}"

# ─── Step 3: Update config.py ──────────────────────
echo -e "${YELLOW}[3/10]${NC} Creating config.py..."
cat > "$WEBAPP_DIR/config.py" << 'EOF'
from pydantic_settings import BaseSettings
import os
from datetime import timedelta

class Settings(BaseSettings):
    # API Server
    api_host: str = os.getenv("API_HOST", "0.0.0.0")
    api_port: int = int(os.getenv("API_PORT", "8080"))
    
    # Docker
    samba_container: str = os.getenv("SAMBA_CONTAINER", "samba-server")
    samba_config_path: str = os.getenv("SAMBA_CONFIG_PATH", "/etc/samba")
    samba_data_path: str = os.getenv("SAMBA_DATA_PATH", "/var/lib/samba")
    
    # Security & Authentication
    secret_key: str = os.getenv("SECRET_KEY", "dev-secret-key-change-in-production")
    algorithm: str = "HS256"
    access_token_expire_minutes: int = 1440  # 24 hours
    
    # Audit & Logging
    audit_log_path: str = os.getenv("AUDIT_LOG_PATH", "/mnt/shared/audit.json")
    admin_credentials_file: str = os.getenv("ADMIN_CREDS_FILE", "/mnt/shared/.admin")
    
    class Config:
        env_file = ".env"

settings = Settings()
EOF
echo -e "${GREEN}✅ config.py created${NC}"

# ─── Step 4: Create models/auth.py ─────────────────
echo -e "${YELLOW}[4/10]${NC} Creating models/auth.py..."
cat > "$WEBAPP_DIR/models/__init__.py" << 'EOF'
# Models package
EOF
cat > "$WEBAPP_DIR/models/auth.py" << 'EOF'
from pydantic import BaseModel, Field
from typing import Optional
from datetime import datetime

class LoginRequest(BaseModel):
    """ฟอร์ม Login"""
    username: str = Field(..., min_length=1, max_length=32)
    password: str = Field(..., min_length=1)

class TokenResponse(BaseModel):
    """Response เมื่อ Login สำเร็จ"""
    access_token: str
    token_type: str = "bearer"
    expires_in: int

class UserInfo(BaseModel):
    """ข้อมูล User ที่ login"""
    username: str
    is_admin: bool = True
    expires_at: datetime

class TokenData(BaseModel):
    """ข้อมูล JWT Token"""
    username: str
    exp: datetime

class AdminCredential(BaseModel):
    """Admin account storage"""
    username: str
    hashed_password: str
    created_at: datetime
    last_login: Optional[datetime] = None
EOF
echo -e "${GREEN}✅ models/auth.py created${NC}"

# ─── Step 5: Create services/auth_service.py ──────
echo -e "${YELLOW}[5/10]${NC} Creating services/auth_service.py..."
cat > "$WEBAPP_DIR/services/__init__.py" << 'EOF'
# Services package
EOF
cat > "$WEBAPP_DIR/services/auth_service.py" << 'SVCEOF'
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
SVCEOF
echo -e "${GREEN}✅ services/auth_service.py created${NC}"

# ─── Step 6: Create services/audit_service.py ─────
echo -e "${YELLOW}[6/10]${NC} Creating services/audit_service.py..."
cat > "$WEBAPP_DIR/services/audit_service.py" << 'AUDITEOF'
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
AUDITEOF
echo -e "${GREEN}✅ services/audit_service.py created${NC}"

# ─── Step 7: Create routers/__init__.py ────────────
echo -e "${YELLOW}[7/10]${NC} Creating routers package..."
cat > "$WEBAPP_DIR/routers/__init__.py" << 'EOF'
# Routers package
EOF
echo -e "${GREEN}✅ routers/__init__.py created${NC}"

# ─── Step 8: Create routers/auth.py ────────────────
echo -e "${YELLOW}[8/10]${NC} Creating routers/auth.py..."
cat > "$WEBAPP_DIR/routers/auth.py" << 'AUTHEOF'
from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import JSONResponse
from models.auth import LoginRequest, TokenResponse, UserInfo
from services.auth_service import AuthService
from services.audit_service import AuditService
from datetime import datetime
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/auth", tags=["authentication"])

def get_client_ip(request: Request) -> str:
    """ดึง Client IP"""
    return request.client.host if request.client else "unknown"

@router.post("/login", response_model=TokenResponse)
async def login(credentials: LoginRequest, request: Request):
    """📌 POST /auth/login - ล็อกอิน"""
    client_ip = get_client_ip(request)
    is_valid, error = AuthService.authenticate(
        credentials.username,
        credentials.password
    )
    
    if not is_valid:
        AuditService.log(
            action="LOGIN",
            actor=credentials.username,
            resource_type="AUTH",
            resource_name="web_login",
            status="failure",
            details={"reason": error},
            client_ip=client_ip
        )
        raise HTTPException(status_code=401, detail=error or "Invalid credentials")
    
    token, expires_in = AuthService.create_access_token(credentials.username)
    
    AuditService.log(
        action="LOGIN",
        actor=credentials.username,
        resource_type="AUTH",
        resource_name="web_login",
        status="success",
        client_ip=client_ip
    )
    
    logger.info(f"User {credentials.username} logged in from {client_ip}")
    
    return TokenResponse(
        access_token=token,
        token_type="bearer",
        expires_in=expires_in
    )

@router.post("/logout")
async def logout(request: Request):
    """📌 POST /auth/logout"""
    username = request.state.username
    client_ip = get_client_ip(request)
    
    AuditService.log(
        action="LOGOUT",
        actor=username,
        resource_type="AUTH",
        resource_name="web_logout",
        status="success",
        client_ip=client_ip
    )
    
    response = JSONResponse({"message": "Logged out successfully"})
    response.delete_cookie("access_token")
    return response

@router.get("/me", response_model=UserInfo)
async def get_current_user(request: Request):
    """📌 GET /auth/me"""
    username = request.state.username
    token_expires = request.state.token_expires
    
    return UserInfo(
        username=username,
        is_admin=True,
        expires_at=token_expires
    )

@router.post("/change-password")
async def change_password(
    old_password: str,
    new_password: str,
    request: Request
):
    """📌 POST /auth/change-password"""
    username = request.state.username
    client_ip = get_client_ip(request)
    
    if not old_password or not new_password:
        raise HTTPException(status_code=400, detail="Missing password fields")
    
    if len(new_password) < 8:
        raise HTTPException(status_code=400, detail="New password must be at least 8 characters")
    
    is_valid, msg = AuthService.change_password(
        username,
        old_password,
        new_password
    )
    
    if not is_valid:
        AuditService.log(
            action="CHANGE_PASSWORD",
            actor=username,
            resource_type="AUTH",
            resource_name=username,
            status="failure",
            details={"reason": msg},
            client_ip=client_ip
        )
        raise HTTPException(status_code=400, detail=msg)
    
    AuditService.log(
        action="CHANGE_PASSWORD",
        actor=username,
        resource_type="AUTH",
        resource_name=username,
        status="success",
        client_ip=client_ip
    )
    
    logger.info(f"Password changed for user {username}")
    return {"message": "Password changed successfully"}
AUTHEOF
echo -e "${GREEN}✅ routers/auth.py created${NC}"

# ─── Step 9: Create routers/audit.py ───────────────
echo -e "${YELLOW}[9/10]${NC} Creating routers/audit.py..."
cat > "$WEBAPP_DIR/routers/audit.py" << 'AUDITROUTER'
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
AUDITROUTER
echo -e "${GREEN}✅ routers/audit.py created${NC}"

# ─── Step 10: Create middleware/auth_middleware.py ──
echo -e "${YELLOW}[10/10]${NC} Creating middleware/auth_middleware.py..."
mkdir -p "$WEBAPP_DIR/middleware"
cat > "$WEBAPP_DIR/middleware/__init__.py" << 'EOF'
# Middleware package
EOF
cat > "$WEBAPP_DIR/middleware/auth_middleware.py" << 'MIDDLEWAREEOF'
from fastapi import Request
from fastapi.responses import RedirectResponse, JSONResponse
from services.auth_service import AuthService
from datetime import datetime
import logging

logger = logging.getLogger(__name__)

PUBLIC_PATHS = [
    "/",
    "/login",
    "/health",
    "/static",
    "/auth/login",
    "/docs",
    "/openapi.json",
]

async def auth_middleware(request: Request, call_next):
    """JWT Authentication Middleware"""
    
    # Check if path is public
    is_public = any(request.url.path.startswith(p) for p in PUBLIC_PATHS)
    if is_public:
        return await call_next(request)
    
    # Check if path requires auth
    if request.url.path.startswith("/api") or request.url.path.startswith("/"):
        token = None
        
        # Try to get token from cookie
        token = request.cookies.get("access_token")
        
        # Try to get token from Authorization header
        if not token and "authorization" in request.headers:
            auth = request.headers.get("authorization")
            if auth.startswith("Bearer "):
                token = auth[7:]
        
        if not token:
            if request.url.path.startswith("/api"):
                return JSONResponse(status_code=401, content={"detail": "Not authenticated"})
            else:
                return RedirectResponse(url="/login", status_code=303)
        
        # Verify token
        username = AuthService.verify_token(token)
        if not username:
            if request.url.path.startswith("/api"):
                return JSONResponse(status_code=401, content={"detail": "Invalid token"})
            else:
                return RedirectResponse(url="/login", status_code=303)
        
        # Attach to request state
        request.state.username = username
        request.state.token_expires = datetime.utcnow()
    
    response = await call_next(request)
    return response
MIDDLEWAREEOF
echo -e "${GREEN}✅ middleware/auth_middleware.py created${NC}"

# ─── Create .env file ──────────────────────────────
echo -e "${YELLOW}[Creating]${NC} Creating .env file..."
cat > "$PROJECT_DIR/.env" << 'ENVEOF'
# Samba
SAMBA_CONTAINER=samba-server
SAMBA_CONFIG_PATH=/etc/samba
SAMBA_DATA_PATH=/var/lib/samba

# API
API_HOST=0.0.0.0
API_PORT=8080

# Security - ⚠️ CHANGE THIS IN PRODUCTION!
SECRET_KEY=super-secret-key-change-this-in-production-12345678

# Audit & Admin
AUDIT_LOG_PATH=/mnt/shared/audit.json
ADMIN_CREDS_FILE=/mnt/shared/.admin
ENVEOF
chmod 600 "$PROJECT_DIR/.env"
echo -e "${GREEN}✅ .env created${NC}"

# ─── Create Dockerfile ─────────────────────────────
echo -e "${YELLOW}[Creating]${NC} Creating Dockerfile..."
cat > "$WEBAPP_DIR/Dockerfile" << 'DOCKEREOF'
FROM python:3.11-slim
WORKDIR /app

RUN apt-get update && apt-get install -y \
    samba-common-bin \
    curl \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["python3", "main.py"]
DOCKEREOF
echo -e "${GREEN}✅ Dockerfile created${NC}"

# ─── Summary ───────────────────────────────────────
echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  ✅ Phase 1 Setup Complete!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""
echo -e "📁 Project : ${YELLOW}$PROJECT_DIR${NC}"
echo ""
echo -e "${BLUE}Files created:${NC}"
find "$WEBAPP_DIR" -type f -name "*.py" | sort | sed 's|^|  ✅ |'
echo -e "  ✅ Dockerfile"
echo -e "  ✅ .env"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo -e "  1️⃣  ${YELLOW}cd $PROJECT_DIR${NC}"
echo -e "  2️⃣  ${YELLOW}docker compose build webapp${NC}"
echo -e "  3️⃣  ${YELLOW}docker compose up -d${NC}"
echo ""
echo -e "${RED}⚠️  IMPORTANT:${NC}"
echo -e "  - Initialize first admin user:"
echo -e "    ${YELLOW}docker compose exec webapp python3 << 'INIT'${NC}"
echo -e "    ${YELLOW}from services.auth_service import AuthService${NC}"
echo -e "    ${YELLOW}AuthService.initialize_admin('admin', 'your-strong-password')${NC}"
echo -e "    ${YELLOW}INIT${NC}"
echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
