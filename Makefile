.PHONY: lint test test.coverage test.coverage.autoparallel test.license.check check.generated.artifacts dev.up dev.setup dev.setup.app dev.setup.go dev.clean.go.cache dev.server dev.server.fg dev.runners doctor-local format.runner profile.cpu profile.heap profile.goroutines check.grpc.actions.status simulate.usage simulate-usage db.reset.billing.trial db.reset.after.onboarding db.snapshot db.restore ensure.bun check.test.ui check.test.ui.shard

MAKE=make
MAKEFLAGS+=--no-print-directory

# Auto-source local overrides from .env so host-side targets (e.g. profiling)
# see the same values docker compose interpolates from it. Command-line
# overrides still take precedence, and a missing .env file is ignored.
-include .env
export

DB_NAME=superplane
DB_PASSWORD=the-cake-is-a-lie
BASE_URL?=https://app.superplane.com

# Quiet BuildKit and Compose progress in CI. Use DEBUG=1 for full logs.
COMPOSE_PROGRESS := auto
COMPOSE_UP_EXTRA :=
ifeq ($(DEBUG),1)
export BUILDKIT_PROGRESS := plain
COMPOSE_PROGRESS := plain
else
export BUILDKIT_PROGRESS := quiet
ifneq ($(strip $(CI)),)
COMPOSE_PROGRESS := quiet
COMPOSE_UP_EXTRA := --quiet-build
endif
endif

PKG_TEST_PACKAGES := ./pkg/...
E2E_TEST_PACKAGES := ./test/e2e/...

# On CI, overlay docker-compose.ci.yml so the Go module and build caches live in
# host directories that the CI cache can restore and store between jobs.
# Runner services live in docker-compose.dev.yml under the local-runner profile.
# CI does not enable that profile, so it does not build the worker image.
COMPOSE_FILES := -f docker-compose.dev.yml
GO_CACHE_DIRS :=
N ?= 10
TASK_BROKER_HOST_PORT ?= 8091
ifneq ($(strip $(CI)),)
COMPOSE_FILES += -f docker-compose.ci.yml
GO_CACHE_DIRS := tmp/go tmp/go-build
endif

COMPOSE=docker compose $(COMPOSE_FILES)
COMPOSE_RUNNER=$(COMPOSE) --profile local-runner
GENERATED_ARTIFACT_PATHS := pkg/protos pkg/openapi_client web_src/src/api-client api/swagger/superplane.swagger.json
OPENAPI_GENERATOR_IMAGE := openapitools/openapi-generator-cli:v7.13.0

# Tests must not inherit compose local-broker defaults. Empty values
# override docker-compose.dev.yml for this command only.
TEST_TASK_BROKER_ENV := -e TASK_BROKER_BASE_URL= -e TASK_BROKER_AUTH_TOKEN= -e TASK_BROKER_FLEET_ID= -e TASK_BROKER_PUBLIC_URL=

#
# Long sausage command to run tests with gotestsum
#
# - starts a docker container for unit tests
# - mounts tmp/screenshots
# - exports junit report
# - sets parallelism to 1
#
GOTESTSUM=$(COMPOSE) run --rm -e DB_NAME=superplane_test $(TEST_TASK_BROKER_ENV) -v $(PWD)/tmp/screenshots:/app/test/screenshots app gotestsum --format short --junitfile junit-report.xml

#
# Targets for test environment
#

lint:
	$(COMPOSE) exec app revive -formatter friendly -config lint.toml -exclude ./tmp/... -exclude ./runner/... ./...

tidy:
	$(COMPOSE) exec app go mod tidy

test.e2e:
	$(COMPOSE) exec -e DB_NAME=superplane_test $(TEST_TASK_BROKER_ENV) app gotestsum --format short --junitfile junit-report.xml --rerun-fails=3 --rerun-fails-max-failures=1 --packages="$(E2E_TEST_PACKAGES)" -- -p 1 -timeout 30m

test.e2e.autoparallel:
	$(COMPOSE) exec -e DB_NAME=superplane_test $(TEST_TASK_BROKER_ENV) -e SHARD_INDEX -e SHARD_COUNT app bash -lc "cd /app && bash scripts/test_e2e_autoparallel.sh"

test.e2e.single:
	bash ./scripts/vscode_run_tests.sh line $(FILE) $(LINE)

test:
	$(GOTESTSUM) --packages="$(PKG_TEST_PACKAGES)" -- -p 1

