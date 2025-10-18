#!/bin/bash
set -e  # Exit on any error

echo "Starting build process for muambr-api..."

# Install Go dependencies
echo "Installing Go dependencies..."
go mod download
go mod tidy

# Build the Go application
echo "Building Go application..."
go build -o muambr-api .

echo "Build completed successfully!"
echo "Pure Go application ready to run."