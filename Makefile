.PHONY: build test fmt docker-build docker-build-fleet-manager docker-build-task-broker docker-build-runner

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
