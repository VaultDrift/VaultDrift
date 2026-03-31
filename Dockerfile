# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make nodejs npm

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build web UI
RUN cd web && npm install && npm run build && cd ..

# Build the server binary
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o vaultdrift-server ./cmd/vaultdrift

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for TLS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh vaultdrift

# Create data directory
RUN mkdir -p /data && chown -R vaultdrift:vaultdrift /data

# Copy binary from builder
COPY --from=builder /build/vaultdrift-server /usr/local/bin/vaultdrift-server

# Copy web assets
COPY --from=builder /build/web/dist /app/web/dist

# Switch to non-root user
USER vaultdrift

# Expose default port
EXPOSE 8443

# Volume for data persistence
VOLUME ["/data"]

# Entrypoint
ENTRYPOINT ["vaultdrift-server"]
CMD ["serve"]
