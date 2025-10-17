# Use official Go image as base
FROM golang:1.19-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download Go dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o muambr-api .

# Start a new stage from Alpine
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/muambr-api .

# Set environment variables
ENV GIN_MODE=release

# Expose port
EXPOSE 8080

# Command to run
CMD ["./muambr-api"]