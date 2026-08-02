# Multi-stage build — keeps final image minimal (distroless)
# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gateway ./cmd/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/backend ./cmd/backend

# ── Final stage ──────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/gateway /gateway
COPY --from=builder /bin/backend /backend
COPY configs/gateway.yaml /configs/gateway.yaml
EXPOSE 8080 9090
ENTRYPOINT ["/gateway", "-config", "/configs/gateway.yaml"]
