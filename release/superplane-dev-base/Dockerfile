ARG GO_VERSION=1.25.3
ARG PLAYWRIGHT_GO_VERSION=v0.5200.1

FROM ubuntu:24.04

ARG GO_VERSION
ARG PLAYWRIGHT_GO_VERSION

WORKDIR /opt/install
COPY scripts/docker /opt/install-scripts
COPY --from=ghcr.io/astral-sh/uv:0.6.6 /uv /uvx /bin/

ENV GO_VERSION=${GO_VERSION}
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOBIN="/go/bin"
ENV PATH="${GOBIN}:${PATH}"
ENV GOPROXY="https://proxy.golang.org,direct"
ENV PLAYWRIGHT_BROWSERS_PATH="/ms-playwright"
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

RUN apt-get update && \
    apt-get install -y --no-install-recommends bash ca-certificates make software-properties-common && \
    add-apt-repository -y ppa:deadsnakes/ppa && \
    apt-get update && \
    apt-get install -y --no-install-recommends python3.13 python3.13-venv python3.13-dev && \
    ln -sf /usr/bin/python3.13 /usr/local/bin/python3 && \
    ln -sf /usr/bin/python3.13 /usr/local/bin/python && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/* /tmp/* && \
    bash /opt/install-scripts/install-go.sh "${GO_VERSION}" && \
    bash /opt/install-scripts/install-nodejs.sh && \
    bash /opt/install-scripts/install-postgresql-client.sh && \
    bash /opt/install-scripts/install-gomigrate.sh && \
    bash /opt/install-scripts/install-protoc.sh

# Install dev tools with retries to mitigate intermittent network flakes.
# Pin protoc plugin versions to ensure consistent proto generation.
RUN bash /opt/install-scripts/retry.sh 6 2s go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6 && \
    bash /opt/install-scripts/retry.sh 6 2s go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0 && \
    bash /opt/install-scripts/retry.sh 6 2s go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.26.3 && \
    bash /opt/install-scripts/retry.sh 6 2s go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.26.3 && \
    bash /opt/install-scripts/retry.sh 6 2s go install github.com/air-verse/air@latest && \
    bash /opt/install-scripts/retry.sh 6 2s go install github.com/mgechev/revive@v1.8.0 && \
    bash /opt/install-scripts/retry.sh 6 2s go install gotest.tools/gotestsum@v1.12.3 && \
    bash /opt/install-scripts/retry.sh 6 2s go install github.com/playwright-community/playwright-go/cmd/playwright@"${PLAYWRIGHT_GO_VERSION}"

WORKDIR /app

# Pre cache Go modules
COPY go.mod go.mod
COPY go.sum go.sum
RUN mkdir -p "${PLAYWRIGHT_BROWSERS_PATH}" && \
    bash /opt/install-scripts/retry.sh 6 2s go mod download && \
    bash /opt/install-scripts/retry.sh 6 2s playwright install chromium-headless-shell --with-deps && \
    rm -rf /opt/install /opt/install-scripts /tmp/*