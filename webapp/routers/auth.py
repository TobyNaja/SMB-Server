from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
from services.auth_service import AuthService
from services.audit_service import AuditService
from datetime import datetime
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/auth", tags=["authentication"])

class LoginRequest(BaseModel):
    username: str = Field(..., min_length=1, max_length=32)
    password: str = Field(..., min_length=1)

class AdminCreate(BaseModel):
    username: str = Field(..., min_length=1, max_length=32)
    password: str = Field(..., min_length=8)

class ChangePasswordRequest(BaseModel):
    old_password: str
    new_password: str = Field(..., min_length=8)

def get_client_ip(request: Request) -> str:
    return request.client.host if request.client else "unknown"

@router.post("/login")
async def login(credentials: LoginRequest, request: Request):
    client_ip = get_client_ip(request)
    is_valid, error = AuthService.authenticate(credentials.username, credentials.password)
    if not is_valid:
        AuditService.log(action="LOGIN", actor=credentials.username,
            resource_type="AUTH", resource_name="web_login",
            status="failure", details={"reason": error}, client_ip=client_ip)
        raise HTTPException(status_code=401, detail=error or "Invalid credentials")
    token, expires_in = AuthService.create_access_token(credentials.username)
    AuditService.log(action="LOGIN", actor=credentials.username,
        resource_type="AUTH", resource_name="web_login",
        status="success", client_ip=client_ip)
    return {"access_token": token, "token_type": "bearer", "expires_in": expires_in}

@router.post("/logout")
async def logout(request: Request):
    username = getattr(request.state, 'username', 'unknown')
    AuditService.log(action="LOGOUT", actor=username,
        resource_type="AUTH", resource_name="web_logout",
        status="success", client_ip=get_client_ip(request))
    response = JSONResponse({"message": "Logged out successfully"})
    response.delete_cookie("access_token")
    return response

@router.get("/me")
async def get_current_user(request: Request):
    try:
        username = getattr(request.state, 'username', None)
        if not username:
            raise HTTPException(status_code=401, detail="Not authenticated")
        return {"username": username, "is_admin": True, "expires_at": datetime.utcnow().isoformat()}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error in /auth/me: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/change-password")
async def change_password(body: ChangePasswordRequest, request: Request):
    username = getattr(request.state, 'username', '')
    is_valid, msg = AuthService.change_password(username, body.old_password, body.new_password)
    AuditService.log(action="CHANGE_PASSWORD", actor=username,
        resource_type="AUTH", resource_name=username,
        status="success" if is_valid else "failure",
        client_ip=get_client_ip(request))
    if not is_valid:
        raise HTTPException(status_code=400, detail=msg)
    return {"message": msg}

@router.get("/admins")
async def list_admins(request: Request):
    admins = AuthService.list_admins()
    return {"admins": admins, "count": len(admins)}

@router.post("/admins")
async def create_admin(body: AdminCreate, request: Request):
    actor = getattr(request.state, 'username', 'unknown')
    is_valid, msg = AuthService.add_admin(body.username, body.password)
    AuditService.log(action="CREATE_ADMIN", actor=actor,
        resource_type="AUTH", resource_name=body.username,
        status="success" if is_valid else "failure",
        client_ip=get_client_ip(request))
    if not is_valid:
        raise HTTPException(status_code=400, detail=msg)
    return {"message": msg}

@router.delete("/admins/{username}")
async def delete_admin(username: str, request: Request):
    actor = getattr(request.state, 'username', 'unknown')
    is_valid, msg = AuthService.delete_admin(username, requester=actor)
    AuditService.log(action="DELETE_ADMIN", actor=actor,
        resource_type="AUTH", resource_name=username,
        status="success" if is_valid else "failure",
        client_ip=get_client_ip(request))
    if not is_valid:
        raise HTTPException(status_code=400, detail=msg)
    return {"message": msg}
