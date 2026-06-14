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
