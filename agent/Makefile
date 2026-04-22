.PHONY: db.create db.delete db.migration.create db.migrate lint lint.fix format format.check typecheck

CONTAINER_ENGINE ?= docker
COMPOSE = $(CONTAINER_ENGINE) compose -f ../docker-compose.dev.yml
DB_NAME ?= agents_dev
DB_PASSWORD ?= the-cake-is-a-lie

# Keep in sync with ../Makefile. Docker BuildKit requires tagged cache_from
# images; Podman's --cache-from rejects tags.
ifeq ($(CONTAINER_ENGINE),podman)
APP_DEV_BASE_CACHE_IMAGE ?= ghcr.io/superplanehq/superplane-dev-base
AGENT_DEV_BASE_CACHE_IMAGE ?= ghcr.io/superplanehq/superplane-dev-base
else
APP_DEV_BASE_CACHE_IMAGE ?= ghcr.io/superplanehq/superplane-dev-base:app-latest
AGENT_DEV_BASE_CACHE_IMAGE ?= ghcr.io/superplanehq/superplane-dev-base:agent-latest
endif
export APP_DEV_BASE_CACHE_IMAGE AGENT_DEV_BASE_CACHE_IMAGE

lint:
	$(COMPOSE) run --rm agent uv run --group dev ruff check

lint.fix:
	$(COMPOSE) run --rm agent uv run --group dev ruff check --fix

format:
	$(COMPOSE) run --rm agent uv run --group dev ruff format

format.check:
	$(COMPOSE) run --rm agent uv run --group dev ruff format --check

typecheck:
	$(COMPOSE) run --rm agent bash -c "uv sync --group dev && .venv/bin/python -m mypy"

db.create:
	-$(COMPOSE) run --rm -e PGPASSWORD=$(DB_PASSWORD) app createdb -h db -p 5432 -U postgres $(DB_NAME)

db.delete:
	$(COMPOSE) run --rm --user $$(id -u):$$(id -g) --rm -e PGPASSWORD=$(DB_PASSWORD) app dropdb -h db -p 5432 -U postgres $(DB_NAME)

db.migration.create:
	$(COMPOSE) run --rm app mkdir -p /app/agent/db/migrations
	$(COMPOSE) run --rm app migrate create -ext sql -dir /app/agent/db/migrations $(NAME)
	ls -lah db/migrations/*$(NAME)*

db.migrate:
	rm -f db/structure.sql
	$(COMPOSE) run --rm --user $$(id -u):$$(id -g) app migrate -source file://agent/db/migrations -database postgres://postgres:$(DB_PASSWORD)@db:5432/$(DB_NAME)?sslmode=disable up
	$(COMPOSE) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --schema-only --no-privileges --restrict-key abcdef123 --no-owner -h db -p 5432 -U postgres -d $(DB_NAME)" > db/structure.sql
	$(COMPOSE) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --data-only --restrict-key abcdef123 --table schema_migrations -h db -p 5432 -U postgres -d $(DB_NAME)" >> db/structure.sql
