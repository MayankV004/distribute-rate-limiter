# Multi-stage build — keeps final image minimal (distroless)
# TODO (Phase 8): fill in once cmd/gateway builds successfully.

# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gateway ./cmd/gateway

# ── Final stage ──────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/gateway /gateway
COPY configs/gateway.yaml /configs/gateway.yaml
EXPOSE 8080 9090
ENTRYPOINT ["/gateway", "-config", "/configs/gateway.yaml"]
