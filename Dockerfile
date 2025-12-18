# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build CLI
RUN CGO_ENABLED=0 GOOS=linux go build -o scanner-cli ./cmd/scanner-cli/main.go

# Stage 2: Runtime
FROM alpine:3.18

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary
COPY --from=builder /app/scanner-cli .

# Set entrypoint
ENTRYPOINT ["./scanner-cli"]