test.coverage:
	$(GOTESTSUM) --packages="$(PKG_TEST_PACKAGES)" -- -p 1 -coverprofile=coverage-go.out -covermode=atomic
	$(COMPOSE) run --rm app go tool cover -func=coverage-go.out | grep '^total:'

test.coverage.check:
	$(MAKE) test.coverage
	$(MAKE) check.coverage.go

test.coverage.autoparallel:
	$(COMPOSE) run --rm -e DB_NAME=superplane_test $(TEST_TASK_BROKER_ENV) -e SHARD_INDEX -e SHARD_COUNT -v $(PWD)/tmp/screenshots:/app/test/screenshots app bash -lc "cd /app && bash scripts/test_unit_autoparallel.sh"

test.coverage.baseline.update:
	$(MAKE) test.coverage
	$(MAKE) check.coverage.go.baseline.update

test.license.check:
	bash ./scripts/license-check.sh

test.watch:
	$(GOTESTSUM) --packages="$(PKG_TEST_PACKAGES)" --watch -- -p 1

test.shell:
	$(COMPOSE) run --rm -e DB_NAME=superplane_test $(TEST_TASK_BROKER_ENV) -v $(PWD)/tmp/screenshots:/app/test/screenshots app /bin/bash

#
# Code formatting
#

format.go:
	$(COMPOSE) exec app bash -c "find . -name '*.go' -not -path './tmp/*' -not -path './runner/*' -print0 | xargs -0 gofmt -s -w"

check.format.go:
	$(COMPOSE) exec app bash -c "find . -name '*.go' -not -path './tmp/*' -not -path './runner/*' -print0 | xargs -0 gofmt -s -l | tee /dev/stderr | if read; then exit 1; else exit 0; fi"

format.runner:
	$(COMPOSE) exec app bash -c "gofmt -s -w runner/fleet-manager runner/runner runner/shared runner/task-broker runner/test"

format.js:
	cd web_src && npm run format

format.js.check:
	cd web_src && npm run format:check

terraform.format:
	$(MAKE) -C release/terraform format

dev.test.is.running:
	@test -n "$$($(COMPOSE) ps --status running -q app 2>/dev/null)" || { echo "Run \`make dev.up\` first (app container is not running)." >&2; exit 1; }

dev.up:
	@mkdir -p tmp/screenshots $(GO_CACHE_DIRS)
	@echo "Starting development containers..."
	$(COMPOSE) --progress $(COMPOSE_PROGRESS) up -d --wait --build --pull always --quiet-pull $(COMPOSE_UP_EXTRA)
ifeq ($(strip $(CI)),)
	$(COMPOSE_RUNNER) --progress $(COMPOSE_PROGRESS) build task-broker runner
	@echo "Runner images built."
endif
	@echo "Development containers are ready."

dev.setup:
	@$(MAKE) dev.test.is.running
	$(MAKE) dev.setup.npm
	$(MAKE) pb.gen
	$(MAKE) dev.setup.go
	$(MAKE) db.create DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_dev
	$(MAKE) db.create DB_NAME=superplane_test
	$(MAKE) db.migrate DB_NAME=superplane_test
ifeq ($(strip $(CI)),)
	$(MAKE) db.create DB_NAME=broker
	$(COMPOSE_RUNNER) --progress $(COMPOSE_PROGRESS) up -d --wait --build task-broker
	$(COMPOSE_RUNNER) run --rm -T --no-deps task-broker-init
	@echo "Task broker ready at http://127.0.0.1:$(TASK_BROKER_HOST_PORT)"
endif

dev.setup.npm:
	@$(COMPOSE) exec app bash -lc "cd /app/web_src && npm install --no-audit --no-fund --loglevel error"

dev.setup.go:
	@$(COMPOSE) exec app bash /app/scripts/go-mod-download
	@$(COMPOSE) exec app go build cmd/server/main.go

dev.clean.go.cache:
	@$(MAKE) dev.test.is.running
	$(COMPOSE) exec app go clean -modcache -cache

dev.setup.no.cache:
	rm -rf tmp
	$(COMPOSE) down -v --remove-orphans
	$(COMPOSE) build --no-cache
	$(MAKE) dev.up
	$(MAKE) dev.setup

dev.server:
	@test -n "$$($(COMPOSE) ps --status running -q app 2>/dev/null)" || { echo "Run \`make dev.up\` first (app container is not running)." >&2; exit 1; }
