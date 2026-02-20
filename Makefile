.PHONY: lint test test.license.check

DB_NAME=superplane
DB_PASSWORD=the-cake-is-a-lie
DOCKER_COMPOSE_OPTS=-f docker-compose.dev.yml
BASE_URL?=https://app.superplane.com

export BUILDKIT_PROGRESS ?= plain

PKG_TEST_PACKAGES := ./pkg/...
E2E_TEST_PACKAGES := ./test/e2e/...

#
# Long sausage command to run tests with gotestsum
#
# - starts a docker container for unit tests
# - mounts tmp/screenshots
# - exports junit report
# - sets parallelism to 1
#
GOTESTSUM=docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e DB_NAME=superplane_test -v $(PWD)/tmp/screenshots:/app/test/screenshots app gotestsum --format short --junitfile junit-report.xml 

#
# Targets for test environment
#

lint:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app revive -formatter friendly -config lint.toml -exclude ./tmp/... ./...

tidy:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app go mod tidy

test.setup:
	@if [ -d "tmp/screenshots" ]; then rm -rf tmp/screenshots; fi
	@mkdir -p tmp/screenshots
	docker compose $(DOCKER_COMPOSE_OPTS) build
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app go mod download
	$(MAKE) db.create DB_NAME=superplane_test
	$(MAKE) db.migrate DB_NAME=superplane_test

test.start:
	docker compose $(DOCKER_COMPOSE_OPTS) up -d
	sleep 5

test.down:
	docker compose $(DOCKER_COMPOSE_OPTS) down --remove-orphans

test.e2e.setup:
	$(MAKE) test.setup
	docker compose $(DOCKER_COMPOSE_OPTS) exec app bash -c "cd web_src && npm ci"

test.e2e:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app gotestsum --format short --junitfile junit-report.xml --rerun-fails=3 --rerun-fails-max-failures=1 --packages="$(E2E_TEST_PACKAGES)" -- -p 1

test.e2e.autoparallel:
	docker compose $(DOCKER_COMPOSE_OPTS) exec -e INDEX -e TOTAL app bash -lc "cd /app && bash scripts/test_e2e_autoparallel.sh"

test.e2e.single:
	bash ./scripts/vscode_run_tests.sh line $(FILE) $(LINE)

test:
	$(GOTESTSUM) --packages="$(PKG_TEST_PACKAGES)" -- -p 1

test.license.check:
	bash ./scripts/license-check.sh

test.watch:
	$(GOTESTSUM) --packages="$(PKG_TEST_PACKAGES)" --watch -- -p 1

test.shell:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e DB_NAME=superplane_test -v $(PWD)/tmp/screenshots:/app/test/screenshots app /bin/bash	

setup.playwright:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app bash -c "bash scripts/docker/retry.sh 6 2s go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1"
	docker compose $(DOCKER_COMPOSE_OPTS) exec app bash -c "if [ -d /app/tmp/ms-playwright ] && [ \"$(ls -A /app/tmp/ms-playwright 2>/dev/null)\" ]; then echo \"Playwright browsers cache present, skipping install\"; else bash scripts/docker/retry.sh 6 2s playwright install chromium-headless-shell --with-deps; fi"

#
# Code formatting
#

format.go:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app bash -c "find . -name '*.go' -not -path './tmp/*' -print0 | xargs -0 gofmt -s -w"

format.go.check:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app bash -c "find . -name '*.go' -not -path './tmp/*' -print0 | xargs -0 gofmt -s -l | tee /dev/stderr | if read; then exit 1; else exit 0; fi"

format.js:
	cd web_src && npm run format

format.js.check:
	cd web_src && npm run format:check

#
# Targets for dev environment
#

dev.setup:
	docker compose $(DOCKER_COMPOSE_OPTS) build
	$(MAKE) db.create DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_dev

dev.setup.no.cache:
	docker compose $(DOCKER_COMPOSE_OPTS) down -v --remove-orphans
	rm -rf tmp
	docker compose $(DOCKER_COMPOSE_OPTS) build --no-cache
	$(MAKE) db.create DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_dev

dev.start.fg:
	docker compose $(DOCKER_COMPOSE_OPTS) up

dev.start:
	docker compose $(DOCKER_COMPOSE_OPTS) up -d
	@bash ./scripts/wait-for-app

dev.logs:
	docker compose $(DOCKER_COMPOSE_OPTS) logs -f

dev.logs.app:
	docker compose $(DOCKER_COMPOSE_OPTS) logs -f app

dev.logs.otel:
	docker compose $(DOCKER_COMPOSE_OPTS) logs -f otel

dev.down:
	docker compose $(DOCKER_COMPOSE_OPTS) down --remove-orphans

dev.console:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app /bin/bash

