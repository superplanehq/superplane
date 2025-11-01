ARG GO_VERSION=1.25.3
ARG UBUNTU_VERSION=22.04
ARG BUILDER_IMAGE="ubuntu:${UBUNTU_VERSION}"
ARG RUNNER_IMAGE="ubuntu:${UBUNTU_VERSION}"

# ----------------------------------------------------------------------------------------------------------------------
# Base stage with common dependencies.
# Based on this, the dev and builder stages are created.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${BUILDER_IMAGE} AS base

WORKDIR /tmp

COPY scripts/docker scripts

ENV GO_VERSION=1.25.3
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOBIN="/go/bin"
ENV PATH="${GOBIN}:${PATH}"

RUN bash scripts/install-go.sh ${GO_VERSION}
RUN bash scripts/install-nodejs.sh
RUN bash scripts/install-postgresql-client.sh
RUN bash scripts/install-gomigrate.sh
RUN bash scripts/install-protoc.sh

WORKDIR /app

COPY pkg pkg
COPY cmd cmd
COPY go.mod go.mod
COPY go.sum go.sum
COPY db/migrations /app/db/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
COPY web_src web_src
COPY protos protos
COPY api/swagger api/swagger
COPY rbac rbac
COPY templates templates

WORKDIR /app

# ----------------------------------------------------------------------------------------------------------------------
# Development stage with tools installed.
# Used for local development and testing.
# ----------------------------------------------------------------------------------------------------------------------

FROM base AS dev

WORKDIR /app

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
RUN go install github.com/air-verse/air@latest
RUN go install github.com/mgechev/revive@v1.8.0
RUN go install gotest.tools/gotestsum@v1.12.3

# Install Playwright
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1
RUN playwright install --with-deps

# Inject test files and dev entrypoint
COPY test test
COPY docker-entrypoint.dev.sh /app/docker-entrypoint.dev.sh

CMD [ "/bin/bash",  "-c \"while sleep 1000; do :; done\"" ]

# ----------------------------------------------------------------------------------------------------------------------
# Builder stage to create production artifacts.
# Used to build the final runner image.
# ----------------------------------------------------------------------------------------------------------------------

FROM base AS builder

ARG BASE_URL=https://app.superplane.com

WORKDIR /app
RUN rm -rf build && go build -o build/superplane cmd/server/main.go

WORKDIR /app/web_src
RUN npm install
RUN VITE_BASE_URL=$BASE_URL npm run build

# ----------------------------------------------------------------------------------------------------------------------
# Runner stage to run the application.
# Used as the final image.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${RUNNER_IMAGE} AS runner

# postgresql-client needs to be installed here too,
# otherwise the createdb command won't work.
# Install PostgreSQL 17.5 client tools
COPY scripts/docker/install-postgresql-client.sh install-postgresql-client.sh
RUN bash install-postgresql-client.sh

# We don't need Docker health checks, since these containers
# are intended to run in Kubernetes pods, which have probes.
HEALTHCHECK NONE

WORKDIR /app
RUN chown nobody /app

# Copy every artifact needed to run the application from previous stages
COPY --from=builder --chown=nobody:root /usr/bin/createdb /usr/bin/createdb
COPY --from=builder --chown=nobody:root /usr/bin/migrate /usr/bin/migrate
COPY --from=builder --chown=nobody:root /app/build/superplane /app/build/superplane
COPY --from=builder --chown=nobody:root /app/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY --from=builder --chown=nobody:root /app/db/migrations /app/db/migrations
COPY --from=builder --chown=nobody:root /app/pkg/web/assets/dist /app/pkg/web/assets/dist
COPY --from=builder --chown=nobody:root /app/api/swagger /app/api/swagger
COPY --from=builder --chown=nobody:root /app/rbac /app/rbac
COPY --from=builder --chown=nobody:root /app/templates /app/templates

USER nobody

CMD ["bash", "/app/docker-entrypoint.sh"]
