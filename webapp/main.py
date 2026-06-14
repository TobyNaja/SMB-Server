from fastapi import FastAPI, Request
from fastapi.responses import HTMLResponse, JSONResponse, RedirectResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from routers import users, groups, shares, auth, audit
from services.auth_service import AuthService
import uvicorn
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s [%(levelname)s] %(name)s: %(message)s')
logger = logging.getLogger(__name__)

app = FastAPI(title="SMB Permission Manager", version="1.0.0")

try:
    app.mount("/static", StaticFiles(directory="static"), name="static")
except:
    logger.warning("Static dir not found")

try:
    templates = Jinja2Templates(directory="templates")
except:
    logger.warning("Templates dir not found")
    templates = None

# ─── MIDDLEWARE ───────────────────────────────────
@app.middleware("http")
async def auth_middleware(request: Request, call_next):
    """Protect all routes except public paths"""
    
    public_paths = ["/login", "/health", "/auth/login", "/docs", "/openapi.json"]
    is_public = any(request.url.path.startswith(p) for p in public_paths) or request.url.path.startswith("/static")
    
    if is_public:
        return await call_next(request)
    
    # Get token from cookie or header
    token = request.cookies.get("access_token")
    if not token and "authorization" in request.headers:
        auth_header = request.headers.get("authorization", "")
        if auth_header.startswith("Bearer "):
            token = auth_header[7:]
    
    # No token - redirect or return 401
    if not token:
        if request.url.path.startswith("/api"):
            return JSONResponse(status_code=401, content={"detail": "Not authenticated"})
        return RedirectResponse(url="/login", status_code=303)
    
    # Verify token
    username = AuthService.verify_token(token)
    if not username:
        if request.url.path.startswith("/api"):
            return JSONResponse(status_code=401, content={"detail": "Invalid token"})
        return RedirectResponse(url="/login", status_code=303)
    
    # Attach username to request
    request.state.username = username
    return await call_next(request)

# ─── ROUTES ───────────────────────────────────────

@app.get("/health")
async def health():
    return {"status": "healthy"}

@app.get("/login", response_class=HTMLResponse)
async def login_page(request: Request):
    if templates:
        return templates.TemplateResponse("login.html", {"request": request})
    return "<h1>Login</h1>"

@app.get("/", response_class=HTMLResponse)
async def dashboard(request: Request):
    if templates:
        return templates.TemplateResponse("index.html", {"request": request})
    return "<h1>Dashboard</h1>"

# Include routers
app.include_router(auth.router)
app.include_router(audit.router)
app.include_router(users.router)
app.include_router(groups.router)
app.include_router(shares.router)

@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    logger.error(f"Error: {exc}")
    return JSONResponse(status_code=500, content={"detail": "Internal error"})

if __name__ == "__main__":
    from config import settings
    uvicorn.run(app, host=settings.api_host, port=settings.api_port, log_level="info")
