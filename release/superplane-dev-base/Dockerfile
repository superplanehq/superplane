ARG GO_VERSION=1.25.3
ARG PLAYWRIGHT_GO_VERSION=v0.5200.1

FROM ubuntu:22.04

ARG GO_VERSION
ARG PLAYWRIGHT_GO_VERSION

WORKDIR /tmp

COPY scripts/docker scripts

ENV GO_VERSION=${GO_VERSION}
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOBIN="/go/bin"
ENV PATH="${GOBIN}:${PATH}"
ENV GOPROXY="https://proxy.golang.org,direct"
ENV PLAYWRIGHT_BROWSERS_PATH="/ms-playwright"

RUN bash scripts/install-go.sh ${GO_VERSION}
RUN bash scripts/install-nodejs.sh
RUN bash scripts/install-postgresql-client.sh
RUN bash scripts/install-gomigrate.sh
RUN bash scripts/install-protoc.sh

# Install dev tools with retries to mitigate intermittent network flakes.
# Pin protoc plugin versions to ensure consistent proto generation.
RUN bash /tmp/scripts/retry.sh 6 2s go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
RUN bash /tmp/scripts/retry.sh 6 2s go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
RUN bash /tmp/scripts/retry.sh 6 2s go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.26.3
RUN bash /tmp/scripts/retry.sh 6 2s go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.26.3
RUN bash /tmp/scripts/retry.sh 6 2s go install github.com/air-verse/air@latest
RUN bash /tmp/scripts/retry.sh 6 2s go install github.com/mgechev/revive@v1.8.0
RUN bash /tmp/scripts/retry.sh 6 2s go install gotest.tools/gotestsum@v1.12.3
RUN bash /tmp/scripts/retry.sh 6 2s go install github.com/playwright-community/playwright-go/cmd/playwright@${PLAYWRIGHT_GO_VERSION}

WORKDIR /app

# Pre cache Go modules
COPY go.mod go.mod
COPY go.sum go.sum
RUN bash /tmp/scripts/retry.sh 6 2s go mod download

# Pre cache Playwright browsers
RUN mkdir -p "${PLAYWRIGHT_BROWSERS_PATH}"
RUN bash /tmp/scripts/retry.sh 6 2s playwright install chromium-headless-shell --with-deps