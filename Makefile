.PHONY: lint test test.coverage test.license.check gen.setup gen.setup.prep gen.setup.backend gen.setup.ui web_src.npm.install.compose check.generated.artifacts check.templates dev.up dev.setup dev.setup.app dev.server dev.server.fg

DB_NAME=superplane
DB_PASSWORD=the-cake-is-a-lie
# Databases to create and migrate in `make dev.setup` (space-separated). CI sets superplane_test alongside superplane_dev.
DEV_SETUP_DBS ?= superplane_dev
BASE_URL?=https://app.superplane.com

export BUILDKIT_PROGRESS ?= plain

PKG_TEST_PACKAGES := ./pkg/...
E2E_TEST_PACKAGES := ./test/e2e/...

COMPOSE=docker compose -f docker-compose.dev.yml
DOCKER_RUN_AS_CURRENT_USER=docker run --rm --user $(shell id -u):$(shell id -g)
GENERATED_ARTIFACT_PATHS := pkg/protos pkg/openapi_client web_src/src/api-client api/swagger/superplane.swagger.json

#
# Long sausage command to run tests with gotestsum
#
# - starts a docker container for unit tests
# - mounts tmp/screenshots
# - exports junit report
# - sets parallelism to 1
#
GOTESTSUM=$(COMPOSE) run --rm -e DB_NAME=superplane_test -v $(PWD)/tmp/screenshots:/app/test/screenshots app gotestsum --format short --junitfile junit-report.xml 

#
# Targets for test environment
#

lint:
	$(COMPOSE) exec app revive -formatter friendly -config lint.toml -exclude ./tmp/... ./...

tidy:
	$(COMPOSE) exec app go mod tidy

test.e2e:
	$(COMPOSE) exec app gotestsum --format short --junitfile junit-report.xml --rerun-fails=3 --rerun-fails-max-failures=1 --packages="$(E2E_TEST_PACKAGES)" -- -p 1

test.e2e.autoparallel:
	$(COMPOSE) exec -e INDEX -e TOTAL app bash -lc "cd /app && bash scripts/test_e2e_autoparallel.sh"

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

test.coverage.baseline.update:
	$(MAKE) test.coverage
	$(MAKE) check.coverage.go.baseline.update

test.license.check:
	$(MAKE) gen.setup.backend
	bash ./scripts/license-check.sh

test.watch:
	$(GOTESTSUM) --packages="$(PKG_TEST_PACKAGES)" --watch -- -p 1

test.shell:
	$(COMPOSE) run --rm -e DB_NAME=superplane_test -v $(PWD)/tmp/screenshots:/app/test/screenshots app /bin/bash	

#
# Code formatting
#

format.go:
	$(COMPOSE) exec app bash -c "find . -name '*.go' -not -path './tmp/*' -print0 | xargs -0 gofmt -s -w"

check.format.go:
	$(COMPOSE) exec app bash -c "find . -name '*.go' -not -path './tmp/*' -print0 | xargs -0 gofmt -s -l | tee /dev/stderr | if read; then exit 1; else exit 0; fi"

format.js:
	cd web_src && npm run format

format.js.check:
	cd web_src && npm run format:check

#
# Targets for dev environment
#
# Typical flow: `make dev.up` then `make dev.setup` (first time / after proto or dependency changes),
# then `make dev.server`. Day-to-day: `make dev.up` then `make dev.server` (or `make dev.server.fg` for attached logs).
# For E2E locally, migrate the test DB too: `DEV_SETUP_DBS="superplane_dev superplane_test" make dev.setup` (after `make dev.up`).

dev.up:
	$(COMPOSE) up -d --wait --build --pull always --quiet-pull

dev.setup:
	@mkdir -p tmp/screenshots
	@test -n "$$($(COMPOSE) ps --status running -q app 2>/dev/null)" || { echo "Run \`make dev.up\` first (app container is not running)." >&2; exit 1; }
	$(COMPOSE) exec -T app bash -lc "cd /app/web_src && npm install"
	$(MAKE) gen.setup SKIP_WEB_SRC_NPM_INSTALL=1
	$(MAKE) dev.setup.app
	@set -euo pipefail; for db in $(DEV_SETUP_DBS); do \
		$(MAKE) db.create DB_NAME=$$db; \
		$(MAKE) db.migrate DB_NAME=$$db; \
	done

