#!/bin/bash

# Install Python dependencies
echo "Installing Python dependencies..."
pip3 install -r requirements.txt

# Build the Go application
echo "Building Go application..."
go build -o muambr-api .

echo "Build completed successfully!"