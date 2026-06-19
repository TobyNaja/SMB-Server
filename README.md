# SMB Permission Manager

Web UI for managing Samba file shares on a server joined to the IT.KMITL.AC.TH Active Directory domain.

## Requirements

- Docker + Docker Compose
- A `.env` file (see below)
- `./data/shared/` directory must exist before first start

## Quick Start

```bash
# 1. Create the shared data directory
mkdir -p data/shared

# 2. Create .env from the template
cp .env.example .env
# Edit .env and fill in SECRET_KEY and LDAP credentials

# 3. Build and start
make up

# 4. Open the web UI
open http://localhost:8080
```

On first login you will be prompted to create the initial admin account.

## Environment Variables (`.env`)

| Variable | Description | Default |
|---|---|---|
| `SECRET_KEY` | JWT signing key — **change in production** | `dev-secret-key-change-in-production` |
| `LDAP_SERVER` | AD/LDAP server IP | `10.70.37.143` |
| `LDAP_PORT` | LDAP port | `389` |
| `LDAP_BASE_DN` | Base DN | `DC=it,DC=kmitl,DC=ac,DC=th` |
| `LDAP_BIND_DN` | Bind account UPN | `ldap-bind-nas@IT.KMITL.AC.TH` |
| `LDAP_BIND_PW` | Bind account password | *(empty)* |
| `LDAP_DOMAIN` | AD domain name | `IT.KMITL.AC.TH` |
| `TOKEN_EXPIRY_MINUTES` | JWT lifetime | `1440` (24 h) |
| `SAMBA_CONTAINER` | Samba container name | `samba-server` |

## Common Operations

```bash
make up          # Build images and start all services (detached)
make down        # Stop and remove containers
make restart     # Restart only the backend (after config change)
make logs        # Follow all logs
make logs-be     # Follow backend logs only
make logs-samba  # Follow samba logs only
make ps          # Show running containers and health status

make build       # Rebuild both images without starting
make build-be    # Rebuild backend image only (faster iteration)

make test        # Run Go unit tests
make test-fe     # Run frontend type-check + build
make lint-fe     # Run ESLint on frontend source

make shell-be    # Open a shell inside the backend container
make shell-samba # Open a shell inside the samba container
```

## Development Workflow

### Backend (Go Fiber)

The Go binary is compiled into the Docker image — code changes require a rebuild:

```bash
# Edit backend/**/*.go, then:
make build-be    # rebuild backend image
make restart     # restart container with new image
make logs-be     # watch logs
```

Run tests locally (no Docker needed):

```bash
make test
```

### Frontend (SvelteKit)

The SvelteKit SPA is compiled into the Docker image as static files:

```bash
# Edit frontend/src/**, then:
make build-be    # rebuilds both frontend and backend
make restart
```

For rapid local iteration without Docker:

```bash
cd frontend
pnpm install
pnpm dev         # starts dev server at http://localhost:5173
                 # API calls proxy to http://localhost:8080
```

## Authentication

- Single or multiple admin accounts stored in `/mnt/shared/.admin` (bcrypt, chmod 600)
- JWT HS256 tokens, 24-hour expiry, stored in an HTTP-only cookie
- Add/remove admins via the **Settings** page (or `make add-admin`)

## Shares Configuration

- Share definitions live in `./samba/shares.conf` (bind-mounted into both containers)
- Changes made through the UI are written immediately and applied via `smbcontrol reload-config`
- The global Samba config (`smb.conf`) is generated from `samba/smb.conf.template` on every container start and is **not** managed by the UI

## Audit Log

All mutating API calls are logged to `/mnt/shared/audit.json` (newest-first, capped at 10 000 entries). View them in the **Audit Log** page or directly:

```bash
make audit       # tail the last 50 audit entries
```

## Troubleshooting

| Symptom | Check |
|---|---|
| Samba container unhealthy | `make logs-samba` — look for Kerberos/AD join errors |
| AD search returns nothing | Verify `LDAP_BIND_PW` in `.env`; run `make shell-samba` and test `wbinfo -u` |
| Login fails after password change | JWT tokens are invalidated on restart; re-login after `make restart` |
| `shares.conf` permission denied | Ensure the file exists and is writable: `ls -la samba/shares.conf` |
