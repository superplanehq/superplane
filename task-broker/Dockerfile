# syntax=docker/dockerfile:1
# Context: repository root — docker build -f task-broker/Dockerfile .

FROM golang:1.22-bookworm AS build
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/task-broker ./task-broker/cmd/task-broker

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /home/nonroot

COPY --from=build /out/task-broker /task-broker

USER nonroot:nonroot
EXPOSE 8081
ENTRYPOINT ["/task-broker"]
