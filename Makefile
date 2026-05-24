DOCKER_SERVICES := fleet-manager task-broker runner

# Local dev defaults (override on the command line)
LOCAL_TMP ?= /tmp
LOCAL_BROKER_DATABASE_URL ?= postgres://broker:broker@127.0.0.1:5432/broker?sslmode=disable
LOCAL_BROKER_LISTEN ?= :8081
LOCAL_BROKER_URL ?= http://127.0.0.1:8081
LOCAL_STACK_AUTH_TOKEN ?= dev-local-token
LOCAL_FLEET_ID ?= local
LOCAL_RUNNER_TRANSPORT ?= http
LOCAL_RUNNER_SHELL_USE_PIPE ?= 1

# Number of runner processes for `make runner` (default 1).
N ?= 1

.PHONY: build test fmt docker-build docker-build-fleet-manager docker-build-task-broker docker-build-runner docker-publish-ghcr runner-linux-amd64 runner-publish-s3
.PHONY: fleet-manager task-broker runner register-local-fleet local-dev-help

build:
	go build -o bin/fleet-manager ./fleet-manager/cmd/fleet-manager
	go build -o bin/runner ./runner/cmd/runner
	go build -o bin/task-broker ./task-broker/cmd/task-broker

test:
	go test ./...

local-dev-help:
	@echo "Local dev (separate terminals):"
	@echo "  make task-broker"
	@echo "  make register-local-fleet"
	@echo "  make runner N=3"
	@echo "Defaults are LOCAL_* / LOCAL_STACK_* / LOCAL_RUNNER_* / N in the Makefile (override on the command line)."

# Long-running: run in its own terminal (EC2 hot pool only; optional for local dev).
fleet-manager: build
	AUTH_TOKEN= ./bin/fleet-manager

# Long-running: run in its own terminal.
task-broker: build
	DATABASE_URL=$(LOCAL_BROKER_DATABASE_URL) LISTEN_ADDR=$(LOCAL_BROKER_LISTEN) \
		AUTH_TOKEN=$(LOCAL_STACK_AUTH_TOKEN) ./bin/task-broker

# After task-broker is listening; registers LOCAL_FLEET_ID on the broker.
register-local-fleet:
	curl -fsS -X POST "$(LOCAL_BROKER_URL)/v1/fleets" \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(LOCAL_STACK_AUTH_TOKEN)" \
		-d '{"id":"$(LOCAL_FLEET_ID)","labels":["local"]}'

# Blocks until all N runner processes exit. N=1 is one foreground-equivalent worker.
runner: build
	@set -e; n="$(N)"; i=1; \
	while [ "$$i" -le "$$n" ]; do \
	  ( cd "$(CURDIR)" && \
	    TASK_BROKER_URL="$(LOCAL_BROKER_URL)" \
	    RUNNER_FLEET_ID="$(LOCAL_FLEET_ID)" \
	    RUNNER_TRANSPORT="$(LOCAL_RUNNER_TRANSPORT)" \
	    RUNNER_SHELL_USE_PIPE="$(LOCAL_RUNNER_SHELL_USE_PIPE)" \
	    RUNNER_ID="runner-$$i" \
	    AUTH_TOKEN="$(LOCAL_STACK_AUTH_TOKEN)" \
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

# Static linux/amd64 runner binary (fleet EC2 host + systemd). Same flags as Docker image build stage.
RUNNER_LINUX_AMD64 ?= bin/runner-linux-amd64

runner-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(RUNNER_LINUX_AMD64) ./runner/cmd/runner

# Expects EC2_PROVISION_RUNNER_S3_URI=s3://bucket/key and aws CLI (same env as fleet-manager + Semaphore runner-s3).
runner-publish-s3: runner-linux-amd64
	@test -n "$(EC2_PROVISION_RUNNER_S3_URI)" || (echo "EC2_PROVISION_RUNNER_S3_URI=s3://bucket/key is required"; exit 1)
	aws s3 cp $(RUNNER_LINUX_AMD64) "$(EC2_PROVISION_RUNNER_S3_URI)" --sse AES256

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
