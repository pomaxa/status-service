# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version info
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o status-incident .

# Run stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies for SQLite
RUN apk add --no-cache ca-certificates tzdata

# Copy binary and assets
COPY --from=builder /app/status-incident .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Create data directory for SQLite
RUN mkdir -p /app/data

EXPOSE 8080

# Set environment variables
ENV TZ=UTC

CMD ["./status-incident"]
