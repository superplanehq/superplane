.PHONY: lint test

APP_NAME=superplane
APP_ENV=prod

test.setup: openapi.spec.gen
	docker-compose build
	docker-compose run --rm app go get ./...
	-$(MAKE) db.test.create
	$(MAKE) db.migrate

lint:
	docker-compose run --rm --no-deps app revive -formatter friendly -config lint.toml ./...

test:
	docker-compose run --rm app gotestsum --format short-verbose --junitfile junit-report.xml --packages="./..." -- -p 1

test.watch:
	docker-compose run --rm app gotestsum --watch --format short-verbose --junitfile junit-report.xml --packages="./..." -- -p 1

tidy:
	docker-compose run --rm app go mod tidy

#
# Database
#

DB_NAME=superplane
DB_PASSWORD=the-cake-is-a-lie

db.test.create:
	-docker-compose run --rm -e PGPASSWORD=the-cake-is-a-lie app createdb -h db -p 5432 -U postgres $(DB_NAME)
	docker-compose run --rm -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME) -c 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp";'

db.migration.create:
	docker-compose run --rm app mkdir -p db/migrations
	docker-compose run --rm app migrate create -ext sql -dir db/migrations $(NAME)
	ls -lah db/migrations/*$(NAME)*

db.migrate:
	rm -f db/structure.sql
	docker-compose run --rm --user $$(id -u):$$(id -g) app migrate -source file://db/migrations -database postgres://postgres:$(DB_PASSWORD)@db:5432/$(DB_NAME)?sslmode=disable up
	# echo dump schema to db/structure.sql
	docker-compose run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --schema-only --no-privileges --no-owner -h db -p 5432 -U postgres -d $(DB_NAME)" > db/structure.sql
	docker-compose run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=$(DB_PASSWORD) app bash -c "pg_dump --data-only --table schema_migrations -h db -p 5432 -U postgres -d $(DB_NAME)" >> db/structure.sql

db.test.console:
	docker-compose run --rm --user $$(id -u):$$(id -g) -e PGPASSWORD=the-cake-is-a-lie app psql -h db -p 5432 -U postgres $(DB_NAME)

db.test.delete:
	docker-compose run --rm --user $$(id -u):$$(id -g) --rm -e PGPASSWORD=$(DB_PASSWORD) app dropdb -h db -p 5432 -U postgres $(DB_NAME)

#
# Protobuf compilation
#

MODULES := superplane
REST_API_MODULES := superplane
pb.gen:
	docker-compose run --rm --no-deps app /app/scripts/protoc.sh $(MODULES)
	docker-compose run --rm --no-deps app /app/scripts/protoc_gateway.sh $(REST_API_MODULES)

openapi.spec.gen:
	DOCKER_DEFAULT_PLATFORM=linux/amd64 docker-compose run --rm --no-deps app /app/scripts/protoc_openapi_spec.sh $(REST_API_MODULES)

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

#
# Image and CLI build
#

cli.build:
	go build -o build/cli cmd/cli/main.go

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

#
# Dev environment helpers
#

dev.setup: db.test.create db.migrate

dev.console: dev.setup
	docker compose run --rm --service-ports app /bin/bash 

dev.server: dev.setup
	docker compose run --rm --service-ports app air 
