FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o /build/proxy ./cmd/proxy

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/proxy .

# Expose port
EXPOSE 8090

# Run the proxy
ENTRYPOINT ["./proxy"]