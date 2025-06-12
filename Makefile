.PHONY: lint test

DB_NAME=superplane
DB_PASSWORD=the-cake-is-a-lie
DOCKER_COMPOSE_OPTS=-f docker-compose.dev.yml
TEST_PACKAGES := ./...

#
# Targets for prod-like environment
#

setup:
	$(MAKE) db.create DOCKER_COMPOSE_OPTS="-f docker-compose.yml"
	$(MAKE) db.migrate DOCKER_COMPOSE_OPTS="-f docker-compose.yml"
	docker compose -f docker-compose.yml build

start:
	docker compose -f docker-compose.yml up

#
# Targets for test environment
#

lint:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app revive -formatter friendly -config lint.toml ./...

tidy:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm app go mod tidy

test.setup:
	docker-compose $(DOCKER_COMPOSE_OPTS) build
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm app go get ./...
	$(MAKE) db.create DB_NAME=superplane_test
	$(MAKE) db.migrate DB_NAME=superplane_test

test.down:
	docker compose $(DOCKER_COMPOSE_OPTS) down --remove-orphans

test:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm -e DB_NAME=superplane_test app gotestsum --format short-verbose --junitfile junit-report.xml --packages="$(TEST_PACKAGES)" -- -p 1

test.watch:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm app gotestsum --watch --format short-verbose --junitfile junit-report.xml --packages="$(TEST_PACKAGES)" -- -p 1

#
# Targets for dev environment
#

dev.setup:
	$(MAKE) db.create DB_NAME=superplane_dev
	$(MAKE) db.migrate DB_NAME=superplane_dev
	docker compose $(DOCKER_COMPOSE_OPTS) build

dev.start:
	docker compose $(DOCKER_COMPOSE_OPTS) up

dev.console:
	docker compose $(DOCKER_COMPOSE_OPTS) run --rm --service-ports app /bin/bash

#
# Database target helpers
#

db.create:
	-docker-compose $(DOCKER_COMPOSE_OPTS) run --rm -e PGPASSWORD=the-cake-is-a-lie app createdb -h db -p 5432 -U postgres $(DB_NAME)
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME) -c 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp";'

db.migration.create:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm app mkdir -p db/migrations
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm app migrate create -ext sql -dir db/migrations $(NAME)
	ls -lah db/migrations/*$(NAME)*

db.migrate:
	rm -f db/structure.sql
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) app migrate -source file://db/migrations -database postgres://postgres:$(DB_PASSWORD)@db:5432/$(DB_NAME)?sslmode=disable up
	# echo dump schema to db/structure.sql
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --schema-only --no-privileges --no-owner -h db -p 5432 -U postgres -d $(DB_NAME)" > db/structure.sql
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --data-only --table schema_migrations -h db -p 5432 -U postgres -d $(DB_NAME)" >> db/structure.sql

db.console:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME)

db.delete:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --user $$(id -u):$$(id -g) --rm -e PGPASSWORD=$(DB_PASSWORD) app dropdb -h db -p 5432 -U postgres $(DB_NAME)

#
# Protobuf compilation
#

MODULES := superplane
REST_API_MODULES := superplane
pb.gen:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app /app/scripts/protoc.sh $(MODULES)
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app /app/scripts/protoc_gateway.sh $(REST_API_MODULES)

openapi.spec.gen:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app /app/scripts/protoc_openapi_spec.sh $(REST_API_MODULES)

openapi.client.gen:
	rm -rf pkg/openapi_client
	openapi-generator generate \
		-i api/swagger/superplane.swagger.json \
		-g go \
		-o pkg/openapi_client \
		--additional-properties=packageName=openapi_client,enumClassPrefix=true,isGoSubmodule=true,withGoMod=false
	rm -rf pkg/openapi_client/test
	rm -rf pkg/openapi_client/docs
	rm -rf pkg/openapi_client/api
	rm -rf pkg/openapi_client/.travis.yml
	rm -rf pkg/openapi_client/README.md
	rm -rf pkg/openapi_client/git_push.sh

openapi.web.client.gen:
	rm -rf web_src/src/api-client
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps app bash -c "cd web_src && npm run generate:api"
#
# Image and CLI build
#

cli.build:
	docker-compose $(DOCKER_COMPOSE_OPTS) run --rm --no-deps -e GOOS=$(OS) -e GOARCH=$(ARCH) app go build -o build/cli cmd/cli/main.go

IMAGE?=superplane
IMAGE_TAG?=$(shell git rev-list -1 HEAD -- .)
REGISTRY_HOST?=ghcr.io/superplanehq
image.build:
	DOCKER_DEFAULT_PLATFORM=linux/amd64 docker build -f Dockerfile --target runner --progress plain -t $(IMAGE):$(IMAGE_TAG) .

image.auth:
	@printf "%s" "$(GITHUB_TOKEN)" | docker login ghcr.io -u superplanehq --password-stdin

image.push:
	docker tag $(IMAGE):$(IMAGE_TAG) $(REGISTRY_HOST)/$(IMAGE):$(IMAGE_TAG)
	docker push $(REGISTRY_HOST)/$(IMAGE):$(IMAGE_TAG)
