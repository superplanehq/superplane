FROM golang:1.26.2-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/fleetmanager ./cmd/fleetmanager
COPY pkg/fleets ./pkg/fleets
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /fleet-manager ./cmd/fleetmanager

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /fleet-manager /usr/local/bin/fleet-manager
ENTRYPOINT ["/usr/local/bin/fleet-manager"]

FROM docker:27-cli AS local
RUN apk add --no-cache postgresql-client
COPY --from=build /fleet-manager /usr/local/bin/fleet-manager
COPY --chmod=755 scripts/entrypoint.local.sh /usr/local/bin/fleet-manager-entrypoint
ENTRYPOINT ["/usr/local/bin/fleet-manager-entrypoint"]