dev.setup.app:
	@test -n "$$($(COMPOSE) ps --status running -q app 2>/dev/null)" || { echo "Run \`make dev.up\` first (app container is not running)." >&2; exit 1; }
	$(COMPOSE) exec -T app go mod download
	$(COMPOSE) exec -T app go build cmd/server/main.go

dev.setup.no.cache:
	rm -rf tmp
	$(COMPOSE) down -v --remove-orphans
	$(COMPOSE) build --no-cache
	$(MAKE) dev.up
	$(MAKE) dev.setup

dev.server:
	@test -n "$$($(COMPOSE) ps --status running -q app 2>/dev/null)" || { echo "Run \`make dev.up\` first (app container is not running)." >&2; exit 1; }
	$(COMPOSE) exec -d app bash /app/docker-entrypoint.dev.sh
	@bash ./scripts/wait-for-app

dev.server.fg:
	@test -n "$$($(COMPOSE) ps --status running -q app 2>/dev/null)" || { echo "Run \`make dev.up\` first (app container is not running)." >&2; exit 1; }
	$(COMPOSE) exec app bash /app/docker-entrypoint.dev.sh

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
	$(COMPOSE) down --remove-orphans

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

check.db.structure:
	bash ./scripts/verify_db_structure_clean.sh

check.db.migrations:
	bash ./scripts/verify_no_future_migrations.sh

check.build.ui:
	$(COMPOSE) exec app bash -c "cd web_src && npm run build"

check.lint.ui:
	$(COMPOSE) exec app bash -c "cd web_src && npm run lint:budget"

check.lint.ui.knip:
	$(COMPOSE) exec app bash -c "cd web_src && npm ci && npm run lint:knip"

check.lint.ui.baseline.update:
	$(COMPOSE) exec app bash -c "cd web_src && npm run lint:baseline:update"

check.templates:
	$(COMPOSE) exec app go run ./scripts/check_canvases_templates/main.go

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
	-$(COMPOSE) exec -T -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres -c 'ALTER DATABASE template1 REFRESH COLLATION VERSION';
	-$(COMPOSE) exec -T -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres -c 'ALTER DATABASE postgres REFRESH COLLATION VERSION';
	-$(COMPOSE) exec -T -e PGPASSWORD=the-cake-is-a-lie app createdb -h db -p 5432 -U postgres $(DB_NAME)
	$(COMPOSE) exec -T -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME) -c 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp";'

