# Migration Design: FastAPI `webapp/` → SvelteKit `frontend/` + Go Fiber `backend/`

**Date:** 2026-06-19
**Status:** ✅ Completed — merged via PR #3 (`migrate/frameworks` → `main`)

## Known Gaps (post-merge — tracked for follow-up sprints)

| Gap | Severity | Notes |
|-----|----------|-------|
| Audit logging not wired | HIGH | `auditSvc` not passed to shares/users/groups/auth/builtin handlers. Zero mutating actions logged. Fix: pass `auditSvc` to each `register*Routes()`, call `auditSvc.Log()` in every mutating handler. |
| `audit.json` world-readable | MED | Saved `0o644` in `audit.go:61`. Change to `0o640`. |
| `abseRequest` dead code | LOW | Struct at `shares.go:45` unused — `toggleABSE` reads `?enabled=` query param not body. Delete struct or migrate handler to use body. |
| `delete` share no existence check | LOW | `shares.go:158` returns 200 even if share missing. Add `ShareExists` guard. |

## Goal

Replace the existing FastAPI `webapp/` (Python, Jinja2 SPA) with two pieces:

- `frontend/` — SvelteKit 2 / Svelte 5 / Tailwind v4 SPA (already scaffolded).
- `backend/` — Go + Fiber API that also serves the built frontend.

This is a **port + restructure**: reproduce all existing behavior, clean up boundaries
and known rough edges (notably the broken multi-admin endpoints), but do not add
unrelated features.

## Decisions (locked)

| Topic | Decision |
|-------|----------|
| Deployment | Single container. Go Fiber serves the static SPA **and** the API on port 8080. Drop-in replacement for the current `webapp` service. |
| Frontend rendering | Static SPA via `@sveltejs/adapter-static` with SPA fallback; `ssr = false`. No Node server. |
| Fidelity | Port + restructure/improve. |
| LDAP | Keep exec'ing `ldapsearch` inside `samba-server` via the Docker socket (reuses the container's Kerberos/AD creds); parse LDIF in Go. |
| Samba control | Official Docker SDK for Go (`github.com/docker/docker/client`), mirroring `DockerExecutor`. |
| Data formats | Keep `shares.conf`, `audit.json`, `builtin_groups.json` byte/format-compatible. `.admin` migrates to a multi-admin list (see below). |
| Multi-admin | Implement real multi-admin support in Go (the current router calls `list_admins`/`add_admin`/`delete_admin`, which `auth_service.py` never implemented). Auto-upgrade an old single-object `.admin` file to a one-element list on startup. |
| API contract | Keep all existing paths identical so behavior is verifiable 1:1. |

## Architecture

### Topology

```
┌──────────────────────────────┐        ┌────────────────────────────┐
│  backend (Go Fiber)           │  exec  │  samba-server              │
│  container: smb-webapp        │───────▶│  (network_mode: host)      │
│  - serves SPA static assets   │ docker │  nmbd / winbindd / smbd    │
│  - serves JSON API :8080      │ socket │  ldapsearch, smbpasswd...  │
└──────────────────────────────┘        └────────────────────────────┘
        ▲ shares.conf, /mnt/shared, /var/run/docker.sock (bind mounts)
```

The `samba` container is unchanged. All Samba control continues to go through the
Docker socket — never network calls.

### Backend layout (`backend/`)

```
backend/
  cmd/server/main.go          # Fiber bootstrap: config load, middleware, routes,
                              #   static asset serving + SPA fallback to index.html
  internal/
    config/
      config.go               # env + .env loading; mirrors webapp/config.py defaults
    samba/
      executor.go             # Executor interface + dockerExecutor (Docker SDK).
                              #   exec_run as root in samba-server; helpers:
                              #   create/delete user, set password, create group,
                              #   add/remove group member, list users/groups,
                              #   reload samba.
      conf.go                 # SmbConfParser port: parse/save shares.conf,
                              #   get/create/update/delete share, global read-only.
      conf_format.go          # user-list parse/format/sanitize helpers
                              #   (quoting, AD backslash, @group handling).
      matrix.go               # syncSharePermissions — ported verbatim from
                              #   _sync_share_permissions (invalid > admin > write > read).
    ldap/
      ldap.go                 # search_users / search_groups / get_user / test_connection
                              #   via Executor-run ldapsearch.
      ldif.go                 # LDIF parser: base64 (`::`) decode, multi-line
                              #   continuation, repeated-attr -> slice.
    auth/
      auth.go                 # bcrypt (cost 12) + JWT HS256 (24h); admin store
                              #   (multi-admin list in .admin, chmod 600);
                              #   authenticate, change_password, list/add/delete admin.
    audit/
      audit.go                # append-only JSON log, cap 10000, newest-first reads,
                              #   action/actor filters.
    httpapi/
      router.go               # mounts all route groups
      middleware.go           # auth middleware (cookie or Bearer; public allowlist)
      users.go groups.go shares.go ad.go builtin.go auth.go audit.go health.go
  go.mod
  go.sum
  Dockerfile                  # multi-stage build (see below)
```

