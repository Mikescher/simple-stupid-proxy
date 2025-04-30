# Stage 1: Build the application
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
# CGO_ENABLED=0 produces a statically linked binary
# -ldflags="-s -w" strips debug information and symbols to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /proxy main.go

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /proxy /app/proxy

# Expose the port the application runs on
EXPOSE 8080

# Set the entrypoint command
ENTRYPOINT ["/app/proxy"]

# Optional: Add metadata labels
LABEL maintainer="Your Name <your.email@example.com>"
LABEL description="Simple Go HTTP Proxy"
