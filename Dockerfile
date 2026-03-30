# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -ldflags "-s -w" -o vaultdrift ./cmd/vaultdrift

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for TLS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh vaultdrift

# Create data directory
RUN mkdir -p /var/lib/vaultdrift && chown -R vaultdrift:vaultdrift /var/lib/vaultdrift

# Copy binary from builder
COPY --from=builder /build/vaultdrift /usr/local/bin/vaultdrift

# Switch to non-root user
USER vaultdrift

# Expose default port
EXPOSE 8443

# Volume for data persistence
VOLUME ["/var/lib/vaultdrift"]

# Entrypoint
ENTRYPOINT ["vaultdrift"]
CMD ["serve"]
