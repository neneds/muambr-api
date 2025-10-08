#!/bin/bash
set -e  # Exit on any error

echo "Starting build process for muambr-api..."

# Install Python dependencies
echo "Installing Python dependencies..."
if command -v pip3 &> /dev/null; then
    echo "Using pip3 to install dependencies..."
    pip3 install -r requirements.txt
elif command -v pip &> /dev/null; then
    echo "Using pip to install dependencies..."  
    pip install -r requirements.txt
else
    echo "Using python3 -m pip to install dependencies..."
    python3 -m pip install -r requirements.txt
fi

echo "Python dependencies installed successfully!"

# Install Go dependencies
echo "Installing Go dependencies..."
go mod download

# Build the Go application
echo "Building Go application..."
go build -o muambr-api .

echo "Build completed successfully!"
echo "Python and Go dependencies ready for extraction scripts."