ifeq ($(strip $(CI)),)
	$(COMPOSE_RUNNER) up -d --no-build --scale runner=$(N) runner
endif
	$(COMPOSE) exec -d app bash /app/docker-entrypoint.dev.sh
	@bash ./scripts/wait-for-app
ifeq ($(strip $(CI)),)
	@echo "Task broker: http://127.0.0.1:$(TASK_BROKER_HOST_PORT)"
	@echo "Runner workers: $(N)"
endif

dev.server.fg:
	@test -n "$$($(COMPOSE) ps --status running -q app 2>/dev/null)" || { echo "Run \`make dev.up\` first (app container is not running)." >&2; exit 1; }
ifeq ($(strip $(CI)),)
	$(COMPOSE_RUNNER) up -d --no-build --scale runner=$(N) runner
endif
	$(COMPOSE) exec app bash /app/docker-entrypoint.dev.sh

# Postgres, the `broker` database, task-broker, fleet registration, and
# runner workers. Does not start the app or RabbitMQ.
# Scale workers with N (default 10): N=1 make dev.runners
dev.runners:
	@echo "Starting Postgres for the task broker..."
	$(COMPOSE) --progress $(COMPOSE_PROGRESS) up -d --wait db
	@$(COMPOSE) exec -T db psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname = 'broker'" | grep -q 1 \
	  || $(COMPOSE) exec -T db psql -U postgres -c "CREATE DATABASE broker"
	@echo "Starting task broker and $(N) runner workers..."
	$(COMPOSE_RUNNER) --progress $(COMPOSE_PROGRESS) up -d --wait --build --scale runner=$(N) task-broker runner
	@echo "Task broker: http://127.0.0.1:$(TASK_BROKER_HOST_PORT)"
	@echo "Runner workers: $(N)"

dev.start.ephemeral:
	bash ./scripts/ephemeral/start-caddy.sh $(BASE_URL)
	bash ./scripts/ephemeral/setup-env.sh $(BASE_URL)
	$(MAKE) dev.up
	$(MAKE) dev.server

dev.logs:
	$(COMPOSE) logs -f

dev.logs.app:
	$(COMPOSE) logs -f app

dev.logs.otel:
	$(COMPOSE) logs -f otel

dev.down:
	$(COMPOSE_RUNNER) down --remove-orphans

doctor-local:
	$(COMPOSE_RUNNER) exec runner sh -c '\
	  missing=0; \
	  for cmd in claude codex opencode playwright node git gh jq python3 bash; do \
	    if ! command -v "$$cmd" >/dev/null 2>&1; then \
	      echo "$$cmd missing" >&2; \
	      missing=1; \
	      continue; \
	    fi; \
	    echo "$$cmd=$$(command -v "$$cmd")"; \
	  done; \
	  echo "claude=$$(claude --version 2>/dev/null | head -n1)"; \
	  echo "codex=$$(codex --version 2>/dev/null | head -n1)"; \
	  echo "opencode=$$(opencode --version 2>/dev/null | head -n1)"; \
	  echo "playwright=$$(playwright --version 2>/dev/null | head -n1)"; \
	  playwright cli --help >/dev/null 2>&1 || missing=1; \
	  echo "node=$$(node --version 2>/dev/null)"; \
	  echo "gh=$$(gh --version 2>/dev/null | head -n1)"; \
	  shot=$$(mktemp /tmp/playwright-doctor.XXXXXX.png); \
	  playwright screenshot about:blank "$$shot" >/dev/null 2>&1 || missing=1; \
	  test -s "$$shot" || missing=1; \
	  rm -f "$$shot"; \
	  test "$$missing" -eq 0'

dev.console:
	$(COMPOSE) run --rm app /bin/bash

dev.db:
	$(COMPOSE) exec -it -e PGPASSWORD=$(DB_PASSWORD) app psql -h db -p 5432 -U postgres -d superplane_dev

dev.db.console:
	$(MAKE) db.console DB_NAME=superplane_dev

dev.pr.clean.checkout:
	bash ./scripts/clean-pr-checkout $(PR)

check.example.payloads:
	$(COMPOSE) run --rm app bash -c "go run scripts/check_example_payloads.go"

check.proto.field.numbers:
	bash ./scripts/check_proto_field_numbers.sh

check.configuration.fields:
	$(COMPOSE) run --rm app bash -c "go run scripts/check_configuration_fields.go"

check.configuration.fields.baseline.update:
	$(COMPOSE) run --rm app bash -c "go run scripts/check_configuration_fields.go --update-baseline"

