# SMB Permission Manager

Web UI สำหรับจัดการ Samba file shares บนเซิร์ฟเวอร์ที่ join domain IT.KMITL.AC.TH

**Stack**: Go Fiber (backend API) · SvelteKit 2 / Svelte 5 (frontend SPA) · Docker

---

## Quick Start

```bash
# 1. สร้าง directory สำหรับเก็บข้อมูล
mkdir -p data/shared

# 2. สร้าง .env จาก template
make setup-env
# แล้วแก้ SECRET_KEY และ LDAP_BIND_PW ใน .env

# 3. Start
make up

# 4. เปิด browser
open http://localhost:8080
```

ครั้งแรกที่เปิดจะให้สร้าง admin account (first-run setup)

---

## Environment Variables (`.env`)

| Variable | Description | Default |
|---|---|---|
| `SECRET_KEY` | JWT signing key — **เปลี่ยนก่อน deploy** | `xxxxxxxxxxxxxxxxxxxxxx` |
| `LDAP_SERVER` | IP ของ AD/LDAP server | `xxx.xxx.xxx.xxx` |
| `LDAP_PORT` | LDAP port | `xxxx` |
| `LDAP_BASE_DN` | Base DN | `DC=xxx` |
| `LDAP_BIND_DN` | Bind account UPN | `xxxxxxxxx@Xxxxxxxxx` |
| `LDAP_BIND_PW` | Bind account password | *(empty)* |
| `LDAP_DOMAIN` | AD domain name | `xxxxxx` |
| `TOKEN_EXPIRY_MINUTES` | JWT lifetime | `xxxx` (24 h) |
| `SAMBA_CONTAINER` | ชื่อ Samba container | `xxxxxxxx` |

---

## Makefile Commands

### Core

```bash
make up           # Build images และ start ทุก service (detached)
make down         # Stop และ remove containers (volumes ยังอยู่)
make restart      # Restart เฉพาะ backend container
make build        # Rebuild ทั้งสอง image โดยไม่ start
make build-be     # Rebuild เฉพาะ backend image (เร็วกว่า)
make ps           # แสดงสถานะ container
```

### Logs

```bash
make logs         # ดู log ทุก service
make logs-be      # ดู log เฉพาะ backend
make logs-samba   # ดู log เฉพาะ samba
```

### Testing

```bash
make test         # รัน Go unit tests
make test-cover   # รัน tests พร้อม coverage report (HTML)
make test-fe      # Type-check + build frontend
make lint-fe      # ESLint บน frontend source
```

### Admin Recovery

```bash
make show-admins                         # แสดง admin username ทั้งหมดใน .admin
make reset-admin USER=admin PASS=secret  # สร้าง admin ผ่าน /auth/setup endpoint
make wipe-admin                          # ลบ .admin ทั้งหมด (ต้องยืนยันก่อน)
```

### Utilities

```bash
make shell-be     # เปิด shell ใน backend container
make shell-samba  # เปิด shell ใน samba container
make audit        # แสดง 50 audit entries ล่าสุด
make clean        # ลบ stopped containers + dangling images
make clean-all    # ลบ containers + images + volumes (ระวัง: ข้อมูลหาย)
```

---

## Features

### Dashboard
- **Stats cards** — จำนวน shares, local users, groups
- **Samba service status** — smbd / nmbd / winbindd (running / stopped)
- **AD connection status** — domain, server, error message
- **Recent activity** — 5 audit entries ล่าสุด พร้อมลิงก์ไปหน้า Audit Log

### Shares
- สร้าง / ลบ share (สร้าง directory ใน container ให้อัตโนมัติ)
- **Clickable property toggles** — คลิก badge เพื่อสลับ `browseable`, `read_only`, `guest_ok` ทันที
- **Permission editor** — เลือก permission list แล้วระบุ user/group ที่ต้องการ
- **Autocomplete suggestions** — แสดง local users + groups ให้คลิกเพิ่มได้เลย, กรองตาม token ที่กำลังพิมพ์
- **Permission help** — ปุ่ม "How it works" อธิบาย priority rules และ username formats
- Search + pagination บน sidebar list

#### Permission Priority (sync ทุกครั้งที่บันทึก)

| ลำดับ | List | พฤติกรรม |
|---|---|---|
| 1 | `invalid_users` | บล็อกจากทุก list อื่น |
| 2 | `admin_users` | เอาออกจาก write/read list, auto-add ไป valid_users |
| 3 | `write_list` | เอาออกจาก read_list, auto-add ไป valid_users |
| 4 | `read_list` | auto-add ไป valid_users |

#### Username Formats

```
alice              # local Samba user
IT\username        # Active Directory user
@groupname         # local Linux group
@"Group Name"      # AD group ที่มีช่องว่าง
```

