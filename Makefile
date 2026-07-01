.PHONY: up down restart build build-be logs logs-be logs-samba ps \
        test test-fe lint-fe shell-be shell-samba audit add-admin clean help fix-perms

# ── Core ──────────────────────────────────────────────────────────────────────

up: ## Build images and start all services (detached)
	docker compose up -d --build

down: ## Stop and remove containers (keeps volumes)
	docker compose down

restart: ## Restart only the backend container
	docker compose restart backend

build: ## Rebuild both images without starting
	docker compose build

build-be: ## Rebuild backend image only (frontend + Go binary)
	docker compose build backend

# ── Logs ──────────────────────────────────────────────────────────────────────

logs: ## Follow logs for all services
	docker compose logs -f

logs-be: ## Follow backend logs
	docker compose logs -f backend

logs-samba: ## Follow samba logs
	docker compose logs -f samba

ps: ## Show container status and health
	docker compose ps

# ── Testing ───────────────────────────────────────────────────────────────────

test: ## Run Go unit tests
	cd backend && go test ./... -v

test-cover: ## Run Go tests with coverage report
	cd backend && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

test-fe: ## Type-check and build frontend (requires pnpm)
	cd frontend && pnpm check && pnpm build

lint-fe: ## Run ESLint on frontend source
	cd frontend && pnpm lint

# ── Shells ────────────────────────────────────────────────────────────────────

shell-be: ## Open a shell inside the backend container
	docker compose exec backend sh

shell-samba: ## Open a shell inside the samba container
	docker exec -it samba-server bash

# ── Utilities ─────────────────────────────────────────────────────────────────

audit: ## Print the last 50 audit log entries
	docker compose exec backend sh -c 'cat /mnt/shared/audit.json 2>/dev/null | head -c 1M' \
		| python3 -c "import sys,json; logs=json.load(sys.stdin); [print(json.dumps(e,indent=2)) for e in logs[:50]]" 2>/dev/null \
		|| docker exec samba-server cat /mnt/shared/audit.json 2>/dev/null | head -c 64K

add-admin: ## Create an admin account (usage: make add-admin USER=alice PASS=secret123)
	@test -n "$(USER)" || (echo "Usage: make add-admin USER=alice PASS=secret123" && exit 1)
	@test -n "$(PASS)" || (echo "Usage: make add-admin USER=alice PASS=secret123" && exit 1)
	curl -s -X POST http://localhost:8080/auth/admins \
		-H "Content-Type: application/json" \
		-b cookies.txt \
		-d '{"username":"$(USER)","password":"$(PASS)"}' | python3 -m json.tool

reset-admin: ## Reset/create admin via setup endpoint (usage: make reset-admin USER=admin PASS=newpassword)
	@test -n "$(USER)" || (echo "Usage: make reset-admin USER=admin PASS=newpassword" && exit 1)
	@test -n "$(PASS)" || (echo "Usage: make reset-admin USER=admin PASS=newpassword" && exit 1)
	@echo "--- Current admins in .admin file ---"
	@docker compose exec backend sh -c 'cat /mnt/shared/.admin 2>/dev/null || echo "(no .admin file yet)"'
	@echo ""
	@echo "--- Attempting setup (works only if no admin exists) ---"
	@curl -s -X POST http://localhost:8080/auth/setup \
		-H "Content-Type: application/json" \
		-d '{"username":"$(USER)","password":"$(PASS)"}' | python3 -m json.tool || true
	@echo ""
	@echo "--- If setup was blocked (admin already exists), wipe .admin and retry ---"
	@echo "    Run: make wipe-admin && make reset-admin USER=$(USER) PASS=$(PASS)"

wipe-admin: ## DELETE the .admin file (all admin accounts lost — use only to recover access)
	@echo "WARNING: this removes all admin accounts from /mnt/shared/.admin"
	@read -p "Continue? [y/N] " ans && [ "$$ans" = "y" ]
	docker compose exec backend sh -c 'rm -f /mnt/shared/.admin && echo "Wiped."'

show-admins: ## Print admin usernames currently in .admin (passwords are bcrypt-hashed)
	@docker compose exec backend sh -c 'cat /mnt/shared/.admin 2>/dev/null || echo "(no .admin file — run: make reset-admin USER=admin PASS=yourpassword)"' \
		| python3 -c "import sys,json; [print('  •', a['username'], '  created:', a.get('created_at','?')) for a in json.load(sys.stdin)]" 2>/dev/null \
		|| echo "(could not parse .admin file)"

fix-perms: ## Re-own existing share dirs to smbshare:smbshare 2770 (one-time remediation)
	docker cp samba/fix-share-perms.sh samba-server:/tmp/fix-share-perms.sh
	docker exec samba-server bash /tmp/fix-share-perms.sh

setup-env: ## Copy .env.example to .env (if .env doesn't exist)
	@test -f .env && echo ".env already exists" || (cp .env.example .env && echo "Created .env — fill in SECRET_KEY and LDAP_BIND_PW")

data-dir: ## Create required data directory
	mkdir -p data/shared
	@echo "data/shared created"

clean: ## Remove stopped containers and dangling images
	docker compose down
	docker image prune -f

clean-all: ## Remove containers, images, AND volumes (destructive — loses share data)
	@echo "WARNING: this will delete all Samba data volumes"
	@read -p "Continue? [y/N] " ans && [ "$$ans" = "y" ]
	docker compose down -v
	docker image prune -f

lint-go: ## Run golangci-lint on backend
	cd backend && golangci-lint run

deploy: ## Deploy production stack
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

rollback: ## Restore previous image after failed deploy
	docker tag smb-webapp:previous smb-webapp:latest
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# ── Help ──────────────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*##"}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
