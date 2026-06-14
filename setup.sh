#!/bin/bash
# =====================================================
#  SMB Permission Manager - Complete Setup Script
#  วิธีใช้: bash setup.sh [ชื่อ Pool]
#  ตัวอย่าง: bash setup.sh tank
# =====================================================

POOL="${1:-tank}"
PROJECT_DIR="/mnt/$POOL/smb-manager"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  SMB Permission Manager Setup${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "Pool     : ${YELLOW}$POOL${NC}"
echo -e "Project  : ${YELLOW}$PROJECT_DIR${NC}"
echo ""

# ─── สร้าง Directories ───────────────────────────────
echo "[1/9] Creating directories..."
mkdir -p "$PROJECT_DIR"/{samba,webapp/{services,routers,templates,static},data/shared}

# ─── docker-compose.yml ──────────────────────────────
echo "[2/9] Creating docker-compose.yml..."
cat > "$PROJECT_DIR/docker-compose.yml" << 'EOF'
version: '3.8'

services:

  samba:
    build:
      context: ./samba
      dockerfile: Dockerfile
    container_name: samba-server
    hostname: samba-server
    networks:
      - smb-network
    ports:
      # NOTE: TrueNAS ใช้ port 445 อยู่แล้ว
      # จึงใช้ port 1445 แทนเพื่อไม่ให้ชนกัน
      # Client เชื่อมต่อด้วย smb://IP:1445/sharename
      - "1137:137/udp"
      - "1138:138/udp"
      - "1139:139/tcp"
      - "1445:445/tcp"
    volumes:
      - samba-config:/etc/samba
      - samba-data:/var/lib/samba
      - ./data/shared:/mnt/shared
    environment:
      - WORKGROUP=WORKGROUP
    healthcheck:
      test: ["CMD-SHELL", "smbclient -L localhost -U % -m SMB3 &>/dev/null && echo ok"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 20s
    restart: unless-stopped

  webapp:
    build:
      context: ./webapp
      dockerfile: Dockerfile
    container_name: smb-webapp
    networks:
      - smb-network
    ports:
      - "8080:8080"
    volumes:
      - samba-config:/etc/samba
      - samba-data:/var/lib/samba
      - ./data/shared:/mnt/shared
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - SAMBA_CONTAINER=samba-server
      - SAMBA_CONFIG_PATH=/etc/samba
      - SAMBA_DATA_PATH=/var/lib/samba
      - API_HOST=0.0.0.0
      - API_PORT=8080
    depends_on:
      samba:
        condition: service_healthy
    restart: unless-stopped

networks:
  smb-network:
    driver: bridge

volumes:
  samba-config:
  samba-data:
EOF

# ─── samba/Dockerfile ────────────────────────────────
echo "[3/9] Creating Samba container..."
cat > "$PROJECT_DIR/samba/Dockerfile" << 'EOF'
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    samba samba-common samba-common-bin \
    smbclient winbind curl \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
COPY smb.conf.template /smb.conf.template
RUN chmod +x /entrypoint.sh && mkdir -p /mnt/shared && chmod 777 /mnt/shared

EXPOSE 137/udp 138/udp 139/tcp 445/tcp
ENTRYPOINT ["/entrypoint.sh"]
CMD ["smbd", "--foreground", "--no-process-group", "--log-stdout"]
EOF

cat > "$PROJECT_DIR/samba/entrypoint.sh" << 'EOF'
#!/bin/bash
set -e
CONF="/etc/samba/smb.conf"

if [ ! -f "$CONF" ]; then
    echo "[*] Init smb.conf from template"
    cp /smb.conf.template "$CONF"
fi

echo "[*] Starting nmbd..."
nmbd --foreground --no-process-group --log-stdout &

echo "[*] Starting smbd..."
exec "$@"
EOF

cat > "$PROJECT_DIR/samba/smb.conf.template" << 'EOF'
[global]
    workgroup = WORKGROUP
    server string = Docker Samba Server
    netbios name = DOCKER-SMB
    security = user
    passdb backend = tdbsam
    map to guest = bad user
    log file = /var/log/samba/log.%m
    max log size = 1000
    dns proxy = no

[shared]
    comment = Shared Storage
    path = /mnt/shared
    browseable = yes
    read only = no
    guest ok = no
    create mask = 0755
    directory mask = 0755
EOF

# ─── webapp/Dockerfile ───────────────────────────────
echo "[4/9] Creating WebApp container..."
cat > "$PROJECT_DIR/webapp/Dockerfile" << 'EOF'
FROM python:3.11-slim
WORKDIR /app

RUN apt-get update && apt-get install -y \
    samba-common-bin curl \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
EXPOSE 8080
CMD ["python", "main.py"]
EOF

cat > "$PROJECT_DIR/webapp/requirements.txt" << 'EOF'
fastapi==0.104.1
uvicorn==0.24.0
docker==7.0.0
pydantic==2.5.0
pydantic-settings==2.1.0
jinja2==3.1.2
python-multipart==0.0.6
EOF

# ─── webapp/config.py ────────────────────────────────
echo "[5/9] Creating Python source files..."
cat > "$PROJECT_DIR/webapp/config.py" << 'EOF'
from pydantic_settings import BaseSettings
import os

class Settings(BaseSettings):
    api_host: str = os.getenv("API_HOST", "0.0.0.0")
    api_port: int = int(os.getenv("API_PORT", "8080"))
    samba_container: str = os.getenv("SAMBA_CONTAINER", "samba-server")
    samba_config_path: str = os.getenv("SAMBA_CONFIG_PATH", "/etc/samba")
    samba_data_path: str = os.getenv("SAMBA_DATA_PATH", "/var/lib/samba")

    class Config:
        env_file = ".env"

settings = Settings()
EOF

# ─── services ────────────────────────────────────────
touch "$PROJECT_DIR/webapp/services/__init__.py"
touch "$PROJECT_DIR/webapp/routers/__init__.py"

cat > "$PROJECT_DIR/webapp/services/docker_executor.py" << 'EOF'
import docker, shlex, re, logging

logger = logging.getLogger(__name__)

class DockerExecutor:
    def __init__(self, container_name: str):
        self.container_name = container_name
        self.client = docker.DockerClient(base_url="unix:///var/run/docker.sock")
        self.container = self.client.containers.get(container_name)

    def execute(self, command: str) -> dict:
        try:
            result = self.container.exec_run(
                cmd=["/bin/bash", "-c", command],
                stdout=True, stderr=True, user="root"
            )
            return {
                "success": result.exit_code == 0,
                "exit_code": result.exit_code,
                "output": result.output.decode("utf-8", errors="replace") if result.output else ""
            }
        except Exception as e:
            logger.error(f"exec error: {e}")
            return {"success": False, "exit_code": -1, "output": str(e)}

    def create_user(self, username: str, password: str) -> dict:
        if not re.match(r'^[a-zA-Z0-9_-]{1,32}$', username):
            return {"success": False, "error": "Invalid username format"}

        # สร้าง system user
        self.execute(f"id {shlex.quote(username)} || useradd -m -s /usr/sbin/nologin {shlex.quote(username)}")

        # ตั้ง Samba password
        escaped = password.replace("'", "'\\''")
        result = self.execute(f"printf '{escaped}\\n{escaped}\\n' | smbpasswd -a -s {shlex.quote(username)}")
        if result['success']:
            return {"success": True, "message": f"User {username} created"}
        return {"success": False, "error": result['output']}

    def delete_user(self, username: str) -> dict:
        self.execute(f"smbpasswd -x {shlex.quote(username)} 2>/dev/null || true")
        self.execute(f"userdel -r {shlex.quote(username)} 2>/dev/null || true")
        return {"success": True, "message": f"User {username} deleted"}

    def set_password(self, username: str, password: str) -> dict:
        escaped = password.replace("'", "'\\''")
        result = self.execute(f"printf '{escaped}\\n{escaped}\\n' | smbpasswd -s {shlex.quote(username)}")
        if result['success']:
            return {"success": True, "message": "Password updated"}
        return {"success": False, "error": result['output']}

    def create_group(self, group_name: str) -> dict:
        result = self.execute(f"groupadd {shlex.quote(group_name)} 2>/dev/null || true")
        return {"success": True, "message": f"Group {group_name} created"}

    def add_user_to_group(self, username: str, group_name: str) -> dict:
        result = self.execute(f"usermod -a -G {shlex.quote(group_name)} {shlex.quote(username)}")
        if result['success']:
            return {"success": True, "message": f"Added {username} to {group_name}"}
        return {"success": False, "error": result['output']}

    def remove_user_from_group(self, username: str, group_name: str) -> dict:
        result = self.execute(f"gpasswd -d {shlex.quote(username)} {shlex.quote(group_name)}")
        if result['success']:
            return {"success": True, "message": f"Removed {username} from {group_name}"}
        return {"success": False, "error": result['output']}

    def get_users(self) -> list:
        result = self.execute("pdbedit -L 2>/dev/null")
        users = []
        if result['success']:
            for line in result['output'].split('\n'):
                if not line.strip():
                    continue
                parts = line.split(':')
                if len(parts) >= 2:
                    users.append({
                        "username": parts[0],
                        "uid": parts[1],
                        "fullname": parts[2] if len(parts) > 2 else "",
                        "disabled": False
                    })
        return users

    def get_groups(self) -> list:
        result = self.execute("getent group")
        groups = []
        skip = {'root','bin','sys','daemon','adm','lp','mail','news',
                'uucp','man','proxy','kmem','dialout','fax','voice',
                'cdrom','floppy','tape','sudo','audio','dip','www-data',
                'backup','operator','list','irc','src','gnats','shadow',
                'utmp','video','sasl','plugdev','staff','games','users',
                'nogroup','crontab','syslog','tty','disk','input','netdev'}
        if result['success']:
            for line in result['output'].split('\n'):
                if not line.strip():
                    continue
                name = line.split(':')[0]
                if name not in skip and not name.startswith('_'):
                    groups.append(name)
        return sorted(groups)

    def reload_samba(self) -> dict:
        result = self.execute("smbcontrol all reload-config 2>/dev/null || pkill -HUP smbd || true")
        return {"success": True, "message": "Samba reloaded"}
EOF

# ─── config_parser.py ────────────────────────────────
cat > "$PROJECT_DIR/webapp/services/config_parser.py" << 'EOF'
import os, logging
from typing import Dict, List, Optional

logger = logging.getLogger(__name__)

SYSTEM_SECTIONS = {'global', 'homes', 'printers', 'print$', 'netlogon', 'sysvol'}

class SmbConfParser:
    def __init__(self, config_path: str):
        self.config_path = config_path
        self.sections: Dict[str, Dict[str, str]] = {}
        self._parse()

    def _parse(self):
        self.sections = {}
        if not os.path.exists(self.config_path):
            logger.warning(f"smb.conf not found: {self.config_path}")
            self.sections = {
                'global': {
                    'workgroup': 'WORKGROUP',
                    'security': 'user',
                    'passdb backend': 'tdbsam'
                }
            }
            return
        with open(self.config_path, 'r') as f:
            section = None
            for line in f:
                line = line.strip()
                if not line or line.startswith(('#', ';')):
                    continue
                if line.startswith('[') and line.endswith(']'):
                    section = line[1:-1].strip()
                    self.sections[section] = {}
                elif section and '=' in line:
                    key, _, val = line.partition('=')
                    self.sections[section][key.strip()] = val.strip()

    def _save(self):
        os.makedirs(os.path.dirname(self.config_path), exist_ok=True)
        with open(self.config_path, 'w') as f:
            f.write("# Auto-managed by SMB Permission Manager\n\n")
            for section, params in self.sections.items():
                f.write(f"[{section}]\n")
                for k, v in params.items():
                    f.write(f"    {k} = {v}\n")
                f.write("\n")
        logger.info("smb.conf saved")

    # ── Shares ──────────────────────────────────────────
    def get_shares(self) -> List[str]:
        return [s for s in self.sections if s not in SYSTEM_SECTIONS]

    def get_share(self, name: str) -> Optional[dict]:
        if name not in self.sections:
            return None
        d = self.sections[name]
        return {
            "name": name,
            "path": d.get('path', ''),
            "comment": d.get('comment', ''),
            "browseable": d.get('browseable', 'yes').lower() == 'yes',
            "read_only": d.get('read only', 'no').lower() == 'yes',
            "guest_ok": d.get('guest ok', 'no').lower() == 'yes',
            "valid_users": [u for u in d.get('valid users', '').split() if u],
            "write_list":  [u for u in d.get('write list',  '').split() if u],
            "read_list":   [u for u in d.get('read list',   '').split() if u],
            "create_mask": d.get('create mask', '0755'),
            "directory_mask": d.get('directory mask', '0755'),
        }

    def get_all_shares(self) -> List[dict]:
        return [self.get_share(s) for s in self.get_shares()]

    def create_share(self, name: str, path: str, comment: str = "") -> bool:
        if name in self.sections:
            return False
        self.sections[name] = {
            'comment': comment or f'{name} share',
            'path': path,
            'browseable': 'yes',
            'read only': 'no',
            'guest ok': 'no',
            'create mask': '0755',
            'directory mask': '0755',
        }
        self._save()
        return True

    def update_share(self, name: str, updates: dict) -> bool:
        if name not in self.sections:
            return False
        bool_keys = {'browseable', 'read only', 'guest ok', 'read_only', 'guest_ok'}
        mapped = {
            'read_only': 'read only',
            'guest_ok': 'guest ok',
        }
        for k, v in updates.items():
            real_key = mapped.get(k, k)
            if k in bool_keys or real_key in bool_keys:
                self.sections[name][real_key] = 'yes' if v else 'no'
            else:
                self.sections[name][real_key] = str(v)
        self._save()
        return True

    def delete_share(self, name: str) -> bool:
        if name in self.sections:
            del self.sections[name]
            self._save()
            return True
        return False

    def set_valid_users(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['valid users'] = ' '.join(users)
        self._save()
        return True

    def set_write_list(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['write list'] = ' '.join(users)
        self._save()
        return True
EOF

# ─── routers ─────────────────────────────────────────
echo "[6/9] Creating API routers..."
cat > "$PROJECT_DIR/webapp/routers/users.py" << 'EOF'
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from typing import List
from services.docker_executor import DockerExecutor
from config import settings

router = APIRouter(prefix="/api/users", tags=["users"])

class UserCreate(BaseModel):
    username: str = Field(..., min_length=1, max_length=32)
    password: str = Field(..., min_length=1)
    fullname: str = ""

def executor():
    return DockerExecutor(settings.samba_container)

@router.get("")
async def list_users():
    return {"users": executor().get_users()}

@router.post("")
async def create_user(user: UserCreate):
    result = executor().create_user(user.username, user.password)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    executor().reload_samba()
    return {"message": f"User {user.username} created"}

@router.delete("/{username}")
async def delete_user(username: str):
    executor().delete_user(username)
    executor().reload_samba()
    return {"message": f"User {username} deleted"}

@router.post("/{username}/password")
async def change_password(username: str, password: str):
    result = executor().set_password(username, password)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    executor().reload_samba()
    return {"message": "Password updated"}
EOF

cat > "$PROJECT_DIR/webapp/routers/groups.py" << 'EOF'
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from services.docker_executor import DockerExecutor
from config import settings

router = APIRouter(prefix="/api/groups", tags=["groups"])

class GroupCreate(BaseModel):
    group_name: str

def executor():
    return DockerExecutor(settings.samba_container)

@router.get("")
async def list_groups():
    return {"groups": executor().get_groups()}

@router.post("")
async def create_group(group: GroupCreate):
    result = executor().create_group(group.group_name)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    return {"message": f"Group {group.group_name} created"}

@router.post("/{group_name}/members/{username}")
async def add_member(group_name: str, username: str):
    result = executor().add_user_to_group(username, group_name)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    return {"message": f"Added {username} to {group_name}"}

@router.delete("/{group_name}/members/{username}")
async def remove_member(group_name: str, username: str):
    result = executor().remove_user_from_group(username, group_name)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    return {"message": f"Removed {username} from {group_name}"}
EOF

cat > "$PROJECT_DIR/webapp/routers/shares.py" << 'EOF'
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import List, Optional
from services.config_parser import SmbConfParser
from services.docker_executor import DockerExecutor
from config import settings
import os

router = APIRouter(prefix="/api/shares", tags=["shares"])

class ShareCreate(BaseModel):
    name: str
    path: str
    comment: str = ""
    browseable: bool = True
    guest_ok: bool = False

class ShareUpdate(BaseModel):
    comment: Optional[str] = None
    browseable: Optional[bool] = None
    guest_ok: Optional[bool] = None
    read_only: Optional[bool] = None

class PermissionUpdate(BaseModel):
    users: List[str] = []
    permission_type: str

def get_config():
    return SmbConfParser(os.path.join(settings.samba_config_path, 'smb.conf'))

def executor():
    return DockerExecutor(settings.samba_container)

@router.get("")
async def list_shares():
    return {"shares": get_config().get_all_shares()}

@router.get("/{share_name}")
async def get_share(share_name: str):
    share = get_config().get_share(share_name)
    if not share:
        raise HTTPException(404, "Share not found")
    return share

@router.post("")
async def create_share(share: ShareCreate):
    ex = executor()
    # สร้าง directory ใน container
    ex.execute(f"mkdir -p {share.path} && chmod 777 {share.path}")
    cfg = get_config()
    if not cfg.create_share(share.name, share.path, share.comment):
        raise HTTPException(400, "Share already exists")
    cfg.update_share(share.name, {"browseable": share.browseable, "guest_ok": share.guest_ok})
    ex.reload_samba()
    return {"message": f"Share '{share.name}' created"}

@router.patch("/{share_name}")
async def update_share(share_name: str, updates: ShareUpdate):
    cfg = get_config()
    if not cfg.get_share(share_name):
        raise HTTPException(404, "Share not found")
    cfg.update_share(share_name, {k: v for k, v in updates.dict().items() if v is not None})
    executor().reload_samba()
    return {"message": f"Share '{share_name}' updated"}

@router.delete("/{share_name}")
async def delete_share(share_name: str):
    get_config().delete_share(share_name)
    executor().reload_samba()
    return {"message": f"Share '{share_name}' deleted"}

@router.post("/{share_name}/permissions")
async def update_permissions(share_name: str, perm: PermissionUpdate):
    cfg = get_config()
    if not cfg.get_share(share_name):
        raise HTTPException(404, "Share not found")
    if perm.permission_type == 'valid_users':
        cfg.set_valid_users(share_name, perm.users)
    elif perm.permission_type == 'write_list':
        cfg.set_write_list(share_name, perm.users)
    else:
        raise HTTPException(400, "Invalid permission_type")
    executor().reload_samba()
    return {"message": "Permissions updated"}
EOF

# ─── main.py ─────────────────────────────────────────
cat > "$PROJECT_DIR/webapp/main.py" << 'EOF'
from fastapi import FastAPI, Request
from fastapi.responses import HTMLResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
import uvicorn, logging

from routers import users, groups, shares
from config import settings

logging.basicConfig(level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s')

app = FastAPI(title="SMB Permission Manager", version="1.0.0")
app.mount("/static", StaticFiles(directory="static"), name="static")
templates = Jinja2Templates(directory="templates")

app.include_router(users.router)
app.include_router(groups.router)
app.include_router(shares.router)

@app.get("/health")
async def health():
    return {"status": "healthy"}

@app.get("/", response_class=HTMLResponse)
async def dashboard(request: Request):
    return templates.TemplateResponse("index.html", {"request": request})

if __name__ == "__main__":
    uvicorn.run(app, host=settings.api_host, port=settings.api_port)
EOF

# ─── Frontend templates ───────────────────────────────
echo "[7/9] Creating Frontend..."
cat > "$PROJECT_DIR/webapp/templates/index.html" << 'HTMLEOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SMB Permission Manager</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.0/font/bootstrap-icons.css" rel="stylesheet">
    <link href="/static/style.css" rel="stylesheet">
</head>
<body>
<div class="wrapper d-flex">
    <!-- Sidebar -->
    <nav class="sidebar bg-dark text-white">
        <div class="sidebar-header p-4 border-bottom border-secondary">
            <h5 class="mb-0 text-white"><i class="bi bi-shield-lock-fill text-primary"></i> SMB Manager</h5>
            <small class="text-secondary">Permission Control</small>
        </div>
        <ul class="nav flex-column p-3">
            <li class="nav-item mb-1">
                <a class="nav-link active" href="#" onclick="showTab('dashboard',this)">
                    <i class="bi bi-speedometer2"></i> <span>Dashboard</span>
                </a>
            </li>
            <li class="nav-item mb-1">
                <a class="nav-link" href="#" onclick="showTab('users',this)">
                    <i class="bi bi-people-fill"></i> <span>Users</span>
                </a>
            </li>
            <li class="nav-item mb-1">
                <a class="nav-link" href="#" onclick="showTab('groups',this)">
                    <i class="bi bi-diagram-3-fill"></i> <span>Groups</span>
                </a>
            </li>
            <li class="nav-item mb-1">
                <a class="nav-link" href="#" onclick="showTab('shares',this)">
                    <i class="bi bi-folder2-open"></i> <span>Shares</span>
                </a>
            </li>
            <li class="nav-item mb-1">
                <a class="nav-link" href="#" onclick="showTab('permissions',this)">
                    <i class="bi bi-lock-fill"></i> <span>Permissions</span>
                </a>
            </li>
        </ul>
        <div class="p-3 mt-auto border-top border-secondary">
            <small class="text-secondary">v1.0.0 · TrueNAS SCALE</small>
        </div>
    </nav>

    <!-- Main Content -->
    <main class="main-content flex-grow-1 d-flex flex-column">
        <header class="topbar navbar navbar-dark bg-dark px-4 py-2 border-bottom border-secondary">
            <span class="navbar-brand mb-0 h6 text-white">
                <i class="bi bi-hdd-network"></i> SMB Share Permission Manager
            </span>
            <span class="ms-auto text-secondary small">
                <i class="bi bi-clock"></i> <span id="clock"></span>
            </span>
        </header>

        <div class="content p-4 flex-grow-1">
            <div id="alert-box"></div>

            <!-- ── DASHBOARD ── -->
            <div id="tab-dashboard" class="tab-pane active">
                <h4 class="mb-4">Dashboard</h4>
                <div class="row g-3 mb-4">
                    <div class="col-6 col-md-3">
                        <div class="card stat-card">
                            <div class="card-body">
                                <div class="stat-icon text-primary"><i class="bi bi-people-fill"></i></div>
                                <div class="stat-num" id="stat-users">—</div>
                                <div class="stat-label">Users</div>
                            </div>
                        </div>
                    </div>
                    <div class="col-6 col-md-3">
                        <div class="card stat-card">
                            <div class="card-body">
                                <div class="stat-icon text-info"><i class="bi bi-diagram-3-fill"></i></div>
                                <div class="stat-num" id="stat-groups">—</div>
                                <div class="stat-label">Groups</div>
                            </div>
                        </div>
                    </div>
                    <div class="col-6 col-md-3">
                        <div class="card stat-card">
                            <div class="card-body">
                                <div class="stat-icon text-warning"><i class="bi bi-folder2-open"></i></div>
                                <div class="stat-num" id="stat-shares">—</div>
                                <div class="stat-label">Shares</div>
                            </div>
                        </div>
                    </div>
                    <div class="col-6 col-md-3">
                        <div class="card stat-card">
                            <div class="card-body">
                                <div class="stat-icon text-success"><i class="bi bi-check-circle-fill"></i></div>
                                <div class="stat-num text-success">Online</div>
                                <div class="stat-label">Status</div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="card">
                    <div class="card-header">Connection Info</div>
                    <div class="card-body">
                        <p class="mb-1"><i class="bi bi-info-circle text-info"></i>
                            Connect to SMB shares using: <code>smb://&lt;TrueNAS-IP&gt;:1445/sharename</code>
                        </p>
                        <p class="mb-0"><i class="bi bi-info-circle text-info"></i>
                            API Docs available at: <a href="/docs" target="_blank">/docs</a>
                        </p>
                    </div>
                </div>
            </div>

            <!-- ── USERS ── -->
            <div id="tab-users" class="tab-pane">
                <div class="d-flex justify-content-between align-items-center mb-3">
                    <h4>Manage Users</h4>
                    <button class="btn btn-primary btn-sm" data-bs-toggle="modal" data-bs-target="#modalUser">
                        <i class="bi bi-plus-lg"></i> Add User
                    </button>
                </div>
                <div class="card">
                    <div class="card-body p-0">
                        <table class="table table-hover mb-0">
                            <thead class="table-dark">
                                <tr><th>Username</th><th>UID</th><th>Full Name</th><th>Status</th><th>Actions</th></tr>
                            </thead>
                            <tbody id="tbody-users">
                                <tr><td colspan="5" class="text-center text-muted py-4">Loading...</td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <!-- ── GROUPS ── -->
            <div id="tab-groups" class="tab-pane">
                <div class="d-flex justify-content-between align-items-center mb-3">
                    <h4>Manage Groups</h4>
                    <button class="btn btn-info btn-sm" data-bs-toggle="modal" data-bs-target="#modalGroup">
                        <i class="bi bi-plus-lg"></i> Add Group
                    </button>
                </div>
                <div class="card" style="max-width:600px">
                    <div class="card-body p-0">
                        <table class="table table-hover mb-0">
                            <thead class="table-dark">
                                <tr><th>Group Name</th><th>Actions</th></tr>
                            </thead>
                            <tbody id="tbody-groups">
                                <tr><td colspan="2" class="text-center text-muted py-4">Loading...</td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <!-- ── SHARES ── -->
            <div id="tab-shares" class="tab-pane">
                <div class="d-flex justify-content-between align-items-center mb-3">
                    <h4>Manage Shares</h4>
                    <button class="btn btn-warning btn-sm" data-bs-toggle="modal" data-bs-target="#modalShare">
                        <i class="bi bi-plus-lg"></i> Add Share
                    </button>
                </div>
                <div class="row g-3" id="shares-grid"></div>
            </div>

            <!-- ── PERMISSIONS ── -->
            <div id="tab-permissions" class="tab-pane">
                <h4 class="mb-3">Share Permissions</h4>
                <div class="card mb-3" style="max-width:400px">
                    <div class="card-body">
                        <label class="form-label fw-bold">Select Share:</label>
                        <select class="form-select" id="perm-share-select" onchange="loadPermissions()">
                            <option value="">-- Select a share --</option>
                        </select>
                    </div>
                </div>
                <div id="perm-container"></div>
            </div>
        </div>
    </main>
</div>

<!-- ── MODAL: ADD USER ── -->
<div class="modal fade" id="modalUser" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header bg-primary text-white">
                <h5 class="modal-title"><i class="bi bi-person-plus"></i> Add New User</h5>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
                <div class="mb-3">
                    <label class="form-label">Username <span class="text-danger">*</span></label>
                    <input type="text" class="form-control" id="in-username"
                           placeholder="e.g. john_doe" pattern="[a-zA-Z0-9_-]+" required>
                    <div class="form-text">Only letters, numbers, _ and - allowed</div>
                </div>
                <div class="mb-3">
                    <label class="form-label">Password <span class="text-danger">*</span></label>
                    <input type="password" class="form-control" id="in-password" required>
                </div>
                <div class="mb-3">
                    <label class="form-label">Full Name</label>
                    <input type="text" class="form-control" id="in-fullname" placeholder="Optional">
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                <button class="btn btn-primary" onclick="createUser()">
                    <i class="bi bi-check-lg"></i> Create
                </button>
            </div>
        </div>
    </div>
</div>

<!-- ── MODAL: ADD GROUP ── -->
<div class="modal fade" id="modalGroup" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header bg-info text-white">
                <h5 class="modal-title"><i class="bi bi-diagram-3"></i> Add New Group</h5>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
                <div class="mb-3">
                    <label class="form-label">Group Name <span class="text-danger">*</span></label>
                    <input type="text" class="form-control" id="in-groupname"
                           placeholder="e.g. developers" required>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                <button class="btn btn-info text-white" onclick="createGroup()">
                    <i class="bi bi-check-lg"></i> Create
                </button>
            </div>
        </div>
    </div>
</div>

<!-- ── MODAL: ADD SHARE ── -->
<div class="modal fade" id="modalShare" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header bg-warning">
                <h5 class="modal-title"><i class="bi bi-folder-plus"></i> Add New Share</h5>
                <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
                <div class="mb-3">
                    <label class="form-label">Share Name <span class="text-danger">*</span></label>
                    <input type="text" class="form-control" id="in-sharename" placeholder="e.g. projects">
                </div>
                <div class="mb-3">
                    <label class="form-label">Path in Container <span class="text-danger">*</span></label>
                    <input type="text" class="form-control" id="in-sharepath"
                           placeholder="/mnt/shared/projects">
                    <div class="form-text">Directory will be created automatically</div>
                </div>
                <div class="mb-3">
                    <label class="form-label">Description</label>
                    <input type="text" class="form-control" id="in-sharecomment">
                </div>
                <div class="form-check mb-2">
                    <input class="form-check-input" type="checkbox" id="in-browseable" checked>
                    <label class="form-check-label">Browseable</label>
                </div>
                <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="in-guestok">
                    <label class="form-check-label">Allow Guest Access</label>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                <button class="btn btn-warning" onclick="createShare()">
                    <i class="bi bi-check-lg"></i> Create
                </button>
            </div>
        </div>
    </div>
</div>

<!-- ── MODAL: PERMISSIONS ── -->
<div class="modal fade" id="modalPerm" tabindex="-1">
    <div class="modal-dialog modal-lg">
        <div class="modal-content">
            <div class="modal-header bg-dark text-white">
                <h5 class="modal-title" id="perm-modal-title">
                    <i class="bi bi-lock-fill"></i> Permissions
                </h5>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body" id="perm-modal-body">Loading...</div>
            <div class="modal-footer">
                <button class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                <button class="btn btn-success" onclick="saveModalPermissions()">
                    <i class="bi bi-shield-check"></i> Save Permissions
                </button>
            </div>
        </div>
    </div>
</div>

<!-- ── MODAL: GROUP MEMBERS ── -->
<div class="modal fade" id="modalMembers" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header bg-info text-white">
                <h5 class="modal-title" id="members-title">Group Members</h5>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
                <div class="input-group mb-3">
                    <input type="text" class="form-control" id="in-member-username"
                           placeholder="Enter username to add">
                    <button class="btn btn-info text-white" onclick="addMember()">
                        <i class="bi bi-plus"></i> Add
                    </button>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
            </div>
        </div>
    </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
<script>
// ─── State ──────────────────────────────────────────────
const S = { users: [], groups: [], shares: [], currentShare: null, currentGroup: null };

// ─── API Helper ─────────────────────────────────────────
async function api(method, url, body) {
    const r = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined
    });
    const d = await r.json();
    if (!r.ok) throw new Error(d.detail || 'API Error');
    return d;
}

// ─── Alert ──────────────────────────────────────────────
function alert(msg, type = 'success') {
    const icons = { success: 'check-circle-fill', danger: 'x-circle-fill', warning: 'exclamation-triangle-fill', info: 'info-circle-fill' };
    const id = 'a' + Date.now();
    document.getElementById('alert-box').insertAdjacentHTML('beforeend', `
        <div id="${id}" class="alert alert-${type} alert-dismissible fade show d-flex align-items-center gap-2" role="alert">
            <i class="bi bi-${icons[type] || icons.info}"></i> ${msg}
            <button type="button" class="btn-close ms-auto" data-bs-dismiss="alert"></button>
        </div>`);
    setTimeout(() => document.getElementById(id)?.remove(), 5000);
}

// ─── Tabs ───────────────────────────────────────────────
function showTab(name, el) {
    document.querySelectorAll('.tab-pane').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
    document.getElementById('tab-' + name).classList.add('active');
    if (el) el.classList.add('active');
    const loaders = { dashboard: loadDashboard, users: loadUsers, groups: loadGroups, shares: loadShares, permissions: initPermissions };
    loaders[name]?.();
}

// ─── Dashboard ──────────────────────────────────────────
async function loadDashboard() {
    try {
        const [u, g, s] = await Promise.all([api('GET','/api/users'), api('GET','/api/groups'), api('GET','/api/shares')]);
        document.getElementById('stat-users').textContent  = u.users.length;
        document.getElementById('stat-groups').textContent = g.groups.length;
        document.getElementById('stat-shares').textContent = s.shares.length;
    } catch(e) { console.error(e); }
}

// ─── Users ──────────────────────────────────────────────
async function loadUsers() {
    try {
        const d = await api('GET', '/api/users');
        S.users = d.users;
        document.getElementById('tbody-users').innerHTML = S.users.length
            ? S.users.map(u => `
                <tr>
                    <td><i class="bi bi-person-circle text-primary me-1"></i><strong>${u.username}</strong></td>
                    <td><code>${u.uid}</code></td>
                    <td>${u.fullname || '<span class="text-muted">—</span>'}</td>
                    <td><span class="badge bg-${u.disabled ? 'danger' : 'success'}">${u.disabled ? 'Disabled' : 'Active'}</span></td>
                    <td>
                        <button class="btn btn-sm btn-outline-warning me-1" onclick="changePassword('${u.username}')" title="Change Password"><i class="bi bi-key-fill"></i></button>
                        <button class="btn btn-sm btn-outline-danger" onclick="deleteUser('${u.username}')" title="Delete"><i class="bi bi-trash-fill"></i></button>
                    </td>
                </tr>`).join('')
            : '<tr><td colspan="5" class="text-center text-muted py-4">No users found</td></tr>';
    } catch(e) { alert('Failed to load users: ' + e.message, 'danger'); }
}

async function createUser() {
    const username = document.getElementById('in-username').value.trim();
    const password = document.getElementById('in-password').value;
    const fullname = document.getElementById('in-fullname').value.trim();
    if (!username || !password) { alert('Username and password are required', 'warning'); return; }
    try {
        await api('POST', '/api/users', { username, password, fullname });
        alert(`User "${username}" created successfully`);
        bootstrap.Modal.getInstance(document.getElementById('modalUser')).hide();
        document.getElementById('in-username').value = '';
        document.getElementById('in-password').value = '';
        document.getElementById('in-fullname').value = '';
        loadUsers();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

async function deleteUser(username) {
    if (!confirm(`Delete user "${username}"? This cannot be undone.`)) return;
    try {
        await api('DELETE', `/api/users/${username}`);
        alert(`User "${username}" deleted`);
        loadUsers();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

async function changePassword(username) {
    const p = prompt(`New password for "${username}":`);
    if (!p) return;
    try {
        await api('POST', `/api/users/${username}/password?password=${encodeURIComponent(p)}`);
        alert(`Password updated for "${username}"`);
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

// ─── Groups ─────────────────────────────────────────────
async function loadGroups() {
    try {
        const d = await api('GET', '/api/groups');
        S.groups = d.groups;
        document.getElementById('tbody-groups').innerHTML = S.groups.length
            ? S.groups.map(g => `
                <tr>
                    <td><i class="bi bi-diagram-3 text-info me-1"></i><strong>${g}</strong></td>
                    <td>
                        <button class="btn btn-sm btn-outline-info" onclick="openGroupMembers('${g}')">
                            <i class="bi bi-people"></i> Members
                        </button>
                    </td>
                </tr>`).join('')
            : '<tr><td colspan="2" class="text-center text-muted py-4">No groups found</td></tr>';
    } catch(e) { alert('Failed to load groups: ' + e.message, 'danger'); }
}

async function createGroup() {
    const gn = document.getElementById('in-groupname').value.trim();
    if (!gn) { alert('Group name required', 'warning'); return; }
    try {
        await api('POST', '/api/groups', { group_name: gn });
        alert(`Group "${gn}" created`);
        bootstrap.Modal.getInstance(document.getElementById('modalGroup')).hide();
        document.getElementById('in-groupname').value = '';
        loadGroups();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

function openGroupMembers(groupName) {
    S.currentGroup = groupName;
    document.getElementById('members-title').textContent = `Add Member → ${groupName}`;
    document.getElementById('in-member-username').value = '';
    new bootstrap.Modal(document.getElementById('modalMembers')).show();
}

async function addMember() {
    const u = document.getElementById('in-member-username').value.trim();
    if (!u) { alert('Enter a username', 'warning'); return; }
    try {
        await api('POST', `/api/groups/${S.currentGroup}/members/${u}`);
        alert(`"${u}" added to group "${S.currentGroup}"`);
        document.getElementById('in-member-username').value = '';
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

// ─── Shares ─────────────────────────────────────────────
async function loadShares() {
    try {
        const d = await api('GET', '/api/shares');
        S.shares = d.shares;
        const grid = document.getElementById('shares-grid');
        grid.innerHTML = S.shares.length
            ? S.shares.map(s => `
                <div class="col-md-6 col-xl-4">
                    <div class="card share-card h-100">
                        <div class="card-header d-flex justify-content-between align-items-center">
                            <span><i class="bi bi-folder2-open text-warning me-1"></i><strong>${s.name}</strong></span>
                            <div>
                                <span class="badge ${s.guest_ok ? 'bg-warning text-dark' : 'bg-secondary'} me-1">${s.guest_ok ? 'Guest' : 'Private'}</span>
                                <span class="badge ${s.read_only ? 'bg-info' : 'bg-success'}">${s.read_only ? 'RO' : 'RW'}</span>
                            </div>
                        </div>
                        <div class="card-body">
                            <p class="text-muted small mb-2"><i class="bi bi-folder"></i> <code>${s.path}</code></p>
                            <p class="text-muted small mb-2">${s.comment || '—'}</p>
                            <div class="mb-2">
                                <small class="fw-bold text-muted">VALID USERS:</small><br>
                                ${s.valid_users.length ? s.valid_users.map(u=>`<span class="badge bg-primary me-1">${u}</span>`).join('') : '<small class="text-muted">All</small>'}
                            </div>
                            <div>
                                <small class="fw-bold text-muted">WRITE LIST:</small><br>
                                ${s.write_list.length ? s.write_list.map(u=>`<span class="badge bg-success me-1">${u}</span>`).join('') : '<small class="text-muted">None</small>'}
                            </div>
                        </div>
                        <div class="card-footer d-flex gap-2">
                            <button class="btn btn-sm btn-dark flex-grow-1" onclick="openPermModal('${s.name}')">
                                <i class="bi bi-lock"></i> Permissions
                            </button>
                            <button class="btn btn-sm btn-outline-danger" onclick="deleteShare('${s.name}')">
                                <i class="bi bi-trash"></i>
                            </button>
                        </div>
                    </div>
                </div>`).join('')
            : '<div class="col"><div class="alert alert-info">No shares configured yet</div></div>';
    } catch(e) { alert('Failed to load shares: ' + e.message, 'danger'); }
}

async function createShare() {
    const payload = {
        name:      document.getElementById('in-sharename').value.trim(),
        path:      document.getElementById('in-sharepath').value.trim(),
        comment:   document.getElementById('in-sharecomment').value.trim(),
        browseable:document.getElementById('in-browseable').checked,
        guest_ok:  document.getElementById('in-guestok').checked,
    };
    if (!payload.name || !payload.path) { alert('Share name and path required', 'warning'); return; }
    try {
        await api('POST', '/api/shares', payload);
        alert(`Share "${payload.name}" created`);
        bootstrap.Modal.getInstance(document.getElementById('modalShare')).hide();
        loadShares();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

async function deleteShare(name) {
    if (!confirm(`Remove share "${name}" from config? Files are NOT deleted.`)) return;
    try {
        await api('DELETE', `/api/shares/${name}`);
        alert(`Share "${name}" removed`);
        loadShares();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

// ─── Permissions (Quick View) ────────────────────────────
async function initPermissions() {
    const d = await api('GET', '/api/shares');
    S.shares = d.shares;
    const sel = document.getElementById('perm-share-select');
    sel.innerHTML = '<option value="">-- Select a share --</option>';
    S.shares.forEach(s => sel.innerHTML += `<option value="${s.name}">${s.name}</option>`);
    document.getElementById('perm-container').innerHTML = '';
}

async function loadPermissions() {
    const name = document.getElementById('perm-share-select').value;
    const container = document.getElementById('perm-container');
    if (!name) { container.innerHTML = ''; return; }
    try {
        const s = await api('GET', `/api/shares/${name}`);
        container.innerHTML = `
            <div class="row g-3">
                <div class="col-md-6">
                    <div class="card">
                        <div class="card-header bg-primary text-white">
                            <i class="bi bi-person-check-fill"></i> Valid Users
                        </div>
                        <div class="card-body">
                            <div class="mb-3">${s.valid_users.length
                                ? s.valid_users.map(u=>`
                                    <span class="badge bg-primary me-1 mb-1">
                                    ${u}
                                    <i class="bi bi-x revoke-x"
                                       onclick="quickRevoke('${name}','valid_users','${u}',${JSON.stringify(s.valid_users)})">
                                    </i></span>`).join('')
                                : '<span class="text-muted small">No restrictions (all users)</span>'}
                            </div>
                            <div class="input-group input-group-sm">
                                <input type="text" class="form-control" id="qi-valid"
                                       placeholder="username or @group">
                                <button class="btn btn-primary"
                                        onclick="quickGrant('${name}','valid_users',${JSON.stringify(s.valid_users)})">
                                    <i class="bi bi-plus-lg"></i> Grant
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="col-md-6">
                    <div class="card">
                        <div class="card-header bg-success text-white">
                            <i class="bi bi-pencil-square"></i> Write List
                        </div>
                        <div class="card-body">
                            <div class="mb-3">${s.write_list.length
                                ? s.write_list.map(u=>`
                                    <span class="badge bg-success me-1 mb-1">
                                    ${u}
                                    <i class="bi bi-x revoke-x"
                                       onclick="quickRevoke('${name}','write_list','${u}',${JSON.stringify(s.write_list)})">
                                    </i></span>`).join('')
                                : '<span class="text-muted small">No write access granted</span>'}
                            </div>
                            <div class="input-group input-group-sm">
                                <input type="text" class="form-control" id="qi-write"
                                       placeholder="username or @group">
                                <button class="btn btn-success"
                                        onclick="quickGrant('${name}','write_list',${JSON.stringify(s.write_list)})">
                                    <i class="bi bi-plus-lg"></i> Grant Write
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="col-12">
                    <div class="card">
                        <div class="card-header"><i class="bi bi-gear-fill"></i> Share Settings</div>
                        <div class="card-body">
                            <div class="row g-3">
                                <div class="col-md-4">
                                    <div class="form-check form-switch">
                                        <input class="form-check-input" type="checkbox" id="sw-readonly"
                                               ${s.read_only ? 'checked' : ''}
                                               onchange="toggleSetting('${name}','read_only',this.checked)">
                                        <label class="form-check-label">
                                            <i class="bi bi-lock text-info"></i> Read-Only
                                        </label>
                                    </div>
                                </div>
                                <div class="col-md-4">
                                    <div class="form-check form-switch">
                                        <input class="form-check-input" type="checkbox" id="sw-guest"
                                               ${s.guest_ok ? 'checked' : ''}
                                               onchange="toggleSetting('${name}','guest_ok',this.checked)">
                                        <label class="form-check-label">
                                            <i class="bi bi-person-dash text-warning"></i> Guest OK
                                        </label>
                                    </div>
                                </div>
                                <div class="col-md-4">
                                    <div class="form-check form-switch">
                                        <input class="form-check-input" type="checkbox" id="sw-browse"
                                               ${s.browseable ? 'checked' : ''}
                                               onchange="toggleSetting('${name}','browseable',this.checked)">
                                        <label class="form-check-label">
                                            <i class="bi bi-eye text-success"></i> Browseable
                                        </label>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>`;
    } catch(e) { alert('Failed: ' + e.message, 'danger'); }
}

// ── Quick Grant ───────────────────────────────────────────
async function quickGrant(shareName, permType, currentList) {
    const inputId = permType === 'valid_users' ? 'qi-valid' : 'qi-write';
    const newUser = document.getElementById(inputId)?.value.trim();
    if (!newUser) { alert('Enter a username or @group', 'warning'); return; }
    if (currentList.includes(newUser)) { alert(`"${newUser}" already in list`, 'warning'); return; }

    try {
        await api('POST', `/api/shares/${shareName}/permissions`, {
            users: [...currentList, newUser],
            permission_type: permType
        });
        alert(`✅ Granted "${newUser}" on share "${shareName}"`);
        loadPermissions();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

// ── Quick Revoke ──────────────────────────────────────────
async function quickRevoke(shareName, permType, username, currentList) {
    if (!confirm(`Revoke "${username}" from ${permType.replace('_',' ')} on "${shareName}"?`)) return;

    try {
        await api('POST', `/api/shares/${shareName}/permissions`, {
            users: currentList.filter(u => u !== username),
            permission_type: permType
        });
        alert(`🚫 Revoked "${username}" from share "${shareName}"`);
        loadPermissions();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

// ── Toggle Setting ────────────────────────────────────────
async function toggleSetting(shareName, key, value) {
    try {
        await api('PATCH', `/api/shares/${shareName}`, { [key]: value });
        alert(`Setting updated: ${key} = ${value}`, 'info');
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

// ── Permission Modal (from Shares card) ───────────────────
async function openPermModal(shareName) {
    S.currentShare = shareName;
    document.getElementById('perm-modal-title').innerHTML =
        `<i class="bi bi-lock-fill"></i> Permissions — ${shareName}`;
    document.getElementById('perm-modal-body').innerHTML =
        '<div class="text-center py-4"><div class="spinner-border text-primary"></div></div>';

    new bootstrap.Modal(document.getElementById('modalPerm')).show();

    try {
        const [share, usersData, groupsData] = await Promise.all([
            api('GET', `/api/shares/${shareName}`),
            api('GET', '/api/users'),
            api('GET', '/api/groups'),
        ]);

        const principals = [
            ...usersData.users.map(u => ({ label: u.username, value: u.username, type: 'user' })),
            ...groupsData.groups.map(g => ({ label: g, value: `@${g}`, type: 'group' })),
        ];

        document.getElementById('perm-modal-body').innerHTML = `
            <div class="mb-3">
                <p class="text-muted small mb-2">
                    <i class="bi bi-info-circle"></i>
                    Tick <strong>Valid</strong> = allow access &nbsp;|&nbsp;
                    Tick <strong>Write</strong> = allow read+write
                </p>
            </div>
            <div class="border rounded" style="max-height:400px;overflow-y:auto">
                ${principals.map(p => {
                    const isValid = share.valid_users.includes(p.value);
                    const isWrite = share.write_list.includes(p.value);
                    return `
                        <div class="d-flex align-items-center px-3 py-2 border-bottom">
                            <span class="me-auto">
                                <i class="bi bi-${p.type === 'group'
                                    ? 'diagram-3 text-info'
                                    : 'person-circle text-primary'} me-2"></i>
                                <strong>${p.label}</strong>
                                ${p.type === 'group'
                                    ? '<span class="badge bg-info text-dark ms-2">group</span>'
                                    : ''}
                            </span>
                            <div class="d-flex gap-4 ms-3">
                                <div class="form-check mb-0">
                                    <input class="form-check-input" type="checkbox"
                                           id="v_${p.value}" value="${p.value}"
                                           data-ptype="valid" ${isValid ? 'checked' : ''}>
                                    <label class="form-check-label small text-muted"
                                           for="v_${p.value}">Valid</label>
                                </div>
                                <div class="form-check mb-0">
                                    <input class="form-check-input" type="checkbox"
                                           id="w_${p.value}" value="${p.value}"
                                           data-ptype="write" ${isWrite ? 'checked' : ''}>
                                    <label class="form-check-label small text-muted"
                                           for="w_${p.value}">Write</label>
                                </div>
                            </div>
                        </div>`;
                }).join('')}
            </div>`;
    } catch(e) {
        document.getElementById('perm-modal-body').innerHTML =
            `<div class="alert alert-danger">Failed to load: ${e.message}</div>`;
    }
}

async function saveModalPermissions() {
    const shareName  = S.currentShare;
    const validUsers = [...document.querySelectorAll('[data-ptype="valid"]:checked')].map(el => el.value);
    const writeList  = [...document.querySelectorAll('[data-ptype="write"]:checked')].map(el => el.value);

    try {
        await api('POST', `/api/shares/${shareName}/permissions`,
                  { users: validUsers, permission_type: 'valid_users' });
        await api('POST', `/api/shares/${shareName}/permissions`,
                  { users: writeList, permission_type: 'write_list' });

        alert(`✅ Permissions saved for "${shareName}"`);
        bootstrap.Modal.getInstance(document.getElementById('modalPerm')).hide();
        loadShares();
    } catch(e) { alert('Error: ' + e.message, 'danger'); }
}

// ─── Clock & Boot ────────────────────────────────────────
setInterval(() => {
    document.getElementById('clock').textContent =
        new Date().toLocaleString('th-TH');
}, 1000);

document.addEventListener('DOMContentLoaded', () => {
    loadDashboard();
});
</script>
</body>
</html>
HTMLEOF

# ─── CSS ─────────────────────────────────────────────────
cat > "$PROJECT_DIR/webapp/static/style.css" << 'EOF'
html, body { height: 100%; margin: 0; background: #f0f2f5; font-family: 'Segoe UI', sans-serif; }

/* Sidebar */
.sidebar {
    width: 230px; min-height: 100vh; flex-shrink: 0;
    position: sticky; top: 0; height: 100vh; overflow-y: auto;
    display: flex; flex-direction: column;
}
.sidebar .nav-link {
    color: #adb5bd; border-radius: 8px;
    padding: 10px 14px; transition: all .2s;
}
.sidebar .nav-link:hover, .sidebar .nav-link.active {
    background: rgba(255,255,255,.12); color: #fff;
}
.sidebar .nav-link i   { margin-right: 8px; }

/* Topbar */
.topbar { flex-shrink: 0; }

/* Content */
.main-content { background: #f0f2f5; overflow-y: auto; }
.content      { max-width: 1400px; }

/* Tabs */
.tab-pane         { display: none; }
.tab-pane.active  { display: block; }

/* Cards */
.card {
    border: none; border-radius: 12px;
    box-shadow: 0 2px 10px rgba(0,0,0,.07);
}
.card-header { border-radius: 12px 12px 0 0 !important; }

/* Stat cards */
.stat-card .card-body { text-align: center; padding: 1.5rem; }
.stat-icon  { font-size: 2rem; margin-bottom: .5rem; }
.stat-num   { font-size: 2rem; font-weight: 700; line-height: 1; }
.stat-label { color: #6c757d; font-size: .85rem; margin-top: .25rem; }

/* Share cards */
.share-card { transition: transform .2s, box-shadow .2s; }
.share-card:hover { transform: translateY(-3px); box-shadow: 0 6px 20px rgba(0,0,0,.12); }

/* Revoke X button inside badge */
.revoke-x { cursor: pointer; opacity: .7; margin-left: 4px; }
.revoke-x:hover { opacity: 1; color: #ff6b6b !important; }

/* Responsive sidebar */
@media (max-width: 768px) {
    .sidebar { width: 58px; }
    .sidebar span, .sidebar small { display: none; }
    .sidebar .nav-link i { margin-right: 0; font-size: 1.3rem; }
    .sidebar-header h5 span { display: none; }
}
EOF

# ─── Step 8: Permissions & Ownership ─────────────────────
echo "[8/9] Setting permissions..."
chmod -R 755 "$PROJECT_DIR"
chmod +x "$PROJECT_DIR/samba/entrypoint.sh"
find "$PROJECT_DIR" -name "*.py" -exec chmod 644 {} \;

# ─── Step 9: Done ─────────────────────────────────────────
echo ""
echo -e "[9/9] ${GREEN}Setup complete!${NC}"
echo ""
echo -e "Project created at: ${YELLOW}$PROJECT_DIR${NC}"
echo ""
echo "Files created:"
find "$PROJECT_DIR" -type f | sort | sed 's|'"$PROJECT_DIR"'/|  ✅ |'
echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  Next: cd $PROJECT_DIR${NC}"
echo -e "${GREEN}  Then: docker compose up -d --build${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"

