FROM --platform=$BUILDPLATFORM golang:1.26.2-bookworm AS build

ARG TARGETOS
ARG TARGETARCH
ARG FLEET_MANAGER_VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/fleetmanager ./cmd/fleetmanager
COPY pkg/fleets ./pkg/fleets
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
  -ldflags="-s -w -X main.Version=${FLEET_MANAGER_VERSION}" \
  -o /fleet-manager ./cmd/fleetmanager

FROM scratch AS artifact
COPY --from=build /fleet-manager /fleet-manager

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
ARG FLEET_MANAGER_VERSION=dev
ARG SOURCE_REVISION=unknown
LABEL org.opencontainers.image.source="https://github.com/superplanehq/superplane" \
  org.opencontainers.image.revision="${SOURCE_REVISION}" \
  org.opencontainers.image.version="${FLEET_MANAGER_VERSION}"
COPY --from=build /fleet-manager /usr/local/bin/fleet-manager
ENTRYPOINT ["/usr/local/bin/fleet-manager"]

FROM docker:27-cli AS local
RUN apk add --no-cache postgresql-client
COPY --from=build /fleet-manager /usr/local/bin/fleet-manager
COPY --chmod=755 scripts/entrypoint.local.sh /usr/local/bin/fleet-manager-entrypoint
ENTRYPOINT ["/usr/local/bin/fleet-manager-entrypoint"]
