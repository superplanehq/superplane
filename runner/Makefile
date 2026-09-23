DOCKER_SERVICES := fleet-manager task-broker runner

# Local dev defaults (override on the command line)
LOCAL_TMP ?= /tmp
LOCAL_BROKER_DATABASE_URL ?= postgres://broker:broker@127.0.0.1:5432/broker?sslmode=disable

# Run integration tests that require Postgres.
# Starts a DB via docker compose, waits for it, runs tests, then tears down.
.PHONY: test-integration
test-integration:
	docker compose up -d db
	TEST_DATABASE_URL=$(LOCAL_BROKER_DATABASE_URL) go test ./task-broker/... ./runner/... ./shared/... -count=1 -p 1
	docker compose stop db

LOCAL_BROKER_LISTEN ?= :8081
LOCAL_BROKER_URL ?= http://127.0.0.1:8081
LOCAL_STACK_AUTH_TOKEN ?= dev-local-token
LOCAL_FLEET_ID ?= local
LOCAL_RUNNER_TRANSPORT ?= http
LOCAL_RUNNER_SHELL_USE_PIPE ?= 1
TASK_BROKER_HOST_PORT ?= 8091
AUTH_TOKEN ?= $(LOCAL_STACK_AUTH_TOKEN)
export TASK_BROKER_HOST_PORT AUTH_TOKEN

# Number of local runner workers for `make dev` and `make runner`.
N ?= 10

.PHONY: build test fmt docker-build docker-build-fleet-manager docker-build-task-broker docker-build-runner docker-publish-ghcr runner-linux-amd64 runner-linux-arm64 runner-linux-all runner-publish-s3 runner-publish-s3-all
.PHONY: dev dev.down doctor-local fleet-manager task-broker runner register-local-fleet register-superplane-fleets local-dev-help

build:
	go build -o bin/fleet-manager ./fleet-manager/cmd/fleet-manager
	go build -o bin/runner ./runner/cmd/runner
	go build -o bin/task-broker ./task-broker/cmd/task-broker

test:
	go test ./...

local-dev-help:
	@echo "Local start from the SuperPlane repo root:"
	@echo "  make dev.up"
	@echo "  make dev.setup"
	@echo "  make dev.server"
	@echo "  make doctor-local"
	@echo "  make dev.down"
	@echo "Manual (separate terminals, this directory):"
	@echo "  make task-broker"
	@echo "  make register-local-fleet"
	@echo "  make register-superplane-fleets"
	@echo "  make runner N=10"
	@echo "Defaults are LOCAL_* / LOCAL_STACK_* / LOCAL_RUNNER_* / N in the Makefile (override on the command line)."

# Local start lives on the SuperPlane repo root. Use those targets.
dev:
	@echo "Start the local stack from the SuperPlane repo root:" >&2
	@echo "  make dev.up" >&2
	@echo "  make dev.setup" >&2
	@echo "  make dev.server" >&2
	@exit 1

dev.down:
	$(MAKE) -C .. dev.down

# Confirm factory host-mode CLIs exist on the Compose worker PATH.
doctor-local:
	$(MAKE) -C .. doctor-local

# Long-running: run in its own terminal. Requires a JSON config; set FM_CONFIG_FILE.
# Example:
#   cp scripts/deploy/fleet-manager.config.example.json /tmp/fleet-manager.config.json
#   $EDITOR /tmp/fleet-manager.config.json
#   FM_CONFIG_FILE=/tmp/fleet-manager.config.json make fleet-manager
fleet-manager: build
	./bin/fleet-manager

# Long-running: run in its own terminal.
task-broker: build
	DATABASE_URL=$(LOCAL_BROKER_DATABASE_URL) LISTEN_ADDR=$(LOCAL_BROKER_LISTEN) \
		AUTH_TOKEN=$(LOCAL_STACK_AUTH_TOKEN) ./bin/task-broker

# After task-broker is listening; registers LOCAL_FLEET_ID on the broker.
register-local-fleet:
	curl -fsS -X POST "$(LOCAL_BROKER_URL)/v1/fleets" \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(LOCAL_STACK_AUTH_TOKEN)" \
		-d '{"id":"$(LOCAL_FLEET_ID)","provisioner":"local","arch":"amd64","size":"local"}'

