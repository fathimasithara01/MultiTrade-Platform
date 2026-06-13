# ─────────────────────────────────────────────────────────────────────────────
# Stage 1: Build
# Uses the official Go image to compile a statically-linked binary.
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

# Install build essentials
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependency downloads separately from source
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" \
    -o /app/bin/multitrade ./cmd/api

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2: Runtime
# Minimal distroless image — no shell, no package manager, minimal attack surface.
# ─────────────────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

# Copy timezone data and CA certs from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy migrations alongside the binary so the server can auto-migrate
COPY --from=builder /app/bin/multitrade /multitrade
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/multitrade"]