check.models.tx.debt:
	@$(COMPOSE) exec app go run ./scripts/check_models_tx_debt.go

check.models.tx.debt.baseline.update:
	@$(COMPOSE) exec app go run ./scripts/check_models_tx_debt.go --update-baseline

check.grpc.actions.status:
	bash ./scripts/verify_grpc_actions_status.sh

check.db.structure:
	bash ./scripts/verify_db_structure_clean.sh

check.db.migrations:
	bash ./scripts/verify_no_future_migrations.sh
	bash ./scripts/verify_branch_migrations_are_latest.sh

check.build.ui:
	$(COMPOSE) exec app bash -c "cd web_src && npm run build"

check.build.storybook:
	$(COMPOSE) exec app bash -c "cd web_src && npm run build-storybook"

ensure.bun:
	$(COMPOSE) exec app bash -lc 'command -v bun >/dev/null || bash /app/scripts/docker/install-bun.sh'

check.test.ui: ensure.bun
	$(COMPOSE) exec -e FILES="$(FILES)" app bash -lc "bash /app/scripts/test_ui_autoparallel.sh"

check.test.ui.shard: ensure.bun
	$(COMPOSE) exec -e SHARD_INDEX="$(SHARD_INDEX)" -e SHARD_COUNT="$(SHARD_COUNT)" app bash -lc "bash /app/scripts/test_ui_autoparallel.sh"

check.format.js:
	$(COMPOSE) exec app bash -c "cd web_src && npm run format:check"

terraform.check:
	$(MAKE) -C release/terraform check

check.tool.configs:
	bash ./scripts/check_tool_config_guard.sh

check.fast.security:
	bash ./scripts/check_fast_security.sh

check.npm.audit.critical:
	@echo "==> npm audit (critical, production, lockfile only)"
	$(COMPOSE) exec app bash -c "cd web_src && npm audit --omit=dev --audit-level=critical --package-lock-only"
	@echo "==> npm audit: PASS"

check.lint.ui:
	$(COMPOSE) exec app bash -c "cd web_src && npm run lint:budget"

check.lint.ui.knip:
	$(COMPOSE) exec app bash -c "cd web_src && npm ci && npm run lint:knip"

check.lint.ui.baseline.update:
	$(COMPOSE) exec app bash -c "cd web_src && npm run lint:baseline:update"

check.build.app:
	$(COMPOSE) exec app go build cmd/server/main.go

check.generated.artifacts:
	@tracked="$$(git ls-files -- $(GENERATED_ARTIFACT_PATHS))"; \
	if [ -n "$$tracked" ]; then \
		echo "Tracked generated artifacts found:"; \
		echo "$$tracked"; \
		exit 1; \
	fi

check.coverage.go:
	go run ./scripts/check_go_coverage_budget.go --profile coverage-go.out

check.coverage.go.baseline.update:
	go run ./scripts/check_go_coverage_budget.go --profile coverage-go.out --update-baseline

#
# Performance profiling against the running dev server (PPROF_ENABLED=yes).
# See docs/contributing/profiling.md
#
profile.cpu:
	$(COMPOSE) exec app go tool pprof -top "http://localhost:$${PPROF_PORT:-6060}/debug/pprof/profile?seconds=$${SECONDS:-30}"

profile.heap:
	$(COMPOSE) exec app go tool pprof -top "http://localhost:$${PPROF_PORT:-6060}/debug/pprof/heap"

profile.goroutines:
	$(COMPOSE) exec app curl -s "http://localhost:$${PPROF_PORT:-6060}/debug/pprof/goroutine?debug=2"


storybook:
	$(COMPOSE) exec app /bin/bash -c "cd web_src && npm install && npm run storybook"

ui.setup:
	npm install

ui.start:
	npm run storybook

#
# Database target helpers (require a running app container: `make dev.up`)
#

db.create:
	@$(COMPOSE) exec app ./scripts/db_create.sh $(DB_NAME)

