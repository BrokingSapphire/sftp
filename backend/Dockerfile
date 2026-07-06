# Build context is the REPO ROOT (see docker-compose.yml) so the canonical
# brand.config.json can be baked in — bind-mounting a single file fails on some
# hosts (e.g. external/mounted drives).

# ── Build stage ──────────────────────────────────────────────
FROM golang:1.26-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" \
    -o /out/sftp_service ./cmd/server

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S sftp && adduser -S -G sftp sftp
WORKDIR /app

COPY --from=build /out/sftp_service /app/sftp_service
COPY backend/config.example.yaml /app/config.yaml
# Canonical white-label config (read at /app/brand.config.json).
COPY brand.config.json /app/brand.config.json

# Storage + logs mounted as volumes in production.
RUN mkdir -p /app/storage/files /app/storage/tmp /app/storage/logs && \
    chown -R sftp:sftp /app
USER sftp

EXPOSE 8080 2222
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/sftp_service"]
