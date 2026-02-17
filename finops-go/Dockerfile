# Multi-stage build for finops-go binaries.
# Produces: worker-finops, api, cli, mcp-finops, shadow-compare

FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/worker-finops   ./cmd/worker-finops
RUN CGO_ENABLED=0 go build -o /out/api              ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/cli              ./cmd/cli
RUN CGO_ENABLED=0 go build -o /out/mcp-finops       ./cmd/mcp-finops
RUN CGO_ENABLED=0 go build -o /out/shadow-compare   ./cmd/shadow-compare

# Runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/ /usr/local/bin/

# Default to the worker binary; override with docker-compose command.
ENTRYPOINT ["worker-finops"]