db.migration.create:
	$(COMPOSE) exec app mkdir -p db/migrations
	$(COMPOSE) exec app migrate create -ext sql -dir db/migrations $(NAME)
	ls -lah db/migrations/*$(NAME)*

db.data_migration.create:
	$(COMPOSE) exec app mkdir -p db/data_migrations
	$(COMPOSE) exec app migrate create -ext sql -dir db/data_migrations $(NAME)
	ls -lah db/data_migrations/*$(NAME)*

db.migrate:
	@$(COMPOSE) exec app ./scripts/db_migrate.sh $(DB_NAME)

db.migrate.all:
	$(MAKE) db.migrate DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_test

# Local only. Writes this worktree's superplane_dev to
# .local/superplane_dev.dump so a later restore can skip owner setup
# and GitHub connection. Postgres in this stack is db:5432.
# The dump is gitignored.
db.snapshot:
	@$(COMPOSE) exec app ./scripts/db_snapshot.sh superplane_dev

# Local only. Replaces this worktree's superplane_dev from
# .local/superplane_dev.dump, then applies pending migrations. Use this
# to create a new local environment without owner setup or GitHub
# connection. It does not touch superplane_test. Restart make
# dev.server after restore if it is already running.
db.restore:
	@$(COMPOSE) exec app ./scripts/db_restore.sh superplane_dev

# Local only. Puts every org on a 14-day trial, clears Polar ids, and
# deletes usage ledger rows in superplane_dev. Cancel the Polar sandbox
# subscription separately.
db.reset.billing.trial:
	@$(COMPOSE) exec app ./scripts/db_reset_billing_trial.sh superplane_dev

# Local only. Keeps org, workspace, GitHub, apps, lines, and intakes in
# superplane_dev. Wipes tasks, runs, usage, subscription, and credits, then
# reseeds intake from GitHub. Cancel the Polar sandbox subscription
# separately. Run `make dev.server` so seeded intake events become tasks.
db.reset.after.onboarding:
	@$(COMPOSE) exec app ./scripts/db_reset_after_onboarding.sh superplane_dev

# Local only. Inserts fake hosted model + runner VM usage into superplane_dev.
# MONEY is the total dollar amount. It is spread across TASKS (default 20)
# over DAYS (default 30). Requires an organization and a factory.
# Example: make simulate.usage MONEY=20 TASKS=30 DAYS=30
# Optional: ORGANIZATION_ID=<uuid> FACTORY_ID=<uuid>
simulate.usage simulate-usage:
	@$(COMPOSE) exec \
		-e MONEY="$(MONEY)" \
		-e TASKS="$(TASKS)" \
		-e DAYS="$(DAYS)" \
		-e ORGANIZATION_ID="$(ORGANIZATION_ID)" \
		-e FACTORY_ID="$(FACTORY_ID)" \
		app ./scripts/db_simulate_usage.sh superplane_dev

db.console:
	$(COMPOSE) exec -it --user $$(id -u):$$(id -g) -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME)

db.delete:
	$(COMPOSE) exec --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app dropdb -h db -p 5432 -U postgres $(DB_NAME)

db.recreate.all.dangerous:
	$(MAKE) dev.down
	$(COMPOSE) up -d --wait
	-$(MAKE) db.delete DB_NAME=superplane_dev
	-$(MAKE) db.delete DB_NAME=superplane_test
	$(MAKE) db.create DB_NAME=superplane_dev
	$(MAKE) db.create DB_NAME=superplane_test
	$(MAKE) db.migrate DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_test

#
# Protobuf / OpenAPI codegen runs in the running `app` container (`make dev.up` first).
#

gen:
	$(MAKE) pb.gen
	$(MAKE) format.go
	$(MAKE) format.js
	$(MAKE) gen.components.docs

gen.components.docs:
	rm -rf docs/components
	go run scripts/generate_components_docs.go

check.components.docs:
	rm -rf docs/components
	$(COMPOSE) run --rm app bash -c "go run scripts/generate_components_docs.go"
	git diff --exit-code docs/components

MODULES := authorization,organizations,integrations,factories,secrets,users,groups,roles,me,configuration,components,actions,triggers,widgets,canvases,api_keys,agents,usage,runners,files
REST_API_MODULES := authorization,organizations,integrations,factories,secrets,users,groups,roles,me,configuration,actions,triggers,widgets,canvases,api_keys,agents,files

pb.gen: dev.test.is.running
	$(MAKE) pb.gen.models
	$(MAKE) pb.gen.gateway
	@$(COMPOSE) exec --user $(shell id -u):$(shell id -g) app bash -lc "find pkg/protos -name '*.go' -print0 | xargs -0 gofmt -s -w"
	$(MAKE) openapi.spec.gen
	$(MAKE) openapi.client.gen
	$(MAKE) openapi.web.client.gen

pb.gen.models:
	@$(COMPOSE) exec app /app/scripts/protoc.sh $(MODULES)

pb.gen.gateway:
	@$(COMPOSE) exec app /app/scripts/protoc_gateway.sh $(REST_API_MODULES)

openapi.spec.gen: dev.test.is.running
	@$(COMPOSE) exec app /app/scripts/protoc_openapi_spec.sh $(REST_API_MODULES)

openapi.client.gen: dev.test.is.running
	@rm -rf pkg/openapi_client
	@./scripts/docker-pull-retry $(OPENAPI_GENERATOR_IMAGE)
	@log=$$(mktemp); trap 'rm -f "$$log"' EXIT; \
	if ! docker run --rm --user $(shell id -u):$(shell id -g) \
		-v ${PWD}:/local $(OPENAPI_GENERATOR_IMAGE) generate \
		-i /local/api/swagger/superplane.swagger.json \
		-g go \
		-o /local/pkg/openapi_client \
		--additional-properties=packageName=openapi_client,enumClassPrefix=true,isGoSubmodule=true,withGoMod=false \
		>"$$log" 2>&1; then cat "$$log"; exit 1; fi
	@rm -rf pkg/openapi_client/test
	@rm -rf pkg/openapi_client/docs
	@rm -rf pkg/openapi_client/api
	@rm -rf pkg/openapi_client/.travis.yml
	@rm -rf pkg/openapi_client/README.md
	@rm -rf pkg/openapi_client/git_push.sh
	@$(COMPOSE) exec --user $(shell id -u):$(shell id -g) app bash -lc "find pkg/openapi_client -name '*.go' -print0 | xargs -0 gofmt -s -w"

openapi.web.client.gen: dev.test.is.running
	@rm -rf web_src/src/api-client
	@$(COMPOSE) exec --user $(shell id -u):$(shell id -g) app bash -lc "export HOME=/tmp && export NPM_CONFIG_CACHE=/tmp/.npm && cd web_src && npm -s run generate:api && npx prettier --log-level silent --write 'src/api-client/**/*.{ts,tsx}'"

