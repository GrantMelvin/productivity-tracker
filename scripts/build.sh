#!/bin/bash

cd ../

# Build the Docker image
echo "Building the Docker image..."
sudo docker build -t prod .

# Run the Docker container
echo "Running the Docker container..."
sudo docker run --rm --name "productivity_tracker" -p 8080:8080 -v "./assets/data:/app/assets/data" -t "prod"