# Registers SuperPlane machine types plus LOCAL_FLEET_ID so canvas nodes can queue locally.
SUPERPLANE_FLEET_IDS ?= local e1-large-amd64 e1-large-arm64 e1-tiny-amd64 e1-tiny-arm64
register-superplane-fleets:
	@set -e; \
	for id in $(SUPERPLANE_FLEET_IDS); do \
	  curl -fsS -X POST "$(LOCAL_BROKER_URL)/v1/fleets" \
	    -H "Content-Type: application/json" \
	    -H "Authorization: Bearer $(LOCAL_STACK_AUTH_TOKEN)" \
	    -d "{\"id\":\"$$id\",\"provisioner\":\"local\",\"arch\":\"amd64\",\"size\":\"local\"}"; \
	  echo; \
	done

# Blocks until all N runner processes exit. N=1 is one foreground-equivalent worker.
runner: build
	@set -e; n="$(N)"; i=1; \
	while [ "$$i" -le "$$n" ]; do \
	  reg=$$(go run ./scripts/mint-runner-registration \
	    -fleet "$(LOCAL_FLEET_ID)" -secret "$(LOCAL_STACK_AUTH_TOKEN)"); \
	  ( cd "$(CURDIR)" && \
	    TASK_BROKER_URL="$(LOCAL_BROKER_URL)" \
	    RUNNER_FLEET_ID="$(LOCAL_FLEET_ID)" \
	    RUNNER_TRANSPORT="$(LOCAL_RUNNER_TRANSPORT)" \
	    RUNNER_SHELL_USE_PIPE="$(LOCAL_RUNNER_SHELL_USE_PIPE)" \
	    RUNNER_RESET_TASK_HOME="1" \
	    RUNNER_ID="runner-$$i" \
	    RUNNER_REGISTRATION_TOKEN="$$reg" \
	    exec ./bin/runner ) & \
	  i=$$((i+1)); \
	done; \
	wait

fmt:
	gofmt -w fleet-manager runner shared task-broker

docker-build-fleet-manager:
	docker build -f fleet-manager/Dockerfile -t superplane/fleet-manager:latest .

docker-build-task-broker:
	docker build -f task-broker/Dockerfile -t superplane/task-broker:latest .

docker-build-runner:
	docker build -f runner/Dockerfile -t superplane/runner:latest .

docker-build: docker-build-fleet-manager docker-build-task-broker docker-build-runner

# Static linux runner binaries (fleet EC2 host + systemd). Same flags as Docker image build stage.
RUNNER_LINUX_AMD64 ?= bin/runner-linux-amd64
RUNNER_LINUX_ARM64 ?= bin/runner-linux-arm64

runner-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(RUNNER_LINUX_AMD64) ./runner/cmd/runner

runner-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(RUNNER_LINUX_ARM64) ./runner/cmd/runner

runner-linux-all: runner-linux-amd64 runner-linux-arm64

# Expects EC2_PROVISION_RUNNER_S3_URI=s3://bucket/key and aws CLI (same env as fleet-manager + Semaphore runner-s3).
runner-publish-s3: runner-linux-amd64
	@test -n "$(EC2_PROVISION_RUNNER_S3_URI)" || (echo "EC2_PROVISION_RUNNER_S3_URI=s3://bucket/key is required"; exit 1)
	aws s3 cp $(RUNNER_LINUX_AMD64) "$(EC2_PROVISION_RUNNER_S3_URI)" --sse AES256

# Expects arch-specific S3 keys for publishing both EC2 runner binaries.
runner-publish-s3-all: runner-linux-all
	@test -n "$(EC2_PROVISION_RUNNER_AMD64_S3_URI)" || (echo "EC2_PROVISION_RUNNER_AMD64_S3_URI=s3://bucket/key is required"; exit 1)
	@test -n "$(EC2_PROVISION_RUNNER_ARM64_S3_URI)" || (echo "EC2_PROVISION_RUNNER_ARM64_S3_URI=s3://bucket/key is required"; exit 1)
	aws s3 cp $(RUNNER_LINUX_AMD64) "$(EC2_PROVISION_RUNNER_AMD64_S3_URI)" --sse AES256
	aws s3 cp $(RUNNER_LINUX_ARM64) "$(EC2_PROVISION_RUNNER_ARM64_S3_URI)" --sse AES256

# Expects IMAGE_PREFIX (e.g. ghcr.io/org/repo) and IMAGE_TAG (short SHA). Pushes that tag plus :latest.
docker-publish-ghcr:
	@set -e; \
	for svc in $(DOCKER_SERVICES); do \
		img="$(IMAGE_PREFIX)/$$svc"; \
		docker build -f $$svc/Dockerfile -t $$img:$(IMAGE_TAG) .; \
		docker push $$img:$(IMAGE_TAG); \
		docker tag $$img:$(IMAGE_TAG) $$img:latest; \
		docker push $$img:latest; \
	done