#
# Image and CLI build
#

CLI_VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")

cli.build:
	$(MAKE) pb.gen
	$(COMPOSE) exec -e GOOS=$(OS) -e GOARCH=$(ARCH) app bash -c 'go build -ldflags "-X github.com/superplanehq/superplane/pkg/cli.Version=$(CLI_VERSION)" -o build/cli cmd/cli/main.go'

cli.build.m1:
	$(MAKE) cli.build OS=darwin ARCH=arm64

IMAGE?=superplane
IMAGE_TAG?=$(shell git rev-list -1 HEAD -- .)
REGISTRY_HOST?=ghcr.io/superplanehq
DEV_BASE_IMAGE?=ghcr.io/superplanehq/superplane-dev-base:app-latest
VITE_ASSET_BASE_URL?=
FRONTEND_PREBUILT?=0
# pb.gen runs in the compose app container; run `make dev.up` first.
# Pass host Go caches into the builder when CI restored tmp/go and tmp/go-build.
image.build:
	$(MAKE) pb.gen
	mkdir -p tmp/go tmp/go-build
	DOCKER_DEFAULT_PLATFORM=linux/amd64 docker build -f Dockerfile --target runner \
	  --build-context ci-go-mod=tmp/go \
	  --build-context ci-go-build=tmp/go-build \
	  --cache-from $(DEV_BASE_IMAGE) \
	  --build-arg BASE_URL=$(BASE_URL) \
	  --build-arg VITE_ASSET_BASE_URL=$(VITE_ASSET_BASE_URL) \
	  --build-arg FRONTEND_PREBUILT=$(FRONTEND_PREBUILT) \
	  --progress plain -t $(IMAGE):$(IMAGE_TAG) .

image.auth:
	@test -n "$$GHCR_PUSH_TOKEN" || (echo "GHCR_PUSH_TOKEN is empty" >&2; exit 1)
	@printf "%s" "$$GHCR_PUSH_TOKEN" | docker login ghcr.io -u superplanehq --password-stdin

image.push:
	docker tag $(IMAGE):$(IMAGE_TAG) $(REGISTRY_HOST)/$(IMAGE):$(IMAGE_TAG)
	docker push $(REGISTRY_HOST)/$(IMAGE):$(IMAGE_TAG)

#
# Tag creation
#

tag.create.patch:
	./release/create_tag.sh patch

tag.create.minor:
	./release/create_tag.sh minor
