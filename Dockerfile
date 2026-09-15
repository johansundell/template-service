# Build stage
FROM golang:1.27.1-alpine AS builder

WORKDIR /app

# Copy go mod and sum files for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Pure Go build (no CGO needed for ncruces/go-sqlite3)
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X 'main.Version=${VERSION}'" \
    -o template-service .

# Run stage
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies, create non-root user, and prepare app directory
RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup \
    && chown -R appuser:appgroup /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app/template-service .

# Copy assets and templates
COPY --from=builder --chown=appuser:appgroup /app/assets ./assets
COPY --from=builder --chown=appuser:appgroup /app/tmpl ./tmpl

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/ || exit 1

CMD ["./template-service"]
