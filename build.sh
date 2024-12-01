#!/bin/bash

# Build the Docker image
echo "Building the Docker image..."
docker build -t prod .

# Run the Docker container
echo "Running the Docker container..."
docker run --rm --name "productivity_tracker" -p 8080:8080 -t "prod"