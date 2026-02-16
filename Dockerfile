# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for private deps
RUN apk add --no-cache git

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download deps
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o tam-backend ./cmd/backend

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create dirs for session and uploads
RUN mkdir -p /app/session /app/uploads && chown -R nobody:nobody /app/session /app/uploads

# Copy binary (migrations embedded via go:embed)
COPY --from=builder /app/tam-backend .

USER root

EXPOSE 8080

ENTRYPOINT ["./tam-backend"]
