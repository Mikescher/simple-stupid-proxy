# Stage 1: Build the application
FROM golang:1-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
# CGO_ENABLED=0 produces a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /proxy './cmd/server'

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /proxy /app/proxy

# Set the entrypoint command
ENTRYPOINT ["/app/proxy"]
