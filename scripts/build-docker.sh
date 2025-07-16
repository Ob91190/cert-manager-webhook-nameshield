#!/bin/bash

# Script to build and test the Docker image locally
# Usage: ./scripts/build-docker.sh [tag]

set -e

# Configuration
REGISTRY="registry.antemeta.io"
IMAGE_NAME="$REGISTRY/devops/cert-manager-webhook-nameshield"
TAG=${1:-"local-$(date +%Y%m%d-%H%M%S)"}
FULL_IMAGE="$IMAGE_NAME:$TAG"

echo "🏗️  Building Docker image: $FULL_IMAGE"

# Build the image
docker build -t "$FULL_IMAGE" .

echo "✅ Build completed successfully!"
echo "📦 Image: $FULL_IMAGE"

# Test the image
echo "🧪 Testing the image..."
if docker run --rm "$FULL_IMAGE" --help >/dev/null 2>&1; then
    echo "✅ Image test passed!"
else
    echo "❌ Image test failed!"
    exit 1
fi

# Show image details
echo "📊 Image details:"
docker images "$IMAGE_NAME" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"

echo ""
echo "🚀 To push the image:"
echo "   docker push $FULL_IMAGE"
echo ""
echo "🧹 To clean up:"
echo "   docker rmi $FULL_IMAGE"
