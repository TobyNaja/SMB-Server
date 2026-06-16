from fastapi import FastAPI, Request
from fastapi.responses import HTMLResponse, JSONResponse, RedirectResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from routers import users, groups, shares, auth, audit, builtin, ad
from services.auth_service import AuthService
from middleware.auth_middleware import auth_middleware
import uvicorn
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s [%(levelname)s] %(name)s: %(message)s')
logger = logging.getLogger(__name__)

app = FastAPI(title="SMB Permission Manager", version="2.0.0")

try:
    app.mount("/static", StaticFiles(directory="static"), name="static")
except:
    logger.warning("Static dir not found")

try:
    templates = Jinja2Templates(directory="templates")
except:
    logger.warning("Templates dir not found")
    templates = None

@app.middleware("http")
async def auth_middleware_wrapper(request: Request, call_next):
    return await auth_middleware(request, call_next)

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

app.include_router(auth.router)
app.include_router(audit.router)
app.include_router(users.router)
app.include_router(groups.router)
app.include_router(shares.router)
app.include_router(builtin.router)
app.include_router(ad.router)

@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    logger.error(f"Error: {exc}")
    return JSONResponse(status_code=500, content={"detail": "Internal error"})

if __name__ == "__main__":
    from config import settings
    uvicorn.run(app, host=settings.api_host, port=settings.api_port, log_level="info")
