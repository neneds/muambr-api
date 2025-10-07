# Use official Go image as base
FROM golang:1.19-alpine AS builder

# Install Python and pip
RUN apk add --no-cache python3 py3-pip

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download Go dependencies
RUN go mod download

# Copy Python requirements
COPY requirements.txt ./

# Install Python dependencies
RUN pip3 install --no-cache-dir -r requirements.txt

# Copy source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o muambr-api .

# Start a new stage from Alpine
FROM alpine:latest

# Install Python3 and ca-certificates
RUN apk --no-cache add ca-certificates python3 py3-pip

# Set working directory
WORKDIR /root/

# Copy Python requirements and install them
COPY requirements.txt ./
RUN pip3 install --no-cache-dir -r requirements.txt

# Copy the binary from builder stage
COPY --from=builder /app/muambr-api .

# Copy Python extractors
COPY --from=builder /app/extractors ./extractors

# Set environment variables
ENV GIN_MODE=release
ENV PYTHON_PATH=python3

# Expose port
EXPOSE 8080

# Command to run
CMD ["./muambr-api"]