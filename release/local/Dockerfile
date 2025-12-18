ARG GO_VERSION=1.25.3
ARG UBUNTU_VERSION=22.04
ARG BUILDER_IMAGE="ubuntu:${UBUNTU_VERSION}"
ARG RUNNER_IMAGE="ubuntu:${UBUNTU_VERSION}"

# ----------------------------------------------------------------------------------------------------------------------
# Base stage with build dependencies (mirrors root Dockerfile's base).
# ----------------------------------------------------------------------------------------------------------------------

FROM ${BUILDER_IMAGE} AS base

WORKDIR /tmp

COPY scripts/docker scripts

ENV GO_VERSION=1.25.3
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
# Builder stage: build backend binary and UI assets.
# ----------------------------------------------------------------------------------------------------------------------

FROM base AS builder

ARG BASE_URL=https://app.superplane.com

WORKDIR /app
RUN rm -rf build && go build -o build/superplane cmd/server/main.go

WORKDIR /app/web_src
RUN npm install
RUN VITE_BASE_URL=$BASE_URL npm run build

# ----------------------------------------------------------------------------------------------------------------------
# Trial runner: single-container image with embedded Postgres.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${RUNNER_IMAGE} AS trial

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      postgresql \
      postgresql-contrib \
      rabbitmq-server \
      ca-certificates \
      curl && \
    # Ensure PostgreSQL server binaries (initdb, pg_ctl, psql, etc.) are on PATH
    ln -s /usr/lib/postgresql/*/bin/* /usr/local/bin/ && \
    rm -rf /var/lib/apt/lists/*

# We still need the PostgreSQL client tools (createdb/migrate) as in the main runner.
COPY scripts/docker/install-postgresql-client.sh /tmp/install-postgresql-client.sh
RUN bash /tmp/install-postgresql-client.sh && rm /tmp/install-postgresql-client.sh

WORKDIR /app

# Copy artifacts from builder
COPY --from=builder /usr/bin/createdb /usr/bin/createdb
COPY --from=builder /usr/bin/migrate /usr/bin/migrate
COPY --from=builder /app/build/superplane /app/build/superplane
COPY --from=builder /app/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY --from=builder /app/db/migrations /app/db/migrations
COPY --from=builder /app/pkg/web/assets/dist /app/pkg/web/assets/dist
COPY --from=builder /app/api/swagger /app/api/swagger
COPY --from=builder /app/rbac /app/rbac
COPY --from=builder /app/templates /app/templates

# Trial entrypoint that runs embedded Postgres and RabbitMQ and then Superplane.
COPY release/local/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh /app/docker-entrypoint.sh

EXPOSE 8000

ENTRYPOINT ["/app/entrypoint.sh"]