### Users (Local Samba)
- สร้าง / ลบ local Samba user
- เปลี่ยน password ได้ inline
- **User → Shares view** — คลิก username เพื่อดูว่า user นั้นอยู่ใน share ไหนบ้าง และมี permission อะไร
- Search + pagination

### Groups (Local Linux)
- สร้าง local Linux group
- เพิ่ม user เข้า group
- Search + pagination

### Active Directory
- ค้นหา AD users / groups
- **Add to share** — ปุ่มบนแต่ละ row เพื่อเพิ่ม AD user/group เข้า share ที่เลือก พร้อม permission list ได้ทันที

### BUILTIN Groups
- จัดการ Windows `BUILTIN\` group membership (Administrators, Users, Guests ฯลฯ)
- sync ผ่าน `net sam addmem/delmem` ใน Samba container
- ข้อมูลเก็บใน `/mnt/shared/builtin_groups.json`

### Audit Log
- บันทึกทุก API call ที่เปลี่ยนแปลงข้อมูล
- Filter ตาม action / actor
- **Export CSV** — ดาวน์โหลด log เป็นไฟล์ `.csv`
- Pagination (default 25/page, ปรับได้ถึง 100/page)
- เก็บสูงสุด 10,000 entries ใน `/mnt/shared/audit.json`

### Settings
- เปลี่ยน password ของ admin ที่ login อยู่
- เพิ่ม / ลบ admin account
- Admin credentials เก็บใน `/mnt/shared/.admin` (bcrypt cost 12, chmod 600)

### อื่นๆ
- **Session timeout warning** — แจ้งเตือน 10 นาทีก่อน JWT หมดอายุ
- **Sidebar collapsible** — คลิก hamburger เพื่อซ่อน/แสดง sidebar
- First-run setup: ถ้ายังไม่มี admin จะ redirect ไปหน้า `/setup` อัตโนมัติ

---

## Architecture

```
┌─────────────────────────────────────┐
│  Browser (SvelteKit SPA)            │
│  http://localhost:8080              │
└─────────────┬───────────────────────┘
              │ HTTP / REST JSON
┌─────────────▼───────────────────────┐
│  backend container (Go Fiber)       │
│  - Serves compiled SPA (./build)    │
│  - /auth/* — JWT auth               │
│  - /api/shares, users, groups …     │
│  - /api/stats, samba/status         │
│  - Mounts Docker socket             │
└─────────────┬───────────────────────┘
              │ docker.sock exec_run
┌─────────────▼───────────────────────┐
│  samba-server container             │
│  - smbd / nmbd / winbindd           │
│  - network_mode: host               │
│  - Joined to IT.KMITL.AC.TH domain  │
└─────────────────────────────────────┘
```

### Config Files

| ไฟล์ | จัดการโดย | หมายเหตุ |
|---|---|---|
| `samba/smb.conf` | Template (auto-gen) | Global AD settings, ไม่แก้ผ่าน UI |
| `samba/shares.conf` | Web UI | Share definitions เท่านั้น |
| `data/shared/.admin` | Web UI / Makefile | Admin accounts (bcrypt) |
| `data/shared/audit.json` | Backend | Audit log |
| `data/shared/builtin_groups.json` | Web UI | BUILTIN group membership |

---

## Development

### Backend (Go Fiber)

```bash
# แก้ backend/**/*.go แล้ว:
make build-be && make restart && make logs-be

# รัน tests โดยไม่ต้อง Docker:
make test
```

### Frontend (SvelteKit)

```bash
# Dev server พร้อม hot-reload (proxy ไป localhost:8080):
cd frontend
pnpm install
pnpm dev      # http://localhost:5173

# Build static files:
pnpm build
```

```bash
# หลังแก้ frontend/src/** แล้วอยากดูใน Docker:
make build-be && make restart
```

---

## Troubleshooting

| อาการ | วิธีตรวจสอบ |
|---|---|
| Samba container unhealthy | `make logs-samba` — หา Kerberos / AD join error |
| AD search ไม่เจอข้อมูล | ตรวจ `LDAP_BIND_PW` ใน `.env`; `make shell-samba` แล้วรัน `wbinfo -u` |
| Login ไม่ได้ / ลืม password | `make show-admins` ดู username; `make reset-admin USER=x PASS=y` หรือ `make wipe-admin` |
| `shares.conf` permission denied | `ls -la samba/shares.conf` — ต้องให้ write permission |
| หน้า Dashboard โหลดนาน | ตรวจว่า samba container กำลัง run: `make ps` |
| JWT หมดอายุ (session warning) | ออกแล้ว login ใหม่; หรือเพิ่ม `TOKEN_EXPIRY_MINUTES` ใน `.env` |
