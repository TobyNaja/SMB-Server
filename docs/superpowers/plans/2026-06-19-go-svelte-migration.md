# webapp → Go Fiber + SvelteKit Migration Implementation Plan

> **Status: ✅ COMPLETED** — All parts shipped, merged via PR #3 to `main` on 2026-06-22.
> Known gaps tracked in `docs/superpowers/specs/2026-06-19-webapp-to-go-svelte-migration-design.md`.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace Python FastAPI `webapp/` with Go Fiber `backend/` (JSON API) + SvelteKit static SPA `frontend/`, deployed as a single Docker container — port 8080, drop-in for existing compose setup.

**Architecture:** Go Fiber serves compiled SvelteKit assets and the API. All Samba control via Docker SDK exec_run. `Executor` is an interface so all packages unit-test without live Docker/Samba.

**Tech Stack:** Go 1.22, Fiber v2, Docker SDK for Go, golang-jwt/jwt/v5, bcrypt, godotenv, testify; SvelteKit 2 + Svelte 5 + Tailwind v4, adapter-static, pnpm.

## Global Constraints

- Module path: `smb-server/backend`
- API paths identical to Python webapp (see design spec)
- `shares.conf` syntax: byte-compatible
- `audit.json`: same record shape, cap 10000, indent 2
- `builtin_groups.json`: same `{group: [usernames]}` shape
- `.admin`: auto-upgrade single-object → multi-admin list on first read
- JWT: HS256, 24h, HTTP-only cookie + Bearer header
- bcrypt cost: 12; password min length: 8
- **Before every backend commit:** `cd backend && go test ./... && go build ./cmd/server/`
- **Before every frontend commit:** `cd frontend && pnpm check && pnpm build`

## Branch Strategy

```
migrate/frameworks  (base)
  └── migrate/backend       ← all Go backend tasks (Parts 1-5)
        merged → migrate/frameworks
  └── migrate/frontend-impl ← all SvelteKit frontend tasks (Parts 6-7)
        merged → migrate/frameworks
```

---

## File Map

```
backend/
  cmd/server/main.go                  — Fiber bootstrap + route wiring + static serve + SPA fallback
  internal/config/config.go           — env/.env loading, Config struct
  internal/auth/
    auth.go                           — bcrypt, JWT, multi-admin store, .admin migration
    auth_test.go
  internal/samba/
    executor.go                       — Executor interface + dockerExecutor (Docker SDK) + FakeExecutor
    executor_test.go
    conf.go                           — SmbConfParser (parse/save shares.conf, CRUD)
    conf_format.go                    — parseUserList / formatUser / sanitizeUsers
    matrix.go                         — syncSharePermissions (permission matrix)
    matrix_test.go
    conf_test.go
  internal/ldap/
    ldif.go                           — LDIF parser (base64, multi-line, repeated attrs)
    ldap.go                           — search_users, search_groups, get_user, test_connection
    ldif_test.go
  internal/audit/
    audit.go                          — append JSON log, cap 10k, filtered read
    audit_test.go
  internal/builtin/
    builtin.go                        — group definitions, load/save store, apply to samba
  internal/httpapi/
    router.go                         — mount all route groups
    middleware.go                     — JWT auth middleware (cookie + Bearer, public allowlist)
    health.go
    auth.go                           — login, logout, me, change-password, admins CRUD
    users.go
    groups.go
    shares.go
    ad.go
    builtin.go
    audit.go
  go.mod / go.sum
  Dockerfile                          — multi-stage: node build FE → go build → alpine runtime

frontend/
  vite.config.ts                      — switch to adapter-static
  src/routes/+layout.ts               — ssr=false, prerender=false
  src/routes/+layout.svelte           — nav shell, auth guard
  src/routes/+page.svelte             — redirect to /shares
  src/routes/login/+page.svelte
  src/routes/shares/+page.svelte      — share list + CRUD + permission matrix editor
  src/routes/users/+page.svelte
  src/routes/groups/+page.svelte
  src/routes/ad/+page.svelte
  src/routes/builtin/+page.svelte
  src/routes/audit/+page.svelte
  src/routes/settings/+page.svelte
  src/lib/api/client.ts               — fetch wrapper, 401→redirect, token attach
  src/lib/api/{auth,users,groups,shares,ad,builtin,audit}.ts
  src/lib/stores/auth.ts              — current user state (svelte 5 rune-based)
  src/lib/components/                 — Toast, Modal, Table, PermissionEditor
```

---

## Part 1 — Go scaffold: config + auth + health (branch: migrate/backend)

**Files:** `backend/go.mod`, `internal/config/config.go`, `internal/auth/auth.go`, `internal/auth/auth_test.go`, `internal/httpapi/middleware.go`, `internal/httpapi/health.go`, `internal/httpapi/auth.go`, `internal/httpapi/router.go`, `cmd/server/main.go`

**Deliverable:** `go test ./...` green, `go build ./cmd/server/` succeeds, `/health` and `/auth/*` endpoints work.

## Part 2 — Executor + Users + Groups (branch: migrate/backend)

**Files:** `internal/samba/executor.go`, `internal/samba/executor_test.go`, `internal/httpapi/users.go`, `internal/httpapi/groups.go`

**Deliverable:** All handler tests pass using FakeExecutor. `/api/users` and `/api/groups` routes registered.

## Part 3 — SmbConfParser + matrix + Shares (branch: migrate/backend)

**Files:** `internal/samba/conf_format.go`, `internal/samba/matrix.go`, `internal/samba/matrix_test.go`, `internal/samba/conf.go`, `internal/samba/conf_test.go`, `internal/httpapi/shares.go`

**Deliverable:** Matrix tests pass (table-driven, all 4 priority rules). conf round-trip test passes. `/api/shares` routes work.

## Part 4 — LDAP + AD handlers (branch: migrate/backend)

**Files:** `internal/ldap/ldif.go`, `internal/ldap/ldif_test.go`, `internal/ldap/ldap.go`, `internal/httpapi/ad.go`

**Deliverable:** LDIF parser tests pass (base64, multi-line, repeated attrs). `/api/ad/*` routes registered.

## Part 5 — Audit + Builtin + route wiring + merge (branch: migrate/backend)

**Files:** `internal/audit/audit.go`, `internal/audit/audit_test.go`, `internal/builtin/builtin.go`, `internal/httpapi/builtin.go`, `internal/httpapi/audit.go` — wire audit into all mutating handlers, finalize `router.go`.

**Deliverable:** All routes complete, all tests pass, build succeeds. Merge `migrate/backend` → `migrate/frameworks`.

## Part 6 — SvelteKit frontend (branch: migrate/frontend-impl)

**Files:** `frontend/vite.config.ts` update, `+layout.ts`, `+layout.svelte`, login page, API client, auth store, all resource pages.

**Deliverable:** `pnpm build` succeeds, login page renders, all resource pages render without errors.

## Part 7 — Dockerfile + docker-compose + E2E verify (branch: migrate/frontend-impl)

**Files:** `backend/Dockerfile` (multi-stage), `docker-compose.yml` update.

**Deliverable:** `docker compose up -d --build` succeeds, full stack working. Merge `migrate/frontend-impl` → `migrate/frameworks`.
