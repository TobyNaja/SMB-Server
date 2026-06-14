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
