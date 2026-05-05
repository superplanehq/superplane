.PHONY: build test fmt

build:
	go build -o bin/fleet-manager ./fleet-manager/cmd/fleet-manager
	go build -o bin/runner ./runner/cmd/runner
	go build -o bin/task-broker ./task-broker/cmd/task-broker

test:
	go test ./...

fmt:
	gofmt -w fleet-manager runner shared task-broker
