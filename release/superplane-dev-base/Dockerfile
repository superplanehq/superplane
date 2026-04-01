ARG GO_VERSION=1.25.3
ARG UBUNTU_VERSION=22.04
ARG BUILDER_IMAGE="ubuntu:${UBUNTU_VERSION}"

FROM ${BUILDER_IMAGE}

ARG GO_VERSION

WORKDIR /tmp

COPY scripts/docker scripts

ENV GO_VERSION=${GO_VERSION}
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOBIN="/go/bin"
ENV PATH="${GOBIN}:${PATH}"
ENV GOPROXY="https://proxy.golang.org,direct"

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

WORKDIR /app
