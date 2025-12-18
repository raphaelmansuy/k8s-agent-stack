# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/agentstack-api \
    ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/agentstack-worker \
    ./cmd/worker

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 agentstack && \
    adduser -u 1000 -G agentstack -s /bin/sh -D agentstack

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/agentstack-api /app/agentstack-api
COPY --from=builder /app/bin/agentstack-worker /app/agentstack-worker

# Copy any static assets or configs if needed
# COPY --from=builder /app/config /app/config

# Change ownership
RUN chown -R agentstack:agentstack /app

# Switch to non-root user
USER agentstack

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/agentstack-api"]
