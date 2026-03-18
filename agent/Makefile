.PHONY: dev.setup dev.start dev.logs dev.down dev.shell dev.console dev.console.once dev.console.reload test.setup test test.watch lint format typecheck

DOCKER_COMPOSE_OPTS=-f docker-compose.dev.yml

dev.setup:
	docker compose $(DOCKER_COMPOSE_OPTS) build
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app uv sync --frozen || docker compose $(DOCKER_COMPOSE_OPTS) run --rm app uv sync

dev.start:
	docker compose $(DOCKER_COMPOSE_OPTS) up -d --wait app

dev.logs:
	docker compose $(DOCKER_COMPOSE_OPTS) logs -f app

dev.down:
	docker compose $(DOCKER_COMPOSE_OPTS) down --remove-orphans

dev.shell:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app bash

dev.console:
	@while true; do \
		docker compose $(DOCKER_COMPOSE_OPTS) run --rm app uv run python -m ai.main --interactive --canvas-id "$(CANVAS_ID)"; \
		status=$$?; \
		if [ $$status -eq 3 ]; then \
			continue; \
		fi; \
		exit $$status; \
	done

dev.console.once:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app uv run python -m ai.main --interactive --canvas-id "$(CANVAS_ID)"

dev.console.reload:
	@$(MAKE) dev.console CANVAS_ID="$(CANVAS_ID)"

test.setup:
	$(MAKE) dev.setup

test:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e IN_DOCKER=1 app uv run pytest -q

test.watch:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e IN_DOCKER=1 app uv run ptw tests -- -q

lint:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app uv run ruff check .

format:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app uv run ruff format .

typecheck:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app uv run mypy src tests
