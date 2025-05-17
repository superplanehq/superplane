ARG GO_VERSION=1.23
ARG UBUNTU_VERSION=22.04
ARG BUILDER_IMAGE="golang:${GO_VERSION}"
ARG RUNNER_IMAGE="ubuntu:${UBUNTU_VERSION}"

FROM ${BUILDER_IMAGE} AS base

ARG APP_NAME
ENV APP_NAME=${APP_NAME}

RUN echo "Build of $APP_NAME started"

# Add PostgreSQL repository for version 17.5
RUN apt-get update -y && apt-get install --no-install-recommends -y ca-certificates unzip curl gnupg lsb-release libc-bin libc6 \
    && sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list' \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg \
    && apt-get update -y \
    && apt-get install --no-install-recommends -y postgresql-client-17 \
    && apt-get clean && rm -f /var/lib/apt/lists/*_*

# Install Node.js 22.4.1 directly from NodeSource repository
RUN apt-get update -y && apt-get install --no-install-recommends -y curl gnupg && \
    mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_22.x nodistro main" > /etc/apt/sources.list.d/nodesource.list && \
    apt-get update -y && \
    apt-get install -y nodejs && \
    apt-get clean && rm -f /var/lib/apt/lists/*_* && \
    node -v && \
    npm -v

WORKDIR /tmp
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz | tar xvz && \
    mv /tmp/migrate /usr/bin/migrate && \
    chmod +x /usr/bin/migrate

# Install protoc for the appropriate architecture
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then \
    curl -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.15.8/protoc-3.15.8-linux-aarch_64.zip -o protoc.zip; \
    else \
    curl -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.15.8/protoc-3.15.8-linux-x86_64.zip -o protoc.zip; \
    fi && \
    unzip protoc.zip && \
    mv bin/protoc /usr/local/bin/protoc && \
    rm -rf protoc.zip


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

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
RUN go install github.com/air-verse/air@latest

WORKDIR /app

FROM base AS dev

COPY test test

WORKDIR /app
RUN go install github.com/mgechev/revive@v1.8.0
RUN go install gotest.tools/gotestsum@v1.12.1

WORKDIR /app/web_src
RUN npm install
RUN npm run build

WORKDIR /app

CMD [ "/bin/bash",  "-c \"while sleep 1000; do :; done\"" ]

FROM base AS builder

RUN rm -rf build && go build -o build/${APP_NAME} cmd/main.go

FROM ${RUNNER_IMAGE} AS runner

# postgresql-client needs to be installed here too,
# otherwise the createdb command won't work.
# Install PostgreSQL 17.5 client tools
RUN apt-get update -y && apt-get install --no-install-recommends -y ca-certificates gnupg lsb-release \
    && sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list' \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg \
    && apt-get update -y \
    && apt-get install --no-install-recommends -y postgresql-client-17 \
    && apt-get clean && rm -f /var/lib/apt/lists/*_*

# We don't need Docker health checks, since these containers
# are intended to run in Kubernetes pods, which have probes.
HEALTHCHECK NONE

WORKDIR /app
RUN chown nobody /app

ARG APP_NAME
ENV APP_NAME=${APP_NAME}

# Only copy the binary, migrations, entrypoint, and a few CLIs needed for the app startup from the build stage
COPY --from=builder --chown=nobody:root /usr/bin/createdb /usr/bin/createdb
COPY --from=builder --chown=nobody:root /usr/bin/migrate /usr/bin/migrate
COPY --from=builder --chown=nobody:root /app/build/${APP_NAME} /app/build/${APP_NAME}
COPY --from=builder --chown=nobody:root /app/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY --from=builder --chown=nobody:root /app/db/migrations /app/db/migrations
COPY --from=builder --chown=nobody:root /app/web/assets/dist /app/web/assets/dist
COPY --from=builder --chown=nobody:root /app/api/swagger /app/api/swagger

USER nobody

CMD ["bash", "/app/docker-entrypoint.sh"]
