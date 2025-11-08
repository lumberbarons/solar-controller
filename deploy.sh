#!/bin/bash

set -e  # Exit on any error

echo "Building ARM64 binary..."
make build-linux-arm64

echo "Copying binary to remote server..."
scp bin/solar-controller-linux-arm64 jim@100.90.141.9:/home/jim/solar-controller

echo "Installing and restarting service on remote server..."
ssh jim@100.90.141.9 'sudo chown root:root solar-controller && sudo mv solar-controller /usr/bin && sudo service solar-controller restart'

echo "Deployment complete!"
