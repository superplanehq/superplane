ARG UBUNTU_VERSION=22.04
ARG GO_VERSION=1.25.3
ARG PLAYWRIGHT_GO_VERSION=v0.5200.1
ARG RUNNER_IMAGE="ubuntu:${UBUNTU_VERSION}"

# ----------------------------------------------------------------------------------------------------------------------
# Development stage with tools installed.
# Used for local development and testing.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${UBUNTU_VERSION} AS dev-base

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

RUN apt-get update && \
  apt-get install -y --no-install-recommends bash ca-certificates make unzip && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

RUN bash /opt/install-scripts/install-go.sh "${GO_VERSION}"
RUN bash /opt/install-scripts/install-nodejs.sh
RUN bash /opt/install-scripts/install-postgresql-client.sh
RUN bash /opt/install-scripts/install-gomigrate.sh
RUN bash /opt/install-scripts/install-protoc.sh

RUN export GOMODCACHE=/tmp/go-mod GOCACHE=/tmp/go-build && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0 && \
  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.26.3 && \
  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.26.3 && \
  go install github.com/air-verse/air@latest && \
  go install github.com/mgechev/revive@v1.8.0 && \
  go install gotest.tools/gotestsum@v1.12.3 && \
  go install github.com/playwright-community/playwright-go/cmd/playwright@"${PLAYWRIGHT_GO_VERSION}" && \
  rm -rf /tmp/go-mod /tmp/go-build /go/pkg/* /root/.cache/* /root/.config/go/telemetry

WORKDIR /app

RUN mkdir -p "${PLAYWRIGHT_BROWSERS_PATH}"
RUN playwright install chromium-headless-shell --with-deps
RUN rm -rf /opt/install /opt/install-scripts /tmp/*

WORKDIR /app
CMD [ "/bin/bash",  "-c", "sleep infinity" ]

# ----------------------------------------------------------------------------------------------------------------------
# Builder stage to create production artifacts.
# Used to build the final runner image.
# ----------------------------------------------------------------------------------------------------------------------

FROM dev-base AS builder

ARG BASE_URL=https://app.superplane.com
ARG VITE_ENABLE_CUSTOM_COMPONENTS=false

WORKDIR /app
COPY pkg /app/pkg
COPY cmd /app/cmd
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY db/migrations /app/db/migrations
COPY db/data_migrations /app/db/data_migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
COPY web_src /app/web_src
COPY protos /app/protos
COPY api/swagger /app/api/swagger
COPY rbac /app/rbac
COPY templates /app/templates
RUN rm -rf build && go build -o build/superplane cmd/server/main.go

WORKDIR /app/web_src
RUN npm install
RUN VITE_BASE_URL=$BASE_URL VITE_ENABLE_CUSTOM_COMPONENTS=$VITE_ENABLE_CUSTOM_COMPONENTS npm run build

# ----------------------------------------------------------------------------------------------------------------------
# Runner stage to run the application.
# Used as the final image.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${RUNNER_IMAGE} AS runner

LABEL org.opencontainers.image.title="superplane" \
  org.opencontainers.image.description="SuperPlane" \
  org.opencontainers.image.vendor="SuperPlane" \
  org.opencontainers.image.source="https://github.com/superplanehq/superplane" \
  org.opencontainers.image.url="https://superplane.com" \
  org.opencontainers.image.documentation="https://docs.superplane.com"

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
COPY --from=builder --chown=nobody:root /app/db/data_migrations /app/db/data_migrations
COPY --from=builder --chown=nobody:root /app/pkg/web/assets/dist /app/pkg/web/assets/dist
COPY --from=builder --chown=nobody:root /app/api/swagger /app/api/swagger
COPY --from=builder --chown=nobody:root /app/rbac /app/rbac
COPY --from=builder --chown=nobody:root /app/templates /app/templates

USER nobody

CMD ["bash", "/app/docker-entrypoint.sh"]