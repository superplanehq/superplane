# syntax=docker/dockerfile:1
# Context: repository root — docker build -f fleet-manager/Dockerfile .
# EC2 images must be linux/amd64: docker build --platform linux/amd64 …

FROM golang:1.23-bookworm AS build
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fleet-manager ./fleet-manager/cmd/fleet-manager

FROM gcr.io/distroless/static-debian12:nonroot

# JSON config lives at /etc/fleet-manager/config.json by default — bind-mount it in.
# Operators can override with -e FM_CONFIG_FILE=/some/other/path.
ENV FM_CONFIG_FILE=/etc/fleet-manager/config.json

COPY --from=build /out/fleet-manager /fleet-manager

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/fleet-manager"]