**Key isolation point:** `samba.Executor` is an interface. `conf.go`, `ldap`, and all
handlers depend on the interface, so they unit-test against a fake executor with no
live Docker/Samba.

### Frontend layout (`frontend/src`)

```
frontend/
  svelte.config.js            # adapter-static, fallback: 'index.html' (SPA mode)
  src/
    app.html
    routes/
      +layout.svelte          # shell: nav/tabs, auth guard, current-user chip
      +layout.ts              # ssr = false; prerender = false
      +page.svelte            # dashboard landing (redirect to shares)
      login/+page.svelte
      shares/+page.svelte     # share list + CRUD + permission matrix editor
      users/+page.svelte      # local Samba users CRUD
      groups/+page.svelte     # local Linux groups
      ad/+page.svelte         # AD user/group search, status, OUs
      builtin/+page.svelte    # BUILTIN\ group membership
      audit/+page.svelte      # audit log viewer with filters
      settings/+page.svelte   # admins list + add/delete + change password
    lib/
      api/
        client.ts             # fetch wrapper: base URL, JSON, attaches token,
                              #   on 401 clears session + redirect to /login
        shares.ts users.ts groups.ts ad.ts builtin.ts audit.ts auth.ts
      stores/
        auth.ts               # current user + token (cookie-backed) state
      components/             # shared UI (table, modal, toast, permission-list, ...)
  static/
```

Tailwind v4 (already configured) for styling. The shares permission editor manages the
five lists — `valid_users`, `write_list`, `read_list`, `admin_users`, `invalid_users` —
sending one list at a time to `POST /api/shares/{name}/permissions`, relying on the
backend matrix sync to keep them consistent.

## API contract (paths unchanged)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/health` | public |
| POST | `/auth/login` | public; returns `{access_token, token_type, expires_in}` |
| POST | `/auth/logout` | clears cookie |
| GET | `/auth/me` | current user |
| POST | `/auth/change-password` | |
| GET | `/auth/admins` | list admins |
| POST | `/auth/admins` | add admin (min password len 8) |
| DELETE | `/auth/admins/{username}` | cannot delete self / last admin |
| GET/POST | `/api/users` | list / create local Samba user |
| DELETE | `/api/users/{username}` | |
| POST | `/api/users/{username}/password` | |
| GET/POST | `/api/groups` | list / create local group |
| POST/DELETE | `/api/groups/{group}/members/{username}` | |
| GET | `/api/shares` | list all shares |
| POST | `/api/shares` | create (mkdir+chmod 777, then config) |
| GET/PATCH/DELETE | `/api/shares/{name}` | |
| GET/PATCH | `/api/shares/global` | global is read-only; PATCH is a no-op that still reloads |
| POST | `/api/shares/{name}/permissions` | `{users, permission_type}` |
| PATCH | `/api/shares/{name}/abse` | `?enabled=` |
| GET | `/api/ad/status` | LDAP connection status |
| GET | `/api/ad/users` | `?q=&ou=&limit=` |
| GET | `/api/ad/users/{username}` | |
| GET | `/api/ad/groups` | `?q=&limit=` |
| GET | `/api/ad/ous` | static OU list |
| GET | `/api/builtin` | groups + members |
| GET | `/api/builtin/{group}/members` | |
| POST | `/api/builtin/{group}/members` | `{username}` |
| DELETE | `/api/builtin/{group}/members/{username}` | |
| GET | `/api/audit/logs` | `?limit=&action=&actor=` |

Every mutating handler writes an audit entry (same `action`/`resource_type` values as
today) and, where applicable, calls `reload samba`.

