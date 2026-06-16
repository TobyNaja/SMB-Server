from fastapi import Request
from fastapi.responses import RedirectResponse, JSONResponse
from services.auth_service import AuthService
from datetime import datetime
import logging

logger = logging.getLogger(__name__)

PUBLIC_PATHS = [
    "/login",
    "/health",
    "/auth/login",
    "/docs",
    "/openapi.json",
    "/static",
]

async def auth_middleware(request: Request, call_next):
    path = request.url.path

    # Check public paths
    is_public = any(path.startswith(p) for p in PUBLIC_PATHS)
    if is_public:
        return await call_next(request)

    # Get token from cookie
    token = request.cookies.get("access_token")

    # Get token from Authorization header
    if not token:
        auth_header = request.headers.get("authorization", "")
        if auth_header.startswith("Bearer "):
            token = auth_header[7:]

    # No token
    if not token:
        if path.startswith("/api") or path.startswith("/auth"):
            return JSONResponse(status_code=401, content={"detail": "Not authenticated"})
        return RedirectResponse(url="/login", status_code=303)

    # Verify token
    username = AuthService.verify_token(token)
    if not username:
        if path.startswith("/api") or path.startswith("/auth"):
            return JSONResponse(status_code=401, content={"detail": "Invalid token"})
        return RedirectResponse(url="/login", status_code=303)

    # Set username in request state
    request.state.username = username
    request.state.token_expires = datetime.utcnow()

    return await call_next(request)