dev.db:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app sh -c 'PGPASSWORD=$(DB_PASSWORD) psql -h db -p 5432 -U postgres -d superplane_dev'

dev.db.console:
	$(MAKE) db.console DB_NAME=superplane_dev

dev.pr.clean.checkout:
	bash ./scripts/clean-pr-checkout $(PR)

check.db.structure:
	bash ./scripts/verify_db_structure_clean.sh

check.db.migrations:
	bash ./scripts/verify_no_future_migrations.sh

check.build.ui:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app bash -c "cd web_src && npm run build"

check.build.app:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app go build cmd/server/main.go


storybook:
	docker compose $(DOCKER_COMPOSE_OPTS) exec app /bin/bash -c "cd web_src && npm install && npm run storybook"

ui.setup:
	npm install

ui.start:
	npm run storybook

#
# Database target helpers
#

db.create:
	-docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres -c 'ALTER DATABASE template1 REFRESH COLLATION VERSION';
	-docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres -c 'ALTER DATABASE postgres REFRESH COLLATION VERSION';
	-docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e PGPASSWORD=the-cake-is-a-lie app createdb -h db -p 5432 -U postgres $(DB_NAME)
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME) -c 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp";'

db.migration.create:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app mkdir -p db/migrations
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app migrate create -ext sql -dir db/migrations $(NAME)
	ls -lah db/migrations/*$(NAME)*

db.data_migration.create:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app mkdir -p db/data_migrations
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm app migrate create -ext sql -dir db/data_migrations $(NAME)
	ls -lah db/data_migrations/*$(NAME)*

db.migrate:
	rm -f db/structure.sql
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) app migrate -source file://db/migrations -database postgres://postgres:$(DB_PASSWORD)@db:5432/$(DB_NAME)?sslmode=disable up
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) app migrate -source file://db/data_migrations -database postgres://postgres:$(DB_PASSWORD)@db:5432/$(DB_NAME)?sslmode=disable\&x-migrations-table=data_migrations up
	# echo dump schema to db/structure.sql
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --schema-only --no-privileges --restrict-key abcdef123 --no-owner -h db -p 5432 -U postgres -d $(DB_NAME)" > db/structure.sql
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --data-only --restrict-key abcdef123 --table schema_migrations -h db -p 5432 -U postgres -d $(DB_NAME)" >> db/structure.sql
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --data-only --restrict-key abcdef123 --table data_migrations -h db -p 5432 -U postgres -d $(DB_NAME)" >> db/structure.sql

db.migrate.all:
	$(MAKE) db.migrate DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_test

db.console:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME)

db.delete:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) --rm -e PGPASSWORD=$(DB_PASSWORD) app dropdb -h db -p 5432 -U postgres $(DB_NAME)

db.recreate.all.dangerous:
	$(MAKE) dev.down
	-$(MAKE) db.delete DB_NAME=superplane_dev
	-$(MAKE) db.delete DB_NAME=superplane_test
	$(MAKE) db.create DB_NAME=superplane_dev
	$(MAKE) db.create DB_NAME=superplane_test
	$(MAKE) db.migrate DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_test

#
# Protobuf compilation
#

gen:
	$(MAKE) pb.gen
	$(MAKE) openapi.spec.gen
	$(MAKE) openapi.client.gen
	$(MAKE) openapi.web.client.gen
	$(MAKE) format.go
	$(MAKE) format.js

gen.components.docs:
	rm -rf docs/components
	go run scripts/generate_components_docs.go

gen.components.local.update: gen.components.docs
	rm -rf ../docs/src/content/docs/components
	cp -R docs/components ../docs/src/content/docs/components

MODULES := authorization,organizations,integrations,secrets,users,groups,roles,me,configuration,components,triggers,widgets,blueprints,canvases,service_accounts
REST_API_MODULES := authorization,organizations,integrations,secrets,users,groups,roles,me,configuration,components,triggers,widgets,blueprints,canvases,service_accounts
pb.gen:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app /app/scripts/protoc.sh $(MODULES)
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app /app/scripts/protoc_gateway.sh $(REST_API_MODULES)

openapi.spec.gen:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app /app/scripts/protoc_openapi_spec.sh $(REST_API_MODULES)

openapi.client.gen:
	rm -rf pkg/openapi_client
	docker run --rm \
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

openapi.web.client.gen:
	rm -rf web_src/src/api-client
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app bash -c "cd web_src && npm run generate:api"

#
# Image and CLI build
#

cli.build:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps -e GOOS=$(OS) -e GOARCH=$(ARCH) app bash -c 'go build -o build/cli cmd/cli/main.go'

IMAGE?=superplane
IMAGE_TAG?=$(shell git rev-list -1 HEAD -- .)
REGISTRY_HOST?=ghcr.io/superplanehq
image.build:
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
		