db.migration.create:
	$(COMPOSE) exec -T app mkdir -p db/migrations
	$(COMPOSE) exec -T app migrate create -ext sql -dir db/migrations $(NAME)
	ls -lah db/migrations/*$(NAME)*

db.data_migration.create:
	$(COMPOSE) exec -T app mkdir -p db/data_migrations
	$(COMPOSE) exec -T app migrate create -ext sql -dir db/data_migrations $(NAME)
	ls -lah db/data_migrations/*$(NAME)*

db.migrate:
	rm -f db/structure.sql
	$(COMPOSE) exec -T --user $$(id -u):$$(id -g) app migrate -source file://db/migrations -database postgres://postgres:$(DB_PASSWORD)@db:5432/$(DB_NAME)?sslmode=disable up
	$(COMPOSE) exec -T --user $$(id -u):$$(id -g) app migrate -source file://db/data_migrations -database postgres://postgres:$(DB_PASSWORD)@db:5432/$(DB_NAME)?sslmode=disable\&x-migrations-table=data_migrations up
	# echo dump schema to db/structure.sql
	$(COMPOSE) exec -T --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --schema-only --no-privileges --restrict-key abcdef123 --no-owner -h db -p 5432 -U postgres -d $(DB_NAME)" > db/structure.sql
	$(COMPOSE) exec -T --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --data-only --restrict-key abcdef123 --table schema_migrations -h db -p 5432 -U postgres -d $(DB_NAME)" >> db/structure.sql
	$(COMPOSE) exec -T --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --data-only --restrict-key abcdef123 --table data_migrations -h db -p 5432 -U postgres -d $(DB_NAME)" >> db/structure.sql

db.migrate.all:
	$(MAKE) db.migrate DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_test

db.console:
	$(COMPOSE) exec -it --user $$(id -u):$$(id -g) -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME)

db.delete:
	$(COMPOSE) exec -T --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app dropdb -h db -p 5432 -U postgres $(DB_NAME)

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
# Protobuf compilation
#

# Install web_src deps via a one-off compose run (CI, `make gen`, release).
# `make dev.setup` installs into the running app container first, then calls
# `gen.setup` with SKIP_WEB_SRC_NPM_INSTALL=1 so this step is skipped.
web_src.npm.install.compose:
	$(COMPOSE) run --rm --no-deps --user $(shell id -u):$(shell id -g) app bash -lc "export HOME=/tmp && export NPM_CONFIG_CACHE=/tmp/.npm && cd web_src && npm install"

gen.setup.prep:
ifneq ($(SKIP_WEB_SRC_NPM_INSTALL),1)
	$(MAKE) web_src.npm.install.compose
endif

gen.setup: gen.setup.prep
	$(MAKE) pb.gen
	$(MAKE) openapi.spec.gen
	$(MAKE) openapi.client.gen
	$(MAKE) openapi.web.client.gen

gen.setup.backend:
	$(MAKE) pb.gen
	$(MAKE) openapi.spec.gen
	$(MAKE) openapi.client.gen

gen.setup.ui:
	$(MAKE) openapi.spec.gen
	$(MAKE) web_src.npm.install.compose
	$(MAKE) openapi.web.client.gen

gen:
	$(MAKE) gen.setup
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

MODULES := authorization,organizations,integrations,secrets,users,groups,roles,me,configuration,components,actions,triggers,widgets,blueprints,canvases,canvas_folders,service_accounts,agents,usage,private/agents
REST_API_MODULES := authorization,organizations,integrations,secrets,users,groups,roles,me,configuration,actions,triggers,widgets,blueprints,canvases,canvas_folders,service_accounts,agents

pb.gen:
	$(COMPOSE) run --rm --no-deps app /app/scripts/protoc.sh $(MODULES)
	$(COMPOSE) run --rm --no-deps app /app/scripts/protoc_gateway.sh $(REST_API_MODULES)
	$(COMPOSE) run --rm --no-deps --user $(shell id -u):$(shell id -g) app bash -lc "find pkg/protos -name '*.go' -print0 | xargs -0 gofmt -s -w"

openapi.spec.gen:
	$(COMPOSE) run --rm --no-deps app /app/scripts/protoc_openapi_spec.sh $(REST_API_MODULES)

openapi.client.gen:
	rm -rf pkg/openapi_client
	$(DOCKER_RUN_AS_CURRENT_USER) \
		-v ${PWD}:/local openapitools/openapi-generator-cli:v7.13.0 generate \
		-i /local/api/swagger/superplane.swagger.json \
		-g go \
		-o /local/pkg/openapi_client \
		--additional-properties=packageName=openapi_client,enumClassPrefix=true,isGoSubmodule=true,withGoMod=false
	rm -rf pkg/openapi_client/test
	rm -rf pkg/openapi_client/docs
	rm -rf pkg/openapi_client/api
	rm -rf pkg/openapi_client/.travis.yml
	rm -rf pkg/openapi_client/README.md
	rm -rf pkg/openapi_client/git_push.sh
	$(COMPOSE) run --rm --no-deps --user $(shell id -u):$(shell id -g) app bash -lc "find pkg/openapi_client -name '*.go' -print0 | xargs -0 gofmt -s -w"

openapi.web.client.gen:
	rm -rf web_src/src/api-client
	$(COMPOSE) run --rm --no-deps --user $(shell id -u):$(shell id -g) app bash -lc "export HOME=/tmp && export NPM_CONFIG_CACHE=/tmp/.npm && cd web_src && npm run generate:api && npx prettier --write 'src/api-client/**/*.{ts,tsx}'"

#
# Image and CLI build
#

CLI_VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")

cli.build:
	$(MAKE) gen.setup.backend
	$(COMPOSE) run --rm --no-deps -e GOOS=$(OS) -e GOARCH=$(ARCH) app bash -c 'go build -ldflags "-X github.com/superplanehq/superplane/pkg/cli.Version=$(CLI_VERSION)" -o build/cli cmd/cli/main.go'

cli.build.m1:
	$(MAKE) cli.build OS=darwin ARCH=arm64

IMAGE?=superplane
IMAGE_TAG?=$(shell git rev-list -1 HEAD -- .)
REGISTRY_HOST?=ghcr.io/superplanehq
image.build:
	$(MAKE) gen.setup
	DOCKER_DEFAULT_PLATFORM=linux/amd64 docker build -f Dockerfile --target runner --build-arg BASE_URL=$(BASE_URL) --progress plain -t $(IMAGE):$(IMAGE_TAG) .

image.auth:
	@printf "%s" "$(GITHUB_TOKEN)" | docker login ghcr.io -u superplanehq --password-stdin

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
