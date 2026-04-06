.PHONY: db.create db.delete db.migration.create db.migrate lint lint.fix format format.check typecheck

COMPOSE=docker compose -f ../docker-compose.dev.yml
DB_NAME ?= agents_dev
DB_PASSWORD ?= the-cake-is-a-lie

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
	@$(COMPOSE) --progress quiet run --rm --no-deps -e PGPASSWORD=$(DB_PASSWORD) app bash scripts/docker/db-create.sh $(DB_NAME)

db.delete:
	@$(COMPOSE) --progress quiet run --rm --no-deps --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app dropdb -h db -p 5432 -U postgres $(DB_NAME)

db.migration.create:
	$(COMPOSE) run --rm app mkdir -p /app/agent/db/migrations
	$(COMPOSE) run --rm app migrate create -ext sql -dir /app/agent/db/migrations $(NAME)
	ls -lah db/migrations/*$(NAME)*

db.migrate:
	@rm -f db/structure.sql
	@$(COMPOSE) --progress quiet run --rm --no-deps --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app \
		bash scripts/docker/db-migrate.sh $(DB_NAME) agent/db/migrations
	@$(COMPOSE) --progress quiet run --rm --no-deps --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app \
		bash scripts/docker/db-dump.sh $(DB_NAME) > db/structure.sql
