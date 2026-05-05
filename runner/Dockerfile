# syntax=docker/dockerfile:1
# Context: repository root — docker build -f runner/Dockerfile .
#
# Docker execution tasks need the daemon socket mounted, e.g. -v /var/run/docker.sock:/var/run/docker.sock
# Default user is root so the client can use a typical docker.sock permission setup.

FROM golang:1.22-alpine AS build
RUN apk add --no-cache git
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/runner ./runner/cmd/runner

FROM alpine:3.20
RUN apk add --no-cache ca-certificates docker-cli

COPY --from=build /out/runner /usr/local/bin/runner

ENTRYPOINT ["/usr/local/bin/runner"]
