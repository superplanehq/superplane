# syntax=docker/dockerfile:1
# Context: repository root — docker build -f fleet-manager/Dockerfile .

FROM golang:1.22-bookworm AS build
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fleet-manager ./fleet-manager/cmd/fleet-manager

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /home/nonroot
ENV DATABASE_PATH=/home/nonroot/fleet.db

COPY --from=build /out/fleet-manager /fleet-manager

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/fleet-manager"]