## Permission matrix (ported verbatim)

`syncSharePermissions` runs before every save when a user list changes:

1. `invalid_users` — removed from all other lists (highest priority).
2. `admin_users` — removed from `write_list`/`read_list`; unioned into `valid_users`.
3. `write_list` — removed from `read_list`; unioned into `valid_users`.
4. `read_list` — unioned into `valid_users`.

Then all five lists are sanitized (strip disallowed chars) and formatted back to
`shares.conf` syntax (AD `IT\user` and `@Group` quoting preserved).

## Auth & multi-admin

- bcrypt cost 12; JWT HS256 with 24h expiry; `secret_key` from env.
- Token accepted from `access_token` cookie or `Authorization: Bearer`.
- Public allowlist: `/login`, `/health`, `/auth/login`, static assets, SPA fallback.
- `.admin` file (chmod 600) becomes a JSON **list** of admin records
  `{username, hashed_password, created_at, last_login}`.
- **Startup migration:** if `.admin` parses as a single object (old format), wrap it in
  a one-element list and rewrite.
- Guards: cannot delete the last remaining admin; cannot delete yourself.

## Data formats

- `shares.conf` — identical syntax, keys, masks (`create mask`/`directory mask` 0775,
  `force create mode`/`force directory mode` 0777), and managed-file header comment.
- `audit.json` — same record shape, cap 10000, indent 2.
- `builtin_groups.json` — same `{group: [usernames]}` shape; six groups with
  description/color/icon metadata served by the API.
- `.admin` — multi-admin list (see above) with one-time upgrade.

## Docker / compose changes

- New `backend/Dockerfile`, multi-stage:
  1. `node` stage: `pnpm install && pnpm build` in `frontend/` → static output.
  2. `golang` stage: `go build` the server binary.
  3. runtime (distroless/alpine): copy binary + frontend build; run server.
- `docker-compose.yml`: repoint the `webapp` service `build.context` to `./backend`
  (build context must reach `frontend/` too — use repo root context with a Dockerfile
  arg, or copy frontend in a pre-step). Keep container name `smb-webapp`, port `8080`,
  and all existing volumes (`samba-data`, `shares.conf`, `./data/shared`,
  `/var/run/docker.sock`). The Python `webapp/` dir stays in the repo until the Go stack
  is verified, then can be removed.
- `.env` keys unchanged (`SECRET_KEY`, `LDAP_*`); add nothing required.

## Testing

- **Go (unit, no live Samba):**
  - matrix: table-driven cases covering each priority rule and overlaps.
  - conf: parse → mutate → save → re-parse round-trip; quoting/backslash/`@group`.
  - auth: bcrypt verify, JWT sign/verify/expiry, `.admin` migration.
  - ldif: fixtures with base64 values, multi-line continuation, repeated attrs.
  - handlers: against a fake `Executor`.
- **Frontend:** Vitest for the API client (401 redirect, token attach) and key
  components (permission-list editor).
- **End-to-end:** manual — `docker compose up -d --build`, exercise login, share CRUD +
  permissions, user/group, AD search, builtin, audit. (No Samba in CI, same as today.)

## Implementation phases

1. **Scaffold:** Go module, Fiber, `config`, `/health`, static + SPA serving, auth
   (bcrypt/JWT, login, middleware, `.admin` multi-admin + migration). Verify login.
2. **Executor + simple resources:** Docker SDK executor behind interface; users +
   groups handlers.
3. **Shares:** `conf.go` + `matrix.go` (with unit tests) → shares handlers incl.
   permissions, abse, global.
4. **AD:** ldapsearch executor calls + LDIF parser → AD handlers.
5. **Audit + builtin:** audit log service; builtin group store + `net sam` apply;
   wire audit into all mutating handlers.
6. **Frontend:** adapter-static config, auth/login + API client, then each resource
   view (shares, users, groups, ad, builtin, audit, settings).
7. **Package & verify:** multi-stage Dockerfile, docker-compose swap, full end-to-end
   pass; then retire `webapp/`.

## Out of scope (migration only)

- Changing Samba global config / `smb.conf` (still template-generated, read-only to UI). *(ABSE sprint adds `access based share enum = yes` to template — tracked in ABSE spec.)*
- New features beyond current parity (except the multi-admin fix already in the router).
- Automated end-to-end / Samba-in-CI.
