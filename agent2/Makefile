.PHONY: build test lint

build:
	go build -o agent2 ./cmd/agent2

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f agent2
