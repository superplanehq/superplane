DOCKER_SERVICES := fleet-manager task-broker runner

.PHONY: build test fmt docker-build docker-build-fleet-manager docker-build-task-broker docker-build-runner docker-publish-ghcr runner-linux-amd64 runner-publish-s3

build:
	go build -o bin/fleet-manager ./fleet-manager/cmd/fleet-manager
	go build -o bin/runner ./runner/cmd/runner
	go build -o bin/task-broker ./task-broker/cmd/task-broker

test:
	go test ./...

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
