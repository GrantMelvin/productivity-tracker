#!/bin/bash

cd ../

# Build the Docker image
echo "Building the Docker image..."
sudo docker build -t prod .

# Run the Docker container
echo "Running the Docker container..."
docker run --rm --name "productivity_tracker" -p 80:8080 -t "prod"
