# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN go build -o scheduler main.go

# Stage 2: Run in a slim image
FROM alpine:latest

WORKDIR /app

# Copy the binary and static files from builder
COPY --from=builder /app/scheduler .
COPY static/ ./static/

# Expose port 8080
EXPOSE 8080

# Run the binary
CMD ["./scheduler"]
